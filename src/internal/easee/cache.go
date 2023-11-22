package easee

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

// ObservationCache is a cache for charger observations.
type ObservationCache interface {
	// ChargerState returns the charger state.
	ChargerState() ChargerState
	// SessionEnergy returns the session energy.
	SessionEnergy() float64
	// CableLocked returns the cable locked state.
	CableLocked() bool
	// TotalPower returns the total power.
	TotalPower() float64
	// LifetimeEnergy returns the lifetime energy.
	LifetimeEnergy() float64

	setChargerState(state ChargerState)
	setSessionEnergy(energy float64)
	setCableLocked(locked bool)
	setTotalPower(power float64)
	setLifetimeEnergy(energy float64)
}

type cache struct {
	mu sync.RWMutex

	chargerState   ChargerState
	cableLocked    bool
	sessionEnergy  float64
	totalPower     float64
	lifetimeEnergy float64
}

func NewObservationCache() ObservationCache {
	return &cache{}
}

func (c *cache) ChargerState() ChargerState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.chargerState
}

func (c *cache) SessionEnergy() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.sessionEnergy
}

func (c *cache) CableLocked() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cableLocked
}

func (c *cache) TotalPower() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.totalPower
}

func (c *cache) LifetimeEnergy() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lifetimeEnergy
}

func (c *cache) setSessionEnergy(energy float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sessionEnergy = energy
}

func (c *cache) setCableLocked(locked bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cableLocked = locked
}

func (c *cache) setTotalPower(power float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totalPower = power
}

func (c *cache) setLifetimeEnergy(energy float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if energy < c.lifetimeEnergy {
		log.
			WithField("old", c.lifetimeEnergy).
			WithField("new", energy).
			Warn("lifetime energy decreased!")

		return
	}

	c.lifetimeEnergy = energy
}

func (c *cache) setChargerState(state ChargerState) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.chargerState = state
}
