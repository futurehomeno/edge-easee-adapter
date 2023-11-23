package easee

import (
	"errors"

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
	chargepoint, err := c.getChargepointService(thing)
	if err != nil {
		return nil, err
	}

	meterElec, err := c.getMeterElecService(thing)
	if err != nil {
		return nil, err
	}

	return NewObservationsHandler(chargepoint, meterElec, c.cache), nil
}

func (c *connector) getChargepointService(thing adapter.Thing) (chargepoint.Service, error) {
	for _, service := range thing.Services(chargepoint.Chargepoint) {
		if service, ok := service.(chargepoint.Service); ok {
			return service, nil
		}
	}

	return nil, errors.New("There are no chargepoint services")
}

func (c *connector) getMeterElecService(thing adapter.Thing) (numericmeter.Service, error) {
	for _, service := range thing.Services(numericmeter.MeterElec) {
		if service, ok := service.(numericmeter.Service); ok {
			return service, nil
		}
	}

	return nil, errors.New("There are no meterelec services")
}
