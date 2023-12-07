package easee

import (
	"github.com/futurehomeno/cliffhanger/adapter"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
)

type connector struct {
	manager    signalr.Manager
	httpClient api.Client

	chargerID string
	cache     config.Cache
}

func NewConnector(manager signalr.Manager, httpClient api.Client, chargerID string, cache config.Cache) adapter.Connector {
	return &connector{
		manager:    manager,
		httpClient: httpClient,
		chargerID:  chargerID,
		cache:      cache,
	}
}

func (c *connector) Connect(thing adapter.Thing) {
	handler, err := signalr.NewObservationsHandler(thing, c.cache)
	if err != nil {
		log.WithError(err).Error("failed to create signalRManager callbacks")

		return
	}

	c.manager.Register(c.chargerID, handler)
}

func (c *connector) Disconnect(_ adapter.Thing) {
	if err := c.manager.Unregister(c.chargerID); err != nil {
		log.WithError(err).Error("failed to unregister charger within signalR manager")
	}
}

func (c *connector) Connectivity() *adapter.ConnectivityDetails {
	ret := adapter.ConnectivityDetails{
		ConnectionStatus: adapter.ConnectionStatusDown,
		ConnectionType:   adapter.ConnectionTypeIndirect,
	}

	if c.manager.Connected(c.chargerID) {
		ret.ConnectionStatus = adapter.ConnectionStatusUp
	}

	return &ret
}

func (c *connector) Ping() *adapter.PingDetails {
	if err := c.httpClient.Ping(); err != nil {
		return &adapter.PingDetails{
			Status: adapter.PingResultFailed,
		}
	}

	if !c.manager.Connected(c.chargerID) {
		return &adapter.PingDetails{
			Status: adapter.PingResultFailed,
		}
	}

	return &adapter.PingDetails{
		Status: adapter.PingResultSuccess,
	}
}
