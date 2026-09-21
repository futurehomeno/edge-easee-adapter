package signalr

import (
	"errors"
	"math"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/alarm"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	"github.com/futurehomeno/cliffhanger/types"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"

	"github.com/futurehomeno/edge-easee-adapter/internal/cache"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/db"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

// Handler handles signalr observations for one charger.
type Handler interface {
	IsOnline() bool
	HandleObservation(observation model.Observation) error
	// Close drains the pending reports and stops the sender. Must be called once the handler
	// is unregistered, or its goroutine outlives the thing it reports for.
	Close()
}

// PhaseStore persists the phase the charger reported using, so the advertised phase mode
// survives a restart. Implemented in the easee package, which owns the thing state.
type PhaseStore interface {
	OutputPhase() types.PhaseMode
	SetOutputPhase(types.PhaseMode) error
}

type observationsHandler struct {
	cache          cache.Cache
	handlers       map[model.ObservationID]func(model.Observation) error
	thing          adapter.Thing
	energyHandler  *energyHandler
	sessionStorage db.ChargingSessionStorage
	chargerID      string
	storedObs      map[model.ObservationID]model.Observation
	phaseStore     PhaseStore

	reports  chan func() error
	senderWG sync.WaitGroup

	// closed guards the send in enqueue: the manager drops a charger from its map and closes
	// the handler without holding the lock an in-flight observation was looked up under, so a
	// dispatch can still reach enqueue after Close and must not send on a closed channel.
	closeMu sync.Mutex
	closed  bool

	isCloudOnline atomic.Bool
	isStateOnline atomic.Bool
	stateSeq      atomic.Uint64
}

func NewObservationsHandler(
	thing adapter.Thing,
	cache cache.Cache,
	confSrv *config.Service,
	sessionStorage db.ChargingSessionStorage,
	chargerID string,
	phaseStore PhaseStore,
) (Handler, error) {
	handler := observationsHandler{
		cache:          cache,
		thing:          thing,
		energyHandler:  newEnergyHandler(cache, thing, confSrv),
		sessionStorage: sessionStorage,
		chargerID:      chargerID,
		storedObs:      make(map[model.ObservationID]model.Observation),
		phaseStore:     phaseStore,
		reports:        make(chan func() error, reportQueueSize),
	}

	handler.isCloudOnline.Store(true)
	handler.isStateOnline.Store(true)

	handler.senderWG.Add(1)

	go handler.runSender()

	handler.handlers = map[model.ObservationID]func(model.Observation) error{
		model.DetectedPowerGridType: handler.handleDetectedPowerGridType,
		model.PhaseMode:             handler.handlePhaseMode,
		model.MaxChargerCurrent:     handler.handleMaxChargerCurrent,
		model.DynamicChargerCurrent: handler.handleDynamicChargerCurrent,
		model.ChargerOPState:        handler.handleChargerState,
		model.OutputPhase:           handler.handleOutPhase,
		model.TotalPower:            handler.handleTotalPower,
		model.LifetimeEnergy:        handler.energyHandler.handle,
		model.EnergySession:         handler.handleEnergySession,
		model.InCurrentT3:           handler.handlePhaseCurrent(cache.SetPhase1Current, "i1", numericmeter.ValueCurrentPhase1),
		model.InCurrentT4:           handler.handlePhaseCurrent(cache.SetPhase2Current, "i2", numericmeter.ValueCurrentPhase2),
		model.InCurrentT5:           handler.handlePhaseCurrent(cache.SetPhase3Current, "i3", numericmeter.ValueCurrentPhase3),
		model.CloudConnected:        handler.handleCloudConnected,
		model.CableLocked:           handler.handleCableLocked,
		model.CableRating:           handler.handleCableRating,
		model.LockCablePermanently:  handler.handleLockCablePermanently,
		model.ChargingSessionStop:   handler.handleChargingSessionStop,
		model.ChargingSessionStart:  handler.handleChargingSessionStart,
		model.ErrorCode:             handler.handleErrorCode,
	}

	return &handler, nil
}

// reportQueueSize bounds the reports waiting on a service lock. A command holds that lock for
// up to CurrentWaitDuration (3s), and a charger streams a handful of observations per second,
// so this absorbs the worst stall without letting an unresponsive service grow the queue.
const reportQueueSize = 64

// enqueue hands a report to the sender goroutine instead of publishing it here. Every report
// takes the service lock, and a chargepoint command holds that lock while it waits up to
// CurrentWaitDuration for its echo observation - an observation this goroutine is the only
// drainer of. Sending inline parks the drain behind the command, so the echo never arrives,
// the command times out and reports a success as a failure, and meanwhile every charger's
// observations stall. One sender rather than a goroutine per send keeps the reports in the
// order the observations arrived: a stale state published after a fresh one is its own defect.
func (h *observationsHandler) enqueue(label string, send func() error) bool {
	h.closeMu.Lock()
	defer h.closeMu.Unlock()

	if h.closed {
		return false
	}

	select {
	case h.reports <- send:
		return true
	default:
		log.Warnf("[%s] Report queue full, dropping %s report", h.chargerID, label)

		return false
	}
}

func (h *observationsHandler) runSender() {
	defer h.senderWG.Done()

	for send := range h.reports {
		if err := send(); err != nil {
			log.Warnf("[%s] Send report err: %v", h.chargerID, err)
		}
	}
}

// Close stops every goroutine that can publish on the thing - the report sender once the reports
// already queued have gone out, and the lifetime-energy timer - and waits for both. Safe to call
// more than once, and safe against an observation still being dispatched concurrently.
func (h *observationsHandler) Close() {
	h.closeMu.Lock()

	if !h.closed {
		h.closed = true

		close(h.reports)
	}

	h.closeMu.Unlock()

	// Stopped before the drain, not after: the lifetime-energy goroutine publishes straight on
	// the thing rather than through the report queue, so closing that queue does not reach it,
	// and a queued report blocked on the service lock holds the wait below open for as long as
	// the lock is held - long enough for the timer to fire and publish on a thing being torn down.
	h.energyHandler.close()

	h.senderWG.Wait()
}

func (h *observationsHandler) IsOnline() bool {
	return h.isCloudOnline.Load() && h.isStateOnline.Load()
}

func (h *observationsHandler) HandleObservation(observation model.Observation) error {
	if prev, ok := h.storedObs[observation.ID]; !ok || prev.Value != observation.Value {
		if log.IsLevelEnabled(log.TraceLevel) {
			log.Trace(observation.Str())
		}

		h.storedObs[observation.ID] = observation
	}

	if handler, ok := h.handlers[observation.ID]; ok {
		return handler(observation)
	}

	return errors.New("not supported")
}

func (h *observationsHandler) handlePhaseMode(observation model.Observation) error {
	val, err := observation.IntValue()
	if err != nil {
		return err
	}

	log.Debugf("[%s] Phase mode=%s (%d)", h.chargerID, model.PhaseModeName(val), val)

	phaseMode, _ := h.cache.PhaseMode()

	if val == phaseMode {
		return nil
	}

	ok := h.cache.SetPhaseMode(val, observation.Timestamp)
	if !ok {
		return nil
	}

	// sup_phase_modes covers everything the charger can be switched to, so the internal mode
	// no longer moves it - only the mode the charger currently reports can change.
	chargepointSrv, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	h.enqueue("phase mode", func() error {
		_, err := chargepointSrv.SendPhaseModeReport(false)

		return err
	})

	return nil
}

func (h *observationsHandler) handleMaxChargerCurrent(observation model.Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	log.Debugf("[%s] Max current=%.1f", h.chargerID, val)

	ok := h.cache.SetMaxCurrent(int(math.Round(val)), observation.Timestamp)
	if !ok {
		return nil
	}

	chargepointSrv, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	h.enqueue("max current", func() error {
		_, err := chargepointSrv.SendMaxCurrentReport(false)

		return err
	})

	return nil
}

func (h *observationsHandler) handleCloudConnected(observation model.Observation) error {
	val, err := observation.BoolValue()
	if err != nil {
		return err
	}

	if was := h.isCloudOnline.Swap(val); was && !val {
		log.Warnf("[%s] Disconnected from cloud", h.chargerID)
	} else if !was && val {
		log.Infof("[%s] Connected to cloud", h.chargerID)
	}

	return nil
}

func (h *observationsHandler) handleDynamicChargerCurrent(observation model.Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	roundedVal := int(math.Round(val))
	if preVal, _ := h.cache.OfferedCurrent(); preVal != roundedVal {
		log.Infof("[%s] Offered current=%d->%d", h.chargerID, preVal, roundedVal)
	} else {
		log.Debugf("[%s] Offered current=%d", h.chargerID, roundedVal)
	}

	ok := h.cache.SetOfferedCurrent(roundedVal, observation.Timestamp)
	if !ok {
		return nil
	}

	chargepointSrv, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	h.enqueue("current session", func() error {
		_, err := chargepointSrv.SendCurrentSessionReport(false)

		return err
	})

	return nil
}

func (h *observationsHandler) handleCableLocked(observation model.Observation) error {
	val, err := observation.BoolValue()
	if err != nil {
		return err
	}

	log.Debugf("[%s] Cable locked=%t", h.chargerID, val)

	ok := h.cache.SetCableLocked(val, observation.Timestamp)
	if !ok {
		return nil
	}

	chargepointSrv, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	h.enqueue("cable lock", func() error {
		_, err := chargepointSrv.SendCableLockReport(false)

		return err
	})

	return nil
}

func (h *observationsHandler) handleCableRating(observation model.Observation) error {
	// Easee sends observation 104 as integer or double depending on the charger, so the
	// strict integer accessor silently dropped half of them.
	val, err := observation.NumericIntValue()
	if err != nil {
		return err
	}

	log.Debugf("[%s] Cable=%dA", h.chargerID, val)

	ok := h.cache.SetCableCurrent(val, observation.Timestamp)
	if !ok {
		return nil
	}

	chargepointSrv, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	h.enqueue("cable lock", func() error {
		_, err := chargepointSrv.SendCableLockReport(false)

		return err
	})

	return nil
}

func (h *observationsHandler) handleChargerState(observation model.Observation) error {
	val, err := observation.IntValue()
	if err != nil {
		return err
	}

	state := model.ChargerState(val)

	if prevState, _ := h.cache.ChargerState(); prevState != state.ToFimpState() {
		log.Infof("[%s] State=%s", h.chargerID, state.Str())
	} else {
		log.Debugf("[%s] State=%s", h.chargerID, state.Str())
	}

	ok := h.cache.SetChargerState(state.ToFimpState(), observation.Timestamp)
	if !ok {
		return nil
	}

	if state.IsSessionFinished() {
		h.cache.SetRequestedOfferedCurrent(0, time.Now())
	}

	chargepointSrv, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	// The report has to cross the controller's checkConnection, which rejects it as
	// ChargerOffline while IsOnline reads false - so the flag has to be on the online side of
	// the send in both directions. Going offline: send first, or the report announcing it is
	// the one report never sent. Coming back: flip first, or the recovery report is refused by
	// the offline state it is there to clear. Both stores stay with the send rather than here,
	// or the flag would move while the report is still queued behind an earlier one. A
	// transition overtaken while it waited leaves the flag to the newer one, and a dropped one
	// moves it here: nothing is left to order it against, and losing the flip would gate every
	// later command on a state the charger has left.
	goingOffline := state == model.ChargerStateOffline
	seq := h.stateSeq.Add(1)

	queued := h.enqueue("state", func() error {
		if !goingOffline && seq == h.stateSeq.Load() {
			h.isStateOnline.Store(true)
		}

		_, err := chargepointSrv.SendStateReport(false)

		if goingOffline && seq == h.stateSeq.Load() {
			h.isStateOnline.Store(false)
		}

		return err
	})

	if !queued {
		h.isStateOnline.Store(!goingOffline)
	}

	return nil
}

func (h *observationsHandler) handleTotalPower(observation model.Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	log.Debugf("[%s] TotalPower=%.2fkW", h.chargerID, val)

	ok := h.cache.SetTotalPower(val*1000, observation.Timestamp)
	if !ok {
		return nil
	}

	meterElecSrv, err := getMeterElecService(h.thing)
	if err != nil {
		return err
	}

	h.enqueue("total power", func() error {
		if _, err := meterElecSrv.SendMeterReport(numericmeter.UnitW, false); err != nil {
			return err
		}

		_, err := meterElecSrv.SendMeterExtendedReport(numericmeter.Values{numericmeter.ValuePowerImport}, false)

		return err
	})

	return nil
}

func (h *observationsHandler) handleEnergySession(observation model.Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	log.Debugf("[%s] EnergySession=%.1f", h.chargerID, val)

	ok := h.cache.SetEnergySession(val, observation.Timestamp)
	if !ok {
		return nil
	}

	chargepointSrv, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	h.enqueue("current session", func() error {
		_, err := chargepointSrv.SendCurrentSessionReport(false)

		return err
	})

	return nil
}

func (h *observationsHandler) handlePhaseCurrent(
	set func(float64, time.Time) bool, label string, value numericmeter.Value,
) func(model.Observation) error {
	return func(observation model.Observation) error {
		val, err := observation.Float64Value()
		if err != nil {
			return err
		}

		if !set(val, observation.Timestamp) {
			return nil
		}

		log.Debugf("[%s] %s=%.1f", h.chargerID, label, val)

		meterElecSrv, err := getMeterElecService(h.thing)
		if err != nil {
			return err
		}

		h.enqueue(label, func() error {
			_, err := meterElecSrv.SendMeterExtendedReport(numericmeter.Values{value}, false)

			return err
		})

		return nil
	}
}

func (h *observationsHandler) handleOutPhase(observation model.Observation) error {
	val, err := observation.IntValue()
	if err != nil {
		return err
	}

	outPhaseType := model.OutputPhaseType(val).ToFimpState()

	// Charger sets outPhaseType parameter to "" if charger not charging, even if it has ongoing charging session.
	if outPhaseType == "" {
		return nil
	}

	ok := h.cache.SetOutputPhaseType(outPhaseType, observation.Timestamp)
	if !ok {
		return nil
	}

	// Logged past the accept check: an outdated replay - the whole observation set arrives
	// again on every reconnect - is not a phase change and costs nothing to skip.
	log.Infof("[%s] PhaseMode=%s", h.chargerID, outPhaseType)

	// Props first: the hub matches the reported mode against the sup_phase_modes it currently
	// holds, so a report naming a leg the old list does not advertise is discarded.
	if err := h.persistOutputPhase(outPhaseType); err != nil {
		return err
	}

	chargepointSrv, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	h.enqueue("phase mode", func() error {
		_, err := chargepointSrv.SendPhaseModeReport(false)

		return err
	})

	return nil
}

// persistOutputPhase stores the phase the charger reported and re-advertises sup_phase_modes if
// that changes the list - the hub otherwise keeps requesting a phase the charger cannot use.
func (h *observationsHandler) persistOutputPhase(outPhaseType types.PhaseMode) error {
	if h.phaseStore == nil || outPhaseType.EffectivePhasesCnt() != 1 {
		return nil
	}

	stored := h.phaseStore.OutputPhase()
	if stored == outPhaseType {
		return nil
	}

	gridType, _ := h.cache.GridType()
	phases, _ := h.cache.Phases()

	before := model.AdvertisedPhaseModes(gridType, phases, stored)
	after := model.AdvertisedPhaseModes(gridType, phases, outPhaseType)

	// Persisted only after a successful republish, so a failed one is retried on the next
	// observation. The republish is queued, so the write goes with it rather than happening here.
	if !slices.Equal(before, after) {
		return h.republishChargepointProps(
			map[string]any{chargepoint.PropertySupportedPhaseModes: after},
			func() error { return h.phaseStore.SetOutputPhase(outPhaseType) },
		)
	}

	return h.phaseStore.SetOutputPhase(outPhaseType)
}

func (h *observationsHandler) handleDetectedPowerGridType(observation model.Observation) error {
	val, err := observation.IntValue()
	if err != nil {
		return err
	}

	gridType, _ := h.cache.GridType()
	phases, _ := h.cache.Phases()

	supportedGridType, supportedPhases := model.GridType(val).ToFimpGridType()

	// Several raw grid types map onto the same FIMP pair, so faults must be reported
	// before the equivalence check below short-circuits an otherwise unchanged topology.
	// Logged rather than returned: the alarm is reportable while the charger is offline, where
	// the send is refused, and abandoning here would drop the grid-type update with it - the
	// topology change is the more durable of the two, and nothing republishes it afterwards.
	if err := h.sendAlarmReports(map[string]bool{
		alarm.EventGroundingFault: model.GridType(val).IsGroundFault(),
		alarm.EventGridTypeFault:  model.GridType(val).IsWiringFault(),
	}, observation.Timestamp); err != nil {
		log.Warnf("[%s] Send grid type alarm reports err: %v", h.chargerID, err)
	}

	// Zero phases is the absence of a topology, not a new one: an unknown grid type yields
	// ("", 0) and the two error types yield (TN, 0) / (IT, 0). Republishing any of them deletes
	// phases and sup_phase_modes - both empty - so cliffhanger rejects every cmd.phase_mode.set
	// and energy guard drops the charger from phase balancing, and the equivalence check below
	// short-circuits every repeat, so nothing restores them while the fault persists. Keeping
	// the last known topology leaves the recovery observation a genuine change. The alarm above
	// is the report for this case.
	if supportedPhases == 0 {
		return nil
	}

	if supportedGridType == gridType && supportedPhases == phases {
		return nil
	}

	log.Debugf("[%s] supGridType=%v supPh=%v", h.chargerID, supportedGridType, supportedPhases)

	outputPhase := types.PhaseMode("")
	if h.phaseStore != nil {
		outputPhase = h.phaseStore.OutputPhase()
	}

	// The cache write is deferred to onPublished, which runs only after the inclusion report has
	// landed. Committing it up front made a dropped or failed republish permanent: the equality
	// check above reads the cache, so every repeat observation short-circuited and nothing
	// re-advertised the topology until the next restart. persistOutputPhase gates its own write
	// the same way.
	return h.republishChargepointProps(map[string]any{
		chargepoint.PropertyGridType:            supportedGridType,
		chargepoint.PropertyPhases:              supportedPhases,
		chargepoint.PropertySupportedPhaseModes: model.AdvertisedPhaseModes(supportedGridType, supportedPhases, outputPhase),
	}, func() error {
		h.cache.SetInstallationParameters(supportedGridType, supportedPhases, observation.Timestamp)

		return nil
	})
}

// republishChargepointProps updates props only. The phase-mode interfaces are derived from
// sup_phase_modes once, in chargepoint.NewService, so a charger created without a grid type -
// stored state that is empty or legacy - gains the property here but not the interfaces until it
// is rebuilt on the next adapter restart. Recreating the service would need the controller here.
func (h *observationsHandler) republishChargepointProps(props map[string]any, onPublished func() error) error {
	service, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	// Replaces the map rather than mutating the one in place: Specification() hands out the
	// live *fimptype.Service, and the chargepoint service reads these props (PropertyStrings,
	// PropertyInteger) while handling a command on another goroutine. Building the replacement
	// first and assigning it makes the props the router sees either the old set or the new one,
	// never a map being written as it is read.
	//
	// Known limitation: if a second republish call lands before this job runs, it reassigns
	// Specification().Props first, and this job's Update/onPublished then run against that
	// second call's props rather than its own. Snapshotting the value at enqueue time and
	// restoring it here would close that window, but it re-introduces a second unsynchronised
	// write racing the synchronous read the rest of the package relies on (see
	// TestObservationsHandler_OutputPhaseNarrowsAdvertisedPhaseModes) - not worth it for a
	// same-charger, back-to-back topology change this narrow.
	service.Specification().Props = chargepointPropsUpdate(service, props)

	// The props assignment above stays on the drain - it is what every later report reads - but
	// the republish waits on MQTT, and the drain is the only drainer of the observation channel.
	// Parking it there fills the observation buffer, and the session start/stop pair is
	// edge-triggered, so a dropped one loses the session's record for good.
	//
	// onPublished runs only after the report lands, on the sender goroutine. persistOutputPhase
	// relies on that: a phase written before a failed republish would never be retried, leaving
	// the charger advertising a phase mode it cannot use until the next restart.
	h.enqueue("inclusion", func() error {
		if err := h.thing.Update(adapter.ThingUpdateRemoveService(service), adapter.ThingUpdateAddService(service)); err != nil {
			return err
		}

		if _, err := h.thing.SendInclusionReport(false); err != nil {
			return err
		}

		if onPublished == nil {
			return nil
		}

		return onPublished()
	})

	// A dropped report is retried: onPublished doesn't run, so persistOutputPhase's caller never
	// marks the phase as persisted, and enqueue already logged the drop.
	return nil
}

func (h *observationsHandler) handleLockCablePermanently(observation model.Observation) error {
	val, err := observation.BoolValue()
	if err != nil {
		return err
	}

	ok := h.cache.SetCableAlwaysLocked(val, observation.Timestamp)
	if !ok {
		return nil
	}

	log.Debugf("[%s] cableAlwaysLock=%t", h.chargerID, val)

	parameterSrv, err := getParametersService(h.thing)
	if err != nil {
		return err
	}

	h.enqueue("parameter", func() error {
		_, err := parameterSrv.SendParameterReport(model.CableAlwaysLockedParameter, true)

		return err
	})

	return nil
}

func (h *observationsHandler) handleChargingSessionStop(observation model.Observation) error {
	var chargingSession model.StopChargingSession

	err := observation.JSONValue(&chargingSession)
	if err != nil {
		return err
	}

	chargepointSrv, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	log.Infof("[%s] Stop session %v", h.chargerID, chargingSession)

	err = h.sessionStorage.RegisterSessionStop(h.chargerID, chargingSession)
	if err != nil {
		return err
	}

	h.enqueue("current session", func() error {
		_, err := chargepointSrv.SendCurrentSessionReport(false)

		return err
	})

	return nil
}

func (h *observationsHandler) handleChargingSessionStart(observation model.Observation) error {
	var chargingSession model.StartChargingSession

	err := observation.JSONValue(&chargingSession)
	if err != nil {
		return err
	}

	log.Infof("[%s] Start session %v", h.chargerID, chargingSession)

	err = h.sessionStorage.RegisterSessionStart(h.chargerID, chargingSession)
	if err != nil {
		return err
	}

	chargepointSrv, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	h.enqueue("current session", func() error {
		_, err := chargepointSrv.SendCurrentSessionReport(false)

		return err
	})

	return nil
}

// chargepointPropsUpdate returns the props the service should carry once props is applied,
// leaving the live specification untouched. Specification() hands out the *fimptype.Service the
// router reads while handling commands, so assigning to its Props here would be an
// unsynchronised write to a map another goroutine may be ranging over.
func chargepointPropsUpdate(srv chargepoint.Service, props map[string]interface{}) map[string]interface{} {
	spec := srv.Specification()
	updated := make(map[string]interface{}, len(spec.Props)+len(props))

	for k, v := range spec.Props {
		updated[k] = v
	}

	for k, v := range props {
		if funk.IsEmpty(v) {
			delete(updated, k)

			continue
		}

		updated[k] = v
	}

	return updated
}

type energyHandler struct {
	cache                 cache.Cache
	thing                 adapter.Thing
	lock                  sync.Mutex
	confSrv               *config.Service
	energyObservationChan chan model.Observation

	// done is closed by the handler's Close. The goroutine selects on it, so a teardown stops it
	// before its timer fires rather than leaving it to publish on a thing that no longer exists.
	done   chan struct{}
	closed bool
	wg     sync.WaitGroup
}

func newEnergyHandler(cache cache.Cache, thing adapter.Thing, confSrv *config.Service) *energyHandler {
	return &energyHandler{
		cache:   cache,
		thing:   thing,
		confSrv: confSrv,
		done:    make(chan struct{}),
	}
}

// close stops the energy goroutine and waits for it. Safe to call more than once.
func (h *energyHandler) close() {
	h.lock.Lock()

	if !h.closed {
		h.closed = true

		close(h.done)
	}

	h.lock.Unlock()

	h.wg.Wait()
}

func (h *energyHandler) handle(observation model.Observation) error {
	observationTime := observation.Timestamp.Truncate(time.Hour)
	_, lastReadingTime := h.cache.LifetimeEnergy()
	lastReadingTime = lastReadingTime.Truncate(time.Hour)

	if !observationTime.After(lastReadingTime) {
		return nil
	}

	h.lock.Lock()

	if h.closed {
		h.lock.Unlock()

		return nil
	}

	if h.energyObservationChan == nil {
		h.energyObservationChan = make(chan model.Observation, 10)

		h.wg.Add(1)

		go h.manageEnergyObservation(h.energyObservationChan)
	}
	ch := h.energyObservationChan
	h.lock.Unlock()

	select {
	case ch <- observation:
	default:
		log.WithField("thing_address", h.thing.Address()).
			Warn("lifetime energy handler: observation buffer full, dropping observation")
	}

	return nil
}

func (h *energyHandler) manageEnergyObservation(ch chan model.Observation) {
	defer h.wg.Done()
	defer func() {
		h.lock.Lock()
		defer h.lock.Unlock()

		if h.energyObservationChan == ch {
			h.energyObservationChan = nil
		}
	}()

	timer := time.NewTimer(h.confSrv.EnergyLifetimeInterval())
	defer timer.Stop()

	var (
		energy   float64
		energyAt time.Time
	)

	for {
		select {
		case <-h.done:
			// Unregister closed the handler: the thing this would publish on is being torn down.
			return

		case val := <-ch:
			v, err := val.Float64Value()
			if err != nil {
				log.Warnf("[%s] Lifetime energy observation parse err: %v", val.ChargerID, err)

				continue
			}

			if val.Timestamp.Before(energyAt) {
				continue
			}

			energy = v
			energyAt = val.Timestamp

		case <-timer.C:
			h.cache.SetLifetimeEnergy(energy, energyAt)

			meterElecSrv, err := getMeterElecService(h.thing)
			if err != nil {
				log.WithField("thing_address", h.thing.Address()).
					WithError(err).
					Error("lifetime energy handler: failed to get meter elec service")

				return
			}

			_, err = meterElecSrv.SendMeterReport(numericmeter.UnitKWh, false)
			if err != nil {
				log.WithField("thing_address", h.thing.Address()).
					WithError(err).
					Error("lifetime energy handler: failed to send meter report")

				return
			}

			_, err = meterElecSrv.SendMeterExtendedReport(numericmeter.Values{numericmeter.ValueEnergyImport}, false)
			if err != nil {
				log.WithField("thing_address", h.thing.Address()).
					WithError(err).
					Error("lifetime energy handler: failed to send meter extend report")

				return
			}

			return
		}
	}
}

// handleErrorCode turns the charger fault code into an alarm. Easee does not document the
// individual codes, so any non-zero code is reported as the generic charger error.
func (h *observationsHandler) handleErrorCode(observation model.Observation) error {
	val, err := observation.IntValue()
	if err != nil {
		return err
	}

	// Only the transition into a fault is worth a warning; a persistent fault is replayed
	// in full on every reconnect, and the raw value is still traced by HandleObservation.
	if val != 0 && !h.cache.AlarmActive(alarm.EventOtherChargeErr) {
		log.Warnf("[%s] ErrorCode=%d", h.chargerID, val)
	}

	return h.sendAlarmReports(map[string]bool{alarm.EventOtherChargeErr: val != 0}, observation.Timestamp)
}

// sendAlarmReports stores the state of each event and reports the ones that changed.
// Dedup is left to the service's reporting cache, which only records an event once it is
// actually published - so a failed publish is retried on the next observation.
//
// The cache write stays on the drain, so the stored state keeps the order the observations
// arrived in; only the publish is queued, because it takes the service lock a chargepoint
// command can hold for the whole of its echo wait.
func (h *observationsHandler) sendAlarmReports(events map[string]bool, timestamp time.Time) error {
	service, err := getAlarmService(h.thing)
	if err != nil {
		return err
	}

	for event, active := range events {
		if !h.cache.SetAlarm(event, active, timestamp) {
			continue
		}

		h.enqueue("alarm", func() error {
			_, err := service.SendAlarmReport(event, false)

			return err
		})
	}

	return nil
}

// getService finds the thing's service of type S. The label spells the error out because the
// FIMP service name and the name used in the message differ (meter_elec vs meterelec).
func getService[S adapter.Service](thing adapter.Thing, name fimptype.ServiceNameT, label string) (S, error) {
	for _, service := range thing.Services(name) {
		if service, ok := service.(S); ok {
			return service, nil
		}
	}

	var zero S

	return zero, errors.New("there are no " + label + " services")
}

func getAlarmService(thing adapter.Thing) (alarm.Service, error) {
	return getService[alarm.Service](thing, alarm.AlarmSystem, "alarm")
}

func getParametersService(thing adapter.Thing) (parameters.Service, error) {
	return getService[parameters.Service](thing, parameters.Parameters, "parameters")
}

func getChargepointService(thing adapter.Thing) (chargepoint.Service, error) {
	return getService[chargepoint.Service](thing, chargepoint.Chargepoint, "chargepoint")
}

func getMeterElecService(thing adapter.Thing) (numericmeter.Service, error) {
	return getService[numericmeter.Service](thing, numericmeter.MeterElec, "meterelec")
}
