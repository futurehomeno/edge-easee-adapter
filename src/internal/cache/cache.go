package cache

import (
	"sync"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	log "github.com/sirupsen/logrus"
)

// Cache is a cache for charger observations.
type Cache interface {
	// ChargerState returns the charger state.
	ChargerState() chargepoint.State
	// MaxCurrent returns the charger max current set by the user.
	MaxCurrent() int64
	// OfferedCurrent returns the desired current determined by the controller.
	OfferedCurrent() int64
	// DynamicCurrent returns the actually used current value.
	DynamicCurrent() int64
	// CableLocked returns the cable locked state.
	CableLocked() bool
	// CableCurrent returns the cable max current.
	CableCurrent() int64
	// TotalPower returns the total power.
	TotalPower() float64
	// LifetimeEnergy returns the lifetime energy.
	LifetimeEnergy() float64
	// EnergySession returns the current session energy value.
	EnergySession() float64
	// Phase1Current return current on phase 1.
	Phase1Current() float64
	// Phase2Current return current on phase 2.
	Phase2Current() float64
	// Phase3Current return current on phase 3.
	Phase3Current() float64

	SetChargerState(state chargepoint.State)
	SetMaxCurrent(current int64)
	SetOfferedCurrent(current int64)
	SetDynamicCurrent(current int64)
	SetCableLocked(locked bool)
	SetCableCurrent(current int64)
	SetTotalPower(power float64)
	SetLifetimeEnergy(energy float64)
	SetEnergySession(energy float64)
	SetPhase1Current(current float64)
	SetPhase2Current(current float64)
	SetPhase3Current(current float64)
}

type cache struct {
	mu sync.RWMutex

	chargerState   chargepoint.State
	maxCurrent     int64
	offeredCurrent int64
	dynamicCurrent int64
	energySession  float64
	cableLocked    bool
	cableCurrent   int64
	totalPower     float64
	lifetimeEnergy float64
	phase1Current  float64
	phase2Current  float64
	phase3Current  float64
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

func (c *cache) OfferedCurrent() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.offeredCurrent
}

func (c *cache) DynamicCurrent() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.dynamicCurrent
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

func (c *cache) EnergySession() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.energySession
}

func (c *cache) Phase1Current() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.phase1Current
}

func (c *cache) Phase2Current() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.phase2Current
}

func (c *cache) Phase3Current() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.phase3Current
}

func (c *cache) SetEnergySession(energy float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.energySession = energy
}

func (c *cache) SetMaxCurrent(current int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.maxCurrent = current
}

func (c *cache) SetOfferedCurrent(current int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.offeredCurrent = current
}

func (c *cache) SetDynamicCurrent(current int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.dynamicCurrent = current
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

func (c *cache) SetPhase1Current(current float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.phase1Current = current
}

func (c *cache) SetPhase2Current(current float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.phase2Current = current
}

func (c *cache) SetPhase3Current(current float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.phase3Current = current
}
