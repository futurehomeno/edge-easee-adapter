package router

import (
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/easee-ad/model"
)

// SendChangerModeEvent sends fimp event
func (fc *FromFimpRouter) SendChangerModeEvent(chargerID string, mode string, oldMsg *fimpgo.Message) error {
	msg := fimpgo.NewStringMessage("evt.mode.report", "chargepoint", mode, nil, nil, oldMsg.Payload)
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
