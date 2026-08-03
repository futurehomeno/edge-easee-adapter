package easee_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/cliffhanger/adapter"
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
