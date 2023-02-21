package easee

import (
	"fmt"
	"strings"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/pkg/errors"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// Controller represents a charger controller.
type Controller interface {
	chargepoint.Controller
	meterelec.Reporter
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

func (c *controller) StartChargepointCharging(mode string) error {
	var current float64

	switch strings.ToLower(mode) {
	case ChargingModeSlow:
		current = c.cfgService.GetSlowChargingCurrentInAmperes()
	default:
		current = c.maxCurrent
	}

	if err := c.client.StartCharging(c.chargerID, current); err != nil {
		return fmt.Errorf("failed to start charging session for charger id %s: %w", c.chargerID, err)
	}

	c.backoff()

	return nil
}

func (c *controller) StopChargepointCharging() error {
	if err := c.client.StopCharging(c.chargerID); err != nil {
		return fmt.Errorf("failed to stop charging session for charger id %s: %w", c.chargerID, err)
	}

	c.backoff()

	return nil
}

func (c *controller) SetChargepointCableLock(locked bool) error {
	if err := c.client.SetCableLock(c.chargerID, locked); err != nil {
		return err
	}

	c.backoff()

	return nil
}

func (c *controller) ChargepointCableLockReport() (bool, error) {
	return c.cache.CableLocked()
}

func (c *controller) ChargepointCurrentSessionReport() (float64, error) {
	return c.cache.SessionEnergy()
}

func (c *controller) ChargepointStateReport() (string, error) {
	power, err := c.cache.TotalPower()
	if err != nil {
		return "", err
	}

	// If a charger reports power usage, assume a charging state.
	if power > 0 {
		return Charging.String(), nil
	}

	return c.cache.ChargerState()
}

func (c *controller) ElectricityMeterReport(unit string) (float64, error) {
	switch unit {
	case meterelec.UnitW:
		return c.cache.TotalPower()
	case meterelec.UnitKWh:
		return c.cache.LifetimeEnergy()
	default:
		return 0, errors.Errorf("unsupported unit: %s", unit)
	}
}

// backoff allows Easee cloud to process the request and invalidates local cache.
func (c *controller) backoff() {
	time.Sleep(c.cfgService.GetEaseeBackoff())
}
