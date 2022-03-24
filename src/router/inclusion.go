package router

import (
	"github.com/futurehomeno/edge-easee-adapter/model"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"
)

// SendInclusionReports sends one inclusion report for each charger
func (fc *FromFimpRouter) SendInclusionReports() {
	for _, charger := range fc.easee.Products {
		fc.SendInclusionReport(charger.Charger.ID, nil)
		log.Debug(charger)
	}
}

// SendInclusionReport sends a report for one charger
func (fc *FromFimpRouter) SendInclusionReport(chargerID string, oldMsg *fimpgo.FimpMessage) error {
	inclusionReport := fc.createInclusionReport(chargerID)
	msg := fimpgo.NewMessage("evt.thing.inclusion_report", model.ServiceName, fimpgo.VTypeObject, inclusionReport, nil, nil, oldMsg)
	msg.Source = model.ServiceName
	addr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: model.ServiceName, ResourceAddress: fc.configs.InstanceAddress}
	err := fc.mqt.Publish(&addr, msg)
	if err != nil {
		log.Debug(err)
		return err
	}
	return nil
}

func (fc *FromFimpRouter) createInclusionReport(chargerID string) fimptype.ThingInclusionReport {
	alias := fc.easee.Products[chargerID].Charger.Name
	manufacturer := model.ServiceName
	productHash := manufacturer
	productName := "Easee Laderobot"
	powerSource := "ac"
	cpService := fc.createChargePointService(chargerID)
	meterService := fc.createMeterElecService(chargerID)
	services := []fimptype.Service{cpService, meterService}

	inclusionReport := fimptype.ThingInclusionReport{
		IntegrationId:     "",
		Address:           chargerID,
		Alias:             alias,
		Type:              "",
		ProductHash:       productHash,
		CommTechnology:    "cloud",
		ProductName:       productName,
		ManufacturerId:    manufacturer,
		DeviceId:          chargerID,
		HwVersion:         "1",
		SwVersion:         "1",
		PowerSource:       powerSource,
		WakeUpInterval:    "-1",
		Security:          "",
		Tags:              nil,
		Groups:            []string{"1"},
		PropSets:          nil,
		TechSpecificProps: nil,
		Services:          services,
	}
	return inclusionReport
}

func (fc *FromFimpRouter) createMeterElecService(addr string) fimptype.Service {
	interfaces := []fimptype.Interface{
		fimptype.Interface{
			Type:      "in",
			MsgType:   "cmd.meter.get_report",
			ValueType: "null",
			Version:   "1",
		},
		fimptype.Interface{
			Type:      "out",
			MsgType:   "evt.meter.report",
			ValueType: "float",
			Version:   "1",
		},
	}
	props := map[string]interface{}{
		"sup_units": []string{"W", "kWh", "V", "A"},
	}
	meterElecService := fimptype.Service{
		Address:          "/rt:dev/rn:" + model.ServiceName + "/ad:" + fc.configs.InstanceAddress + "/sv:meter_elec/ad:" + addr,
		Alias:            "meter elec",
		Enabled:          true,
		Groups:           []string{"1"},
		Interfaces:       interfaces,
		Name:             "meter_elec",
		PropSetReference: "",
		Props:            props,
		Tags:             nil,
	}
	return meterElecService
}

func (fc *FromFimpRouter) createChargePointService(addr string) fimptype.Service {

	interfaces := []fimptype.Interface{
		fimptype.Interface{
			MsgType:   "cmd.mode.set",
			Type:      "in",
			ValueType: "string",
			Version:   "1",
		},
		fimptype.Interface{
			MsgType:   "cmd.mode.get_report",
			Type:      "in",
			ValueType: "null",
			Version:   "1",
		},
		fimptype.Interface{
			MsgType:   "evt.mode.report",
			Type:      "out",
			ValueType: "string",
			Version:   "1",
		},
		fimptype.Interface{
			MsgType:   "cmd.state.get_report",
			Type:      "in",
			ValueType: "null",
			Version:   "1",
		},
		fimptype.Interface{
			MsgType:   "evt.state.report",
			Type:      "out",
			ValueType: "string",
			Version:   "1",
		},
		fimptype.Interface{
			MsgType:   "cmd.smart_charge.set",
			Type:      "in",
			ValueType: "bool",
			Version:   "1",
		},
		fimptype.Interface{
			MsgType:   "cmd.smart_charge.get_report",
			Type:      "in",
			ValueType: "null",
			Version:   "1",
		},
		fimptype.Interface{
			MsgType:   "evt.smart_charge.report",
			Type:      "out",
			ValueType: "bool",
			Version:   "1",
		},
		fimptype.Interface{
			MsgType:   "cmd.cable_lock.set",
			Type:      "in",
			ValueType: "bool",
			Version:   "1",
		},
		fimptype.Interface{
			MsgType:   "cmd.cable_lock.get_report",
			Type:      "in",
			ValueType: "null",
			Version:   "1",
		},
		fimptype.Interface{
			MsgType:   "evt.cable_lock.report",
			Type:      "out",
			ValueType: "string",
			Version:   "1",
		},
		fimptype.Interface{
			MsgType:   "evt.error.report",
			Type:      "out",
			ValueType: "string",
			Version:   "1",
		},
		fimptype.Interface{
			MsgType:   "cmd.charge.start",
			Type:      "out",
			ValueType: "null",
			Version:   "1",
		},
		fimptype.Interface{
			MsgType:   "cmd.charge.stop",
			Type:      "out",
			ValueType: "null",
			Version:   "1",
		},
		fimptype.Interface{
			MsgType:   "cmd.current_session.get_report",
			Type:      "in",
			ValueType: "null",
			Version:   "1",
		},
		fimptype.Interface{
			MsgType:   "evt.current_session.report",
			Type:      "out",
			ValueType: "float",
			Version:   "1",
		},
	}

	props := map[string]interface{}{
		"sup_modes":  []string{"start", "stop", "pause", "resume"},
		"sup_states": []string{"available", "preparing", "charging", "paused", "finished", "unknown"},
	}

	chargePointService := fimptype.Service{
		Address:          "/rt:dev/rn:" + model.ServiceName + "/ad:" + fc.configs.InstanceAddress + "/sv:chargepoint/ad:" + addr,
		Alias:            "ev charger",
		Enabled:          true,
		Groups:           []string{"1"},
		Interfaces:       interfaces,
		Name:             "chargepoint",
		PropSetReference: "",
		Props:            props,
		Tags:             nil,
	}
	return chargePointService
}
