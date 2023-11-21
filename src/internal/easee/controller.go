package easee

import (
	"strings"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/pkg/errors"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// Controller represents a charger controller.
type Controller interface {
	chargepoint.Controller
	numericmeter.Reporter
}

// NewController returns a new instance of Controller.
func NewController(client APIClient, cache ObservationCache, cfgService *config.Service, chargerID string, maxCurrent float64) Controller {
	return &controller{
		client:     client,
		cache:      cache,
		cfgService: cfgService,
		chargerID:  chargerID,
		maxCurrent: maxCurrent,
	}
}

type controller struct {
	client     APIClient
	cache      ObservationCache
	cfgService *config.Service
	chargerID  string
	maxCurrent float64
}

func (c *controller) StartChargepointCharging(settings *chargepoint.ChargingSettings) error {
	var current float64

	switch strings.ToLower(settings.Mode) {
	case ChargingModeSlow:
		current = c.cfgService.GetSlowChargingCurrentInAmperes()
	default:
		current = c.maxCurrent
	}

	return c.client.StartCharging(c.chargerID, current)
}

func (c *controller) StopChargepointCharging() error {
	return c.client.StopCharging(c.chargerID)
}

func (c *controller) SetChargepointCableLock(locked bool) error {
	return c.client.SetCableLock(c.chargerID, locked)
}

func (c *controller) ChargepointCableLockReport() (*chargepoint.CableReport, error) {
	isLocked, err := c.cache.CableLocked()
	if err != nil {
		return nil, err
	}
	return &chargepoint.CableReport{
		CableLock:    isLocked,
		CableCurrent: 0, // TODO
	}, nil
}

func (c *controller) ChargepointCurrentSessionReport() (*chargepoint.SessionReport, error) {
	energy, err := c.cache.SessionEnergy()
	if err != nil {
		return nil, err
	}

	return &chargepoint.SessionReport{
		SessionEnergy:         energy,
		PreviousSessionEnergy: 0,           // TODO
		StartedAt:             time.Time{}, // TODO
		FinishedAt:            time.Time{}, // TODO
		OfferedCurrent:        0,           // TODO
	}, nil
}

func (c *controller) ChargepointStateReport() (chargepoint.State, error) {
	power, err := c.cache.TotalPower()
	if err != nil {
		return "", err
	}

	// If a charger reports power usage, assume a charging state.
	if power > 0 {
		return chargepoint.StateCharging, nil
	}

	state, err := c.cache.ChargerState()
	if err != nil {
		return "", err
	}

	return state.ToFimpState(), nil
}

func (c *controller) MeterReport(unit numericmeter.Unit) (float64, error) {
	switch unit {
	case numericmeter.UnitW:
		return c.cache.TotalPower()
	case numericmeter.UnitKWh:
		return c.cache.LifetimeEnergy()
	default:
		return 0, errors.Errorf("unsupported unit: %s", unit)
	}
}
