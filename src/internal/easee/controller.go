package easee

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/cache"
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
		client:                  client,
		manager:                 manager,
		cache:                   cache,
		cfgService:              cfgService,
		chargerID:               chargerID,
		maxCurrent:              maxCurrent,
		chargeSessionsRefresher: newChargeSessionsRefresher(client, chargerID, cfgService.GetPollingInterval()),
	}
}

type controller struct {
	client                  api.APIClient
	manager                 signalr.Manager
	cache                   config.Cache
	cfgService              *config.Service
	chargerID               string
	maxCurrent              float64 // TODO: needed?
	chargeSessionsRefresher cache.Refresher[api.ChargeSessions]
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

	sessions, err := c.retrieveChargeSessions()
	if err != nil {
		return nil, err
	}

	ret := chargepoint.SessionReport{}
	ret.SessionEnergy = c.cache.EnergySession()

	if latest := sessions.LatestSession(); latest != nil {
		ret.StartedAt = latest.CarConnected
		ret.FinishedAt = latest.CarDisconnected

		if !latest.IsComplete {
			ret.OfferedCurrent = c.cache.OfferedCurrent()
		}
	}

	if prev := sessions.PreviousSession(); prev != nil {
		ret.PreviousSessionEnergy = prev.KiloWattHours
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
		return c.cache.LifetimeEnergy(), nil
	default:
		return 0, fmt.Errorf("unsupported unit: %s", unit)
	}
}

func (c *controller) UpdateInfo(info *Info) error {
	configErr := c.updateChargerConfigInfo(info)
	siteErr := c.updateChargerSiteInfo(info)

	return errors.Join(configErr, siteErr)
}

func (c *controller) updateChargerConfigInfo(info *Info) error {
	cfg, err := c.client.ChargerConfig(info.ChargerID)
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

func (c *controller) updateChargerSiteInfo(info *Info) error {
	siteInfo, err := c.client.ChargerSiteInfo(info.ChargerID)
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

// retrieveChargeSessions retrieves charge sessions from refresher cache.
func (c *controller) retrieveChargeSessions() (api.ChargeSessions, error) {
	sessions, err := c.chargeSessionsRefresher.Refresh()
	if err != nil {
		return nil, fmt.Errorf("controller: failed to refresh charge sessions: %w", err)
	}

	return sessions, nil
}

// newChargeSessionsRefresher creates new instance of a charge sessions refresher cache.
func newChargeSessionsRefresher(client api.APIClient, id string, interval time.Duration) cache.Refresher[api.ChargeSessions] {
	refresh := func() (api.ChargeSessions, error) {
		sessions, err := client.ChargerSessions(id)
		if err != nil {
			return nil, fmt.Errorf("controller: failed to get charges history: %w", err)
		}

		return sessions, nil
	}

	return cache.NewRefresher(refresh, interval)
}
