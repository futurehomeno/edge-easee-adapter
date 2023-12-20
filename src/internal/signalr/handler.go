package signalr

import (
	"errors"
	"math"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/event"

	"github.com/futurehomeno/edge-easee-adapter/internal/cache"
	"github.com/futurehomeno/edge-easee-adapter/internal/pubsub"
)

// Handler interface handles signalr observations.
type Handler interface {
	// HandleObservation handles signalr observation callback.
	HandleObservation(observation Observation) error
}

type observationsHandler struct {
	chargepoint  chargepoint.Service
	meterElec    numericmeter.Service
	cache        cache.Cache
	eventManager event.Manager
	callbacks    map[ObservationID]func(Observation) error
}

// NewObservationsHandler creates new observation handler.
func NewObservationsHandler(thing adapter.Thing, cache cache.Cache, eventManager event.Manager) (Handler, error) {
	chargepoint, err := getChargepointService(thing)
	if err != nil {
		return nil, err
	}

	meterElec, err := getMeterElecService(thing)
	if err != nil {
		return nil, err
	}

	handler := observationsHandler{
		chargepoint:  chargepoint,
		meterElec:    meterElec,
		cache:        cache,
		eventManager: eventManager,
	}

	handler.callbacks = map[ObservationID]func(Observation) error{
		MaxChargerCurrent:     handler.handleMaxChargerCurrent,
		DynamicChargerCurrent: handler.handleDynamicChargerCurrent,
		CableLocked:           handler.handleCableLocked,
		CableRating:           handler.handleCableRating,
		ChargerOPState:        handler.handleChargerState,
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

func (o *observationsHandler) HandleObservation(observation Observation) error {
	if callback, ok := o.callbacks[observation.ID]; ok {
		return callback(observation)
	}

	return nil
}

func (o *observationsHandler) handleMaxChargerCurrent(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	current := int64(math.Round(val))
	o.cache.SetMaxCurrent(current)

	o.eventManager.Publish(pubsub.NewMaxCurrentRefreshEvent(current))

	return nil
}

func (o *observationsHandler) handleCloudConnected(observation Observation) error {
	val, err := observation.BoolValue()
	if err != nil {
		return err
	}

	if !val {
		o.cache.SetTotalPower(0)
		o.cache.SetChargerState(chargepoint.StateUnavailable)
		_, err = o.chargepoint.SendStateReport(true)
	}

	return err
}

func (o *observationsHandler) handleDynamicChargerCurrent(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	current := int64(math.Round(val))
	o.cache.SetDynamicCurrent(current)

	o.eventManager.Publish(pubsub.NewOfferedCurrentRefreshEvent(current))

	return nil
}

func (o *observationsHandler) handleCableLocked(observation Observation) error {
	val, err := observation.BoolValue()
	if err != nil {
		return err
	}

	o.cache.SetCableLocked(val)

	_, err = o.chargepoint.SendCableLockReport(false)

	return err
}

func (o *observationsHandler) handleCableRating(observation Observation) error {
	val, err := observation.IntValue()
	if err != nil {
		return err
	}

	o.cache.SetCableCurrent(int64(val))

	_, err = o.chargepoint.SendCableLockReport(false)

	return err
}

func (o *observationsHandler) handleChargerState(observation Observation) error {
	val, err := observation.IntValue()
	if err != nil {
		return err
	}

	chargerState := ChargerState(val)
	o.cache.SetChargerState(chargerState.ToFimpState())

	if chargerState.IsSessionFinished() {
		o.cache.SetOfferedCurrent(0)
	}

	_, err = o.chargepoint.SendStateReport(false)

	return err
}

func (o *observationsHandler) handleTotalPower(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.SetTotalPower(val * 1000)

	_, err = o.meterElec.SendMeterReport(numericmeter.UnitW, false)
	if err != nil {
		return err
	}

	_, err = o.meterElec.SendMeterExtendedReport(numericmeter.Values{numericmeter.ValuePowerImport}, false)

	return err
}

func (o *observationsHandler) handleLifetimeEnergy(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.SetLifetimeEnergy(val)

	_, err = o.meterElec.SendMeterReport(numericmeter.UnitKWh, false)
	if err != nil {
		return err
	}

	_, err = o.meterElec.SendMeterExtendedReport(numericmeter.Values{numericmeter.ValueEnergyImport}, false)

	return err
}

func (o *observationsHandler) handleEnergySession(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.SetEnergySession(val)

	_, err = o.meterElec.SendMeterReport(numericmeter.UnitKWh, false)

	return err
}

func (o *observationsHandler) handleInCurrentT3(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.SetPhase1Current(val)

	_, err = o.meterElec.SendMeterExtendedReport(numericmeter.Values{numericmeter.ValueCurrentPhase1}, false)

	return err
}

func (o *observationsHandler) handleInCurrentT4(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.SetPhase2Current(val)

	_, err = o.meterElec.SendMeterExtendedReport(numericmeter.Values{numericmeter.ValueCurrentPhase2}, false)

	return err
}

func (o *observationsHandler) handleInCurrentT5(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.SetPhase3Current(val)

	_, err = o.meterElec.SendMeterExtendedReport(numericmeter.Values{numericmeter.ValueCurrentPhase3}, false)

	return err
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
