package signalr

import (
	"encoding/json"
	"sync/atomic"

	"github.com/philippseith/signalr"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

type receiver struct {
	signalr.Receiver

	observations chan<- model.Observation

	// Set while the buffer is full, so a streak of drops warns once instead of once per
	// observation. Written from the library's per-call goroutines, hence atomic.
	dropping atomic.Bool
}

func newReceiver(observations chan<- model.Observation) *receiver {
	return &receiver{
		observations: observations,
	}
}

func (r *receiver) ProductUpdate(o model.Observation) {
	select {
	case r.observations <- o:
		r.dropping.Store(false)
	default:
		// The signalR library dispatches every hub call on its own goroutine, so blocking here
		// parks one per observation for as long as the manager is not draining - and after the
		// manager stops, forever. Dropping is the lesser failure, not a free one: most
		// observations are cache refreshes the charger re-sends on the next reconnect, but the
		// session start/stop pair is edge-triggered and its record is simply lost. The buffer
		// only fills while the run loop is stalled, which today means a subscribe invoke
		// blocking it for up to SignalRInvokeTimeout - so a line per dropped observation would
		// put synchronous logging on the overload path. First of the streak only.
		if !r.dropping.Swap(true) {
			log.Warnf("signalR: observation buffer full, dropping obs='%s' chargerID=%s", o.ID.Str(), o.ChargerID)
		}
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
