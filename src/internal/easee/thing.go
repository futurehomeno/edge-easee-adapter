package easee

import (
	"fmt"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// Info is an object representing charger persisted information.
type Info struct {
	ChargerID  string  `json:"chargerID"`
	MaxCurrent float64 `json:"maxCurrent"`
}

type thingFactory struct {
	client         APIClient
	cfgService     *config.Service
	signalRManager SignalRManager
}

// NewThingFactory returns a new instance of adapter.ThingFactory.
func NewThingFactory(client APIClient, cfgService *config.Service, signalRManager SignalRManager) adapter.ThingFactory {
	return &thingFactory{
		client:         client,
		cfgService:     cfgService,
		signalRManager: signalRManager,
	}
}

func (t *thingFactory) Create(ad adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
	info := &Info{}

	if err := thingState.Info(info); err != nil {
		return nil, fmt.Errorf("factory: failed to retrieve information: %w", err)
	}

	cache := NewObservationCache()
	controller := NewController(t.client, t.signalRManager, cache, t.cfgService, info.ChargerID, info.MaxCurrent)

	groups := []string{"ch_0"}

	return thing.NewCarCharger(publisher, thingState, &thing.CarChargerConfig{
		ThingConfig: &adapter.ThingConfig{
			Connector:       NewConnector(t.signalRManager, t.client, info.ChargerID, cache),
			InclusionReport: t.inclusionReport(info, thingState, groups),
		},
		ChargepointConfig: &chargepoint.Config{
			Specification: t.chargepointSpecification(ad, thingState, groups),
			Controller:    controller,
		},
		MeterElecConfig: &numericmeter.Config{
			Specification: t.meterElecSpecification(ad, thingState, groups),
			Reporter:      controller,
		},
	}), nil
}

func (t *thingFactory) inclusionReport(info *Info, thingState adapter.ThingState, groups []string) *fimptype.ThingInclusionReport {
	return &fimptype.ThingInclusionReport{
		Address:        thingState.Address(),
		ProductHash:    "Easee - Easee - Easee Home",
		ProductName:    "Easee Home",
		DeviceId:       info.ChargerID,
		CommTechnology: "cloud",
		ManufacturerId: "Easee",
		PowerSource:    "ac",
		WakeUpInterval: "-1",
		Groups:         groups,
	}
}

func (t *thingFactory) chargepointSpecification(adapter adapter.Adapter, thingState adapter.ThingState, groups []string) *fimptype.Service {
	var supportedStates []chargepoint.State
	for _, s := range SupportedChargingStates() {
		supportedStates = append(supportedStates, s.ToFimpState())
	}

	var supportedModes []string
	for _, m := range SupportedChargingModes() {
		supportedModes = append(supportedModes, m)
	}

	return chargepoint.Specification(
		adapter.Name(),
		adapter.Address(),
		thingState.Address(),
		groups,
		supportedStates,
		chargepoint.WithChargingModes(supportedModes...),
		// TODO
		// chargepoint.WithPhases(info.Phases),
		// chargepoint.WithSupportedMaxCurrent(info.MaxCurrent),
		// chargepoint.WithGridType(info.GridType),
	)
}

func (t *thingFactory) meterElecSpecification(adapter adapter.Adapter, thingState adapter.ThingState, groups []string) *fimptype.Service {
	return numericmeter.Specification(
		numericmeter.MeterElec,
		adapter.Name(),
		adapter.Address(),
		thingState.Address(),
		groups,
		[]numericmeter.Unit{numericmeter.UnitW, numericmeter.UnitKWh},
	)
}
