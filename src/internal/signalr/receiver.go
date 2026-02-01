package signalr

import (
	"encoding/json"

	"github.com/philippseith/signalr"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

type receiver struct {
	signalr.Receiver

	observations chan<- model.Observation
}

func newReceiver(observations chan<- model.Observation) *receiver {
	return &receiver{
		observations: observations,
	}
}

func (r *receiver) ProductUpdate(o model.Observation) {
	r.observations <- o
}

func (r *receiver) CommandResponse(resp any) {
	res, err := json.MarshalIndent(resp, "", "\t")

	if err != nil {
		log.Errorf("Marshal command response err: %v", err)
	} else {
		log.Debugf("Command response: %s", string(res))
	}
}
