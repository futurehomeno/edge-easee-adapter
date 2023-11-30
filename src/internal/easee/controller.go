package easee

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
)

// Controller represents a charger controller.
type Controller interface {
	chargepoint.Controller
	numericmeter.Reporter
	UpdateInfo(*Info) error
}

// NewController returns a new instance of Controller.
func NewController(client api.APIClient, manager signalr.Manager, cache config.Cache,
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
	client     api.APIClient
	manager    signalr.Manager
	cache      config.Cache
	cfgService *config.Service
	chargerID  string
	// TODO: needed?
	maxCurrent float64
}

func (c *controller) SetChargepointMaxCurrent(current int64) error {
	return c.client.UpdateMaxCurrent(c.chargerID, float64(current))
}

func (c *controller) ChargepointMaxCurrentReport() (int64, error) {
	if err := c.checkConnection(); err != nil {
		return 0, err
	}

	return c.cache.MaxCurrent(), nil
}

func (c *controller) SetChargepointOfferedCurrent(current int64) error {
	return c.client.UpdateDynamicCurrent(c.chargerID, float64(current))
}

func (c *controller) StartChargepointCharging(settings *chargepoint.ChargingSettings) error {
	// TODO: Remove Mode?

	// switch strings.ToLower(settings.Mode) {
	// case ChargingModeSlow:
	// 	current = c.cfgService.GetSlowChargingCurrentInAmperes()
	// default:
	// 	current = c.maxCurrent
	// }

	return c.client.StartCharging(c.chargerID)
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
		CableCurrent: c.cache.CableCurrent(),
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
		return c.cache.LifetimeEnergy(), nil
	default:
		return 0, fmt.Errorf("unsupported unit: %s", unit)
	}
}

func (a *controller) UpdateInfo(info *Info) error {
	configErr := a.updateChargerConfigInfo(info)
	siteErr := a.updateChargerSiteInfo(info)

	return errors.Join(configErr, siteErr)
}

func (a *controller) updateChargerConfigInfo(info *Info) error {
	cfg, err := a.client.ChargerConfig(info.ChargerID)
	if err != nil {
		if info.GridType == "" {
			return fmt.Errorf("failed to fetch a charger config ID %s: %w", info.ChargerID, err)
		}

		return nil
	}

	gridType, phases := cfg.DetectedPowerGridType.ToFimpGridType()

	info.MaxCurrent = cfg.MaxChargerCurrent
	info.GridType = gridType
	info.Phases = phases

	return nil
}

func (a *controller) updateChargerSiteInfo(info *Info) error {
	siteInfo, err := a.client.ChargerSiteInfo(info.ChargerID)
	if err != nil {
		if info.SupportedMaxCurrent == 0 {
			return fmt.Errorf("failed to fetch a charger site info ID %s: %w", info.ChargerID, err)
		}

		return nil
	}

	info.SupportedMaxCurrent = int64(math.Round(siteInfo.RatedCurrent))

	return nil
}

func (c *controller) checkConnection() error {
	if !c.manager.Connected(c.chargerID) {
		return errors.New("signalR connection is inactive, cannot determine actual state")
	}

	return nil
}
