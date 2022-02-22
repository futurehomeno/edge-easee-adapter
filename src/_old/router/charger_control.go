package router

import (
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/_old/model"
)

// SendChangerModeEvent sends fimp event
func (fc *FromFimpRouter) SendChangerStateEvent(chargerID string, state string, oldMsg *fimpgo.Message) error {
	msg := fimpgo.NewStringMessage("evt.state.report", "chargepoint", state, nil, nil, oldMsg.Payload)
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
		log.Debug("Err in SendChargerStateEvent: ", err)

		return err
	}

	return err
}
