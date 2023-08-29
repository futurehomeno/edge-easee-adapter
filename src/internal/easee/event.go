package easee

import (
	"github.com/futurehomeno/cliffhanger/adapter"

	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
)

type ObservationHandler struct {
	thing      adapter.Thing
	cache      Cache
	strategies map[signalr.ObservationID]func()

	chargerID string
}

func NewObservationHandler(chargerID string, thing adapter.Thing, cache Cache) *ObservationHandler {
	h := &ObservationHandler{
		thing:      thing,
		cache:      cache,
		chargerID:  chargerID,
		strategies: make(map[signalr.ObservationID]func()),
	}

	h.strategies[signalr.ChargerOPState] = h.handleChargerState
	h.strategies[signalr.SessionEnergy] = h.handleChargerState // TODO and below...
	h.strategies[signalr.CableLocked] = h.handleChargerState
	h.strategies[signalr.TotalPower] = h.handleChargerState
	h.strategies[signalr.LifetimeEnergy] = h.handleChargerState

	return h
}

func (h *ObservationHandler) Handle(obs signalr.Observation) {

}

func (h *ObservationHandler) handleChargerState() {

}
