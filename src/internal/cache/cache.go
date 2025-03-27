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

	SetChargerState(state chargepoint.State, timestamp time.Time) bool
	SetMaxCurrent(current int64, timestamp time.Time) bool
	SetRequestedOfferedCurrent(current int64)
	SetOfferedCurrent(current int64, timestamp time.Time) bool
	SetTotalPower(power float64, timestamp time.Time) bool
	SetLifetimeEnergy(energy float64, timestamp time.Time) bool
	SetEnergySession(energy float64, timestamp time.Time) bool
	SetPhase1Current(current float64, timestamp time.Time) bool
	SetPhase2Current(current float64, timestamp time.Time) bool
	SetPhase3Current(current float64, timestamp time.Time) bool

	WaitForMaxCurrent(current int64, duration time.Duration) bool
	WaitForOfferedCurrent(current int64, duration time.Duration) bool
}

type cache struct {
	mu sync.RWMutex

	chargerID string

	chargerState            chargepoint.State
	chargerStateAt          time.Time
	maxCurrent              int64
	maxCurrentAt            time.Time
	requestedOfferedCurrent int64
	offeredCurrent          int64
	offeredCurrentAt        time.Time
	energySession           float64
	energySessionAt         time.Time
	totalPower              float64
	totalPowerAt            time.Time
	lifetimeEnergy          float64
	lifetimeEnergyAt        time.Time
	phase1Current           float64
	phase1CurrentAt         time.Time
	phase2Current           float64
	phase2CurrentAt         time.Time
	phase3Current           float64
	phase3CurrentAt         time.Time

	currentListeners map[waitGroup][]chan<- int64
}

func NewCache(chargerID string) Cache {
	return &cache{
		chargerID:        chargerID,
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

func (c *cache) SetEnergySession(energy float64, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.energySessionAt) {
		c.logOutdatedObservation("session energy", c.energySessionAt, timestamp)

		return false
	}

	c.energySession = energy
	c.energySessionAt = timestamp

	return true
}

func (c *cache) SetMaxCurrent(current int64, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.maxCurrentAt) {
		c.logOutdatedObservation("max current", c.maxCurrentAt, timestamp)

		return false
	}

	c.maxCurrent = current
	c.maxCurrentAt = timestamp

	if listeners, ok := c.currentListeners[waitGroupMaxCurrent]; ok {
		for _, c := range listeners {
			select {
			case c <- current:
			default:
				log.Warn("Unable to publish max current change")
			}
		}
	}

	return true
}

func (c *cache) SetRequestedOfferedCurrent(current int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.requestedOfferedCurrent = current
}

func (c *cache) SetOfferedCurrent(current int64, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.offeredCurrentAt) {
		c.logOutdatedObservation("offered current", c.offeredCurrentAt, timestamp)

		return false
	}

	c.offeredCurrent = current
	c.offeredCurrentAt = timestamp

	if listeners, ok := c.currentListeners[waitGroupOfferedCurrent]; ok {
		for _, c := range listeners {
			select {
			case c <- current:
			default:
				log.Warn("Unable to publish offered current change")
			}
		}
	}

	return true
}

func (c *cache) SetTotalPower(power float64, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.totalPowerAt) {
		c.logOutdatedObservation("total power", c.totalPowerAt, timestamp)

		return false
	}

	c.totalPower = power
	c.totalPowerAt = timestamp

	return true
}

func (c *cache) SetLifetimeEnergy(energy float64, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.lifetimeEnergyAt) {
		c.logOutdatedObservation("lifetime energy", c.lifetimeEnergyAt, timestamp)

		return false
	}

	if energy < c.lifetimeEnergy {
		log.
			WithField("charger_id", c.chargerID).
			WithField("old", c.lifetimeEnergy).
			WithField("new", energy).
			Warn("cache: setting lifetime energy skipped: received observation with decreased value")

		return false
	}

	c.lifetimeEnergy = energy
	c.lifetimeEnergyAt = timestamp

	return true
}

func (c *cache) SetChargerState(state chargepoint.State, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.chargerStateAt) {
		c.logOutdatedObservation("charger state", c.chargerStateAt, timestamp)

		return false
	}

	c.chargerState = state
	c.chargerStateAt = timestamp

	return true
}

func (c *cache) SetPhase1Current(current float64, timestamp time.Time) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if timestamp.Before(c.phase1CurrentAt) {
		c.logOutdatedObservation("phase 1 current", c.phase1CurrentAt, timestamp)

		return false
	}

	c.phase1Current = current
	c.phase1CurrentAt = timestamp

	return true
}

func (c *cache) SetPhase2Current(current float64, timestamp time.Time) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if timestamp.Before(c.phase2CurrentAt) {
		c.logOutdatedObservation("phase 2 current", c.phase2CurrentAt, timestamp)

		return false
	}

	c.phase2Current = current
	c.phase2CurrentAt = timestamp

	return true
}

func (c *cache) SetPhase3Current(current float64, timestamp time.Time) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if timestamp.Before(c.phase3CurrentAt) {
		c.logOutdatedObservation("phase 3 current", c.phase3CurrentAt, timestamp)

		return false
	}

	c.phase3Current = current
	c.phase3CurrentAt = timestamp

	return true
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

func (c *cache) logOutdatedObservation(operation string, oldTimestamp, newTimestamp time.Time) {
	log.WithField("charger_id", c.chargerID).
		WithField("old", oldTimestamp.Format(time.RFC3339)).
		WithField("new", newTimestamp.Format(time.RFC3339)).
		Debugf("cache: setting %s skipped: outdated observation", operation)
}
