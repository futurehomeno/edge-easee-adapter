package easee

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/service/alarm"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	"github.com/futurehomeno/cliffhanger/types"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/cache"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/db"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
)

const maxCurrentValue = 32

var extendedReportMapping = map[numericmeter.Value]func(cache.Cache) (float64, time.Time){
	numericmeter.ValueCurrentPhase1: cache.Cache.Phase1Current,
	numericmeter.ValueCurrentPhase2: cache.Cache.Phase2Current,
	numericmeter.ValueCurrentPhase3: cache.Cache.Phase3Current,
	numericmeter.ValuePowerImport:   cache.Cache.TotalPower,
	numericmeter.ValueEnergyImport:  cache.Cache.LifetimeEnergy,
}

type Controller interface {
	chargepoint.Controller
	chargepoint.AdjustablePhaseModeController
	chargepoint.AdjustableMaxCurrentController
	chargepoint.AdjustableOfferedCurrentController
	chargepoint.CableLockAwareController
	parameters.Controller
	numericmeter.Reporter
	numericmeter.ExtendedReporter
	alarm.Reporter
	UpdateState(chargerID string, state *State) error
}

func NewController(
	manager signalr.Manager,
	client api.Client,
	chargerID string,
	cache cache.Cache,
	cfgService *config.Service,
	sessionStorage db.ChargingSessionStorage,
) Controller {
	return &controller{
		client:         client,
		manager:        manager,
		cache:          cache,
		cfgService:     cfgService,
		chargerID:      chargerID,
		sessionStorage: sessionStorage,
	}
}

type controller struct {
	client         api.Client
	manager        signalr.Manager
	cache          cache.Cache
	cfgService     *config.Service
	chargerID      string
	sessionStorage db.ChargingSessionStorage
}

func (c *controller) SetParameter(p *parameters.Parameter) error {
	if p.ID != model.CableAlwaysLockedParameter {
		return fmt.Errorf("parameter: %v not supported", p.ID)
	}

	val, err := p.BoolValue()
	if err != nil {
		return err
	}

	return c.client.SetCableAlwaysLocked(c.chargerID, val)
}

func (c *controller) GetParameter(id string) (*parameters.Parameter, error) {
	if id != model.CableAlwaysLockedParameter {
		return nil, fmt.Errorf("parameter: %v not supported", id)
	}

	alwaysLocked, _ := c.cache.CableAlwaysLocked()

	return parameters.NewBoolParameter(id, alwaysLocked), nil
}

func (c *controller) GetParameterSpecifications() ([]*parameters.ParameterSpecification, error) {
	return []*parameters.ParameterSpecification{
		parameterSpecificationCableAlwaysLocked(),
	}, nil
}

func (c *controller) ChargepointCableLockReport() (*chargepoint.CableReport, error) {
	if err := c.checkConnection(); err != nil {
		return nil, err
	}

	locked, _ := c.cache.CableLocked()
	report := chargepoint.CableReport{CableLock: locked}

	if !locked {
		zero := 0
		report.CableCurrent = &zero

		return &report, nil
	}

	if cable, cableTime := c.cache.CableCurrent(); !cableTime.IsZero() && cable >= 0 {
		report.CableCurrent = &cable
	}

	return &report, nil
}

func (c *controller) ChargepointPhaseModeReport() (types.PhaseMode, error) {
	if err := c.checkConnection(); err != nil {
		return "", err
	}

	outputPhase, outputPhaseSet := c.cache.OutputPhaseType()

	// A mode we requested ourselves outranks the output phase until the charger reports a
	// newer one: outputPhase goes unassigned between sessions and handleOutPhase drops that
	// observation, so the cached value survives as a stale echo of the previous session.
	if requested, requestedAt := c.cache.RequestedPhaseMode(); requested != "" && requestedAt.After(outputPhaseSet) && c.requestStillHolds(requested, requestedAt) {
		return requested, nil
	}

	if outputPhase != "" && !c.outputPhaseStale(outputPhase, outputPhaseSet) {
		return outputPhase, nil
	}

	// outputPhase is unassigned when not charging
	// if not previous value was recorded, default first value from sup_phase_modes is used
	state := State{}
	if err := c.UpdateState(c.chargerID, &state); err != nil {
		return "", err
	}

	if modes := model.SupportedPhaseModes(state.GridType, state.PhaseMode, state.Phases); len(modes) > 0 {
		// The auto row ends with the multi-phase mode, which is what the setter maps a
		// three-phase request onto. Reporting modes[0] here would answer a request the user
		// just made with a single leg whenever the cache is empty - after an adapter restart.
		if state.PhaseMode == model.EaseePhaseModeAuto {
			return modes[len(modes)-1], nil
		}

		return modes[0], nil
	}

	errMsg := "unable to map phase modes"

	log.WithField("charger_id", c.chargerID).
		WithField("grid_type", state.GridType).
		WithField("phases", state.Phases).
		WithField("internal_phase_mode", state.PhaseMode).
		Error(errMsg)

	return "", errors.New(errMsg)
}

// requestStillHolds reports whether the charger is still set to the mode we asked for. A
// newer internal phase mode that maps to something else means it was changed elsewhere, so
// the request stops outranking the charger's own state - nothing else ever clears it.
func (c *controller) requestStillHolds(requested types.PhaseMode, requestedAt time.Time) bool {
	internal, internalAt := c.cache.PhaseMode()
	if !internalAt.After(requestedAt) {
		return true
	}

	gridType, _ := c.cache.GridType()
	phases, _ := c.cache.Phases()

	target, err := model.ToEaseePhaseMode(gridType, phases, requested)

	return err == nil && target == internal
}

// outputPhaseStale reports whether the cached leg predates an internal phase mode that no longer
// covers it. Nothing ever clears outputPhase, so without this an internal mode changed elsewhere -
// in the Easee app - republishes the leg of the previous session. A charging charger keeps its
// leg: Easee applies a new mode only at a session boundary, so the leg in use is still the old one.
func (c *controller) outputPhaseStale(outputPhase types.PhaseMode, outputPhaseSet time.Time) bool {
	internal, internalAt := c.cache.PhaseMode()
	if !internalAt.After(outputPhaseSet) {
		return false
	}

	gridType, _ := c.cache.GridType()
	phases, _ := c.cache.Phases()

	if slices.Contains(model.SupportedPhaseModes(gridType, internal, phases), outputPhase) {
		return false
	}

	state, _ := c.ChargepointStateReport()

	return state != chargepoint.StateCharging
}

func (c *controller) SetChargepointPhaseMode(mode types.PhaseMode) error {
	if err := c.checkConnection(); err != nil {
		return err
	}

	gridType, _ := c.cache.GridType()
	phases, _ := c.cache.Phases()

	target, err := model.ToEaseePhaseMode(gridType, phases, mode)
	if err != nil {
		return err
	}

	// A grid offering a single mode has nothing to switch between, yet sup_phase_modes still
	// has to advertise it - the property gates evt.phase_mode.report too. Flipping the internal
	// mode here would bounce a charging session for a change nothing can observe.
	if len(model.SettablePhaseModes(gridType, phases)) < 2 {
		return nil
	}

	// Nothing is pending when the charger already sits in the target mode, so the request
	// is not recorded: a fresh timestamp here would outrank a live outputPhase observation
	// and report a leg the charger is not actually on.
	current, internalAt := c.cache.PhaseMode()
	if target == current {
		return nil
	}

	if err := c.client.SetPhaseMode(c.chargerID, target); err != nil {
		return err
	}

	// Stamped in the observation clock rather than the hub's: the request is only ever compared
	// against SignalR timestamps, so a skewed hub clock would otherwise let a live observation
	// outrank a fresh request, or keep a stale request winning after the charger moved on.
	_, requestedAt := c.cache.OutputPhaseType()
	if internalAt.After(requestedAt) {
		requestedAt = internalAt
	}

	c.cache.SetRequestedPhaseMode(mode, requestedAt.Add(time.Millisecond))

	return c.restartForPhaseMode(target)
}

// restartForPhaseMode bounces an in-progress session, because the charger applies a new
// phase mode only at a session boundary. Failing to pause is not fatal - the mode is stored
// and takes effect on the next session anyway - but a failed resume leaves the charger
// stopped, so that one is reported back rather than only logged.
func (c *controller) restartForPhaseMode(target int) error {
	state, err := c.ChargepointStateReport()
	if err != nil {
		log.Warnf("[%s] Phase mode set, but the charger state is unknown: %v", c.chargerID, err)

		return nil
	}

	if state != chargepoint.StateCharging {
		return nil
	}

	// Read before the stop: the session-finished observation clears the cached value
	// asynchronously, so afterwards it may no longer describe the session being bounced.
	resume, _ := c.cache.RequestedOfferedCurrent()
	if resume <= 0 {
		resume, _ = c.cache.MaxCurrent()
	}

	// setOfferedCurrent only clamps the upper bound, so a zero would "resume" the session at
	// 0A and report success while the charger stays paused.
	if resume <= 0 {
		return fmt.Errorf("phase mode set to %d, but no current is known to resume at", target)
	}

	if err := c.StopChargepointCharging(); err != nil {
		log.Warnf("[%s] Phase mode set, but pausing to apply it failed: %v", c.chargerID, err)

		return nil
	}

	// Resumed at the session's own current rather than through StartChargepointCharging: a
	// normal-mode start floors the current to initial_charging_current, which would silently
	// raise a slow session - the mode is not recorded anywhere, so it cannot be restored.
	if err := c.setOfferedCurrent(resume, true); err != nil {
		return fmt.Errorf("phase mode set to %d, but the charger was left stopped: %w", target, err)
	}

	return nil
}

func (c *controller) SetChargepointMaxCurrent(current int) error {
	err := c.client.UpdateMaxCurrent(c.chargerID, float64(current))
	if err != nil {
		return err
	}

	c.cache.WaitForMaxCurrent(current, c.cfgService.CurrentWaitDuration())

	return nil
}

func (c *controller) ChargepointMaxCurrentReport() (int, error) {
	if err := c.checkConnection(); err != nil {
		return 0, err
	}

	current, _ := c.cache.MaxCurrent()

	return current, nil
}

func (c *controller) SetChargepointOfferedCurrent(current int) error {
	return c.setOfferedCurrent(current, false)
}

// setOfferedCurrent is the shared implementation behind SetChargepointOfferedCurrent and the
// Start path. When force is true the recent-value dedup is bypassed - this matters for
// (re)starting a stopped session, where the charger needs the UpdateDynamicCurrent call to
// resume charging even if the cached value matches what was sent before the stop.
func (c *controller) setOfferedCurrent(current int, force bool) error {
	limit, _ := c.cache.MaxCurrent()
	if limit == 0 {
		limit = maxCurrentValue
	}

	if current > limit {
		log.Warnf("[%s] Clamp offered current %dA to max %dA", c.chargerID, current, limit)
		current = limit
	}

	if !force {
		lastValue, lastSet := c.cache.RequestedOfferedCurrent()

		if time.Since(lastSet) < c.cfgService.OfferedCurrentWaitTime() && current == lastValue {
			return nil
		}
	}

	err := c.client.UpdateDynamicCurrent(c.chargerID, float64(current))
	if err != nil {
		return err
	}

	c.cache.SetRequestedOfferedCurrent(current, time.Now())

	c.cache.WaitForOfferedCurrent(current, c.cfgService.CurrentWaitDuration())

	return nil
}

func (c *controller) StartChargepointCharging(settings *chargepoint.ChargingSettings) error {
	slow := strings.ToLower(settings.Mode) == model.ChargingModeSlow

	startCurrent, _ := c.cache.RequestedOfferedCurrent()

	switch {
	case startCurrent <= 0:
		// A cached offered current of 0 means "unknown" - either no load balancer ever set one, or
		// the session-finished observation cleared it - so the charger starts at the user's max.
		startCurrent, _ = c.cache.MaxCurrent()
	case !slow:
		// Slow mode is deliberately exempt from the floor: with no slow current configured the
		// throttled cached value is the closest thing to what the user asked for.
		startCurrent = max(startCurrent, c.cfgService.InitialChargingCurrent())
	}

	if slow {
		if slowCurrent := c.cfgService.SlowChargingCurrentInAmperes(); slowCurrent > 0 {
			startCurrent = int(math.Round(slowCurrent))
		}
	}

	if startCurrent == 0 {
		return errors.New("invalid start current")
	}

	// resume charging request is not used because it clears dynamic current value.
	// update current will resume charging. Bypass the dedup - if the user pressed Start
	// within OfferedCurrentWaitTime of a Stop the cached value still matches startCurrent
	// (cache is only cleared async via the SignalR session-finished observation), and
	// dedup-suppressing the call would leave the charger stopped.
	return c.setOfferedCurrent(startCurrent, true)
}

func (c *controller) StopChargepointCharging() error {
	return c.client.StopCharging(c.chargerID)
}

func (c *controller) ChargepointCurrentSessionReport() (*chargepoint.SessionReport, error) {
	if err := c.checkConnection(); err != nil {
		return nil, err
	}

	energy, _ := c.cache.EnergySession()

	ret := chargepoint.SessionReport{
		SessionEnergy: energy,
	}

	sessions, err := c.sessionStorage.LatestSessionsByChargerID(c.chargerID)
	if err != nil {
		return nil, err
	}

	if latest := sessions.Latest(); latest != nil {
		ret.StartedAt = latest.Start
		ret.FinishedAt = latest.Stop

		if latest.Stop.IsZero() {
			offeredCurrent, _ := c.cache.OfferedCurrent()
			maxCurrent, _ := c.cache.MaxCurrent()

			if maxCurrent > 0 {
				offeredCurrent = min(offeredCurrent, maxCurrent)
			}

			ret.OfferedCurrent = offeredCurrent
		}
	}

	if prev := sessions.Previous(); prev != nil {
		ret.PreviousSessionEnergy = prev.Energy
	}

	return &ret, nil
}

func (c *controller) AlarmReport(event string) (*alarm.Report, error) {
	if err := c.checkConnection(); err != nil {
		return nil, err
	}

	status := alarm.StatusDeactivate

	if c.cache.AlarmActive(event) {
		status = alarm.StatusActivate
	}

	return &alarm.Report{Event: event, Status: status}, nil
}

func (c *controller) ChargepointStateReport() (chargepoint.State, error) {
	if err := c.checkConnection(); err != nil {
		return "", err
	}

	if power, _ := c.cache.TotalPower(); power > 0 {
		return chargepoint.StateCharging, nil
	}

	state, _ := c.cache.ChargerState()

	return state, nil
}

func (c *controller) MeterReport(unit numericmeter.Unit) (float64, error) {
	if err := c.checkConnection(); err != nil {
		return 0, err
	}

	switch unit { //nolint:exhaustive
	case numericmeter.UnitW:
		power, _ := c.cache.TotalPower()

		return power, nil
	case numericmeter.UnitKWh:
		energy, timestamp := c.cache.LifetimeEnergy()

		if timestamp.IsZero() {
			return 0, errors.New("energy value not updated")
		}

		return energy, nil
	default:
		return 0, fmt.Errorf("unsupported unit: %s", unit)
	}
}

func (c *controller) MeterExtendedReport(values numericmeter.Values) (numericmeter.ValuesReport, error) {
	if err := c.checkConnection(); err != nil {
		return nil, err
	}

	ret := make(numericmeter.ValuesReport, len(values))

	for _, value := range values {
		read, ok := extendedReportMapping[value]
		if !ok {
			continue
		}

		v, timestamp := read(c.cache)

		// Lifetime energy is the one value with no meaningful zero: never having observed it
		// must leave it out of the report rather than report 0 kWh.
		if value == numericmeter.ValueEnergyImport && timestamp.IsZero() {
			continue
		}

		ret[value] = v
	}

	return ret, nil
}

func (c *controller) UpdateState(chargerID string, state *State) error {
	configErr := c.updateChargerConfigState(chargerID, state)
	siteErr := c.updateChargerSiteState(chargerID, state)

	return errors.Join(configErr, siteErr)
}

func (c *controller) updateChargerConfigState(chargerID string, state *State) error {
	cfg, err := c.client.ChargerConfig(chargerID)
	if err != nil {
		if state.IsConfigUpdateNeeded() {
			return fmt.Errorf("failed to fetch a charger config ID %s: %w", chargerID, err)
		}

		return nil
	}

	gridType, phases := cfg.DetectedPowerGridType.ToFimpGridType()

	state.GridType = gridType
	state.Phases = phases
	state.PhaseMode = cfg.PhaseMode

	return nil
}

func (c *controller) updateChargerSiteState(chargerID string, state *State) error {
	siteInfo, err := c.client.ChargerSiteInfo(chargerID)
	if err != nil {
		if state.IsSiteUpdateNeeded() {
			return fmt.Errorf("failed to fetch a charger site info ID %s: %w", chargerID, err)
		}

		return nil
	}

	state.SupportedMaxCurrent = min(int(math.Round(siteInfo.RatedCurrent)), maxCurrentValue)

	return nil
}

func (c *controller) checkConnection() error {
	connected, reason := c.manager.Connected(c.chargerID)
	if !connected {
		return fmt.Errorf("charger %s is not connected: %s", c.chargerID, reason)
	}

	return nil
}
