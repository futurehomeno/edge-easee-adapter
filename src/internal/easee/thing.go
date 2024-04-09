package easee

import (
	"fmt"
	"slices"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	cliffCache "github.com/futurehomeno/cliffhanger/adapter/cache"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/cache"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
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
}

// NewThingFactory returns a new instance of adapter.ThingFactory.
func NewThingFactory(client api.Client, cfgService *config.Service, signalRManager signalr.Manager) adapter.ThingFactory {
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

	thingCache := cache.NewCache()
	controller := NewController(t.signalRManager, t.client, info.ChargerID, thingCache, t.cfgService)

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

	groups := []string{"ch_0"}

	return thing.NewCarCharger(publisher, thingState, &thing.CarChargerConfig{
		ThingConfig: &adapter.ThingConfig{
			Connector:       NewConnector(t.signalRManager, t.client, info.ChargerID, thingCache),
			InclusionReport: t.inclusionReport(info, thingState, groups),
		},
		ChargepointConfig: &chargepoint.Config{
			Specification: t.chargepointSpecification(ad, thingState, groups, state),
			Controller:    controller,
		},
		MeterElecConfig: &numericmeter.Config{
			Specification:     t.meterElecSpecification(ad, thingState, groups),
			Reporter:          controller,
			ReportingStrategy: cliffCache.ReportAtLeastEvery(time.Minute),
		},
	}), nil
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

func (t *thingFactory) chargepointSpecification(adapter adapter.Adapter, thingState adapter.ThingState, groups []string, state *State) *fimptype.Service {
	return chargepoint.Specification(
		adapter.Name(),
		adapter.Address(),
		thingState.Address(),
		groups,
		t.supportedStates(),
		chargepoint.WithChargingModes(SupportedChargingModes()...),
		chargepoint.WithPhases(state.Phases),
		chargepoint.WithSupportedPhaseModes(SupportedPhaseModes(state)...),
		chargepoint.WithSupportedMaxCurrent(state.SupportedMaxCurrent),
		chargepoint.WithGridType(state.GridType),
	)
}

func (t *thingFactory) supportedStates() []chargepoint.State {
	var supportedStates []chargepoint.State

	for _, s := range signalr.SupportedChargingStates() {
		if !slices.Contains(supportedStates, s.ToFimpState()) {
			supportedStates = append(supportedStates, s.ToFimpState())
		}
	}

	return supportedStates
}

func SupportedPhaseModes(state *State) []chargepoint.PhaseMode {
	if state.Phases == 1 {
		if state.GridType == chargepoint.GridTypeTN {
			return []chargepoint.PhaseMode{chargepoint.PhaseModeNL1}
		}

		if state.GridType == chargepoint.GridTypeIT || state.GridType == chargepoint.GridTypeTT {
			return []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2}
		}
	}

	if state.Phases == 3 {
		if state.GridType == chargepoint.GridTypeTN {
			switch state.PhaseMode {
			case 1:
				return []chargepoint.PhaseMode{chargepoint.PhaseModeNL1, chargepoint.PhaseModeNL2, chargepoint.PhaseModeNL3}
			case 2:
				return []chargepoint.PhaseMode{chargepoint.PhaseModeNL1, chargepoint.PhaseModeNL2, chargepoint.PhaseModeNL3, chargepoint.PhaseModeNL1L2L3}
			case 3:
				return []chargepoint.PhaseMode{chargepoint.PhaseModeNL1L2L3}
			}
		}

		if state.GridType == chargepoint.GridTypeIT || state.GridType == chargepoint.GridTypeTT {
			switch state.PhaseMode {
			case 1:
				return []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2, chargepoint.PhaseModeL2L3, chargepoint.PhaseModeL3L1}
			case 2:
				return []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2, chargepoint.PhaseModeL2L3, chargepoint.PhaseModeL3L1, chargepoint.PhaseModeL1L2L3}
			case 3:
				return []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2L3}
			}
		}
	}

	return []chargepoint.PhaseMode{}
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
