package cache

import (
	"slices"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	log "github.com/sirupsen/logrus"
)

// Cache is a cache for charger observations.
type Cache interface {
	// ChargerState returns the charger state.
	ChargerState() chargepoint.State
	// MaxCurrent returns the charger max current set by the user.
	MaxCurrent() int64
	// RequestedOfferedCurrent returns the offered current requested by controller.
	RequestedOfferedCurrent() int64
	// OfferedCurrent returns the current accepted by evse.
	OfferedCurrent() int64
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
	SetRequestedOfferedCurrent(current int64)
	SetOfferedCurrent(current int64)
	SetTotalPower(power float64)
	SetLifetimeEnergy(energy float64)
	SetEnergySession(energy float64)
	SetPhase1Current(current float64)
	SetPhase2Current(current float64)
	SetPhase3Current(current float64)

	WaitForMaxCurrent(current int64, duration time.Duration) bool
	WaitForOfferedCurrent(current int64, duration time.Duration) bool
}

type cache struct {
	mu sync.RWMutex

	chargerState            chargepoint.State
	maxCurrent              int64
	requestedOfferedCurrent int64
	offeredCurrent          int64
	energySession           float64
	totalPower              float64
	lifetimeEnergy          float64
	phase1Current           float64
	phase2Current           float64
	phase3Current           float64

	currentListeners map[waitGroup][]chan<- int64
}

func NewCache() Cache {
	return &cache{
		currentListeners: make(map[waitGroup][]chan<- int64),
	}
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

func (c *cache) RequestedOfferedCurrent() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.requestedOfferedCurrent
}

func (c *cache) OfferedCurrent() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.offeredCurrent
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

	if listeners, ok := c.currentListeners[waitGroupMaxCurrent]; ok {
		for _, c := range listeners {
			select {
			case c <- current:
			default:
				log.Warn("Unable to publish max current change")
			}
		}
	}
}

func (c *cache) SetRequestedOfferedCurrent(current int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.requestedOfferedCurrent = current
}

func (c *cache) SetOfferedCurrent(current int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.offeredCurrent = current

	if listeners, ok := c.currentListeners[waitGroupOfferedCurrent]; ok {
		for _, c := range listeners {
			select {
			case c <- current:
			default:
				log.Warn("Unable to publish offered current change")
			}
		}
	}
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

type waitGroup int

const (
	waitGroupMaxCurrent waitGroup = iota
	waitGroupOfferedCurrent
)

func (c *cache) WaitForMaxCurrent(current int64, duration time.Duration) bool {
	return c.waitForCurrent(waitGroupMaxCurrent, current, duration)
}

func (c *cache) WaitForOfferedCurrent(current int64, duration time.Duration) bool {
	return c.waitForCurrent(waitGroupOfferedCurrent, current, duration)
}

func (c *cache) waitForCurrent(group waitGroup, current int64, duration time.Duration) bool {
	c.mu.Lock()

	var value int64

	switch group {
	case waitGroupMaxCurrent:
		value = c.maxCurrent
	case waitGroupOfferedCurrent:
		value = c.offeredCurrent
	default:
		log.Warnf("invalid waitGroup: %v", group)
		c.mu.Unlock()

		return false
	}

	if current == value {
		c.mu.Unlock()

		return true
	}

	channel := make(chan int64, 1)
	c.currentListeners[group] = append(c.currentListeners[group], channel)
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		defer c.mu.Unlock()

		close(channel)

		c.currentListeners[group] = slices.DeleteFunc(c.currentListeners[group], func(c chan<- int64) bool {
			return c == channel
		})
	}()

	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case v := <-channel:
			if v == current {
				return true
			}
		case <-timer.C:
			return false
		}
	}
}
