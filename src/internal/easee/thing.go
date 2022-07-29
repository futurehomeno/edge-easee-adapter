package easee

import (
	"fmt"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// Info is an object representing charger persisted information.
type Info struct {
	ChargerID  string  `json:"chargerID"`
	MaxCurrent float64 `json:"maxCurrent"`
}

type thingFactory struct {
	client     Client
	cfgService *config.Service
}

// NewThingFactory returns a new instance of adapter.ThingFactory.
func NewThingFactory(client Client, cfgService *config.Service) adapter.ThingFactory {
	return &thingFactory{
		client:     client,
		cfgService: cfgService,
	}
}

func (t *thingFactory) Create(mqtt *fimpgo.MqttTransport, adapter adapter.ExtendedAdapter, thingState adapter.ThingState) (adapter.Thing, error) {
	info := &Info{}

	if err := thingState.Info(info); err != nil {
		return nil, fmt.Errorf("factory: failed to retrieve information: %w", err)
	}

	controller := NewController(t.client, t.cfgService, info.ChargerID, info.MaxCurrent)
	groups := []string{"ch_0"}

	return thing.NewCarCharger(mqtt, &thing.CarChargerConfig{
		InclusionReport: t.inclusionReport(info, thingState, groups),
		ChargepointConfig: &chargepoint.Config{
			Specification: t.chargepointSpecification(adapter, thingState, groups),
			Controller:    controller,
		},
		MeterElecConfig: &meterelec.Config{
			Specification: t.meterElecSpecification(adapter, thingState, groups),
			Reporter:      controller,
		},
	}), nil
}

func (t *thingFactory) inclusionReport(info *Info, thingState adapter.ThingState, groups []string) *fimptype.ThingInclusionReport {
	return &fimptype.ThingInclusionReport{
		Address:        thingState.Address(),
		Alias:          "easee",
		ProductHash:    "EaseeHome" + info.ChargerID,
		CommTechnology: "cloud",
		ProductName:    "Easee Home",
		ManufacturerId: "easee",
		DeviceId:       info.ChargerID,
		PowerSource:    "ac",
		Groups:         groups,
	}
}

func (t *thingFactory) chargepointSpecification(adapter adapter.ExtendedAdapter, thingState adapter.ThingState, groups []string) *fimptype.Service {
	return chargepoint.Specification(
		adapter.Name(),
		adapter.Address(),
		thingState.Address(),
		groups,
		SupportedChargingStates(),
		SupportedChargingModes(),
	)
}

func (t *thingFactory) meterElecSpecification(adapter adapter.ExtendedAdapter, thingState adapter.ThingState, groups []string) *fimptype.Service {
	return meterelec.Specification(
		adapter.Name(),
		adapter.Address(),
		thingState.Address(),
		groups,
		[]string{meterelec.UnitW, meterelec.UnitKWh},
		nil,
	)
}
