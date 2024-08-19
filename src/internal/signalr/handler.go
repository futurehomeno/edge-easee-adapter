package signalr

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"

	"github.com/futurehomeno/edge-easee-adapter/internal/cache"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

// Handler interface handles signalr observations.
type Handler interface {
	// IsOnline return if the charger is online.
	IsOnline() bool

	// HandleObservation handles signalr observation callback.
	HandleObservation(observation model.Observation) error
}

type observationsHandler struct {
	cache         cache.Cache
	handlers      map[model.ObservationID]func(model.Observation) error
	thing         adapter.Thing
	energyHandler *energyHandler

	isCloudOnline atomic.Bool
	isStateOnline atomic.Bool
}

// NewObservationsHandler creates new observation handler.
func NewObservationsHandler(thing adapter.Thing, cache cache.Cache, confSrv *config.Service) (Handler, error) {
	handler := observationsHandler{
		cache:         cache,
		thing:         thing,
		energyHandler: newEnergyHandler(cache, thing, confSrv),
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
		model.InCurrentT3:           handler.handleInCurrentT3,
		model.InCurrentT4:           handler.handleInCurrentT4,
		model.InCurrentT5:           handler.handleInCurrentT5,
		model.CloudConnected:        handler.handleCloudConnected,
		model.CableLocked:           handler.handleCableLocked,
		model.CableRating:           handler.handleCableRating,
		model.LockCablePermanently:  handler.handleLockCablePermanently,
	}

	return &handler, nil
}

func (h *observationsHandler) IsOnline() bool {
	return h.isCloudOnline.Load() && h.isStateOnline.Load()
}

func (h *observationsHandler) HandleObservation(observation model.Observation) error {
	if handler, ok := h.handlers[observation.ID]; ok {
		return handler(observation)
	}

	return nil
}

func (h *observationsHandler) handlePhaseMode(observation model.Observation) error {
	val, err := observation.IntValue()
	if err != nil {
		return err
	}

	if val == h.cache.PhaseMode() {
		return nil
	}

	h.cache.SetPhaseMode(val)

	service, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	service = h.ensureChargepointProps(service, map[string]interface{}{
		chargepoint.PropertySupportedPhaseModes: model.SupportedPhaseModes(h.cache.GridType(), h.cache.PhaseMode(), h.cache.Phases()),
	})

	if err := h.thing.Update(adapter.ThingUpdateRemoveService(service), adapter.ThingUpdateAddService(service)); err != nil {
		return err
	}

	_, err = h.thing.SendInclusionReport(false)

	return err
}

func (h *observationsHandler) handleMaxChargerCurrent(observation model.Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	current := int64(math.Round(val))
	h.cache.SetMaxCurrent(current)

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

	h.isCloudOnline.Store(val)

	return err
}

func (h *observationsHandler) handleDynamicChargerCurrent(observation model.Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	current := int64(math.Round(val))
	h.cache.SetOfferedCurrent(current)

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

	h.cache.SetCableLocked(val)

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

	h.cache.SetCableCurrent(int64(val))

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

	chargerState := model.ChargerState(val)
	h.cache.SetChargerState(chargerState.ToFimpState())
	h.isStateOnline.Store(chargerState != model.ChargerStateOffline)

	if chargerState.IsSessionFinished() {
		h.cache.SetRequestedOfferedCurrent(0)
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

	h.cache.SetTotalPower(val * 1000)

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

	h.cache.SetEnergySession(val)

	chargepointSrv, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	_, err = chargepointSrv.SendCurrentSessionReport(false)

	return err
}

func (h *observationsHandler) handleInCurrentT3(observation model.Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	h.cache.SetPhase1Current(val)

	meterElecSrv, err := getMeterElecService(h.thing)
	if err != nil {
		return err
	}

	_, err = meterElecSrv.SendMeterExtendedReport(numericmeter.Values{numericmeter.ValueCurrentPhase1}, false)

	return err
}

func (h *observationsHandler) handleInCurrentT4(observation model.Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	h.cache.SetPhase2Current(val)

	meterElecSrv, err := getMeterElecService(h.thing)
	if err != nil {
		return err
	}

	_, err = meterElecSrv.SendMeterExtendedReport(numericmeter.Values{numericmeter.ValueCurrentPhase2}, false)

	return err
}

func (h *observationsHandler) handleInCurrentT5(observation model.Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	h.cache.SetPhase3Current(val)

	meterElecSrv, err := getMeterElecService(h.thing)
	if err != nil {
		return err
	}

	_, err = meterElecSrv.SendMeterExtendedReport(numericmeter.Values{numericmeter.ValueCurrentPhase3}, false)

	return err
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

	h.cache.SetOutputPhaseType(outPhaseType)

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

	supportedGridType, supportedPhases := model.GridType(val).ToFimpGridType()
	if supportedGridType == h.cache.GridType() && supportedPhases == h.cache.Phases() {
		return nil
	}

	h.cache.SetGridType(supportedGridType)
	h.cache.SetPhases(supportedPhases)

	service, err := getChargepointService(h.thing)
	if err != nil {
		return err
	}

	service = h.ensureChargepointProps(service, map[string]interface{}{
		chargepoint.PropertyGridType:            supportedGridType,
		chargepoint.PropertyPhases:              supportedPhases,
		chargepoint.PropertySupportedPhaseModes: model.SupportedPhaseModes(h.cache.GridType(), h.cache.PhaseMode(), h.cache.Phases()),
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

	h.cache.SetCableAlwaysLocked(val)

	parameterSrv, err := getParametersService(h.thing)
	if err != nil {
		return err
	}

	_, err = parameterSrv.SendParameterReport(model.CableAlwaysLockedParameter, val)

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
	lastReadingTime := h.cache.LifetimeEnergy().Timestamp.Truncate(time.Hour)

	if !observationTime.After(lastReadingTime) {
		return nil
	}

	if h.energyObservationChan == nil {
		h.lock.Lock()
		h.energyObservationChan = make(chan model.Observation, 10)
		h.lock.Unlock()

		go h.manageEnergyObservation()
	}

	h.energyObservationChan <- observation

	return nil
}

func (h *energyHandler) manageEnergyObservation() {
	defer func() {
		h.lock.Lock()
		defer h.lock.Unlock()

		h.energyObservationChan = nil
	}()

	timer := time.NewTimer(h.confSrv.GetEnergyLifetimeInterval())
	defer timer.Stop()

	energy := model.TimestampedValue[float64]{}

	for {
		select {
		case val := <-h.energyObservationChan:
			v, err := val.Float64Value()
			if err != nil {
				log.WithError(err)

				continue
			}

			if val.Timestamp.Before(energy.Timestamp) {
				continue
			}

			energy.Value = v
			energy.Timestamp = val.Timestamp

		case <-timer.C:
			h.cache.SetLifetimeEnergy(energy)

			meterElecSrv, err := getMeterElecService(h.thing)
			if err != nil {
				log.WithField("thing-address", h.thing.Address()).
					WithError(err).
					Error("lifetime energy handler: failed to get meter elec service")

				return
			}

			_, err = meterElecSrv.SendMeterReport(numericmeter.UnitKWh, false)
			if err != nil {
				log.WithField("thing-address", h.thing.Address()).
					WithError(err).
					Error("lifetime energy handler: failed to send meter report")

				return
			}

			_, err = meterElecSrv.SendMeterExtendedReport(numericmeter.Values{numericmeter.ValueEnergyImport}, false)
			if err != nil {
				log.WithField("thing-address", h.thing.Address()).
					WithError(err).
					Error("lifetime energy handler: failed to send meter extend report")

				return
			}

			return
		}
	}
}

func getParametersService(thing adapter.Thing) (parameters.Service, error) {
	for _, service := range thing.Services(parameters.Parameters) {
		if service, ok := service.(parameters.Service); ok {
			return service, nil
		}
	}

	return nil, errors.New("there are no parameters services")
}

func getChargepointService(thing adapter.Thing) (chargepoint.Service, error) {
	for _, service := range thing.Services(chargepoint.Chargepoint) {
		if service, ok := service.(chargepoint.Service); ok {
			return service, nil
		}
	}

	return nil, errors.New("there are no chargepoint services")
}

func getMeterElecService(thing adapter.Thing) (numericmeter.Service, error) {
	for _, service := range thing.Services(numericmeter.MeterElec) {
		if service, ok := service.(numericmeter.Service); ok {
			return service, nil
		}
	}

	return nil, errors.New("there are no meterelec services")
}
