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
	fail      int
}

func (t *phaseThing) Services(fimptype.ServiceNameT) []adapter.Service {
	return []adapter.Service{t.srv}
}
func (t *phaseThing) Update(...adapter.ThingUpdate) error { return nil }

func (t *phaseThing) SendInclusionReport(bool) (bool, error) {
	t.inclusion++

	if t.fail > 0 {
		t.fail--

		return false, assert.AnError
	}

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

// An Easee always uses the phase it is wired to, so the first OutputPhase observation must narrow
// sup_phase_modes to that phase - a hub asking for any other one retries forever. A repeat
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

	assert.Equal(t, 1, thing.inclusion, "an unchanged phase must not republish the inclusion report")
}

// A republish that fails must not be recorded as done: the next observation has to retry it.
func TestObservationsHandler_OutputPhaseRepublishRetriedAfterFailure(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("SetOutputPhaseType", types.PhaseModeNL3, now).Return(true)
	cacheMock.On("GridType").Return(types.GridTypeTN, now)
	cacheMock.On("Phases").Return(3, now)

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("Specification").Return(&fimptype.Service{Props: map[string]interface{}{}})
	srv.On("SendPhaseModeReport", false).Return(true, nil).Maybe()

	thing := &phaseThing{srv: srv, fail: 1}
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

	require.Error(t, handler.HandleObservation(observation))
	assert.Equal(t, types.PhaseModeUnknown, store.mode, "a failed republish must not be persisted")

	require.NoError(t, handler.HandleObservation(observation))
	assert.Equal(t, types.PhaseModeNL3, store.mode)
	assert.Equal(t, 2, thing.inclusion)
}

// The offline state report is the one report that must survive the offline gate. The handler
// used to flip isStateOnline before sending, so the controller's checkConnection rejected the
// very report announcing the charger had gone offline, and the app kept the last online state
// until it came back.
func TestObservationsHandler_OfflineStateIsReportedBeforeTheFlagFlips(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("ChargerState").Return(chargepoint.StateCharging, now)
	cacheMock.On("SetChargerState", chargepoint.StateUnknown, now).Return(true)
	// Offline counts as a finished session, so the requested current is cleared on the way.
	cacheMock.On("SetRequestedOfferedCurrent", 0, mock.Anything).Return(true).Maybe()

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()

	handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, cacheMock, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	// isCloudOnline starts true, so the handler is online until this observation lands.
	online := make(chan bool, 1)
	srv.On("SendStateReport", false).Run(func(mock.Arguments) {
		online <- handler.IsOnline()
	}).Return(true, nil)

	require.NoError(t, handler.HandleObservation(model.Observation{
		ID:        model.ChargerOPState,
		ChargerID: testChargerID,
		DataType:  model.ObservationDataTypeInteger,
		Timestamp: now,
		Value:     strconv.Itoa(int(model.ChargerStateOffline)),
	}))

	assert.True(t, <-online, "the report must be sent while the charger still counts as online, or the offline gate refuses it")
	assert.False(t, handler.IsOnline(), "the flag flips once the report is away")
}

// A faulted grid type maps to ("", 0), which is the absence of a topology rather than a new
// one. Writing it deleted grid_type, phases and sup_phase_modes from the props, and the
// equivalence check short-circuits every repeat, so the charger could not restore them while
// the fault persisted. The alarm is the report for this case.
func TestObservationsHandler_FaultGridTypeKeepsTheChargepointProps(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(types.GridTypeTN, now)
	cacheMock.On("Phases").Return(3, now)

	props := map[string]interface{}{
		chargepoint.PropertyGridType: types.GridTypeTN,
		chargepoint.PropertyPhases:   3,
	}

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("Specification").Return(&fimptype.Service{Props: props}).Maybe()

	thing := &phaseThing{srv: srv}

	handler, err := signalr.NewObservationsHandler(thing, cacheMock, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	require.NoError(t, handler.HandleObservation(model.Observation{
		ID:        model.DetectedPowerGridType,
		ChargerID: testChargerID,
		DataType:  model.ObservationDataTypeInteger,
		Timestamp: now,
		Value:     strconv.Itoa(int(model.GridTypeWarningTN3PhaseGNDFault)),
	}))

	assert.Equal(t, types.GridTypeTN, srv.Specification().Props[chargepoint.PropertyGridType],
		"a fault must not delete the known grid type")
	assert.Equal(t, 3, srv.Specification().Props[chargepoint.PropertyPhases],
		"a fault must not delete the known phase count")
	assert.Zero(t, thing.inclusion, "nothing to republish when the topology did not change")
}
