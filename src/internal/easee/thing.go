package easee

import (
	"fmt"
	"slices"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	cliffCache "github.com/futurehomeno/cliffhanger/adapter/cache"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/cache"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/db"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
)

// Info is an object representing charger persisted information.
type Info struct {
	ChargerID string `json:"chargerID"`
	Product   string `json:"product"`
}

// State is an object representing charger persisted mutable information.
type State struct {
	GridType            chargepoint.GridType `json:"gridType"`
	Phases              int                  `json:"phases"`
	PhaseMode           int                  `json:"phaseMode"`
	SupportedMaxCurrent int64                `json:"supportedMaxCurrent"`
}

func (s *State) IsConfigUpdateNeeded() bool {
	return s.GridType == ""
}

func (s *State) IsSiteUpdateNeeded() bool {
	return s.SupportedMaxCurrent == 0
}

type thingFactory struct {
	client         api.Client
	cfgService     *config.Service
	signalRManager signalr.Manager
	sessionStorage db.ChargingSessionStorage
}

// NewThingFactory returns a new instance of adapter.ThingFactory.
func NewThingFactory(
	client api.Client,
	cfgService *config.Service,
	signalRManager signalr.Manager,
	sessionStorage db.ChargingSessionStorage,
) adapter.ThingFactory {
	return &thingFactory{
		client:         client,
		cfgService:     cfgService,
		signalRManager: signalRManager,
		sessionStorage: sessionStorage,
	}
}

func (t *thingFactory) Create(ad adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
	info := &Info{}

	if err := thingState.Info(info); err != nil {
		return nil, fmt.Errorf("factory: failed to retrieve information: %w", err)
	}

	thingCache := cache.NewCache()
	controller := NewController(t.signalRManager, t.client, info.ChargerID, thingCache, t.cfgService, t.sessionStorage)

	state := &State{}
	if err := thingState.State(state); err != nil {
		log.WithError(err).Warnf("factory: failed to retrieve state: %v", err)
	}

	if err := controller.UpdateState(info.ChargerID, state); err != nil {
		return nil, err
	}

	if err := thingState.SetState(state); err != nil {
		log.WithError(err).Warnf("factory: failed to set state: %v", err)
	}

	thingCache.SetGridType(state.GridType)
	thingCache.SetPhases(state.Phases)
	thingCache.SetPhaseMode(state.PhaseMode)

	groups := []string{"ch_0"}
	services := []adapter.Service{
		t.newChargepointService(publisher, ad, thingState, groups, controller, state),
		t.newMeterElecService(publisher, ad, thingState, groups, controller),
		t.newParametersService(publisher, ad, thingState, groups, controller),
	}

	return adapter.NewThing(publisher, thingState, &adapter.ThingConfig{
		Connector:       NewConnector(t.signalRManager, t.client, info.ChargerID, thingCache, t.cfgService, t.sessionStorage),
		InclusionReport: t.inclusionReport(info, thingState, groups),
	}, services...), nil
}

func (t *thingFactory) inclusionReport(info *Info, thingState adapter.ThingState, groups []string) *fimptype.ThingInclusionReport {
	return &fimptype.ThingInclusionReport{
		Address:        thingState.Address(),
		ProductHash:    "Easee - Easee - " + info.Product,
		ProductName:    info.Product,
		DeviceId:       info.ChargerID,
		CommTechnology: "cloud",
		ManufacturerId: "Easee",
		PowerSource:    "ac",
		WakeUpInterval: "-1",
		Groups:         groups,
	}
}

func (t *thingFactory) chargepointSpecification(ad adapter.Adapter, thingState adapter.ThingState, groups []string, state *State) *fimptype.Service {
	options := []adapter.SpecificationOption{
		chargepoint.WithChargingModes(model.SupportedChargingModes()...),
		chargepoint.WithSupportedMaxCurrent(state.SupportedMaxCurrent),
	}

	if phases := state.Phases; phases > 0 {
		options = append(options, chargepoint.WithPhases(phases))
	}

	if gridType := state.GridType; gridType != "" {
		options = append(options, chargepoint.WithGridType(gridType))
	}

	if maxCurrent := state.SupportedMaxCurrent; maxCurrent > 0 {
		options = append(options, chargepoint.WithSupportedMaxCurrent(maxCurrent))
	}

	if phaseModes := model.SupportedPhaseModes(state.GridType, state.PhaseMode, state.Phases); len(phaseModes) > 0 {
		options = append(options, chargepoint.WithSupportedPhaseModes(phaseModes...))
	}

	return chargepoint.Specification(
		ad.Name(),
		ad.Address(),
		thingState.Address(),
		groups,
		t.supportedStates(),
		options...,
	)
}

func (t *thingFactory) supportedStates() []chargepoint.State {
	var supportedStates []chargepoint.State

	for _, s := range model.SupportedChargingStates() {
		if !slices.Contains(supportedStates, s.ToFimpState()) {
			supportedStates = append(supportedStates, s.ToFimpState())
		}
	}

	return supportedStates
}

func (t *thingFactory) meterElecSpecification(adapter adapter.Adapter, thingState adapter.ThingState, groups []string) *fimptype.Service {
	return numericmeter.Specification(
		numericmeter.MeterElec,
		adapter.Name(),
		adapter.Address(),
		thingState.Address(),
		groups,
		[]numericmeter.Unit{numericmeter.UnitW, numericmeter.UnitKWh},
		numericmeter.WithExtendedValues(
			numericmeter.ValueCurrentPhase1,
			numericmeter.ValueCurrentPhase2,
			numericmeter.ValueCurrentPhase3,
			numericmeter.ValueEnergyImport,
			numericmeter.ValuePowerImport,
		),
	)
}

func (t *thingFactory) newChargepointService(
	publisher adapter.ServicePublisher,
	ad adapter.Adapter,
	thingState adapter.ThingState,
	groups []string,
	controller Controller,
	state *State,
) adapter.Service {
	return chargepoint.NewService(publisher, &chargepoint.Config{
		Specification: t.chargepointSpecification(ad, thingState, groups, state),
		Controller:    controller,
	})
}

func (t *thingFactory) newMeterElecService(publisher adapter.ServicePublisher,
	ad adapter.Adapter,
	thingState adapter.ThingState,
	groups []string,
	controller Controller,
) adapter.Service {
	return numericmeter.NewService(publisher, &numericmeter.Config{
		Specification:     t.meterElecSpecification(ad, thingState, groups),
		Reporter:          controller,
		ReportingStrategy: cliffCache.ReportAtLeastEvery(time.Minute),
	})
}

func (t *thingFactory) newParametersService(publisher adapter.ServicePublisher,
	ad adapter.Adapter,
	thingState adapter.ThingState,
	groups []string,
	controller Controller,
) adapter.Service {
	return parameters.NewService(publisher, &parameters.Config{
		Specification: t.parametersSpecification(ad, thingState, groups),
		Controller:    controller,
	})
}

func (t *thingFactory) parametersSpecification(adapter adapter.Adapter, thingState adapter.ThingState, groups []string) *fimptype.Service {
	return parameters.Specification(adapter.Name(), adapter.Address(), thingState.Address(), groups)
}

// parameterSpecificationCableAlwaysLocked returns parameter specification for the associated configuration option.
func parameterSpecificationCableAlwaysLocked() *parameters.ParameterSpecification {
	return &parameters.ParameterSpecification{
		ID:          model.CableAlwaysLockedParameter,
		Name:        "Cable always locked",
		Description: "Maintains locked cable at all times.",
		ValueType:   parameters.ValueTypeBool,
		WidgetType:  parameters.WidgetTypeSelect,
		Options: parameters.SelectOptions{
			{
				Label: "Yes",
				Value: true,
			},
			{
				Label: "No",
				Value: false,
			},
		},
		DefaultValue: false,
		ReadOnly:     false,
	}
}
