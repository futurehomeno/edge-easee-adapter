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
func NewController(client APIClient, manager SignalRManager, cache ObservationCache,
	cfgService *config.Service, chargerID string, maxCurrent float64) Controller {
	return &controller{
		client:     client,
		manager:    manager,
		cache:      cache,
		cfgService: cfgService,
		chargerID:  chargerID,
		maxCurrent: maxCurrent,
	}
}

type controller struct {
	client     APIClient
	manager    SignalRManager
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
	if err := c.checkConnection(); err != nil {
		return nil, err
	}

	return &chargepoint.CableReport{
		CableLock:    c.cache.CableLocked(),
		CableCurrent: 0, // TODO
	}, nil
}

func (c *controller) ChargepointCurrentSessionReport() (*chargepoint.SessionReport, error) {
	if err := c.checkConnection(); err != nil {
		return nil, err
	}

	return &chargepoint.SessionReport{
		SessionEnergy:         c.cache.SessionEnergy(),
		PreviousSessionEnergy: 0,           // TODO
		StartedAt:             time.Time{}, // TODO
		FinishedAt:            time.Time{}, // TODO
		OfferedCurrent:        0,           // TODO
	}, nil
}

func (c *controller) ChargepointStateReport() (chargepoint.State, error) {
	if err := c.checkConnection(); err != nil {
		return "", err
	}

	// If a charger reports power usage, assume a charging state.
	if power := c.cache.TotalPower(); power > 0 {
		return chargepoint.StateCharging, nil
	}

	state := c.cache.ChargerState()

	return state.ToFimpState(), nil
}

func (c *controller) MeterReport(unit numericmeter.Unit) (float64, error) {
	if err := c.checkConnection(); err != nil {
		return 0, err
	}

	switch unit { //nolint:exhaustive
	case numericmeter.UnitW:
		return c.cache.TotalPower(), nil
	case numericmeter.UnitKWh:
		return c.cache.LifetimeEnergy(), nil
	default:
		return 0, errors.Errorf("unsupported unit: %s", unit)
	}
}

func (c *controller) checkConnection() error {
	if !c.manager.Connected() {
		return errors.New("signalR connection is inactive, cannot determine actual state")
	}

	return nil
}
