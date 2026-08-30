package easee

import (
	"fmt"
	"slices"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	cliffCache "github.com/futurehomeno/cliffhanger/adapter/cache"
	"github.com/futurehomeno/cliffhanger/adapter/service/alarm"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	"github.com/futurehomeno/cliffhanger/types"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/cache"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/db"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
)

// Info is charger information persisted with the thing.
type Info struct {
	ChargerID string `json:"chargerID"`
	Product   string `json:"product"`
}

// State is the mutable charger information persisted with the thing.
type State struct {
	GridType            types.GridType `json:"gridType"`
	Phases              int            `json:"phases"`
	PhaseMode           int            `json:"phaseMode"`
	SupportedMaxCurrent int            `json:"supportedMaxCurrent"`
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

	thingCache := cache.NewCache(info.ChargerID)
	controller := NewController(t.signalRManager, t.client, info.ChargerID, thingCache, t.cfgService, t.sessionStorage)

	state := &State{}
	if err := thingState.State(state); err != nil {
		log.WithError(err).Warnf("factory: failed to retrieve state: %v", err)
	}

	usingStoredState := false

	// Any refresh failure falls back to the stored state rather than aborting: the adapter
	// creates things in a loop that stops at the first error, so one charger behind a 5xx
	// used to take every remaining charger down with it.
	if err := controller.UpdateState(info.ChargerID, state); err != nil {
		log.Warnf("factory: [%s] state refresh failed, creating thing with stored state: %v", info.ChargerID, err)

		usingStoredState = true
	}

	// A failed refresh means state was not read from the cloud - persisting it here would
	// overwrite the stored state with zeros if it had failed to load above.
	if !usingStoredState {
		if err := thingState.SetState(state); err != nil {
			log.WithError(err).Warnf("factory: failed to set state: %v", err)
		}
	}

	// using zero time, because we have no idea about the exact time those parameters were set
	thingCache.SetInstallationParameters(state.GridType, state.Phases, time.Time{})
	thingCache.SetPhaseMode(state.PhaseMode, time.Time{})

	groups := []string{"ch_0"}
	services := []adapter.Service{
		chargepoint.NewService(publisher, &chargepoint.Config{
			Specification: t.chargepointSpecification(ad, thingState, groups, state),
			Controller:    controller,
		}),
		numericmeter.NewService(publisher, &numericmeter.Config{
			Specification:     t.meterElecSpecification(ad, thingState, groups),
			Reporter:          controller,
			ReportingStrategy: cliffCache.ReportAtLeastEvery(time.Minute),
		}),
		parameters.NewService(publisher, &parameters.Config{
			Specification: parameters.Specification(ad.Name().Str(), ad.Address(), thingState.Address(), groups),
			Controller:    controller,
		}),
		alarm.NewService(publisher, &alarm.Config{
			Specification: alarm.Specification(
				ad.Name().Str(),
				ad.Address(),
				thingState.Address(),
				groups,
				model.SupportedAlarmEvents(),
			),
			Reporter: controller,
		}),
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
	// sup_max_current must be *present* to keep the max-current interfaces: cliffhanger derives
	// them from the property once, in NewService, so omitting it on a charger created without
	// state drops cmd.max_current.set for the lifetime of the process. A present 0 is worse
	// than absent, though - validateCurrent rejects everything above it, so every legal current
	// fails "must not exceed 0A". Fall back to the same ceiling setOfferedCurrent applies when
	// the cached maximum is unknown; the real site limit clamps it once site data arrives.
	supportedMaxCurrent := state.SupportedMaxCurrent
	if supportedMaxCurrent <= 0 {
		supportedMaxCurrent = maxCurrentValue
	}

	options := []adapter.SpecificationOption{
		chargepoint.WithChargingModes(model.SupportedChargingModes()...),
		chargepoint.WithSupportedMaxCurrent(supportedMaxCurrent),
	}

	if phases := state.Phases; phases > 0 {
		options = append(options, chargepoint.WithPhases(phases))
	}

	if gridType := state.GridType; gridType != "" {
		options = append(options, chargepoint.WithGridType(gridType))
	}

	if phaseModes := model.SettablePhaseModes(state.GridType, state.Phases); len(phaseModes) > 0 {
		options = append(options, chargepoint.WithSupportedPhaseModes(phaseModes...))
	}

	return chargepoint.Specification(
		ad.Name().Str(),
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
		if state := s.ToFimpState(); !slices.Contains(supportedStates, state) {
			supportedStates = append(supportedStates, state)
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
