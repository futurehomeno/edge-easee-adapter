package easee

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	cliffCache "github.com/futurehomeno/cliffhanger/adapter/cache"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/event"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/cache"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/pubsub"
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
		report[numericmeter.ValueEnergyImport] = c.LifetimeEnergy()
	},
}

type specFunc func(report numericmeter.ValuesReport, c cache.Cache)

// Controller represents a charger controller.
type Controller interface {
	chargepoint.Controller
	numericmeter.Reporter
	numericmeter.ExtendedReporter
	UpdateInfo(*Info) error
}

// NewController returns a new instance of Controller.
func NewController(
	client api.Client,
	manager signalr.Manager,
	cache cache.Cache,
	cfgService *config.Service,
	chargerID string,
	eventManager event.Manager,
) Controller {
	return &controller{
		client:                  client,
		manager:                 manager,
		cache:                   cache,
		cfgService:              cfgService,
		chargerID:               chargerID,
		chargeSessionsRefresher: newChargeSessionsRefresher(client, chargerID, cfgService.GetPollingInterval()),
		eventManager:            eventManager,
	}
}

type controller struct {
	client                  api.Client
	manager                 signalr.Manager
	cache                   cache.Cache
	cfgService              *config.Service
	chargerID               string
	chargeSessionsRefresher cliffCache.Refresher[api.ChargeSessions]
	eventManager            event.Manager
}

func (c *controller) SetChargepointMaxCurrent(current int64) error {
	done := make(chan struct{})

	listener, err := c.startMaxCurrentListener(current, done)
	if err != nil {
		return err
	}

	defer stopListener(listener)

	err = c.client.UpdateMaxCurrent(c.chargerID, float64(current))
	if err != nil {
		return err
	}

	return c.waitForCurrentEvent(c.cache.OfferedCurrent(), done)
}

func (c *controller) ChargepointMaxCurrentReport() (int64, error) {
	if err := c.checkConnection(); err != nil {
		return 0, err
	}

	return c.cache.MaxCurrent(), nil
}

func (c *controller) SetChargepointOfferedCurrent(current int64) error {
	done := make(chan struct{})

	listener, err := c.startOfferedCurrentListener(current, done)
	if err != nil {
		return err
	}

	defer stopListener(listener)

	err = c.client.UpdateDynamicCurrent(c.chargerID, float64(current))
	if err != nil {
		return err
	}

	return c.waitForCurrentEvent(current, done)
}

func (c *controller) StartChargepointCharging(settings *chargepoint.ChargingSettings) error {
	startCurrent := float64(c.cache.MaxCurrent())

	if offered := c.cache.OfferedCurrent(); offered > 0 {
		startCurrent = float64(offered)
	}

	if strings.ToLower(settings.Mode) == ChargingModeSlow {
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

	ret := chargepoint.SessionReport{
		SessionEnergy: c.cache.EnergySession(),
	}

	if latest := sessions.Latest(); latest != nil {
		ret.StartedAt = latest.CarConnected
		ret.FinishedAt = latest.CarDisconnected

		if !latest.IsComplete {
			ret.OfferedCurrent = min(c.cache.DynamicCurrent(), c.cache.MaxCurrent())
		}
	}

	if prev := sessions.Previous(); prev != nil {
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

func (c *controller) MeterExtendedReport(values numericmeter.Values) (numericmeter.ValuesReport, error) {
	if err := c.checkConnection(); err != nil {
		return nil, err
	}

	ret := make(numericmeter.ValuesReport)

	for _, value := range values {
		if f, ok := extendedReportMapping[value]; ok {
			f(ret, c.cache)
		}
	}

	return ret, nil
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

	info.SupportedMaxCurrent = min(int64(math.Round(siteInfo.RatedCurrent)), maxCurrentValue)

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
func newChargeSessionsRefresher(client api.Client, id string, interval time.Duration) cliffCache.Refresher[api.ChargeSessions] {
	refresh := func() (api.ChargeSessions, error) {
		sessions, err := client.ChargerSessions(id)
		if err != nil {
			return nil, fmt.Errorf("controller: failed to get charges history: %w", err)
		}

		return sessions, nil
	}

	return cliffCache.NewRefresher(refresh, interval)
}

func (c *controller) startOfferedCurrentListener(current int64, done chan struct{}) (event.Listener, error) {
	processor := event.ProcessorFn(func(event *event.Event) {
		close(done)
	})

	listener := pubsub.NewOfferedCurrentListener(c.eventManager, current, processor)

	err := listener.Start()

	return listener, err
}

func (c *controller) startMaxCurrentListener(current int64, done chan struct{}) (event.Listener, error) {
	processor := event.ProcessorFn(func(event *event.Event) {
		close(done)
	})

	listener := pubsub.NewMaxCurrentListener(c.eventManager, current, processor)

	err := listener.Start()

	return listener, err
}

func (c *controller) waitForCurrentEvent(offeredCurrent int64, done chan struct{}) error {
	timer := time.NewTimer(c.cfgService.GetPollingInterval())
	defer timer.Stop()

	select {
	case <-timer.C:
		return errors.New("timeout")
	case <-done:
		if offeredCurrent != c.cache.OfferedCurrent() {
			c.cache.SetOfferedCurrent(offeredCurrent)
		}

		return nil
	}
}

func stopListener(listener event.Listener) {
	if err := listener.Stop(); err != nil {
		log.Errorf("error during stopping listener: %v", err)
	}
}
