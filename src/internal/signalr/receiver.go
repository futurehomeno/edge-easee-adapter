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
	select {
	case r.observations <- o:
	default:
		// The signalR library dispatches every hub call on its own goroutine, so blocking here
		// parks one per observation for as long as the manager is not draining - and after the
		// manager stops, forever. Dropping is the better failure: the cache is timestamp
		// guarded and the charger replays its state on the next reconnect.
		log.Warnf("signalR: observation buffer full, dropping obs='%s' chargerID=%s", o.ID.Str(), o.ChargerID)
	}
}

func (r *receiver) CommandResponse(resp any) {
	res, err := json.MarshalIndent(resp, "", "\t")

	if err != nil {
		log.Errorf("Marshal command response err: %v", err)
	} else {
		log.Debugf("Command response: %s", string(res))
	}
}
