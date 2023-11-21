package easee

import (
	"fmt"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
)

type connector struct {
	manager       SignalRManager
	httpClient    APIClient
	signalRClient signalr.Client

	chargerID string
	cache     ObservationCache
}

func NewConnector(manager SignalRManager, httpClient APIClient, signalRClient signalr.Client, chargerID string, cache ObservationCache) adapter.Connector {
	return &connector{
		manager:       manager,
		httpClient:    httpClient,
		signalRClient: signalRClient,
		chargerID:     chargerID,
		cache:         cache,
	}
}

func (c *connector) Connect(t adapter.Thing) {
	callbacks, err := c.signalRCallbacks(t)
	if err != nil {
		log.WithError(err).Error("failed to create signalRManager callbacks")

		return
	}

	if err := c.manager.Register(c.chargerID, c.cache, callbacks); err != nil {
		log.WithError(err).Error("failed to register charger within signalR manager")

		return
	}
}

func (c *connector) Disconnect(_ adapter.Thing) {
	if err := c.manager.Unregister(c.chargerID); err != nil {
		log.WithError(err).Error("failed to unregister charger within signalR manager")
	}
}

func (c *connector) Connectivity() *adapter.ConnectivityDetails {
	if c.signalRClient.Connected() {
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

	if !c.signalRClient.Connected() {
		return &adapter.PingDetails{
			Status: adapter.PingResultFailed,
		}
	}

	return &adapter.PingDetails{
		Status: adapter.PingResultSuccess,
	}
}

//nolint:cyclop
func (c *connector) signalRCallbacks(thing adapter.Thing) (map[signalr.ObservationID]func(), error) {
	chargepoints, err := c.extractChargepointServices(thing)
	if err != nil {
		return nil, err
	}

	meterElecs, err := c.extractMeterElecServices(thing)
	if err != nil {
		return nil, err
	}

	return map[signalr.ObservationID]func(){
		signalr.ChargerOPState: func() {
			for _, cp := range chargepoints {
				if _, err := cp.SendStateReport(false); err != nil {
					log.WithError(err).Error()
				}
			}
		},
		signalr.SessionEnergy: func() {
			for _, cp := range chargepoints {
				if _, err := cp.SendCurrentSessionReport(false); err != nil {
					log.WithError(err).Error()
				}
			}
		},
		signalr.CableLocked: func() {
			for _, cp := range chargepoints {
				if _, err := cp.SendCableLockReport(false); err != nil {
					log.WithError(err).Error()
				}
			}
		},
		signalr.TotalPower: func() {
			for _, cp := range meterElecs {
				if _, err := cp.SendMeterReport(numericmeter.UnitW, false); err != nil {
					log.WithError(err).Error()
				}
			}
		},
		signalr.LifetimeEnergy: func() {
			for _, cp := range meterElecs {
				if _, err := cp.SendMeterReport(numericmeter.UnitKWh, false); err != nil {
					log.WithError(err).Error()
				}
			}
		},
	}, nil
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
