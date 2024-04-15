package signalr

import (
	"errors"
	"math"
	"sync/atomic"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"

	"github.com/futurehomeno/edge-easee-adapter/internal/cache"
	"github.com/futurehomeno/edge-easee-adapter/internal/helper"
)

// Handler interface handles signalr observations.
type Handler interface {
	// IsOnline return if the charger is online.
	IsOnline() bool

	// HandleObservation handles signalr observation callback.
	HandleObservation(observation Observation) error
}

type observationsHandler struct {
	cache     cache.Cache
	callbacks map[ObservationID]func(Observation) error
	thing     adapter.Thing

	isCloudOnline atomic.Bool
	isStateOnline atomic.Bool
}

// NewObservationsHandler creates new observation handler.
func NewObservationsHandler(thing adapter.Thing, cache cache.Cache) (Handler, error) {
	handler := observationsHandler{
		cache: cache,
		thing: thing,
	}

	handler.isCloudOnline.Store(true)
	handler.isStateOnline.Store(true)

	handler.callbacks = map[ObservationID]func(Observation) error{
		PhaseMode:             handler.handlePhaseMode,
		MaxChargerCurrent:     handler.handleMaxChargerCurrent,
		DynamicChargerCurrent: handler.handleDynamicChargerCurrent,
		ChargerOPState:        handler.handleChargerState,
		OutputPhase:           handler.handleOutPhase,
		TotalPower:            handler.handleTotalPower,
		LifetimeEnergy:        handler.handleLifetimeEnergy,
		EnergySession:         handler.handleEnergySession,
		InCurrentT3:           handler.handleInCurrentT3,
		InCurrentT4:           handler.handleInCurrentT4,
		InCurrentT5:           handler.handleInCurrentT5,
		CloudConnected:        handler.handleCloudConnected,
	}

	return &handler, nil
}

func (o *observationsHandler) IsOnline() bool {
	return o.isCloudOnline.Load() && o.isStateOnline.Load()
}

func (o *observationsHandler) HandleObservation(observation Observation) error {
	if callback, ok := o.callbacks[observation.ID]; ok {
		return callback(observation)
	}

	return nil
}

func (o *observationsHandler) handlePhaseMode(observation Observation) error {
	val, err := observation.IntValue()
	if err != nil {
		return err
	}

	o.cache.SetPhaseMode(val)

	chargepointSrv, err := o.getChargepointService()
	if err != nil {
		return err
	}

	o.cache.OutputPhaseType()

	newChargepointSrv := chargepointSrv
	newChargepointSrv.Specification().Props["sup_phase_modes"] = helper.SupportedPhaseModes(o.cache.GridType(), o.cache.PhaseMode(), o.cache.Phases())

	if err := o.thing.Update(adapter.ThingUpdateRemoveService(chargepointSrv), adapter.ThingUpdateAddService(newChargepointSrv)); err != nil {
		return err
	}

	_, err = o.thing.SendInclusionReport(false)

	return err
}

func (o *observationsHandler) handleMaxChargerCurrent(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	current := int64(math.Round(val))
	o.cache.SetMaxCurrent(current)

	chargepointSrv, err := o.getChargepointService()
	if err != nil {
		return err
	}

	_, err = chargepointSrv.SendMaxCurrentReport(false)

	return err
}

func (o *observationsHandler) handleCloudConnected(observation Observation) error {
	val, err := observation.BoolValue()
	if err != nil {
		return err
	}

	o.isCloudOnline.Store(val)

	return err
}

func (o *observationsHandler) handleDynamicChargerCurrent(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	current := int64(math.Round(val))
	o.cache.SetOfferedCurrent(current)

	chargepointSrv, err := o.getChargepointService()
	if err != nil {
		return err
	}

	_, err = chargepointSrv.SendCurrentSessionReport(false)

	return err
}

func (o *observationsHandler) handleChargerState(observation Observation) error {
	val, err := observation.IntValue()
	if err != nil {
		return err
	}

	chargerState := ChargerState(val)
	o.cache.SetChargerState(chargerState.ToFimpState())
	o.isStateOnline.Store(chargerState != ChargerStateOffline)

	if chargerState.IsSessionFinished() {
		o.cache.SetRequestedOfferedCurrent(0)
	}

	chargepointSrv, err := o.getChargepointService()
	if err != nil {
		return err
	}

	_, err = chargepointSrv.SendStateReport(false)

	return err
}

func (o *observationsHandler) handleTotalPower(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.SetTotalPower(val * 1000)

	meterElecSrv, err := o.getMeterElecService()
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

func (o *observationsHandler) handleLifetimeEnergy(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.SetLifetimeEnergy(val)

	meterElecSrv, err := o.getMeterElecService()
	if err != nil {
		return err
	}

	_, err = meterElecSrv.SendMeterReport(numericmeter.UnitKWh, false)
	if err != nil {
		return err
	}

	_, err = meterElecSrv.SendMeterExtendedReport(numericmeter.Values{numericmeter.ValueEnergyImport}, false)

	return err
}

func (o *observationsHandler) handleEnergySession(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.SetEnergySession(val)

	chargepointSrv, err := o.getChargepointService()
	if err != nil {
		return err
	}

	_, err = chargepointSrv.SendCurrentSessionReport(false)

	return err
}

func (o *observationsHandler) handleInCurrentT3(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.SetPhase1Current(val)

	meterElecSrv, err := o.getMeterElecService()
	if err != nil {
		return err
	}

	_, err = meterElecSrv.SendMeterExtendedReport(numericmeter.Values{numericmeter.ValueCurrentPhase1}, false)

	return err
}

func (o *observationsHandler) handleInCurrentT4(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.SetPhase2Current(val)

	meterElecSrv, err := o.getMeterElecService()
	if err != nil {
		return err
	}

	_, err = meterElecSrv.SendMeterExtendedReport(numericmeter.Values{numericmeter.ValueCurrentPhase2}, false)

	return err
}

func (o *observationsHandler) handleInCurrentT5(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.SetPhase3Current(val)

	meterElecSrv, err := o.getMeterElecService()
	if err != nil {
		return err
	}

	_, err = meterElecSrv.SendMeterExtendedReport(numericmeter.Values{numericmeter.ValueCurrentPhase3}, false)

	return err
}

func (o *observationsHandler) handleOutPhase(observation Observation) error {
	val, err := observation.IntValue()
	if err != nil {
		return err
	}

	outPhaseType := OutputPhaseType(val)
	o.cache.SetOutputPhaseType(outPhaseType.ToFimpState())

	chargepointSrv, err := o.getChargepointService()
	if err != nil {
		return err
	}

	_, err = chargepointSrv.SendPhaseModeReport(false)

	return err
}

func (o *observationsHandler) getChargepointService() (chargepoint.Service, error) {
	for _, service := range o.thing.Services(chargepoint.Chargepoint) {
		if service, ok := service.(chargepoint.Service); ok {
			return service, nil
		}
	}

	return nil, errors.New("there are no chargepoint services")
}

func (o *observationsHandler) getMeterElecService() (numericmeter.Service, error) {
	for _, service := range o.thing.Services(numericmeter.MeterElec) {
		if service, ok := service.(numericmeter.Service); ok {
			return service, nil
		}
	}

	return nil, errors.New("there are no meterelec services")
}
