package easee

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/cache"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/michalkurzeja/go-clock"
	"github.com/pkg/errors"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

const (
	modeRefresher           = "mode"
	sessionEnergyRefresher  = "session-energy"
	cableLockedRefresher    = "cable-locked"
	totalPowerRefresher     = "total-power"
	lifetimeEnergyRefresher = "lifetime-energy"
)

// Controller represents a charger controller.
type Controller interface {
	chargepoint.Controller
	meterelec.Reporter
}

// NewController returns a new instance of Controller.
func NewController(client Client, cfgService *config.Service, chargerID string, maxCurrent float64) Controller {
	ctrl := &controller{
		client:           client,
		cfgService:       cfgService,
		chargerID:        chargerID,
		maxCurrent:       maxCurrent,
		refresherManager: newRefresherManager(),
	}

	ctrl.refresherManager.register(modeRefresher, newObservationRefresher(client, chargerID, ChargerOPMode, cfgService.GetPollingInterval()))
	ctrl.refresherManager.register(sessionEnergyRefresher, newObservationRefresher(client, chargerID, SessionEnergy, cfgService.GetPollingInterval()))
	ctrl.refresherManager.register(cableLockedRefresher, newObservationRefresher(client, chargerID, CableLocked, cfgService.GetPollingInterval()))
	ctrl.refresherManager.register(totalPowerRefresher, newObservationRefresher(client, chargerID, TotalPower, cfgService.GetPollingInterval()))
	ctrl.refresherManager.register(lifetimeEnergyRefresher, newObservationRefresher(client, chargerID, LifetimeEnergy, cfgService.GetPollingInterval()))

	return ctrl
}

type controller struct {
	client           Client
	cfgService       *config.Service
	refresherManager *refresherManager
	chargerID        string
	maxCurrent       float64
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
	rawLock, err := c.refresherManager.getValue(cableLockedRefresher)
	if err != nil {
		return false, errors.Wrap(err, "failed to get current cable locked state from cable locked refresher")
	}

	locked, ok := rawLock.(bool)
	if !ok {
		return false, errors.Errorf("expected bool, got %s instead", reflect.TypeOf(rawLock))
	}

	return locked, nil
}

func (c *controller) ChargepointCurrentSessionReport() (float64, error) {
	mode, err := c.ChargepointStateReport()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get charger mode")
	}

	if !c.sessionReportAvailable(mode) {
		return 0, nil
	}

	rawEnergy, err := c.refresherManager.getValue(sessionEnergyRefresher)
	if err != nil {
		return 0, errors.Wrap(err, "failed to fetch session energy")
	}

	energy, ok := rawEnergy.(float64)
	if !ok {
		return 0, errors.Errorf("expected float64, got %s instead", reflect.TypeOf(rawEnergy))
	}

	return energy, nil
}

func (c *controller) ChargepointStateReport() (string, error) {
	rawMode, err := c.refresherManager.getValue(modeRefresher)
	if err != nil {
		return "", errors.Wrap(err, "failed to get current charger mode from mode refresher")
	}

	mode, ok := rawMode.(float64)
	if !ok {
		return "", errors.Errorf("expected float64, got %s instead", reflect.TypeOf(rawMode))
	}

	return ChargerMode(mode).String(), nil
}

func (c *controller) ElectricityMeterReport(unit string) (float64, error) {
	switch unit {
	case meterelec.UnitW:
		return c.totalPower()
	case meterelec.UnitKWh:
		return c.lifetimeEnergy()
	default:
		return 0, errors.Errorf("unsupported unit: %s", unit)
	}
}

func (c *controller) totalPower() (float64, error) {
	rawPower, err := c.refresherManager.getValue(totalPowerRefresher)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get current total power from total power refresher")
	}

	power, ok := rawPower.(float64)
	if !ok {
		return 0, errors.Errorf("expected float64, got %s instead", reflect.TypeOf(rawPower))
	}

	return power * 1000, nil // convert to watts from kW
}

func (c *controller) lifetimeEnergy() (float64, error) {
	rawEnergy, err := c.refresherManager.getValue(lifetimeEnergyRefresher)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get current lifetime energy from lifetime energy refresher")
	}

	energy, ok := rawEnergy.(float64)
	if !ok {
		return 0, errors.Errorf("expected float64, got %s instead", reflect.TypeOf(rawEnergy))
	}

	return energy, nil
}

// backoff allows Easee cloud to process the request and invalidates local cache.
func (c *controller) backoff() {
	time.Sleep(c.cfgService.GetEaseeBackoff())

	c.refresherManager.reset()
}

func (c *controller) sessionReportAvailable(mode string) bool {
	return mode == ChargerModeCharging || mode == ChargerModeFinished
}

func newObservationRefresher(client Client, chargerID string, obID ObservationID, interval time.Duration) cache.Refresher {
	return cache.NewRefresher(
		observationsRefreshFn(client, chargerID, obID),
		cache.OffsetInterval(interval),
	)
}

func observationsRefreshFn(client Client, chargerID string, obID ObservationID) func() (interface{}, error) {
	return func() (interface{}, error) {
		now := clock.Now().UTC()
		yearAgo := now.Add(-365 * 24 * time.Hour)

		obs, err := client.Observations(chargerID, obID, yearAgo, now)
		if err != nil {
			return nil, fmt.Errorf("controller: failed to fetch observations for charger ID %s and observation ID %d: %w", chargerID, obID, err)
		}

		if len(obs) == 0 {
			return nil, fmt.Errorf("controller: no observations found for charger ID %s and observation ID %d", chargerID, obID)
		}

		last := obs[len(obs)-1]

		return last.Value, nil
	}
}

type refresherManager struct {
	refreshers map[string]cache.Refresher
}

func newRefresherManager() *refresherManager {
	return &refresherManager{
		refreshers: make(map[string]cache.Refresher),
	}
}

func (r *refresherManager) register(name string, refresher cache.Refresher) {
	r.refreshers[name] = refresher
}

func (r *refresherManager) getValue(name string) (any, error) {
	refresher, ok := r.refreshers[name]
	if !ok {
		return nil, fmt.Errorf("controller: no refresher found for name %s", name)
	}

	value, err := refresher.Refresh()
	if err != nil {
		return nil, err
	}

	return value, nil
}

func (r *refresherManager) reset() {
	for _, ref := range r.refreshers {
		ref.Reset()
	}
}
