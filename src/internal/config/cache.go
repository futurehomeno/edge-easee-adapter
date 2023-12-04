package config

import (
	"sync"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	log "github.com/sirupsen/logrus"
)

// Cache is a cache for charger observations.
type Cache interface {
	// ChargerState returns the charger state.
	ChargerState() chargepoint.State
	// MaxCurrent returns the charger max current.
	MaxCurrent() int64
	// CableLocked returns the cable locked state.
	CableLocked() bool
	// CableCurrent returns the cable max current.
	CableCurrent() int64
	// TotalPower returns the total power.
	TotalPower() float64
	// LifetimeEnergy returns the lifetime energy.
	LifetimeEnergy() float64
	// OfferedCurrent returns the current value.
	OfferedCurrent() int64
	// EnergySession returns the current session energy value.
	EnergySession() float64

	SetChargerState(state chargepoint.State)
	SetMaxCurrent(current int64)
	SetOfferedCurrent(current int64)
	SetCableLocked(locked bool)
	SetCableCurrent(current int64)
	SetTotalPower(power float64)
	SetLifetimeEnergy(energy float64)
	SetEnergySession(energy float64)
}

type cache struct {
	mu sync.RWMutex

	chargerState   chargepoint.State
	maxCurrent     int64
	offeredCurrent int64
	energySession  float64
	cableLocked    bool
	cableCurrent   int64
	totalPower     float64
	lifetimeEnergy float64
}

func (c *cache) EnergySession() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.energySession
}

func (c *cache) SetEnergySession(energy float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.energySession = energy
}

func NewCache() Cache {
	return &cache{}
}

func (c *cache) ChargerState() chargepoint.State {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.chargerState
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

func (c *cache) OfferedCurrent() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.offeredCurrent
}

func (c *cache) SetMaxCurrent(current int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.maxCurrent = current
}

func (c *cache) SetCableLocked(locked bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cableLocked = locked
}

func (c *cache) SetCableCurrent(current int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cableCurrent = current
}

func (c *cache) SetTotalPower(power float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totalPower = power
}

func (c *cache) SetLifetimeEnergy(energy float64) {
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

func (c *cache) SetChargerState(state chargepoint.State) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.chargerState = state
}

func (c *cache) SetOfferedCurrent(current int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.offeredCurrent = current
}
