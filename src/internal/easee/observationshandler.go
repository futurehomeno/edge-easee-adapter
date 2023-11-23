package easee

import (
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"

	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
)

type ObservationsHandler interface {
	// HandleObservation handles signalr observation callback.
	HandleObservation(observation signalr.Observation) error
}

type observationsHandler struct {
	chargepoint chargepoint.Service
	meterElec   numericmeter.Service
	cache       ObservationCache
	callbacks   map[signalr.ObservationID]func(signalr.Observation) error
}

func NewObservationsHandler(chargepoint chargepoint.Service, meterElec numericmeter.Service, cache ObservationCache) ObservationsHandler {
	handler := observationsHandler{
		chargepoint: chargepoint,
		meterElec:   meterElec,
		cache:       cache,
	}

	handler.callbacks = map[signalr.ObservationID]func(signalr.Observation) error{
		signalr.CableLocked:    handler.handleCableLocked,
		signalr.CableRating:    handler.handleCableRating,
		signalr.ChargerOPState: handler.handleChargerState,
		signalr.TotalPower:     handler.handleTotalPower,
		signalr.SessionEnergy:  handler.handleSessionEnergy,
		signalr.LifetimeEnergy: handler.handleLifetimeEnergy,
	}

	return &handler
}

func (o *observationsHandler) HandleObservation(observation signalr.Observation) error {
	if callback, ok := o.callbacks[observation.ID]; ok {
		return callback(observation)
	}

	return nil
}

func (o *observationsHandler) handleCableLocked(observation signalr.Observation) error {
	val, err := observation.BoolValue()
	if err != nil {
		return err
	}

	o.cache.setCableLocked(val)

	_, err = o.chargepoint.SendCableLockReport(false)

	return err
}

func (o *observationsHandler) handleCableRating(observation signalr.Observation) error {
	val, err := observation.IntValue()
	if err != nil {
		return err
	}

	o.cache.setCableCurrent(int64(val))

	_, err = o.chargepoint.SendCableLockReport(false)

	return err
}

func (o *observationsHandler) handleChargerState(observation signalr.Observation) error {
	val, err := observation.IntValue()
	if err != nil {
		return err
	}

	o.cache.setChargerState(ChargerState(val))

	_, err = o.chargepoint.SendStateReport(false)

	return err
}

func (o *observationsHandler) handleTotalPower(observation signalr.Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.setTotalPower(val * 1000)

	_, err = o.meterElec.SendMeterReport(numericmeter.UnitW, false)

	return err
}

func (o *observationsHandler) handleSessionEnergy(observation signalr.Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.setSessionEnergy(val)

	_, err = o.chargepoint.SendCurrentSessionReport(false)

	return err
}

func (o *observationsHandler) handleLifetimeEnergy(observation signalr.Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.setLifetimeEnergy(val)

	_, err = o.meterElec.SendMeterReport(numericmeter.UnitKWh, false)

	return err
}
