package easee

import (
	"errors"
	"fmt"
	"math"
	"strings"

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
		report[numericmeter.ValueCurrentPhase1] = c.Phase1Current()
	},
	numericmeter.ValueCurrentPhase2: func(report numericmeter.ValuesReport, c cache.Cache) {
		report[numericmeter.ValueCurrentPhase2] = c.Phase2Current()
	},
	numericmeter.ValueCurrentPhase3: func(report numericmeter.ValuesReport, c cache.Cache) {
		report[numericmeter.ValueCurrentPhase3] = c.Phase3Current()
	},
	numericmeter.ValuePowerImport: func(report numericmeter.ValuesReport, c cache.Cache) {
		report[numericmeter.ValuePowerImport] = c.TotalPower()
	},
	numericmeter.ValueEnergyImport: func(report numericmeter.ValuesReport, c cache.Cache) {
		report[numericmeter.ValueEnergyImport] = c.LifetimeEnergy().Value
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

	return parameters.NewBoolParameter(id, c.cache.CableAlwaysLocked()), nil
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

	return &chargepoint.CableReport{
		CableLock:    c.cache.CableLocked(),
		CableCurrent: c.cache.CableCurrent(),
	}, nil
}

func (c *controller) ChargepointPhaseModeReport() (chargepoint.PhaseMode, error) {
	if err := c.checkConnection(); err != nil {
		return "", err
	}

	if outputPhase := c.cache.OutputPhaseType(); outputPhase != "" {
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

	return c.cache.MaxCurrent(), nil
}

func (c *controller) SetChargepointOfferedCurrent(current int64) error {
	err := c.client.UpdateDynamicCurrent(c.chargerID, float64(current))
	if err != nil {
		return err
	}

	c.cache.SetRequestedOfferedCurrent(current)

	c.cache.WaitForOfferedCurrent(current, c.cfgService.GetCurrentWaitDuration())

	return nil
}

func (c *controller) StartChargepointCharging(settings *chargepoint.ChargingSettings) error {
	startCurrent := float64(c.cache.MaxCurrent())

	if offered := c.cache.RequestedOfferedCurrent(); offered > 0 {
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

	ret := chargepoint.SessionReport{
		SessionEnergy: c.cache.EnergySession(),
	}

	sessions, err := c.sessionStorage.LatestSessionsByChargerID(c.chargerID)
	if err != nil {
		return nil, err
	}

	if latest := sessions.Latest(); latest != nil {
		ret.StartedAt = latest.Start
		ret.FinishedAt = latest.Stop

		if !latest.Stop.IsZero() {
			ret.OfferedCurrent = min(c.cache.OfferedCurrent(), c.cache.MaxCurrent())
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
	if power := c.cache.TotalPower(); power > 0 {
		return chargepoint.StateCharging, nil
	}

	return c.cache.ChargerState(), nil
}

func (c *controller) MeterReport(unit numericmeter.Unit) (float64, error) {
	if err := c.checkConnection(); err != nil {
		return 0, err
	}

	switch unit { //nolint:exhaustive
	case numericmeter.UnitW:
		return c.cache.TotalPower(), nil
	case numericmeter.UnitKWh:
		if c.cache.LifetimeEnergy().Timestamp.IsZero() {
			return 0, fmt.Errorf("energy value not updated")
		}

		return c.cache.LifetimeEnergy().Value, nil
	default:
		return 0, fmt.Errorf("unsupported unit: %s", unit)
	}
}

func (c *controller) MeterExtendedReport(values numericmeter.Values) (numericmeter.ValuesReport, error) {
	if err := c.checkConnection(); err != nil {
		return nil, err
	}

	ret := make(numericmeter.ValuesReport)

	for _, value := range values {
		if value == numericmeter.ValueEnergyImport && c.cache.LifetimeEnergy().Timestamp.IsZero() {
			continue
		}

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
