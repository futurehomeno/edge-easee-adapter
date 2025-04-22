package easee

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/cache"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/db"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
)

const maxCurrentValue = 32

var extendedReportMapping = map[numericmeter.Value]specFunc{
	numericmeter.ValueCurrentPhase1: func(report numericmeter.ValuesReport, c cache.Cache) {
		current, _ := c.Phase1Current()
		report[numericmeter.ValueCurrentPhase1] = current
	},
	numericmeter.ValueCurrentPhase2: func(report numericmeter.ValuesReport, c cache.Cache) {
		current, _ := c.Phase2Current()
		report[numericmeter.ValueCurrentPhase2] = current
	},
	numericmeter.ValueCurrentPhase3: func(report numericmeter.ValuesReport, c cache.Cache) {
		current, _ := c.Phase3Current()
		report[numericmeter.ValueCurrentPhase3] = current
	},
	numericmeter.ValuePowerImport: func(report numericmeter.ValuesReport, c cache.Cache) {
		power, _ := c.TotalPower()
		report[numericmeter.ValuePowerImport] = power
	},
	numericmeter.ValueEnergyImport: func(report numericmeter.ValuesReport, c cache.Cache) {
		energy, timestamp := c.LifetimeEnergy()
		if timestamp.IsZero() {
			return
		}

		report[numericmeter.ValueEnergyImport] = energy
	},
}

type specFunc func(report numericmeter.ValuesReport, c cache.Cache)

// Controller represents a charger controller.
type Controller interface {
	chargepoint.Controller
	chargepoint.PhaseModeAwareController
	chargepoint.AdjustableMaxCurrentController
	chargepoint.AdjustableOfferedCurrentController
	chargepoint.CableLockAwareController
	parameters.Controller
	numericmeter.Reporter
	numericmeter.ExtendedReporter
	UpdateState(chargerID string, state *State) error
}

// NewController returns a new instance of Controller.
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
	current, _ := c.cache.CableCurrent()

	if !locked || (current != nil && *current < 0) {
		locked = false

		current = new(int64)
		*current = 0
	}

	return &chargepoint.CableReport{
		CableLock:    locked,
		CableCurrent: current,
	}, nil
}

func (c *controller) ChargepointPhaseModeReport() (chargepoint.PhaseMode, error) {
	if err := c.checkConnection(); err != nil {
		return "", err
	}

	outputPhase, _ := c.cache.OutputPhaseType()
	if outputPhase != "" {
		return outputPhase, nil
	}

	// outputPhase is unassigned when not charging
	// if not previous value was recorded, default first value from sup_phase_modes is used
	state := State{}
	if err := c.UpdateState(c.chargerID, &state); err != nil {
		return "", err
	}

	if modes := model.SupportedPhaseModes(state.GridType, state.PhaseMode, state.Phases); len(modes) > 0 {
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

func (c *controller) SetChargepointMaxCurrent(current int64) error {
	err := c.client.UpdateMaxCurrent(c.chargerID, float64(current))
	if err != nil {
		return err
	}

	c.cache.WaitForMaxCurrent(current, c.cfgService.GetCurrentWaitDuration())

	return nil
}

func (c *controller) ChargepointMaxCurrentReport() (int64, error) {
	if err := c.checkConnection(); err != nil {
		return 0, err
	}

	current, _ := c.cache.MaxCurrent()

	return current, nil
}

func (c *controller) SetChargepointOfferedCurrent(current int64) error {
	err := c.client.UpdateDynamicCurrent(c.chargerID, float64(current))
	if err != nil {
		return err
	}

	c.cache.SetRequestedOfferedCurrent(current, time.Now())

	c.cache.WaitForOfferedCurrent(current, c.cfgService.GetCurrentWaitDuration())

	return nil
}

func (c *controller) StartChargepointCharging(settings *chargepoint.ChargingSettings) error {
	maxCurrent, _ := c.cache.MaxCurrent()
	startCurrent := float64(maxCurrent)

	if offered, _ := c.cache.RequestedOfferedCurrent(); offered > 0 {
		startCurrent = float64(offered)
	}

	if strings.ToLower(settings.Mode) == model.ChargingModeSlow {
		slowCurrent := c.cfgService.GetSlowChargingCurrentInAmperes()

		if slowCurrent > 0 {
			startCurrent = slowCurrent
		}
	}

	if startCurrent == 0 {
		return errors.New("invalid start current")
	}

	// resume charing request is not used because it clears dynamic current value.
	// update current will resume charging.
	return c.client.UpdateDynamicCurrent(c.chargerID, startCurrent)
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

		// if session is not finished
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

func (c *controller) ChargepointStateReport() (chargepoint.State, error) {
	if err := c.checkConnection(); err != nil {
		return "", err
	}

	// If a charger reports power usage, assume a charging state.
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
			return 0, fmt.Errorf("energy value not updated")
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
		if f, ok := extendedReportMapping[value]; ok {
			f(ret, c.cache)
		}
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

	state.SupportedMaxCurrent = min(int64(math.Round(siteInfo.RatedCurrent)), maxCurrentValue)

	return nil
}

func (c *controller) checkConnection() error {
	connected, reason := c.manager.Connected(c.chargerID)
	if !connected {
		return fmt.Errorf("charger %s is not connected: %s", c.chargerID, reason)
	}

	return nil
}
