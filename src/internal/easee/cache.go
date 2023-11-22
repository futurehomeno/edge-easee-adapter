package easee

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

// ObservationCache is a cache for charger observations.
type ObservationCache interface {
	// ChargerState returns the charger state.
	ChargerState() (ChargerState, error)
	// SessionEnergy returns the session energy.
	SessionEnergy() (float64, error)
	// CableLocked returns the cable locked state.
	CableLocked() (bool, error)
	// TotalPower returns the total power.
	TotalPower() (float64, error)
	// LifetimeEnergy returns the lifetime energy.
	LifetimeEnergy() (float64, error)

	setChargerState(state ChargerState)
	setSessionEnergy(energy float64)
	setCableLocked(locked bool)
	setTotalPower(power float64)
	setLifetimeEnergy(energy float64)

	isConnected() bool
	setConnected(connected bool)
}

type cache struct {
	mu sync.RWMutex

	connected bool

	chargerState   ChargerState
	cableLocked    bool
	sessionEnergy  float64
	totalPower     float64
	lifetimeEnergy float64
}

func NewObservationCache() ObservationCache {
	return &cache{}
}

func (c *cache) ChargerState() (ChargerState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return Error, errNotConnected
	}

	return c.chargerState, nil
}

func (c *cache) SessionEnergy() (float64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return 0, errNotConnected
	}

	return c.sessionEnergy, nil
}

func (c *cache) CableLocked() (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return false, errNotConnected
	}

	return c.cableLocked, nil
}

func (c *cache) TotalPower() (float64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return 0, errNotConnected
	}

	return c.totalPower, nil
}

func (c *cache) LifetimeEnergy() (float64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return 0, errNotConnected
	}

	return c.lifetimeEnergy, nil
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

func (c *cache) isConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.connected
}

func (c *cache) setConnected(connected bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.connected = connected
}
