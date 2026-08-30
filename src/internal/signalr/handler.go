package signalr

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/alarm"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
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
}

type observationsHandler struct {
	cache          cache.Cache
	handlers       map[model.ObservationID]func(model.Observation) error
	thing          adapter.Thing
	energyHandler  *energyHandler
	sessionStorage db.ChargingSessionStorage
	chargerID      string
	storedObs      map[model.ObservationID]model.Observation

	isCloudOnline atomic.Bool
	isStateOnline atomic.Bool
}

func NewObservationsHandler(
	thing adapter.Thing,
	cache cache.Cache,
	confSrv *config.Service,
	sessionStorage db.ChargingSessionStorage,
	chargerID string,
) (Handler, error) {
	handler := observationsHandler{
		cache:          cache,
		thing:          thing,
		energyHandler:  newEnergyHandler(cache, thing, confSrv),
		sessionStorage: sessionStorage,
		chargerID:      chargerID,
		storedObs:      make(map[model.ObservationID]model.Observation),
	}

	handler.isCloudOnline.Store(true)
	handler.isStateOnline.Store(true)

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

	log.Debugf("[%s] Connected phases=%d", h.chargerID, val)

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

	_, err = chargepointSrv.SendPhaseModeReport(false)

	return err
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

	_, err = chargepointSrv.SendMaxCurrentReport(false)

	return err
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

	_, err = chargepointSrv.SendCurrentSessionReport(false)

	return err
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

	_, err = chargepointSrv.SendCableLockReport(false)

	return err
}

func (h *observationsHandler) handleCableRating(observation model.Observation) error {
	val, err := observation.IntValue()
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

	_, err = chargepointSrv.SendCableLockReport(false)

	return err
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

	h.isStateOnline.Store(state != model.ChargerStateOffline)

	if state.IsSessionFinished() {
		h.cache.SetRequestedOfferedCurrent(0, time.Now())
	}

	chargepointSrv, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	_, err = chargepointSrv.SendStateReport(false)

	return err
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

	_, err = meterElecSrv.SendMeterReport(numericmeter.UnitW, false)
	if err != nil {
		return err
	}

	_, err = meterElecSrv.SendMeterExtendedReport(numericmeter.Values{numericmeter.ValuePowerImport}, false)

	return err
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

	_, err = chargepointSrv.SendCurrentSessionReport(false)

	return err
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

		_, err = meterElecSrv.SendMeterExtendedReport(numericmeter.Values{value}, false)

		return err
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

	log.Infof("[%s] PhaseMode=%s", h.chargerID, outPhaseType)

	ok := h.cache.SetOutputPhaseType(outPhaseType, observation.Timestamp)
	if !ok {
		return nil
	}

	chargepointSrv, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	_, err = chargepointSrv.SendPhaseModeReport(false)

	return err
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
	if err := h.sendAlarmReports(map[string]bool{
		alarm.EventGroundingFault: model.GridType(val).IsGroundFault(),
		alarm.EventGridTypeFault:  model.GridType(val).IsWiringFault(),
	}, observation.Timestamp); err != nil {
		return err
	}

	if supportedGridType == gridType && supportedPhases == phases {
		return nil
	}

	log.Debugf("[%s] supGridType=%v supPh=%v", h.chargerID, supportedGridType, supportedPhases)

	ok := h.cache.SetInstallationParameters(supportedGridType, supportedPhases, observation.Timestamp)
	if !ok {
		return nil
	}

	service, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	supportedModes := model.SettablePhaseModes(supportedGridType, supportedPhases)

	service = h.ensureChargepointProps(service, map[string]any{
		chargepoint.PropertyGridType:            supportedGridType,
		chargepoint.PropertyPhases:              supportedPhases,
		chargepoint.PropertySupportedPhaseModes: supportedModes,
	})

	if err := h.thing.Update(adapter.ThingUpdateRemoveService(service), adapter.ThingUpdateAddService(service)); err != nil {
		return err
	}

	_, err = h.thing.SendInclusionReport(false)

	return err
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

	_, err = parameterSrv.SendParameterReport(model.CableAlwaysLockedParameter, true)

	return err
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

	_, err = chargepointSrv.SendCurrentSessionReport(false)

	return err
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

	_, err = chargepointSrv.SendCurrentSessionReport(false)

	return err
}

func (h *observationsHandler) ensureChargepointProps(srv chargepoint.Service, props map[string]interface{}) chargepoint.Service {
	for k, v := range props {
		if funk.IsEmpty(v) {
			delete(srv.Specification().Props, k)

			continue
		}

		srv.Specification().Props[k] = v
	}

	return srv
}

type energyHandler struct {
	cache                 cache.Cache
	thing                 adapter.Thing
	lock                  sync.Mutex
	confSrv               *config.Service
	energyObservationChan chan model.Observation
}

func newEnergyHandler(cache cache.Cache, thing adapter.Thing, confSrv *config.Service) *energyHandler {
	return &energyHandler{
		cache:   cache,
		thing:   thing,
		confSrv: confSrv,
	}
}

func (h *energyHandler) handle(observation model.Observation) error {
	observationTime := observation.Timestamp.Truncate(time.Hour)
	_, lastReadingTime := h.cache.LifetimeEnergy()
	lastReadingTime = lastReadingTime.Truncate(time.Hour)

	if !observationTime.After(lastReadingTime) {
		return nil
	}

	h.lock.Lock()
	if h.energyObservationChan == nil {
		h.energyObservationChan = make(chan model.Observation, 10)
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
		case val := <-ch:
			v, err := val.Float64Value()
			if err != nil {
				log.WithError(err)

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
func (h *observationsHandler) sendAlarmReports(events map[string]bool, timestamp time.Time) error {
	service, err := getAlarmService(h.thing)
	if err != nil {
		return err
	}

	for event, active := range events {
		if !h.cache.SetAlarm(event, active, timestamp) {
			continue
		}

		if _, err := service.SendAlarmReport(event, false); err != nil {
			return err
		}
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
