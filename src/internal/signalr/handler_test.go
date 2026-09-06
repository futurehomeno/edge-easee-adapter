package signalr_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	mockedchargepoint "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/types"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
	mockedcache "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/cache"
)

// serviceLessThing exposes no services, so a handler stops with an error the moment it looks
// one up - right after the cache writes, which is as far as the test below needs to get.
type serviceLessThing struct {
	adapter.Thing
}

func (serviceLessThing) Services(fimptype.ServiceNameT) []adapter.Service { return nil }

// The session-finished clear is a controller-convention write: it carries the current time
// rather than the observation's, so a state observation stamped before the last set command
// still zeroes the cached request instead of being turned away by the timestamp guard.
func TestObservationsHandler_SessionFinishedClearsRequestedCurrentWithNow(t *testing.T) {
	t.Parallel()

	stale := time.Now().Add(-time.Hour)

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("ChargerState").Return(chargepoint.StateCharging, stale)
	cacheMock.On("SetChargerState", chargepoint.StateDisconnected, stale).Return(true)
	cacheMock.On("SetRequestedOfferedCurrent", 0, mock.MatchedBy(func(ts time.Time) bool {
		return ts.After(stale)
	})).Return(true)

	handler, err := signalr.NewObservationsHandler(serviceLessThing{}, cacheMock, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	require.Error(t, handler.HandleObservation(model.Observation{
		ID:        model.ChargerOPState,
		ChargerID: testChargerID,
		DataType:  model.ObservationDataTypeInteger,
		Timestamp: stale,
		Value:     strconv.Itoa(int(model.ChargerStateDisconnected)),
	}))
}

// phaseThing exposes a single chargepoint service and counts inclusion reports.
type phaseThing struct {
	adapter.Thing

	srv       *mockedchargepoint.Service
	inclusion int
}

func (t *phaseThing) Services(fimptype.ServiceNameT) []adapter.Service {
	return []adapter.Service{t.srv}
}
func (t *phaseThing) Update(...adapter.ThingUpdate) error { return nil }

func (t *phaseThing) SendInclusionReport(bool) (bool, error) {
	t.inclusion++

	return true, nil
}

type fakePhaseStore struct {
	mode types.PhaseMode
}

func (s *fakePhaseStore) OutputPhase() types.PhaseMode { return s.mode }

func (s *fakePhaseStore) SetOutputPhase(mode types.PhaseMode) error {
	s.mode = mode

	return nil
}

// An Easee always uses the leg it is wired to, so the first OutputPhase observation must narrow
// sup_phase_modes to that leg - a hub asking for any other one retries forever. A repeat
// observation carries no news, so it must not republish the inclusion report.
func TestObservationsHandler_OutputPhaseNarrowsAdvertisedPhaseModes(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("SetOutputPhaseType", types.PhaseModeNL3, now).Return(true).Once()
	cacheMock.On("SetOutputPhaseType", types.PhaseModeNL3, now).Return(false)
	cacheMock.On("GridType").Return(types.GridTypeTN, now)
	cacheMock.On("Phases").Return(3, now)

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("Specification").Return(&fimptype.Service{Props: map[string]interface{}{
		chargepoint.PropertySupportedPhaseModes: []types.PhaseMode{
			types.PhaseModeNL1, types.PhaseModeNL2, types.PhaseModeNL3, types.PhaseModeNL1L2L3,
		},
	}})
	srv.On("SendPhaseModeReport", false).Return(true, nil).Maybe()

	thing := &phaseThing{srv: srv}
	store := &fakePhaseStore{}

	handler, err := signalr.NewObservationsHandler(thing, cacheMock, nil, nil, testChargerID, store)
	require.NoError(t, err)

	observation := model.Observation{
		ID:        model.OutputPhase,
		ChargerID: testChargerID,
		DataType:  model.ObservationDataTypeInteger,
		Timestamp: now,
		Value:     strconv.Itoa(int(model.P1T2T5TN)),
	}

	require.NoError(t, handler.HandleObservation(observation))

	assert.Equal(t, types.PhaseModeNL3, store.mode)
	assert.Equal(t, []types.PhaseMode{types.PhaseModeNL3, types.PhaseModeNL1L2L3},
		srv.Specification().Props[chargepoint.PropertySupportedPhaseModes])
	assert.Equal(t, 1, thing.inclusion)

	require.NoError(t, handler.HandleObservation(observation))

	assert.Equal(t, 1, thing.inclusion, "an unchanged leg must not republish the inclusion report")
}
