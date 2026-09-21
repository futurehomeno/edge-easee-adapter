package easee

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/service/alarm"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	"github.com/futurehomeno/cliffhanger/types"
	"github.com/michalkurzeja/go-clock"
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
	persistedPhase func() types.PhaseMode,
) Controller {
	return &controller{
		client:         client,
		manager:        manager,
		cache:          cache,
		cfgService:     cfgService,
		chargerID:      chargerID,
		sessionStorage: sessionStorage,
		persistedPhase: persistedPhase,
	}
}

type controller struct {
	client         api.Client
	manager        signalr.Manager
	cache          cache.Cache
	cfgService     *config.Service
	chargerID      string
	sessionStorage db.ChargingSessionStorage
	persistedPhase func() types.PhaseMode

	// Easee ignores a dynamic-current write that reaches the charger within ~20s of the
	// previous one, or of a stop, although the cloud accepts it. Every such write goes through
	// this channel: sent at once on a quiet channel, otherwise stored in the single slot and
	// sent at sentAt + OfferedCurrentWaitTime. A newer command replaces the payload and never
	// moves that deadline.
	pace    sync.Mutex
	sentAt  time.Time
	pending *chargerCommand
}

type chargerCommand struct {
	stop    bool
	current int
}

func (c chargerCommand) String() string {
	if c.stop {
		return "stop"
	}

	return fmt.Sprintf("set_current %d", c.current)
}

// dispatch sends cmd now or stores it, reporting which. Stamped on the attempt rather than on
// success: a call that timed out may still have reached Easee.
func (c *controller) dispatch(cmd chargerCommand) (bool, error) {
	c.pace.Lock()

	if c.pending == nil && clock.Since(c.sentAt) >= c.cfgService.OfferedCurrentWaitTime() {
		c.sentAt = clock.Now()
		c.pace.Unlock()

		return true, c.send(cmd)
	}

	delay := c.cfgService.OfferedCurrentWaitTime() - clock.Since(c.sentAt)
	if c.pending == nil {
		clock.AfterFunc(delay, c.sendPending)
	}

	c.pending = &cmd
	c.pace.Unlock()

	log.Infof("[%s] Deferred %s, sending in %s", c.chargerID, cmd, delay.Round(time.Second))

	return false, nil
}

// pendingDiffers reports whether the slot holds a current this write would supersede.
// RequestedOfferedCurrent is stamped only on a send, so the dedup compares against the last
// value on the wire: without this, a write matching that value is dropped while a newer,
// different one stays in the slot and reaches the charger at the deadline instead.
//
// A pending stop is deliberately not counted. Letting a same-value write displace it would
// cancel a stop - including an emergency pause - whenever a balancing refresh lands inside the
// window, which the base never did.
func (c *controller) pendingDiffers(current int) bool {
	c.pace.Lock()
	defer c.pace.Unlock()

	return c.pending != nil && !c.pending.stop && c.pending.current != current
}

func (c *controller) send(cmd chargerCommand) error {
	if cmd.stop {
		return c.client.StopCharging(c.chargerID)
	}

	if err := c.client.UpdateDynamicCurrent(c.chargerID, float64(cmd.current)); err != nil {
		return err
	}

	c.cache.SetRequestedOfferedCurrent(cmd.current, clock.Now())

	return nil
}

// sendPending runs on the timer goroutine. The command that stored the payload has already
// answered, so the outcome is log-only: the echo shows up in the SignalR-driven reports.
func (c *controller) sendPending() {
	c.pace.Lock()

	cmd := c.pending
	c.pending = nil

	if cmd == nil {
		c.pace.Unlock()

		return
	}

	c.sentAt = clock.Now()
	c.pace.Unlock()

	if err := c.send(*cmd); err != nil {
		log.Errorf("[%s] Deferred %s failed: %v", c.chargerID, cmd, err)

		return
	}

	if !cmd.stop && !c.cache.WaitForOfferedCurrent(cmd.current, c.cfgService.CurrentWaitDuration()) {
		log.Warnf("[%s] Deferred %s was not echoed back by the charger", c.chargerID, cmd)
	}
}

func (c *controller) SetParameter(p *parameters.Parameter) error {
	if p.ID != model.CableAlwaysLockedParameter {
		return fmt.Errorf("parameter: %v not supported", p.ID)
	}

	val, err := p.BoolValue()
	if err != nil {
		return err
	}

	if err := c.client.SetCableAlwaysLocked(c.chargerID, val); err != nil {
		return err
	}

	// Seeded optimistically: cliffhanger answers cmd.param.set with a forced report, which the
	// reporting cache cannot suppress, so without this it echoes the old value and corrects
	// itself only once the observation lands. The seed bypasses the ordering guard rather than
	// picking a timestamp: time.Now() is the hub's clock and would suppress an observation
	// carrying Easee's, while the zero time would itself be rejected against a populated cache.
	c.cache.SeedCableAlwaysLocked(val)

	return nil
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

		// The cache is empty after a restart; the persisted phase keeps the report on the
		// same phase the inclusion report advertises.
		if persisted := c.persistedPhase(); slices.Contains(modes, persisted) {
			return persisted, nil
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

	return !c.charging()
}

func (c *controller) charging() bool {
	state, _ := c.ChargepointStateReport()

	return state == chargepoint.StateCharging
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

	current, internalAt := c.cache.PhaseMode()
	if target == current {
		// Nothing to send: Easee stores "one phase", not a chosen leg. Skip the record only
		// while a live observation says which leg is actually in use - the charger picks
		// it, so a request outranking that observation would report a leg it is not on.
		// An idle charger is on no leg at all: its cached value is left over from a
		// finished session, and outputPhaseStale keeps it only because no internal mode
		// change has happened since. Without the record, a mode the charger never echoes
		// (NL1 -> NL2) leaves the report republishing the old leg and the UI reverting
		// the user's choice.
		outputPhase, outputPhaseSet := c.cache.OutputPhaseType()
		if outputPhase == "" || c.outputPhaseStale(outputPhase, outputPhaseSet) || !c.charging() {
			c.cache.SetRequestedPhaseMode(c.legToRecord(mode, gridType, phases), c.recordAt(outputPhaseSet, internalAt))

			return nil
		}

		// The live leg answers the report on its own, but only while it outranks what is
		// already recorded. A three-phase request from the previous balance tick is stamped
		// after it, so re-record the leg over that request rather than leaving it to win.
		if requested, requestedAt := c.cache.RequestedPhaseMode(); requested != "" &&
			requestedAt.After(outputPhaseSet) && c.requestStillHolds(requested, requestedAt) {
			c.cache.SetRequestedPhaseMode(c.legToRecord(mode, gridType, phases), requestedAt.Add(time.Millisecond))
		}

		return nil
	}

	if err := c.client.SetPhaseMode(c.chargerID, target); err != nil {
		return err
	}

	_, outputPhaseSet := c.cache.OutputPhaseType()

	c.cache.SetRequestedPhaseMode(c.legToRecord(mode, gridType, phases), c.recordAt(outputPhaseSet, internalAt))

	return c.restartForPhaseMode(target)
}

// recordAt stamps a record in the observation clock rather than the hub's: the request is only
// ever compared against SignalR timestamps, so a skewed hub clock would otherwise let a live
// observation outrank a fresh request, or keep a stale request winning after the charger moved on.
func (c *controller) recordAt(outputPhaseSet, internalAt time.Time) time.Time {
	at := outputPhaseSet
	if internalAt.After(at) {
		at = internalAt
	}

	// A re-recorded leg is stamped off an older request, so it can already outrank both
	// observations; the cache drops a record stamped behind it and the report keeps the leg.
	if existing, existingAt := c.cache.RequestedPhaseMode(); existing != "" && existingAt.After(at) {
		at = existingAt
	}

	return at.Add(time.Millisecond)
}

// legToRecord substitutes the leg the charger is known to use for a single-phase request naming
// a different one. An Easee cannot choose its leg, so the report must name it rather than echo
// the request: the hub learns the charger is fixed only from that mismatch, and echoing leaves
// it re-requesting a leg the charger will never use. Recorded rather than left unrecorded,
// because the record is what outranks the other answers - a leftover three-phase request from
// the previous balance tick, or the multi-phase leg of a running auto session, both of which
// would otherwise reach the forced report instead. A leg persisted under another topology is no
// evidence about this one.
func (c *controller) legToRecord(mode types.PhaseMode, gridType types.GridType, phases int) types.PhaseMode {
	if mode.EffectivePhasesCnt() != 1 {
		return mode
	}

	known := c.persistedPhase()

	// The leg the charger is delivering on right now outranks the stored one, which lags it
	// whenever a props republish failed: substituting the stale store there would name a leg
	// the charger is demonstrably not on, and the hub would learn that as its fixed phase.
	if outputPhase, outputPhaseSet := c.cache.OutputPhaseType(); outputPhase.EffectivePhasesCnt() == 1 &&
		!c.outputPhaseStale(outputPhase, outputPhaseSet) && c.charging() {
		known = outputPhase
	}

	if known.EffectivePhasesCnt() == 1 && known != mode &&
		slices.Contains(model.SettablePhaseModes(gridType, phases), known) {
		return known
	}

	return mode
}

// restartForPhaseMode bounces an in-progress session, because the charger applies a new
// phase mode only at a session boundary. Failing to pause is not fatal - the mode is stored
// and takes effect on the next session anyway. The resume always follows the pause inside
// the channel's window, so it is stored and goes out at the deadline; a pause that was
// itself stored is replaced by it, which leaves the session running - the same outcome.
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

	// Only this adapter writes RequestedOfferedCurrent, so it is empty after a restart even
	// though the session is still running. OfferedCurrent is the charger's own observation of
	// what it is delivering, so it describes that session; falling straight through to
	// MaxCurrent would resume a 6A session at 32A - the silent raise this whole path avoids.
	if resume <= 0 {
		resume, _ = c.cache.OfferedCurrent()
	}

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
	if _, err := c.setOfferedCurrent(resume, true); err != nil {
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
	_, err := c.setOfferedCurrent(current, false)

	return err
}

// setOfferedCurrent is the shared implementation behind SetChargepointOfferedCurrent and the
// Start path. When force is true the recent-value dedup is bypassed - this matters for
// (re)starting a stopped session, where the charger needs the UpdateDynamicCurrent call to
// resume charging even if the cached value matches what was sent before the stop.
//
// The bool reports whether the charger echoed the new current back over SignalR within
// CurrentWaitDuration; a deferred write counts as confirmed, its echo is checked by the
// deferred send. Callers that need to know the change actually landed check it.
func (c *controller) setOfferedCurrent(current int, force bool) (bool, error) {
	limit, _ := c.cache.MaxCurrent()
	if limit == 0 {
		limit = maxCurrentValue
	}

	if current > limit {
		log.Warnf("[%s] Clamp offered current %dA to max %dA", c.chargerID, current, limit)
		current = limit
	}

	if !force && !c.pendingDiffers(current) {
		lastValue, lastSet := c.cache.RequestedOfferedCurrent()

		if clock.Since(lastSet) < c.cfgService.OfferedCurrentWaitTime() && current == lastValue {
			return true, nil
		}
	}

	sent, err := c.dispatch(chargerCommand{current: current})
	if err != nil {
		return false, err
	}

	if !sent {
		return true, nil
	}

	return c.cache.WaitForOfferedCurrent(current, c.cfgService.CurrentWaitDuration()), nil
}

func (c *controller) StartChargepointCharging(settings *chargepoint.ChargingSettings) error {
	mode := strings.ToLower(settings.Mode)
	slow := mode == model.ChargingModeSlow

	startCurrent, _ := c.cache.RequestedOfferedCurrent()

	switch {
	case startCurrent <= 0:
		// A cached offered current of 0 means "unknown" - either no load balancer ever set one, or
		// the session-finished observation cleared it - so the charger starts at the user's max.
		startCurrent, _ = c.cache.MaxCurrent()
	case mode == model.ChargingModeNormal:
		// Only an explicit normal-mode start gets the floor. The mode is optional, so a start
		// without one may be a load balancer resuming a session it paused, and the cached value
		// the budget it balanced us to; raising that offers more than it allowed for a whole
		// throttle window. Kept, it costs a user's bare start at most one balancer tick, or a
		// set_current. Slow mode is exempt too: with no slow current configured the throttled
		// cached value is the closest thing to what the user asked for.
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
	confirmed, err := c.setOfferedCurrent(startCurrent, true)
	if err != nil {
		return err
	}

	// Easee accepting the call is not the charger acting on it, and reporting success on a
	// start that left the charger paused hides that.
	if !confirmed {
		return fmt.Errorf("start accepted, but the charger did not resume at %dA", startCurrent)
	}

	return nil
}

func (c *controller) StopChargepointCharging() error {
	_, err := c.dispatch(chargerCommand{stop: true})

	return err
}

func (c *controller) ChargepointCurrentSessionReport() (*chargepoint.SessionReport, error) {
	if err := c.checkConnection(); err != nil {
		return nil, err
	}

	energy, _ := c.cache.EnergySession()
	offeredCurrent, _ := c.cache.OfferedCurrent()

	if maxCurrent, _ := c.cache.MaxCurrent(); maxCurrent > 0 {
		offeredCurrent = min(offeredCurrent, maxCurrent)
	}

	// Cliffhanger stamps offered_current on every session report, so gating it on an open
	// session row publishes 0 for a charger that is offering current - energy-guard then
	// anchors its ramp on that zero.
	ret := chargepoint.SessionReport{
		SessionEnergy:  energy,
		OfferedCurrent: offeredCurrent,
	}

	sessions, err := c.sessionStorage.LatestSessionsByChargerID(c.chargerID)
	if err != nil {
		return nil, err
	}

	if latest := sessions.Latest(); latest != nil {
		ret.StartedAt = latest.Start
		ret.FinishedAt = latest.Stop
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
