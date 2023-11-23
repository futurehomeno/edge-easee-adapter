package easee

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

// ObservationCache is a cache for charger observations.
type ObservationCache interface {
	// ChargerState returns the charger state.
	ChargerState() ChargerState
	// MaxCurrent returns the charger max current.
	MaxCurrent() int64
	// CableLocked returns the cable locked state.
	CableLocked() bool
	// CableCurrent returns the cable max current.
	CableCurrent() int64
	// SessionEnergy returns the session energy.
	SessionEnergy() float64
	// TotalPower returns the total power.
	TotalPower() float64
	// LifetimeEnergy returns the lifetime energy.
	LifetimeEnergy() float64

	setChargerState(state ChargerState)
	setMaxCurrent(current int64)
	setCableLocked(locked bool)
	setCableCurrent(current int64)
	setSessionEnergy(energy float64)

	setTotalPower(power float64)
	setLifetimeEnergy(energy float64)
}

type cache struct {
	mu sync.RWMutex

	chargerState   ChargerState
	maxCurrent     int64
	cableLocked    bool
	cableCurrent   int64
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

func (c *cache) MaxCurrent() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.maxCurrent
}

func (c *cache) CableLocked() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cableLocked
}

func (c *cache) CableCurrent() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cableCurrent
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

func (c *cache) setMaxCurrent(current int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.maxCurrent = current
}

func (c *cache) setCableLocked(locked bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cableLocked = locked
}

func (c *cache) setCableCurrent(current int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cableCurrent = current
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
