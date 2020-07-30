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
	msg := fimpgo.NewMessage("evt.thing.exclusion_report", model.ServiceName, fimpgo.VTypeObject, val, nil, nil, oldMsg)
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
