package signalr

import (
	"errors"
	"math"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

type ObservationsHandler interface {
	// HandleObservation handles signalr observation callback.
	HandleObservation(observation Observation) error
}

type observationsHandler struct {
	chargepoint chargepoint.Service
	meterElec   numericmeter.Service
	cache       config.Cache
	callbacks   map[ObservationID]func(Observation) error
}

func NewObservationsHandler(thing adapter.Thing, cache config.Cache) (ObservationsHandler, error) {
	chargepoint, err := getChargepointService(thing)
	if err != nil {
		return nil, err
	}

	meterElec, err := getMeterElecService(thing)
	if err != nil {
		return nil, err
	}

	handler := observationsHandler{
		chargepoint: chargepoint,
		meterElec:   meterElec,
		cache:       cache,
	}

	handler.callbacks = map[ObservationID]func(Observation) error{
		MaxChargerCurrent:     handler.handleMaxChargerCurrent,
		DynamicChargerCurrent: handler.handleDynamicChargerCurrent,
		CableLocked:           handler.handleCableLocked,
		CableRating:           handler.handleCableRating,
		ChargerOPState:        handler.handleChargerState,
		TotalPower:            handler.handleTotalPower,
		LifetimeEnergy:        handler.handleLifetimeEnergy,
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

	_, err = o.chargepoint.SendMaxCurrentReport(false)

	return err
}

func (o *observationsHandler) handleDynamicChargerCurrent(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	current := int64(math.Round(val))
	o.cache.SetOfferedCurrent(current)

	return err
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

	o.cache.SetChargerState(ChargerState(val).ToFimpState())

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

	return err
}

func (o *observationsHandler) handleLifetimeEnergy(observation Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.SetLifetimeEnergy(val)

	_, err = o.meterElec.SendMeterReport(numericmeter.UnitKWh, false)

	return err
}

func getChargepointService(thing adapter.Thing) (chargepoint.Service, error) {
	for _, service := range thing.Services(chargepoint.Chargepoint) {
		if service, ok := service.(chargepoint.Service); ok {
			return service, nil
		}
	}

	return nil, errors.New("There are no chargepoint services")
}

func getMeterElecService(thing adapter.Thing) (numericmeter.Service, error) {
	for _, service := range thing.Services(numericmeter.MeterElec) {
		if service, ok := service.(numericmeter.Service); ok {
			return service, nil
		}
	}

	return nil, errors.New("There are no meterelec services")
}
