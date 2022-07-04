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
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// Controller represents a charger controller.
type Controller interface {
	chargepoint.Controller
	meterelec.Reporter
}

// NewController returns a new instance of Controller.
func NewController(client Client, cfgService *config.Service, chargerID string, maxCurrent float64) Controller {
	return &controller{
		client:         client,
		cfgService:     cfgService,
		chargerID:      chargerID,
		maxCurrent:     maxCurrent,
		stateRefresher: newStateRefresher(client, chargerID, cfgService.GetPollingInterval()),
	}
}

type controller struct {
	client         Client
	cfgService     *config.Service
	chargerID      string
	maxCurrent     float64
	stateRefresher cache.Refresher
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
	state, err := c.cachedChargerState()
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch charger state")
	}

	return state.ChargerOpMode.String(), nil
}

func (c *controller) ElectricityMeterReport(unit string) (float64, error) {
	switch unit {
	case meterelec.UnitW:
		return c.power()
	case meterelec.UnitKWh:
		return c.energy()
	case meterelec.UnitV:
		return c.voltage()
	default:
		return 0, errors.Errorf("unsupported unit: %s", unit)
	}
}

func (c *controller) cachedChargerState() (*ChargerState, error) {
	rawState, err := c.stateRefresher.Refresh()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current charger state from state refresher")
	}

	state, ok := rawState.(*ChargerState)
	if !ok {
		return nil, errors.Errorf("expected %s, got %s instead", reflect.TypeOf(&ChargerState{}), reflect.TypeOf(rawState))
	}

	return state, nil
}

// backoff allows Easee cloud to process the request and invalidates local cache.
func (c *controller) backoff() {
	time.Sleep(c.cfgService.GetEaseeBackoff())

	c.stateRefresher.Reset()
}

func (c *controller) energy() (float64, error) {
	now := clock.Now().UTC().Truncate(time.Hour)
	lastMeasurement := c.cfgService.GetLastEnergyReport()

	if lastMeasurement.Timestamp.Equal(now) {
		return 0, nil
	}

	if lastMeasurement.Timestamp.IsZero() {
		return c.reportEnergy(lastMeasurement, now.Add(-2*time.Hour), now, lastEnergyValue)
	}

	return c.reportEnergy(lastMeasurement, lastMeasurement.Timestamp.Add(time.Second), now, sumEnergyValues)
}

func (c *controller) reportEnergy(measurement config.EnergyReport, from, to time.Time, strategy func(measurements []Measurement) float64) (float64, error) {
	measurements, err := c.client.EnergyPerHour(c.chargerID, from, to)
	if err != nil {
		return 0, errors.Wrap(err, "failed to fetch lifetime energy measurements")
	}

	if len(measurements) == 0 {
		log.Warn("controller: energy measurements are not available, skipping...")

		return 0, nil
	}

	measurement.Timestamp = to
	measurement.Value = strategy(measurements)

	if err := c.cfgService.SetLastEnergyReport(measurement); err != nil {
		log.Error("failed to save energy measurement", err)
	}

	return measurement.Value, nil
}

func (c *controller) power() (float64, error) {
	state, err := c.cachedChargerState()
	if err != nil {
		return 0, errors.Wrap(err, "failed to fetch charger state")
	}

	return state.TotalPower * 1000, nil // TotalPower is in kW
}

func (c *controller) voltage() (float64, error) {
	state, err := c.cachedChargerState()
	if err != nil {
		return 0, errors.Wrap(err, "failed to fetch charger state")
	}

	return state.Voltage, nil
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

func sumEnergyValues(measurements []Measurement) float64 {
	var sum float64
	for _, m := range measurements {
		sum += m.Value
	}

	return sum
}

func lastEnergyValue(measurements []Measurement) float64 {
	return measurements[len(measurements)-1].Value
}
