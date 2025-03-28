package cache

import (
	"slices"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

// Cache is a cache for charger observations.
type Cache interface {
	// PhaseMode returns the charger phase mode.
	PhaseMode() (int, time.Time)
	// ChargerState returns the charger state.
	ChargerState() (chargepoint.State, time.Time)
	// MaxCurrent returns the charger max current set by the user.
	MaxCurrent() (int64, time.Time)
	// RequestedOfferedCurrent returns the offered current requested by controller.
	RequestedOfferedCurrent() (int64, time.Time)
	// OfferedCurrent returns the current accepted by evse.
	OfferedCurrent() (int64, time.Time)
	// TotalPower returns the total power.
	TotalPower() (float64, time.Time)
	// LifetimeEnergy returns the lifetime energy.
	LifetimeEnergy() (float64, time.Time)
	// EnergySession returns the current session energy value.
	EnergySession() (float64, time.Time)
	// Phase1Current return current on phase 1.
	Phase1Current() (float64, time.Time)
	// Phase2Current return current on phase 2.
	Phase2Current() (float64, time.Time)
	// Phase3Current return current on phase 3.
	Phase3Current() (float64, time.Time)
	// OutputPhaseType return output phase type.
	OutputPhaseType() (chargepoint.PhaseMode, time.Time)
	// GridType return GridType.
	GridType() (chargepoint.GridType, time.Time)
	// Phases return phases.
	Phases() (int, time.Time)
	// CableLocked returns the cable locked state.
	CableLocked() (bool, time.Time)
	// CableCurrent returns the cable max current.
	CableCurrent() (*int64, time.Time)
	// CableAlwaysLocked returns state of cable always locked parameter.
	CableAlwaysLocked() (bool, time.Time)

	SetPhaseMode(mode int, timestamp time.Time) bool
	SetChargerState(state chargepoint.State, timestamp time.Time) bool
	SetMaxCurrent(current int64, timestamp time.Time) bool
	SetRequestedOfferedCurrent(current int64, timestamp time.Time) bool
	SetOfferedCurrent(current int64, timestamp time.Time) bool
	SetTotalPower(power float64, timestamp time.Time) bool
	SetLifetimeEnergy(energy float64, timestamp time.Time) bool
	SetOutputPhaseType(mode chargepoint.PhaseMode, timestamp time.Time) bool
	SetInstallationParameters(gridType chargepoint.GridType, phases int, timestamp time.Time) bool
	SetCableLocked(locked bool, timestamp time.Time) bool
	SetCableCurrent(current int64, timestamp time.Time) bool
	SetCableAlwaysLocked(alwaysLocked bool, timestamp time.Time) bool
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

	requestedOfferedCurrent model.TimestampedValue[int64]
	chargerState            model.TimestampedValue[chargepoint.State]
	phaseMode               model.TimestampedValue[int]
	maxCurrent              model.TimestampedValue[int64]
	offeredCurrent          model.TimestampedValue[int64]
	energySession           model.TimestampedValue[float64]
	totalPower              model.TimestampedValue[float64]
	lifetimeEnergy          model.TimestampedValue[float64]
	phase1Current           model.TimestampedValue[float64]
	phase2Current           model.TimestampedValue[float64]
	phase3Current           model.TimestampedValue[float64]
	outputPhase             model.TimestampedValue[chargepoint.PhaseMode]
	gridType                model.TimestampedValue[chargepoint.GridType]
	phases                  model.TimestampedValue[int]
	cableLocked             model.TimestampedValue[bool]
	cableCurrent            model.TimestampedValue[*int64]
	cableAlwaysLocked       model.TimestampedValue[bool]

	currentListeners map[waitGroup][]chan<- int64
}

func NewCache(chargerID string) Cache {
	return &cache{
		chargerID:        chargerID,
		currentListeners: make(map[waitGroup][]chan<- int64),
	}
}

func (c *cache) PhaseMode() (int, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.phaseMode.Value, c.phaseMode.Timestamp
}

func (c *cache) OutputPhaseType() (chargepoint.PhaseMode, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.outputPhase.Value, c.outputPhase.Timestamp
}

func (c *cache) ChargerState() (chargepoint.State, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.chargerState.Value, c.chargerState.Timestamp
}

func (c *cache) MaxCurrent() (int64, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.maxCurrent.Value, c.maxCurrent.Timestamp
}

func (c *cache) RequestedOfferedCurrent() (int64, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.requestedOfferedCurrent.Value, c.requestedOfferedCurrent.Timestamp
}

func (c *cache) OfferedCurrent() (int64, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.offeredCurrent.Value, c.offeredCurrent.Timestamp
}

func (c *cache) TotalPower() (float64, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.totalPower.Value, c.totalPower.Timestamp
}

func (c *cache) LifetimeEnergy() (float64, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lifetimeEnergy.Value, c.lifetimeEnergy.Timestamp
}

func (c *cache) EnergySession() (float64, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.energySession.Value, c.energySession.Timestamp
}

func (c *cache) Phase1Current() (float64, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.phase1Current.Value, c.phase1Current.Timestamp
}

func (c *cache) Phase2Current() (float64, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.phase2Current.Value, c.phase2Current.Timestamp
}

func (c *cache) Phase3Current() (float64, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.phase3Current.Value, c.phase3Current.Timestamp
}

func (c *cache) GridType() (chargepoint.GridType, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.gridType.Value, c.gridType.Timestamp
}

func (c *cache) Phases() (int, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.phases.Value, c.phases.Timestamp
}

func (c *cache) CableLocked() (bool, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cableLocked.Value, c.cableLocked.Timestamp
}

func (c *cache) CableCurrent() (*int64, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cableCurrent.Value, c.cableCurrent.Timestamp
}

func (c *cache) CableAlwaysLocked() (bool, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cableAlwaysLocked.Value, c.cableAlwaysLocked.Timestamp
}

func (c *cache) SetCableAlwaysLocked(alwaysLocked bool, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.cableAlwaysLocked.Timestamp) {
		c.logOutdatedObservation("cable always locked", c.cableAlwaysLocked.Timestamp, timestamp)

		return false
	}

	c.cableAlwaysLocked = model.TimestampedValue[bool]{
		Value:     alwaysLocked,
		Timestamp: timestamp,
	}

	return true
}

func (c *cache) SetCableLocked(locked bool, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.cableLocked.Timestamp) {
		c.logOutdatedObservation("cable locked", c.cableLocked.Timestamp, timestamp)

		return false
	}

	c.cableLocked = model.TimestampedValue[bool]{
		Value:     locked,
		Timestamp: timestamp,
	}

	return true
}

func (c *cache) SetCableCurrent(current int64, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.cableCurrent.Timestamp) {
		c.logOutdatedObservation("cable current", c.cableCurrent.Timestamp, timestamp)

		return false
	}

	c.cableCurrent = model.TimestampedValue[*int64]{
		Value:     &current,
		Timestamp: timestamp,
	}

	return true
}

func (c *cache) SetPhaseMode(phaseMode int, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.phaseMode.Timestamp) {
		c.logOutdatedObservation("phase mode", c.phaseMode.Timestamp, timestamp)

		return false
	}

	c.phaseMode = model.TimestampedValue[int]{
		Value:     phaseMode,
		Timestamp: timestamp,
	}

	return true
}

func (c *cache) SetOutputPhaseType(mode chargepoint.PhaseMode, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.outputPhase.Timestamp) {
		c.logOutdatedObservation("output phase", c.outputPhase.Timestamp, timestamp)

		return false
	}

	c.outputPhase = model.TimestampedValue[chargepoint.PhaseMode]{
		Value:     mode,
		Timestamp: timestamp,
	}

	return true
}

func (c *cache) SetEnergySession(energy float64, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.energySession.Timestamp) {
		c.logOutdatedObservation("session energy", c.energySession.Timestamp, timestamp)

		return false
	}

	c.energySession = model.TimestampedValue[float64]{
		Value:     energy,
		Timestamp: timestamp,
	}

	return true
}

func (c *cache) SetMaxCurrent(current int64, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.maxCurrent.Timestamp) {
		c.logOutdatedObservation("max current", c.maxCurrent.Timestamp, timestamp)

		return false
	}

	c.maxCurrent = model.TimestampedValue[int64]{
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

func (c *cache) SetRequestedOfferedCurrent(current int64, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.requestedOfferedCurrent.Timestamp) {
		c.logOutdatedObservation("requested offered current", c.requestedOfferedCurrent.Timestamp, timestamp)

		return false
	}

	c.requestedOfferedCurrent = model.TimestampedValue[int64]{
		Value:     current,
		Timestamp: timestamp,
	}

	return true
}

func (c *cache) SetOfferedCurrent(current int64, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.offeredCurrent.Timestamp) {
		c.logOutdatedObservation("offered current", c.offeredCurrent.Timestamp, timestamp)

		return false
	}

	c.offeredCurrent = model.TimestampedValue[int64]{
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
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.totalPower.Timestamp) {
		c.logOutdatedObservation("total power", c.totalPower.Timestamp, timestamp)

		return false
	}

	c.totalPower = model.TimestampedValue[float64]{
		Value:     power,
		Timestamp: timestamp,
	}

	return true
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
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.chargerState.Timestamp) {
		c.logOutdatedObservation("charger state", c.chargerState.Timestamp, timestamp)

		return false
	}

	c.chargerState = model.TimestampedValue[chargepoint.State]{
		Value:     state,
		Timestamp: timestamp,
	}

	return true
}

func (c *cache) SetPhase1Current(current float64, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.phase1Current.Timestamp) {
		c.logOutdatedObservation("phase 1 current", c.phase1Current.Timestamp, timestamp)

		return false
	}

	c.phase1Current = model.TimestampedValue[float64]{
		Value:     current,
		Timestamp: timestamp,
	}

	return true
}

func (c *cache) SetPhase2Current(current float64, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.phase2Current.Timestamp) {
		c.logOutdatedObservation("phase 2 current", c.phase2Current.Timestamp, timestamp)

		return false
	}

	c.phase2Current = model.TimestampedValue[float64]{
		Value:     current,
		Timestamp: timestamp,
	}

	return true
}

func (c *cache) SetPhase3Current(current float64, timestamp time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if timestamp.Before(c.phase3Current.Timestamp) {
		c.logOutdatedObservation("phase 3 current", c.phase3Current.Timestamp, timestamp)

		return false
	}

	c.phase3Current = model.TimestampedValue[float64]{
		Value:     current,
		Timestamp: timestamp,
	}

	return true
}

func (c *cache) SetInstallationParameters(gridType chargepoint.GridType, phases int, timestamp time.Time) bool {
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

	c.gridType = model.TimestampedValue[chargepoint.GridType]{
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
		value = c.maxCurrent.Value
	case waitGroupOfferedCurrent:
		value = c.offeredCurrent.Value
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
