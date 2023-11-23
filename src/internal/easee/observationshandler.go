package easee

import (
	"errors"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"

	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
)

type ObservationsHandler interface {
	// HandleObservation handles signalr observation callback.
	HandleObservation(observation signalr.Observation) error
}

type observationsHandler struct {
	chargepoints []chargepoint.Service
	meterElecs   []numericmeter.Service
	cache        ObservationCache
	callbacks    map[signalr.ObservationID]func(signalr.Observation) error
}

func NewObservationsHandler(chargepoints []chargepoint.Service, meterElecs []numericmeter.Service, cache ObservationCache) ObservationsHandler {
	handler := observationsHandler{
		chargepoints: chargepoints,
		meterElecs:   meterElecs,
		cache:        cache,
	}

	handler.callbacks = map[signalr.ObservationID]func(signalr.Observation) error{
		signalr.ChargerOPState: handler.handleChargerState,
		signalr.SessionEnergy:  handler.handleSessionEnergy,
		signalr.CableLocked:    handler.handleCableLocked,
		signalr.TotalPower:     handler.handleTotalPower,
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

func (o *observationsHandler) handleChargerState(observation signalr.Observation) error {
	val, err := observation.IntValue()
	if err != nil {
		return err
	}

	o.cache.setChargerState(ChargerState(val))

	var ret error
	for _, cp := range o.chargepoints {
		if _, err := cp.SendStateReport(false); err != nil {
			ret = errors.Join(ret, err)
		}
	}
	return err
}

func (o *observationsHandler) handleSessionEnergy(observation signalr.Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.setSessionEnergy(val)

	var ret error
	for _, cp := range o.chargepoints {
		if _, err := cp.SendCurrentSessionReport(false); err != nil {
			ret = errors.Join(ret, err)
		}
	}

	return err
}
func (o *observationsHandler) handleCableLocked(observation signalr.Observation) error {
	val, err := observation.BoolValue()
	if err != nil {
		return err
	}

	o.cache.setCableLocked(val)

	var ret error
	for _, cp := range o.chargepoints {
		if _, err := cp.SendCableLockReport(false); err != nil {
			ret = errors.Join(ret, err)
		}
	}

	return err
}

func (o *observationsHandler) handleTotalPower(observation signalr.Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.setTotalPower(val * 1000)

	var ret error
	for _, cp := range o.meterElecs {
		if _, err := cp.SendMeterReport(numericmeter.UnitW, false); err != nil {
			ret = errors.Join(ret, err)
		}
	}

	return err
}

func (o *observationsHandler) handleLifetimeEnergy(observation signalr.Observation) error {
	val, err := observation.Float64Value()
	if err != nil {
		return err
	}

	o.cache.setLifetimeEnergy(val)

	var ret error
	for _, cp := range o.meterElecs {
		if _, err := cp.SendMeterReport(numericmeter.UnitKWh, false); err != nil {
			ret = errors.Join(ret, err)
		}
	}

	return err
}
