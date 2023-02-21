package easee

import (
	"fmt"
	"reflect"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
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
		log.WithError(err).Error("failed to create signalr callbacks")

		return
	}

	c.manager.Register(c.chargerID, c.cache, callbacks)
}

func (c *connector) Disconnect(_ adapter.Thing) {
	c.manager.Unregister(c.chargerID)
}

func (c *connector) Connectivity() *adapter.ConnectivityDetails {
	if c.signalRClient.Connected() {
		return &adapter.ConnectivityDetails{
			ConnectionStatus: adapter.ConnectionStatusUp,
		}
	}

	return &adapter.ConnectivityDetails{
		ConnectionStatus: adapter.ConnectionStatusDown,
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
func (c *connector) signalRCallbacks(thing adapter.Thing) (map[ObservationID]func(), error) {
	chargepoints, err := c.extractChargepointServices(thing)
	if err != nil {
		return nil, err
	}

	meterElecs, err := c.extractMeterElecServices(thing)
	if err != nil {
		return nil, err
	}

	return map[ObservationID]func(){
		ChargerOPState: func() {
			for _, cp := range chargepoints {
				if _, err := cp.SendStateReport(false); err != nil {
					log.WithError(err).Error()
				}
			}
		},
		SessionEnergy: func() {
			for _, cp := range chargepoints {
				if _, err := cp.SendCurrentSessionReport(false); err != nil {
					log.WithError(err).Error()
				}
			}
		},
		CableLocked: func() {
			for _, cp := range chargepoints {
				if _, err := cp.SendCableLockReport(false); err != nil {
					log.WithError(err).Error()
				}
			}
		},
		TotalPower: func() {
			for _, cp := range meterElecs {
				if _, err := cp.SendMeterReport(meterelec.UnitW, false); err != nil {
					log.WithError(err).Error()
				}
			}
		},
		LifetimeEnergy: func() {
			for _, cp := range meterElecs {
				if _, err := cp.SendMeterReport(meterelec.UnitKWh, false); err != nil {
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
			return nil, fmt.Errorf("expected a service to be a chargepoint, got %s instead", reflect.TypeOf(service))
		}

		chargepoints = append(chargepoints, cp)
	}

	return chargepoints, nil
}

func (c *connector) extractMeterElecServices(thing adapter.Thing) ([]meterelec.Service, error) {
	raw := thing.Services(meterelec.MeterElec)
	meterElecs := make([]meterelec.Service, 0, len(raw))

	for _, service := range raw {
		me, ok := service.(meterelec.Service)
		if !ok {
			return nil, fmt.Errorf("expected a service to be a meter_elec, got %s instead", reflect.TypeOf(service))
		}

		meterElecs = append(meterElecs, me)
	}

	return meterElecs, nil
}
