package router

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/easee-ad/model"
)

// SendChargerState sends the charger state
func (fc *FromFimpRouter) SendChargerState(chargerID string, oldMsg *fimpgo.Message) error {
	var fimpChargeState = map[int]string{
		1: "available",
		2: "paused",
		3: "charging",
		4: "finished",
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
			value = fc.easee.Products[chargerID].ChargerState.TotalPower
		case "kWh":
			value = fc.easee.Products[chargerID].ChargerState.SessionEnergy
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
