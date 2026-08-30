package easee_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/alarm"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/fakes"
	mockapi "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/api"
)

// fakeAdapter satisfies adapter.Adapter with just enough behavior for thingFactory.Create to
// build service specifications; every other method is unused by Create and left to panic.
type fakeAdapter struct {
	adapter.Adapter
}

func (fakeAdapter) Name() fimptype.ResourceNameT { return "easee" }
func (fakeAdapter) Address() string              { return "1" }

// fakePublisher satisfies adapter.Publisher - Create only wires it into services for later
// use, it never publishes anything during construction.
type fakePublisher struct {
	adapter.Publisher
}

// fakeThingState is a minimal adapter.ThingState letting the test control what State()
// returns and observe whether SetState was called and with what.
type fakeThingState struct {
	adapter.ThingState

	info          easee.Info
	stateErr      error
	setStateCalls int
	setStateArg   easee.State
}

func (f *fakeThingState) Address() string { return "1" }

func (f *fakeThingState) Info(m any) error {
	*m.(*easee.Info) = f.info //nolint:forcetypeassert

	return nil
}

func (f *fakeThingState) State(any) error {
	return f.stateErr
}

func (f *fakeThingState) SetState(m any) error {
	f.setStateCalls++
	f.setStateArg = *m.(*easee.State) //nolint:forcetypeassert

	return nil
}

// TestThingFactory_Create_NotLoggedIn_DoesNotOverwriteStoredState covers the finding from
// PR #88 review: when the charger is not logged in AND the previously stored state failed to
// load, Create must not persist the zero-valued state over whatever was on disk.
func TestThingFactory_Create_NotLoggedIn_DoesNotOverwriteStoredState(t *testing.T) {
	clientMock := mockapi.NewClient(t)
	clientMock.On("ChargerConfig", "test-charger").Return((*model.ChargerConfig)(nil), api.ErrNotLoggedIn)
	clientMock.On("ChargerSiteInfo", "test-charger").Return((*model.ChargerSiteInfo)(nil), api.ErrNotLoggedIn)

	cfg := &config.Config{}
	storage := fakes.NewConfigStorage(t, cfg, config.Factory)
	cfgService := config.NewService(storage)

	factory := easee.NewThingFactory(clientMock, cfgService, nil, nil)

	ts := &fakeThingState{
		info:     easee.Info{ChargerID: "test-charger", Product: "Home"},
		stateErr: errors.New("stored state is corrupt"),
	}

	thing, err := factory.Create(fakeAdapter{}, fakePublisher{}, ts)
	require.NoError(t, err)
	require.NotNil(t, thing)

	assert.Zero(t, ts.setStateCalls, "SetState must not run when the fresh state failed to load and the charger is not logged in")
}

// sup_max_current must be advertised even at 0. cliffhanger derives the max-current
// interfaces from the property once, in NewService, and the runtime repair path re-adds the
// same specification - so a charger created without state used to lose cmd.max_current.set
// for the lifetime of the process.
func TestThingFactory_Create_KeepsMaxCurrentCommandsWithoutState(t *testing.T) {
	clientMock := mockapi.NewClient(t)
	clientMock.On("ChargerConfig", "test-charger").Return(&model.ChargerConfig{}, nil)
	clientMock.On("ChargerSiteInfo", "test-charger").Return(&model.ChargerSiteInfo{}, nil)

	storage := fakes.NewConfigStorage(t, &config.Config{}, config.Factory)
	factory := easee.NewThingFactory(clientMock, config.NewService(storage), nil, nil)

	thing, err := factory.Create(fakeAdapter{}, fakePublisher{}, &fakeThingState{
		info: easee.Info{ChargerID: "test-charger", Product: "Home"},
	})
	require.NoError(t, err)

	services := thing.Services(chargepoint.Chargepoint)
	require.Len(t, services, 1)

	assert.Contains(t, messageTypes(services[0].Specification().Interfaces), chargepoint.CmdMaxCurrentSet)
}

// The adapter stops creating things at the first factory error, so a single charger behind a
// transient 5xx used to take every remaining charger down with it.
func TestThingFactory_Create_FallsBackToStoredStateOnFetchError(t *testing.T) {
	clientMock := mockapi.NewClient(t)
	clientMock.On("ChargerConfig", "test-charger").Return((*model.ChargerConfig)(nil), errors.New("internal server error"))
	clientMock.On("ChargerSiteInfo", "test-charger").Return((*model.ChargerSiteInfo)(nil), errors.New("internal server error"))

	storage := fakes.NewConfigStorage(t, &config.Config{}, config.Factory)
	factory := easee.NewThingFactory(clientMock, config.NewService(storage), nil, nil)

	ts := &fakeThingState{info: easee.Info{ChargerID: "test-charger", Product: "Home"}}

	thing, err := factory.Create(fakeAdapter{}, fakePublisher{}, ts)
	require.NoError(t, err)
	require.NotNil(t, thing)

	assert.Zero(t, ts.setStateCalls, "a state that was never refreshed must not be persisted")
}

func messageTypes(interfaces []fimptype.Interface) []string {
	types := make([]string, 0, len(interfaces))

	for _, i := range interfaces {
		types = append(types, i.MsgType)
	}

	return types
}

func TestThingFactory_Create_RegistersAlarmSystemService(t *testing.T) {
	clientMock := mockapi.NewClient(t)
	clientMock.On("ChargerConfig", "test-charger").Return(&model.ChargerConfig{}, nil)
	clientMock.On("ChargerSiteInfo", "test-charger").Return(&model.ChargerSiteInfo{}, nil)

	storage := fakes.NewConfigStorage(t, &config.Config{}, config.Factory)
	factory := easee.NewThingFactory(clientMock, config.NewService(storage), nil, nil)

	thing, err := factory.Create(fakeAdapter{}, fakePublisher{}, &fakeThingState{
		info: easee.Info{ChargerID: "test-charger", Product: "Home"},
	})
	require.NoError(t, err)

	services := thing.Services(alarm.AlarmSystem)
	require.Len(t, services, 1)

	spec := services[0].Specification()
	assert.Equal(t, "/rt:dev/rn:easee/ad:1/sv:alarm_system/ad:1", spec.Address)
	assert.Equal(t, model.SupportedAlarmEvents(), spec.PropertyStrings(alarm.PropertySupportedEvents))
	assert.ElementsMatch(t, []fimptype.Interface{
		{Type: fimptype.TypeIn, MsgType: alarm.CmdAlarmGetReport, ValueType: fimptype.VTypeNull, Version: "1"},
		{Type: fimptype.TypeOut, MsgType: alarm.EvtAlarmReport, ValueType: fimptype.VTypeStrMap, Version: "1"},
		{Type: fimptype.TypeOut, MsgType: router.EvtErrorReport, ValueType: fimptype.VTypeString, Version: "1"},
	}, spec.Interfaces)
}

func TestThingFactory_Create_RegistersPhaseModeSet(t *testing.T) {
	clientMock := mockapi.NewClient(t)
	clientMock.On("ChargerConfig", "test-charger").Return(&model.ChargerConfig{
		DetectedPowerGridType: model.GridTypeTN3Phase,
		PhaseMode:             3,
	}, nil)
	clientMock.On("ChargerSiteInfo", "test-charger").Return(&model.ChargerSiteInfo{RatedCurrent: 32}, nil)

	storage := fakes.NewConfigStorage(t, &config.Config{}, config.Factory)
	factory := easee.NewThingFactory(clientMock, config.NewService(storage), nil, nil)

	thing, err := factory.Create(fakeAdapter{}, fakePublisher{}, &fakeThingState{
		info: easee.Info{ChargerID: "test-charger", Product: "Home"},
	})
	require.NoError(t, err)

	services := thing.Services(chargepoint.Chargepoint)
	require.Len(t, services, 1)

	spec := services[0].Specification()

	// The charger sits in Easee's locked-3-phase mode, yet every mode stays advertised -
	// otherwise switching back to a single phase would be impossible.
	assert.Equal(t,
		[]string{"NL1", "NL2", "NL3", "NL1L2L3"},
		spec.PropertyStrings(chargepoint.PropertySupportedPhaseModes),
	)
	assert.Contains(t, spec.Interfaces, fimptype.Interface{
		Type: fimptype.TypeIn, MsgType: chargepoint.CmdPhaseModeSet, ValueType: fimptype.VTypeString, Version: "1",
	})
}

// A 1-phase grid has a single settable mode, but sup_phase_modes must stay advertised: the
// property also gates evt.phase_mode.report, so dropping it would blind the hub to the leg
// in use. Suppressing the pointless switch is the setter's job, not the specification's.
func TestThingFactory_Create_SinglePhaseGridKeepsPhaseModeReport(t *testing.T) {
	clientMock := mockapi.NewClient(t)
	clientMock.On("ChargerConfig", "test-charger").Return(&model.ChargerConfig{
		DetectedPowerGridType: model.GridTypeTN1Phase,
		PhaseMode:             2,
	}, nil)
	clientMock.On("ChargerSiteInfo", "test-charger").Return(&model.ChargerSiteInfo{RatedCurrent: 32}, nil)

	storage := fakes.NewConfigStorage(t, &config.Config{}, config.Factory)
	factory := easee.NewThingFactory(clientMock, config.NewService(storage), nil, nil)

	thing, err := factory.Create(fakeAdapter{}, fakePublisher{}, &fakeThingState{
		info: easee.Info{ChargerID: "test-charger", Product: "Home"},
	})
	require.NoError(t, err)

	services := thing.Services(chargepoint.Chargepoint)
	require.Len(t, services, 1)

	spec := services[0].Specification()

	assert.Equal(t, []string{"NL1"}, spec.PropertyStrings(chargepoint.PropertySupportedPhaseModes))
	assert.Contains(t, spec.Interfaces, fimptype.Interface{
		Type: fimptype.TypeOut, MsgType: chargepoint.EvtPhaseModeReport, ValueType: fimptype.VTypeString, Version: "1",
	})
}
