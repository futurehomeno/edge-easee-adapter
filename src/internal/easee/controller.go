package easee

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/cache"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/pkg/errors"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

const (
	SlowChargingCurrentInAmpers = 10.0
)

// Controller represents a charger controller.
type Controller interface {
	chargepoint.Controller
	meterelec.Reporter
}

// NewController returns a new instance of Controller.
func NewController(client Client, cfgService *config.Service, chargerID string) Controller {
	return &controller{
		client:         client,
		chargerID:      chargerID,
		stateRefresher: newStateRefresher(client, chargerID, cfgService.GetPollingInterval()),
		cfgRefresher:   newConfigRefresher(client, chargerID, cfgService.GetPollingInterval()),
	}
}

type controller struct {
	client         Client
	chargerID      string
	stateRefresher cache.Refresher
	cfgRefresher   cache.Refresher
}

func (c *controller) StartChargepointCharging(mode string) error {
	current, err := c.chargingCurrent(mode)
	if err != nil {
		return fmt.Errorf("failed to get charging current: %w", err)
	}

	if err := c.client.StartCharging(c.chargerID, current); err != nil {
		return fmt.Errorf("failed to start charging session for charger id %s: %w", c.chargerID, err)
	}

	return nil
}

func (c *controller) StopChargepointCharging() error {
	if err := c.client.StopCharging(c.chargerID); err != nil {
		return fmt.Errorf("failed to stop charging session for charger id %s: %w", c.chargerID, err)
	}

	return nil
}

func (c *controller) SetChargepointCableLock(locked bool) error {
	if err := c.client.SetCableLock(c.chargerID, locked); err != nil {
		return err
	}

	return nil
}

func (c *controller) ChargepointCableLockReport() (bool, error) {
	state, err := c.cachedChargerState()
	if err != nil {
		return false, errors.Wrap(err, "failed to fetch charger state")
	}

	return state.CableLocked, nil
}

func (c *controller) ChargepointCurrentSessionReport() (float64, error) {
	state, err := c.cachedChargerState()
	if err != nil {
		return 0, errors.Wrap(err, "failed to fetch charger state")
	}

	mode := state.ChargerOpMode.String()
	if mode == ChargerModeCharging || mode == ChargerModeFinished {
		return state.SessionEnergy, nil
	}

	return 0, nil
}

func (c *controller) ChargepointStateReport() (string, error) {
	chargerState, err := c.client.ChargerState(c.chargerID)
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch charger state")
	}

	return chargerState.ChargerOpMode.String(), nil
}

func (c *controller) ElectricityMeterReport(unit string) (float64, error) {
	state, err := c.cachedChargerState()
	if err != nil {
		return 0, errors.Wrap(err, "failed to fetch charger state")
	}

	switch unit {
	case meterelec.UnitW:
		return state.TotalPower * 1000, nil
	case meterelec.UnitKWh:
		return state.LifetimeEnergy, nil
	case meterelec.UnitV:
		return state.Voltage, nil
	default:
		return 0, errors.Errorf("unsupported unit: %s", unit)
	}
}

func (c *controller) chargingCurrent(mode string) (float64, error) {
	cfg, err := c.cachedChargerConfig()
	if err != nil {
		return 0, err
	}

	if cfg.MaxChargerCurrent < SlowChargingCurrentInAmpers {
		return 0, fmt.Errorf("returned max charger current (%f) is lower than minimum (%f)", cfg.MaxChargerCurrent, SlowChargingCurrentInAmpers)
	}

	switch strings.ToLower(mode) {
	case ChargingModeSlow:
		return SlowChargingCurrentInAmpers, nil
	default:
		return cfg.MaxChargerCurrent, nil
	}
}

func (c *controller) cachedChargerState() (*ChargerState, error) {
	rawState, err := c.stateRefresher.Refresh()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current charger state from stateRefresher")
	}

	state, ok := rawState.(*ChargerState)
	if !ok {
		return nil, errors.Errorf("expected %s, got %s instead", reflect.TypeOf(&ChargerState{}), reflect.TypeOf(rawState))
	}

	return state, nil
}

func (c *controller) cachedChargerConfig() (*ChargerConfig, error) {
	rawCfg, err := c.cfgRefresher.Refresh()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current charger cfg from stateRefresher")
	}

	cfg, ok := rawCfg.(*ChargerConfig)
	if !ok {
		return nil, errors.Errorf("expected %s, got %s instead", reflect.TypeOf(&ChargerState{}), reflect.TypeOf(rawCfg))
	}

	return cfg, nil
}

// newStateRefresher creates new instance of a stateRefresher cache.
func newStateRefresher(client Client, chargerID string, interval time.Duration) cache.Refresher {
	refreshFn := func() (interface{}, error) {
		state, err := client.ChargerState(chargerID)
		if err != nil {
			return nil, fmt.Errorf("controller: failed to fetch charger state ID %s: %w", chargerID, err)
		}

		return state, nil
	}

	return cache.NewRefresher(refreshFn, cache.OffsetInterval(interval))
}

// newConfigRefresher creates new instance of a stateRefresher cache.
func newConfigRefresher(client Client, chargerID string, interval time.Duration) cache.Refresher {
	refreshFn := func() (interface{}, error) {
		cfg, err := client.ChargerConfig(chargerID)
		if err != nil {
			return nil, fmt.Errorf("controller: failed to fetch charger config ID %s: %w", chargerID, err)
		}

		return cfg, nil
	}

	return cache.NewRefresher(refreshFn, cache.OffsetInterval(interval))
}
