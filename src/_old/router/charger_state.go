package router

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/_old/model"
)

// SendChargerState sends the charger state
func (fc *FromFimpRouter) SendChargerState(chargerID string, oldMsg *fimpgo.Message) error {
	log.Debug("ChargerOpMode: ", fc.easee.Products[chargerID].ChargerState.ChargerOpMode)

	var fimpChargeState = map[int]string{
		0: "unavailable",
		1: "disconnected",
		2: "ready_to_charge",
		3: "charging",
		4: "finished",
		5: "error",
		6: "requesting",
	}

	var oldPayload *fimpgo.FimpMessage
	if oldMsg != nil {
		oldPayload = oldMsg.Payload
	}
	state := fc.easee.Products[chargerID].ChargerState.ChargerOpMode
	msg := fimpgo.NewStringMessage("evt.state.report", "chargepoint", fimpChargeState[state], nil, nil, oldPayload)
	msg.Source = model.ServiceName
	addr := fimpgo.Address{
		MsgType:         fimpgo.MsgTypeEvt,
		ResourceType:    fimpgo.ResourceTypeDevice,
		ResourceName:    model.ServiceName,
		ResourceAddress: fc.configs.InstanceAddress,
		ServiceName:     "chargepoint",
		ServiceAddress:  chargerID,
	}
	err := fc.mqt.Publish(&addr, msg)
	if err != nil {
		log.Debug(err)
		return err
	}
	return err
}

// SendChangedStateForAllChargers sends a new FIMP message if the state for the charger has changed.
func (fc *FromFimpRouter) SendChangedStateForAllChargers() error {
	for _, product := range fc.easee.Products {
		if product.ChargeStateHasChanged() {
			err := fc.SendChargerState(product.Charger.ID, nil)
			if err != nil {
				return err
			}
			err = fc.SendSessionEnergyReport(product.Charger.ID, nil) // sends session report on changed state, in case new state != (charging || finished) -> session will be forced to 0.
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SendStateForAllChargers will send a FIMP state message for all chargers
func (fc *FromFimpRouter) SendStateForAllChargers() error {
	for _, product := range fc.easee.Products {
		err := fc.SendChargerState(product.Charger.ID, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// SendMeterReport sends evt.meter.report
func (fc *FromFimpRouter) SendMeterReport(chargerID string, unit string, oldMsg *fimpgo.Message) error {
	var oldPayload *fimpgo.FimpMessage
	if oldMsg != nil {
		oldPayload = oldMsg.Payload
	}
	if unit == "W" || unit == "kWh" || unit == "V" {
		props := fimpgo.Props{
			"unit": unit,
		}
		var value float64
		switch unit {
		case "W":
			value = fc.easee.Products[chargerID].ChargerState.TotalPower * 1000
		case "kWh":
			value = fc.easee.Products[chargerID].ChargerState.LifetimeEnergy
		case "V":
			value = fc.easee.Products[chargerID].ChargerState.Voltage
		default:
			return fmt.Errorf("Not a valid unit")
		}
		msg := fimpgo.NewFloatMessage("evt.meter.report", "meter_elec", value, props, nil, oldPayload)
		msg.Source = model.ServiceName
		addr := fimpgo.Address{
			MsgType:         fimpgo.MsgTypeEvt,
			ResourceType:    fimpgo.ResourceTypeDevice,
			ResourceName:    model.ServiceName,
			ResourceAddress: fc.configs.InstanceAddress,
			ServiceName:     "meter_elec",
			ServiceAddress:  chargerID,
		}
		err := fc.mqt.Publish(&addr, msg)
		if err != nil {
			log.Debug(err)
			return err
		}
		return nil
	}
	return fmt.Errorf("Not a valid unit")
}

// SendSessionEnergyReport sends evt.current_session.report
func (fc *FromFimpRouter) SendSessionEnergyReport(chargerID string, oldMsg *fimpgo.Message) error {
	var oldPayload *fimpgo.FimpMessage
	if oldMsg != nil {
		oldPayload = oldMsg.Payload
	}
	props := fimpgo.Props{
		"unit": "kWh",
	}
	var value float64
	if fc.easee.Products[chargerID].ChargerState.ChargerOpMode != 3 && fc.easee.Products[chargerID].ChargerState.ChargerOpMode != 4 {
		log.Debug("!= (charging || finished)")
		value = 0
	} else {
		log.Debug("charging || finished")
		value = fc.easee.Products[chargerID].ChargerState.SessionEnergy
	}
	log.Debug("Getted session energy: ", fc.easee.Products[chargerID].ChargerState.SessionEnergy)
	msg := fimpgo.NewFloatMessage("evt.current_session.report", "chargepoint", value, props, nil, oldPayload)
	msg.Source = model.ServiceName
	addr := fimpgo.Address{
		MsgType:         fimpgo.MsgTypeEvt,
		ResourceType:    fimpgo.ResourceTypeDevice,
		ResourceName:    model.ServiceName,
		ResourceAddress: fc.configs.InstanceAddress,
		ServiceName:     "chargepoint",
		ServiceAddress:  chargerID,
	}
	err := fc.mqt.Publish(&addr, msg)
	if err != nil {
		log.Debug(err)
		return err
	}
	return nil
}

// SendCableReport sends evt.cable_lock.report
func (fc *FromFimpRouter) SendCableReport(chargerID string, oldMsg *fimpgo.Message) error {
	var oldPayload *fimpgo.FimpMessage
	if oldMsg != nil {
		oldPayload = oldMsg.Payload
	}
	msg := fimpgo.NewBoolMessage("evt.cable_lock.report", "chargepoint", fc.easee.Products[chargerID].ChargerState.CableLocked, nil, nil, oldPayload)
	msg.Source = model.ServiceName
	addr := fimpgo.Address{
		MsgType:         fimpgo.MsgTypeEvt,
		ResourceType:    fimpgo.ResourceTypeDevice,
		ResourceName:    model.ServiceName,
		ResourceAddress: fc.configs.InstanceAddress,
		ServiceName:     "chargepoint",
		ServiceAddress:  chargerID,
	}
	err := fc.mqt.Publish(&addr, msg)
	if err != nil {
		log.Debug(err)
		return err
	}
	return nil
}

// SendWattReportForAllProducts sends evt.meter.report with unit W for all products
func (fc *FromFimpRouter) SendWattReportForAllProducts() error {
	for _, product := range fc.easee.Products {
		err := fc.SendMeterReport(product.Charger.ID, "W", nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// SendWattReportIfValueChanged sends a FIMP message if the wattage has changed.
func (fc *FromFimpRouter) SendWattReportIfValueChanged() error {
	for _, product := range fc.easee.Products {
		if product.WattHasChanged() {
			err := fc.SendMeterReport(product.Charger.ID, "W", nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SendLifetimeEnergyReportIfValueChanged sends evt.meter.report if the lifetime energy value has changed.
func (fc *FromFimpRouter) SendLifetimeEnergyReportIfValueChanged() error {
	for _, product := range fc.easee.Products {
		if product.LifetimeEnergyHasChanged() {
			err := fc.SendMeterReport(product.Charger.ID, "kWh", nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SendSessionEnergyReportForAllProducts sends evt.current_session.report with unit kWh for all products
func (fc *FromFimpRouter) SendSessionEnergyReportForAllProducts() error {
	for _, product := range fc.easee.Products {
		err := fc.SendSessionEnergyReport(product.Charger.ID, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// SendSessionEnergyReportIfValueChanged sends a FIMP message if the session energy has changed.
func (fc *FromFimpRouter) SendSessionEnergyReportIfValueChanged() error {
	for _, product := range fc.easee.Products {
		if product.SessionEnergyHasChanged() {
			err := fc.SendSessionEnergyReport(product.Charger.ID, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SendCableReportForAllProducts sends evt.cable_lock.report for all products
func (fc *FromFimpRouter) SendCableReportForAllProducts() error {
	for _, product := range fc.easee.Products {
		err := fc.SendCableReport(product.Charger.ID, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// SendCableReportIfChanged sends a FIMP message if the cable lock state has changed.
func (fc *FromFimpRouter) SendCableReportIfChanged() error {
	for _, product := range fc.easee.Products {
		if product.CableLockHasChanged() {
			err := fc.SendCableReport(product.Charger.ID, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
