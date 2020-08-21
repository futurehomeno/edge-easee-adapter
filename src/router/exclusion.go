package router

import (
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/easee-ad/model"
)

// SendExclusionReport sends exclusion report for one charger
func (fc *FromFimpRouter) SendExclusionReport(chargerID string, oldMsg *fimpgo.FimpMessage) error {
	val := map[string]interface{}{
		"address": chargerID,
	}
	var oldPayload *fimpgo.FimpMessage
	if oldMsg != nil {
		oldPayload = oldMsg
	}
	msg := fimpgo.NewMessage("evt.thing.exclusion_report", model.ServiceName, fimpgo.VTypeObject, val, nil, nil, oldPayload)
	msg.Source = model.ServiceName
	addr := fimpgo.Address{
		MsgType:         fimpgo.MsgTypeEvt,
		ResourceType:    fimpgo.ResourceTypeAdapter,
		ResourceName:    model.ServiceName,
		ResourceAddress: fc.configs.InstanceAddress,
	}
	err := fc.mqt.Publish(&addr, msg)
	if err != nil {
		log.Debug(err)
		return err
	}
	return nil
}

// SendExclusionReportForAllChargers sends a report for all chargers
func (fc *FromFimpRouter) SendExclusionReportForAllChargers() {
	for _, product := range fc.easee.Products {
		err := fc.SendExclusionReport(product.Charger.ID, nil)
		if err != nil {
			log.Error(err)
		}
	}
}
