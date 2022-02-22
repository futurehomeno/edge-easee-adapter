package easee

import (
	"fmt"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/pkg/errors"
)

type Controller interface {
	chargepoint.Controller
	meterelec.Reporter
}

func NewController(client Client, chargerID string) Controller {
	return &controller{client: client, chargerID: chargerID}
}

type controller struct {
	client    Client
	chargerID string
}

func (c *controller) StartChargepointCharging() error {
	if err := c.client.StartCharging(c.chargerID); err != nil {
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
	state, err := c.client.ChargerState(c.chargerID)
	if err != nil {
		return false, errors.Wrap(err, "failed to fetch charger state")
	}

	return state.CableLocked, nil
}

func (c *controller) ChargepointCurrentSessionReport() (float64, error) {
	state, err := c.client.ChargerState(c.chargerID)
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
	state, err := c.client.ChargerState(c.chargerID)
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
