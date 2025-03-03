package signalr

import (
	"encoding/json"

	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/philippseith/signalr"
	log "github.com/sirupsen/logrus"
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
	res, _ := json.MarshalIndent(resp, "", "\t")
	log.Info("command response: ", string(res))
}
