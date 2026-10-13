package cache

import (
	"slices"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/types"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

// Cache holds the latest observation per value, each with the timestamp it was observed at.
type Cache interface {
	PhaseMode() (int, time.Time)
	ChargerState() (chargepoint.State, time.Time)
	// MaxCurrent is the ceiling set by the user.
	MaxCurrent() (int, time.Time)
	// RequestedOfferedCurrent is what the controller last asked for.
	RequestedOfferedCurrent() (int, time.Time)
	// OfferedCurrent is what the evse accepted.
	OfferedCurrent() (int, time.Time)
	TotalPower() (float64, time.Time)
	LifetimeEnergy() (float64, time.Time)
	EnergySession() (float64, time.Time)
	Phase1Current() (float64, time.Time)
	Phase2Current() (float64, time.Time)
	Phase3Current() (float64, time.Time)
	OutputPhaseType() (types.PhaseMode, time.Time)
	// RequestedPhaseMode is what the controller last asked for.
	RequestedPhaseMode() (types.PhaseMode, time.Time)
	GridType() (types.GridType, time.Time)
	Phases() (int, time.Time)
	CableLocked() (bool, time.Time)
	// CableCurrent is the cable's max current.
	CableCurrent() (int, time.Time)
	CableAlwaysLocked() (bool, time.Time)
	AlarmActive(event string) bool

	SetPhaseMode(mode int, timestamp time.Time) bool
	SetChargerState(state chargepoint.State, timestamp time.Time) bool
	SetMaxCurrent(current int, timestamp time.Time) bool
	SetRequestedOfferedCurrent(current int, timestamp time.Time) bool
	SetOfferedCurrent(current int, timestamp time.Time) bool
	SetTotalPower(power float64, timestamp time.Time) bool
	SetLifetimeEnergy(energy float64, timestamp time.Time) bool
	SetOutputPhaseType(mode types.PhaseMode, timestamp time.Time) bool
	SetRequestedPhaseMode(mode types.PhaseMode, timestamp time.Time) bool
	SetInstallationParameters(gridType types.GridType, phases int, timestamp time.Time) bool
	SetCableLocked(locked bool, timestamp time.Time) bool
	SetCableCurrent(current int, timestamp time.Time) bool
	SetCableAlwaysLocked(alwaysLocked bool, timestamp time.Time) bool
	// SeedCableAlwaysLocked stores an optimistic local value without the ordering guard, so
	// the next observation — whatever its clock — still wins.
	SeedCableAlwaysLocked(alwaysLocked bool)
	// SetAlarm stores the state of an alarm event and reports whether the observation was accepted.
	SetAlarm(event string, active bool, timestamp time.Time) bool
	SetEnergySession(energy float64, timestamp time.Time) bool
	SetPhase1Current(current float64, timestamp time.Time) bool
	SetPhase2Current(current float64, timestamp time.Time) bool
	SetPhase3Current(current float64, timestamp time.Time) bool

	WaitForMaxCurrent(current int, duration time.Duration) bool
	WaitForOfferedCurrent(current int, duration time.Duration) bool
}

type cache struct {
	mu sync.RWMutex

	chargerID string

	requestedOfferedCurrent model.TimestampedValue[int]
	chargerState            model.TimestampedValue[chargepoint.State]
	phaseMode               model.TimestampedValue[int]
	maxCurrent              model.TimestampedValue[int]
	offeredCurrent          model.TimestampedValue[int]
	energySession           model.TimestampedValue[float64]
	totalPower              model.TimestampedValue[float64]
	lifetimeEnergy          model.TimestampedValue[float64]
	phase1Current           model.TimestampedValue[float64]
	phase2Current           model.TimestampedValue[float64]
	phase3Current           model.TimestampedValue[float64]
	outputPhase             model.TimestampedValue[types.PhaseMode]
	requestedPhaseMode      model.TimestampedValue[types.PhaseMode]
	gridType                model.TimestampedValue[types.GridType]
	phases                  model.TimestampedValue[int]
	cableLocked             model.TimestampedValue[bool]
	cableCurrent            model.TimestampedValue[int]
	cableAlwaysLocked       model.TimestampedValue[bool]

	activeAlarms map[string]model.TimestampedValue[bool]

	currentListeners map[waitGroup][]chan<- int
}

func NewCache(chargerID string) Cache {
	return &cache{
		chargerID:        chargerID,
		activeAlarms:     make(map[string]model.TimestampedValue[bool]),
		currentListeners: make(map[waitGroup][]chan<- int),
	}
}

// value reads a timestamped field under the shared read lock.
func value[T any](c *cache, v *model.TimestampedValue[T]) (T, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return v.Value, v.Timestamp
}

// store rejects an observation older than the one already held, so a replayed or out-of-order
// SignalR message cannot overwrite fresher state.
func store[T any](c *cache, dst *model.TimestampedValue[T], name string, v T, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(dst.Timestamp) {
		c.logOutdatedObservation(name, dst.Timestamp, timestamp)

		return false
	}

	*dst = model.TimestampedValue[T]{Value: v, Timestamp: timestamp}

	return true
}

func (c *cache) PhaseMode() (int, time.Time) { return value(c, &c.phaseMode) }

func (c *cache) OutputPhaseType() (types.PhaseMode, time.Time) { return value(c, &c.outputPhase) }

func (c *cache) RequestedPhaseMode() (types.PhaseMode, time.Time) {
	return value(c, &c.requestedPhaseMode)
}

func (c *cache) ChargerState() (chargepoint.State, time.Time) { return value(c, &c.chargerState) }

func (c *cache) MaxCurrent() (int, time.Time) { return value(c, &c.maxCurrent) }

func (c *cache) RequestedOfferedCurrent() (int, time.Time) {
	return value(c, &c.requestedOfferedCurrent)
}

func (c *cache) OfferedCurrent() (int, time.Time) { return value(c, &c.offeredCurrent) }

func (c *cache) TotalPower() (float64, time.Time) { return value(c, &c.totalPower) }

func (c *cache) LifetimeEnergy() (float64, time.Time) { return value(c, &c.lifetimeEnergy) }

func (c *cache) EnergySession() (float64, time.Time) { return value(c, &c.energySession) }

func (c *cache) Phase1Current() (float64, time.Time) { return value(c, &c.phase1Current) }

func (c *cache) Phase2Current() (float64, time.Time) { return value(c, &c.phase2Current) }

func (c *cache) Phase3Current() (float64, time.Time) { return value(c, &c.phase3Current) }

func (c *cache) GridType() (types.GridType, time.Time) { return value(c, &c.gridType) }

func (c *cache) Phases() (int, time.Time) { return value(c, &c.phases) }

func (c *cache) CableLocked() (bool, time.Time) { return value(c, &c.cableLocked) }

func (c *cache) CableCurrent() (int, time.Time) { return value(c, &c.cableCurrent) }

func (c *cache) CableAlwaysLocked() (bool, time.Time) { return value(c, &c.cableAlwaysLocked) }

func (c *cache) AlarmActive(event string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.activeAlarms[event].Value
}

func (c *cache) SetAlarm(event string, active bool, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if prev := c.activeAlarms[event]; timestamp.Before(prev.Timestamp) {
		c.logOutdatedObservation("alarm "+event, prev.Timestamp, timestamp)

		return false
	}

	c.activeAlarms[event] = model.TimestampedValue[bool]{
		Value:     active,
		Timestamp: timestamp,
	}

	return true
}

func (c *cache) SetCableAlwaysLocked(alwaysLocked bool, timestamp time.Time) bool {
	return store(c, &c.cableAlwaysLocked, "cable always locked", alwaysLocked, timestamp)
}

func (c *cache) SeedCableAlwaysLocked(alwaysLocked bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cableAlwaysLocked = model.TimestampedValue[bool]{Value: alwaysLocked}
}

func (c *cache) SetCableLocked(locked bool, timestamp time.Time) bool {
	return store(c, &c.cableLocked, "cable locked", locked, timestamp)
}

func (c *cache) SetCableCurrent(current int, timestamp time.Time) bool {
	return store(c, &c.cableCurrent, "cable current", current, timestamp)
}

func (c *cache) SetPhaseMode(phaseMode int, timestamp time.Time) bool {
	return store(c, &c.phaseMode, "phase mode", phaseMode, timestamp)
}

func (c *cache) SetOutputPhaseType(mode types.PhaseMode, timestamp time.Time) bool {
	return store(c, &c.outputPhase, "output phase", mode, timestamp)
}

func (c *cache) SetRequestedPhaseMode(mode types.PhaseMode, timestamp time.Time) bool {
	return store(c, &c.requestedPhaseMode, "requested phase mode", mode, timestamp)
}

func (c *cache) SetEnergySession(energy float64, timestamp time.Time) bool {
	return store(c, &c.energySession, "session energy", energy, timestamp)
}

func (c *cache) SetMaxCurrent(current int, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.maxCurrent.Timestamp) {
		c.logOutdatedObservation("max current", c.maxCurrent.Timestamp, timestamp)

		return false
	}

	c.maxCurrent = model.TimestampedValue[int]{
		Value:     current,
		Timestamp: timestamp,
	}

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

func (c *cache) SetRequestedOfferedCurrent(current int, timestamp time.Time) bool {
	return store(c, &c.requestedOfferedCurrent, "requested offered current", current, timestamp)
}

func (c *cache) SetOfferedCurrent(current int, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.offeredCurrent.Timestamp) {
		c.logOutdatedObservation("offered current", c.offeredCurrent.Timestamp, timestamp)

		return false
	}

	c.offeredCurrent = model.TimestampedValue[int]{
		Value:     current,
		Timestamp: timestamp,
	}

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
	return store(c, &c.totalPower, "total power", power, timestamp)
}

func (c *cache) SetLifetimeEnergy(energy float64, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.lifetimeEnergy.Timestamp) {
		c.logOutdatedObservation("lifetime energy", c.lifetimeEnergy.Timestamp, timestamp)

		return false
	}

	if energy < c.lifetimeEnergy.Value {
		log.
			WithField("charger_id", c.chargerID).
			WithField("old", c.lifetimeEnergy).
			WithField("new", energy).
			Warn("cache: setting lifetime energy skipped: received observation with decreased value")

		return false
	}

	c.lifetimeEnergy = model.TimestampedValue[float64]{
		Value:     energy,
		Timestamp: timestamp,
	}

	return true
}

func (c *cache) SetChargerState(state chargepoint.State, timestamp time.Time) bool {
	return store(c, &c.chargerState, "charger state", state, timestamp)
}

func (c *cache) SetPhase1Current(current float64, timestamp time.Time) bool {
	return store(c, &c.phase1Current, "phase 1 current", current, timestamp)
}

func (c *cache) SetPhase2Current(current float64, timestamp time.Time) bool {
	return store(c, &c.phase2Current, "phase 2 current", current, timestamp)
}

func (c *cache) SetPhase3Current(current float64, timestamp time.Time) bool {
	return store(c, &c.phase3Current, "phase 3 current", current, timestamp)
}

func (c *cache) SetInstallationParameters(gridType types.GridType, phases int, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.gridType.Timestamp) {
		c.logOutdatedObservation("grid type", c.gridType.Timestamp, timestamp)

		return false
	}

	if timestamp.Before(c.phases.Timestamp) {
		c.logOutdatedObservation("phases", c.phases.Timestamp, timestamp)

		return false
	}

	c.gridType = model.TimestampedValue[types.GridType]{
		Value:     gridType,
		Timestamp: timestamp,
	}
	c.phases = model.TimestampedValue[int]{
		Value:     phases,
		Timestamp: timestamp,
	}

	return true
}

type waitGroup int

const (
	waitGroupMaxCurrent waitGroup = iota
	waitGroupOfferedCurrent
)

func (c *cache) WaitForMaxCurrent(current int, duration time.Duration) bool {
	return c.waitForCurrent(waitGroupMaxCurrent, current, duration)
}

func (c *cache) WaitForOfferedCurrent(current int, duration time.Duration) bool {
	return c.waitForCurrent(waitGroupOfferedCurrent, current, duration)
}

// currentOf returns the current tracked by a wait group. The caller must hold c.mu.
func (c *cache) currentOf(group waitGroup) (int, bool) {
	switch group {
	case waitGroupMaxCurrent:
		return c.maxCurrent.Value, true
	case waitGroupOfferedCurrent:
		return c.offeredCurrent.Value, true
	default:
		return 0, false
	}
}

func (c *cache) waitForCurrent(group waitGroup, current int, duration time.Duration) bool {
	c.mu.Lock()

	value, ok := c.currentOf(group)
	if !ok {
		log.Warnf("Invalid waitGroup: %v", group)
		c.mu.Unlock()

		return false
	}

	if current == value {
		c.mu.Unlock()

		return true
	}

	channel := make(chan int, 1)
	c.currentListeners[group] = append(c.currentListeners[group], channel)
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		defer c.mu.Unlock()

		close(channel)

		c.currentListeners[group] = slices.DeleteFunc(c.currentListeners[group], func(c chan<- int) bool {
			return c == channel
		})
	}()

	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case v := <-channel:
			// The delivered value is checked first: a confirmation immediately superseded by
			// another observation is gone from the cache by the time this runs, and the
			// notifiers drop on a full buffer, so re-reading alone would wait for a
			// notification that never comes.
			if v == current {
				return true
			}

			// Conversely the delivered value can be the stale one - two observations in quick
			// succession deliver the first and drop the second - so the cache still decides.
			c.mu.RLock()
			value, _ := c.currentOf(group)
			c.mu.RUnlock()

			if value == current {
				return true
			}
		case <-timer.C:
			return false
		}
	}
}

func (c *cache) logOutdatedObservation(operation string, oldTimestamp, newTimestamp time.Time) {
	// Called under the write lock from every setter; formatting two timestamps for a line the
	// hub discards would extend the critical section on every replayed observation.
	if !log.IsLevelEnabled(log.DebugLevel) {
		return
	}

	log.WithField("charger_id", c.chargerID).
		WithField("old", oldTimestamp.Format(time.RFC3339)).
		WithField("new", newTimestamp.Format(time.RFC3339)).
		Debugf("cache: setting %s skipped: outdated observation", operation)
}
