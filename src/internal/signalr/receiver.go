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
		// manager stops, forever. Dropping is the lesser failure, not a free one: most
		// observations are cache refreshes the charger re-sends on the next reconnect, but the
		// session start/stop pair is edge-triggered and its record is simply lost. The buffer
		// only fills while the run loop is stalled, which today means a subscribe invoke
		// blocking it for up to SignalRInvokeTimeout.
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
