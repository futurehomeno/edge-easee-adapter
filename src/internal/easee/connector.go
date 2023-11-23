package easee

import (
	"fmt"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	log "github.com/sirupsen/logrus"
)

type connector struct {
	manager    SignalRManager
	httpClient APIClient

	chargerID string
	cache     ObservationCache
}

func NewConnector(manager SignalRManager, httpClient APIClient, chargerID string, cache ObservationCache) adapter.Connector {
	return &connector{
		manager:    manager,
		httpClient: httpClient,
		chargerID:  chargerID,
		cache:      cache,
	}
}

func (c *connector) Connect(thing adapter.Thing) {
	handler, err := c.getObservationsHandler(thing)
	if err != nil {
		log.WithError(err).Error("failed to create signalRManager callbacks")

		return
	}

	if err := c.manager.Register(c.chargerID, handler); err != nil {
		log.WithError(err).Error("failed to register charger within signalR manager")
	}
}

func (c *connector) Disconnect(_ adapter.Thing) {
	if err := c.manager.Unregister(c.chargerID); err != nil {
		log.WithError(err).Error("failed to unregister charger within signalR manager")
	}
}

func (c *connector) Connectivity() *adapter.ConnectivityDetails {
	if c.manager.Connected() {
		return &adapter.ConnectivityDetails{
			ConnectionStatus: adapter.ConnectionStatusUp,
			ConnectionType:   adapter.ConnectionTypeIndirect,
		}
	}

	return &adapter.ConnectivityDetails{
		ConnectionStatus: adapter.ConnectionStatusDown,
		ConnectionType:   adapter.ConnectionTypeIndirect,
	}
}

func (c *connector) Ping() *adapter.PingDetails {
	if err := c.httpClient.Ping(); err != nil {
		return &adapter.PingDetails{
			Status: adapter.PingResultFailed,
		}
	}

	if !c.manager.Connected() {
		return &adapter.PingDetails{
			Status: adapter.PingResultFailed,
		}
	}

	return &adapter.PingDetails{
		Status: adapter.PingResultSuccess,
	}
}

func (c *connector) getObservationsHandler(thing adapter.Thing) (ObservationsHandler, error) {
	chargepoints, err := c.extractChargepointServices(thing)
	if err != nil {
		return nil, err
	}

	meterElecs, err := c.extractMeterElecServices(thing)
	if err != nil {
		return nil, err
	}

	return NewObservationsHandler(chargepoints, meterElecs, c.cache), nil
}

func (c *connector) extractChargepointServices(thing adapter.Thing) ([]chargepoint.Service, error) {
	raw := thing.Services(chargepoint.Chargepoint)
	chargepoints := make([]chargepoint.Service, 0, len(raw))

	for _, service := range raw {
		cp, ok := service.(chargepoint.Service)
		if !ok {
			return nil, fmt.Errorf("expected a service to be a chargepoint, got %T instead", service)
		}

		chargepoints = append(chargepoints, cp)
	}

	return chargepoints, nil
}

func (c *connector) extractMeterElecServices(thing adapter.Thing) ([]numericmeter.Service, error) {
	raw := thing.Services(numericmeter.MeterElec)
	meterElecs := make([]numericmeter.Service, 0, len(raw))

	for _, service := range raw {
		nm, ok := service.(numericmeter.Service)
		if !ok {
			return nil, fmt.Errorf("expected a service to be a numeric_meter, got %T instead", service)
		}

		meterElecs = append(meterElecs, nm)
	}

	return meterElecs, nil
}
