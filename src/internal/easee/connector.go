package easee

import (
	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/types"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/cache"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/db"
	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
)

type connector struct {
	manager    signalr.Manager
	httpClient api.Client
	confSrv    *config.Service

	chargerID      string
	cache          cache.Cache
	sessionStorage db.ChargingSessionStorage
	thingState     adapter.ThingState
}

func NewConnector(
	manager signalr.Manager,
	httpClient api.Client,
	chargerID string,
	cache cache.Cache,
	confSrv *config.Service,
	sessionStorage db.ChargingSessionStorage,
	thingState adapter.ThingState,
) adapter.Connector {
	return &connector{
		manager:        manager,
		httpClient:     httpClient,
		chargerID:      chargerID,
		cache:          cache,
		confSrv:        confSrv,
		sessionStorage: sessionStorage,
		thingState:     thingState,
	}
}

func (c *connector) Connect(thing adapter.Thing) {
	handler, err := signalr.NewObservationsHandler(thing, c.cache, c.confSrv, c.sessionStorage, c.chargerID, &phaseStore{state: c.thingState})
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
		ConnStatus: adapter.ConnStatusDown,
		ConnType:   adapter.ConnTypeIndirect,
	}

	connected, reason := c.manager.Connected(c.chargerID)

	if !connected {
		log.Debugf("Charger %s not connected reason=%s", c.chargerID, reason)
		return &ret
	}

	ret.ConnStatus = adapter.ConnStatusUp

	return &ret
}

func (c *connector) Ping() *adapter.PingDetails {
	if err := c.httpClient.Ping(); err != nil {
		return &adapter.PingDetails{
			Status: adapter.PingResultFailed,
		}
	}

	if connected, reason := c.manager.Connected(c.chargerID); !connected {
		log.Debugf("Charger %s not connected to SignalR manager reason=%s", c.chargerID, reason)
		return &adapter.PingDetails{
			Status: adapter.PingResultFailed,
		}
	}

	return &adapter.PingDetails{
		Status: adapter.PingResultSuccess,
	}
}

// phaseStore reads and writes the observed leg in the persisted thing state.
type phaseStore struct {
	state adapter.ThingState
}

func (p *phaseStore) OutputPhase() types.PhaseMode {
	state := State{}
	if err := p.state.State(&state); err != nil {
		log.WithError(err).Error("connector: failed to read state")
	}

	return state.OutputPhase
}

func (p *phaseStore) SetOutputPhase(mode types.PhaseMode) error {
	state := State{}
	if err := p.state.State(&state); err != nil {
		return err
	}

	state.OutputPhase = mode

	return p.state.SetState(&state)
}
