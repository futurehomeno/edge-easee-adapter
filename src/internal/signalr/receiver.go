package signalr

import (
	"encoding/json"

	"github.com/philippseith/signalr"
	log "github.com/sirupsen/logrus"
)

type receiver struct {
	signalr.Receiver

	observations chan Observation
}

func newReceiver() *receiver {
	return &receiver{
		observations: make(chan Observation, 100),
	}
}

func (r *receiver) ProductUpdate(o Observation) {
	r.observations <- o
}

func (r *receiver) CommandResponse(resp any) {
	res, _ := json.MarshalIndent(resp, "", "\t")
	log.Info("command response: ", string(res))
}

func (r *receiver) observationC() <-chan Observation {
	return r.observations
}
