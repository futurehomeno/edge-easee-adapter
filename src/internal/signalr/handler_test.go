package signalr_test

import (
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/alarm"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	mockedalarm "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/alarm"
	mockedchargepoint "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/chargepoint"
	mockednumericmeter "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/types"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/cache"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/fakes"
	mockedcache "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/cache"
)

// serviceLessThing exposes no services, so a handler stops with an error the moment it looks
// one up - right after the cache writes, which is as far as the test below needs to get.
type serviceLessThing struct {
	adapter.Thing
}

func (serviceLessThing) Services(fimptype.ServiceNameT) []adapter.Service { return nil }

// flushReports drains the handler's report queue. Close publishes everything already queued and
// waits for the sender, so it is the flush point a test needs once a report is asynchronous.
func flushReports(t *testing.T, handler signalr.Handler) {
	t.Helper()

	handler.Close()
}

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

// A re-sync replays the snapshot with the timestamps already cached; that is no new end of a
// session, so the current requested since is kept.
func TestObservationsHandler_ReplayedSessionFinishedKeepsRequestedCurrent(t *testing.T) {
	t.Parallel()

	at := time.Now().Add(-time.Hour)

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("ChargerState").Return(chargepoint.StateFinished, at)
	cacheMock.On("SetChargerState", chargepoint.StateFinished, at).Return(true)

	handler, err := signalr.NewObservationsHandler(serviceLessThing{}, cacheMock, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	require.Error(t, handler.HandleObservation(model.Observation{
		ID:        model.ChargerOPState,
		ChargerID: testChargerID,
		DataType:  model.ObservationDataTypeInteger,
		Timestamp: at,
		Value:     strconv.Itoa(int(model.ChargerStateCompleted)),
	}))
}

// phaseThing exposes a single chargepoint service and counts inclusion reports.
type phaseThing struct {
	adapter.Thing

	srv *mockedchargepoint.Service

	// The inclusion report is published on the sender goroutine, so the counter and the failure
	// budget are both touched off the test goroutine.
	mu           sync.Mutex
	inclusion    int
	fail         int
	connectivity []bool
	online       func() bool
	block        chan struct{}
}

func (t *phaseThing) Services(fimptype.ServiceNameT) []adapter.Service {
	return []adapter.Service{t.srv}
}
func (t *phaseThing) Update(...adapter.ThingUpdate) error { return nil }

func (t *phaseThing) SendInclusionReport(bool) (bool, error) {
	if t.block != nil {
		<-t.block
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.inclusion++

	if t.fail > 0 {
		t.fail--

		return false, assert.AnError
	}

	return true, nil
}

func (t *phaseThing) SendConnectivityReport(bool) (bool, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.connectivity = append(t.connectivity, t.online != nil && t.online())

	return true, nil
}

func (t *phaseThing) connectivityReports() []bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.connectivity
}

func (t *phaseThing) inclusionCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.inclusion
}

// waitFor polls until cond holds, for reports that are published on the sender goroutine.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}

		time.Sleep(time.Millisecond)
	}

	t.Fatal("condition not met before the deadline")
}

type fakePhaseStore struct {
	mode     types.PhaseMode
	gridType model.GridType
	block    chan struct{}
	err      error
}

func (s *fakePhaseStore) OutputPhase() types.PhaseMode     { return s.mode }
func (s *fakePhaseStore) DetectedGridType() model.GridType { return s.gridType }

func (s *fakePhaseStore) SetOutputPhase(mode types.PhaseMode) error {
	if s.block != nil {
		<-s.block
	}

	if s.err != nil {
		return s.err
	}

	s.mode = mode

	return nil
}

// The leg is written to flash, so the write goes with the queued report: on the drain it would
// park the only reader of the observation channel, and its failure would cost the hub the report.
func TestObservationsHandler_OutputPhaseStoreWriteStaysOffTheDrain(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("SetOutputPhaseType", types.PhaseModeL3L1, now).Return(true).Once()

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("Specification").Return(&fimptype.Service{Props: map[string]interface{}{
		chargepoint.PropertyGridType: types.GridTypeIT,
	}})
	srv.On("IsPhaseModeAware").Return(true).Maybe()
	srv.On("SendPhaseModeReport", false).Return(true, nil).Once()

	store := &fakePhaseStore{mode: types.PhaseModeL1L2, block: make(chan struct{}), err: assert.AnError}

	handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, cacheMock, nil, nil, testChargerID, store)
	require.NoError(t, err)

	handled := make(chan error, 1)

	go func() {
		handled <- handler.HandleObservation(model.Observation{
			ID:        model.OutputPhase,
			ChargerID: testChargerID,
			DataType:  model.ObservationDataTypeInteger,
			Timestamp: now,
			Value:     strconv.Itoa(int(model.P1T2T4IT)),
		})
	}()

	select {
	case err := <-handled:
		require.NoError(t, err)
	case <-time.After(time.Second):
		close(store.block)
		t.Fatal("the store write parked the observation drain")
	}

	close(store.block)
	flushReports(t, handler)
}

// One sender publishes every report of the charger in order, so a write that blocks has to come
// after the report it would otherwise hold back.
func TestObservationsHandler_OutputPhaseReportDoesNotWaitForTheStoreWrite(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("SetOutputPhaseType", types.PhaseModeL3L1, now).Return(true).Once()

	reported := make(chan struct{})

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("Specification").Return(&fimptype.Service{Props: map[string]interface{}{
		chargepoint.PropertyGridType: types.GridTypeIT,
	}})
	srv.On("IsPhaseModeAware").Return(true).Maybe()
	srv.On("SendPhaseModeReport", false).Run(func(mock.Arguments) { close(reported) }).Return(true, nil).Once()

	store := &fakePhaseStore{mode: types.PhaseModeL1L2, block: make(chan struct{})}

	handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, cacheMock, nil, nil, testChargerID, store)
	require.NoError(t, err)

	require.NoError(t, handler.HandleObservation(model.Observation{
		ID:        model.OutputPhase,
		ChargerID: testChargerID,
		DataType:  model.ObservationDataTypeInteger,
		Timestamp: now,
		Value:     strconv.Itoa(int(model.P1T2T4IT)),
	}))

	select {
	case <-reported:
	case <-time.After(time.Second):
		t.Error("the report waited for the store write")
	}

	close(store.block)
	flushReports(t, handler)

	assert.Equal(t, types.PhaseModeL3L1, store.mode)
}

// An Easee picks its leg per session (hub 8ee88ca8: T2T3 and T2T4 in turn on an IT grid), so a new
// leg must not narrow sup_phase_modes: cliffhanger would reject the leg a hub requested a moment ago.
func TestObservationsHandler_OutputPhaseKeepsEveryLegAdvertised(t *testing.T) {
	t.Parallel()

	now := time.Now()
	all := []types.PhaseMode{types.PhaseModeL1L2, types.PhaseModeL2L3, types.PhaseModeL3L1, types.PhaseModeL1L2L3}

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("SetOutputPhaseType", types.PhaseModeL3L1, now).Return(true).Once()

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("Specification").Return(&fimptype.Service{Props: map[string]interface{}{
		chargepoint.PropertyGridType:            types.GridTypeIT,
		chargepoint.PropertyPhases:              3,
		chargepoint.PropertySupportedPhaseModes: all,
	}})
	srv.On("IsPhaseModeAware").Return(true).Maybe()
	srv.On("SendPhaseModeReport", false).Return(true, nil).Maybe()

	thing := &phaseThing{srv: srv}
	store := &fakePhaseStore{mode: types.PhaseModeL1L2}

	handler, err := signalr.NewObservationsHandler(thing, cacheMock, nil, nil, testChargerID, store)
	require.NoError(t, err)

	require.NoError(t, handler.HandleObservation(model.Observation{
		ID:        model.OutputPhase,
		ChargerID: testChargerID,
		DataType:  model.ObservationDataTypeInteger,
		Timestamp: now,
		Value:     strconv.Itoa(int(model.P1T2T4IT)),
	}))

	flushReports(t, handler)

	assert.Equal(t, all, srv.Specification().Props[chargepoint.PropertySupportedPhaseModes])
	assert.Equal(t, types.PhaseModeL3L1, store.mode)
	assert.Zero(t, thing.inclusionCount(), "a new leg must not republish the inclusion report")
}

// An Easee on an IT grid has been seen reporting the TN variant of its output phase; the grid it
// detected decides which legs those terminals are. The props carry it: a topology republish still
// queued has replaced them but not yet written the cache.
func TestObservationsHandler_OutputPhaseFollowsDetectedGridType(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("SetOutputPhaseType", types.PhaseModeL1L2, now).Return(true).Once()
	cacheMock.On("GridType").Return(types.GridTypeTN, now).Maybe()

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("Specification").Return(&fimptype.Service{Props: map[string]interface{}{
		chargepoint.PropertyGridType: types.GridTypeIT,
		chargepoint.PropertyPhases:   1,
	}})
	srv.On("IsPhaseModeAware").Return(true).Maybe()
	srv.On("SendPhaseModeReport", false).Return(true, nil).Once()

	store := &fakePhaseStore{}

	handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, cacheMock, nil, nil, testChargerID, store)
	require.NoError(t, err)

	require.NoError(t, handler.HandleObservation(model.Observation{
		ID:        model.OutputPhase,
		ChargerID: testChargerID,
		DataType:  model.ObservationDataTypeInteger,
		Timestamp: now,
		Value:     strconv.Itoa(int(model.P1T2T3TN)),
	}))

	flushReports(t, handler)

	assert.Equal(t, types.PhaseModeL1L2, store.mode)
}

// The controller caches a sent target with the timestamp of the observation it replaced, so the
// echo of that target must record its own: the library dispatches each hub call on its own
// goroutine, and an older observation landing after the echo would otherwise win.
func TestObservationsHandler_PhaseModeRepeatKeepsItsTimestamp(t *testing.T) {
	t.Parallel()

	t0 := time.Now()
	t1, t2 := t0.Add(time.Second), t0.Add(2*time.Second)

	phases := cache.NewCache(testChargerID)
	phases.SetPhaseMode(1, t0)

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("SendPhaseModeReport", false).Return(true, nil).Maybe()

	handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, phases, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	for _, obs := range []struct {
		mode int
		at   time.Time
	}{{1, t2}, {2, t1}} {
		require.NoError(t, handler.HandleObservation(model.Observation{
			ID: model.PhaseMode, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
			Timestamp: obs.at, Value: strconv.Itoa(obs.mode),
		}))
	}

	flushReports(t, handler)

	mode, at := phases.PhaseMode()
	assert.Equal(t, 1, mode)
	assert.Equal(t, t2, at)
	srv.AssertNotCalled(t, "SendPhaseModeReport", false)
}

type chargepointThing struct {
	adapter.Thing

	srv chargepoint.Service
}

func (t chargepointThing) Services(fimptype.ServiceNameT) []adapter.Service {
	return []adapter.Service{t.srv}
}

// Grid type 30, or one not yet detected, advertises no sup_phase_modes, and cliffhanger refuses
// a phase-mode report on such a chargepoint, so sending one anyway warns on every change.
func TestObservationsHandler_NoPhaseModesSendsNoPhaseModeReport(t *testing.T) {
	hook := logtest.NewLocal(log.StandardLogger())
	t.Cleanup(hook.Reset)

	now := time.Now()

	srv := chargepoint.NewService(nil, &chargepoint.Config{
		Specification: &fimptype.Service{Props: map[string]interface{}{
			chargepoint.PropertyGridType: types.GridTypeTN,
			chargepoint.PropertyPhases:   2,
		}},
		Controller: mockedchargepoint.NewController(t),
	})
	store := &fakePhaseStore{}

	handler, err := signalr.NewObservationsHandler(chargepointThing{srv: srv}, cache.NewCache(testChargerID), nil, nil, testChargerID, store)
	require.NoError(t, err)

	for _, obs := range []struct {
		id  model.ObservationID
		val int
	}{{model.PhaseMode, 1}, {model.OutputPhase, int(model.P1T2T3TN)}} {
		require.NoError(t, handler.HandleObservation(model.Observation{
			ID: obs.id, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
			Timestamp: now, Value: strconv.Itoa(obs.val),
		}))
	}

	flushReports(t, handler)

	for _, entry := range hook.AllEntries() {
		assert.NotContains(t, entry.Message, "aware phase modes is not supported")
	}

	assert.Equal(t, types.PhaseModeNL1, store.mode)
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

	// The report is published off the drain, so the flag flips once the sender is done with it.
	handler.Close()

	assert.False(t, handler.IsOnline(), "the flag flips once the report is away")
}

// The mirror of the case above: coming back online, the flag has to flip BEFORE the send, or
// the recovery report is refused by the very offline state it exists to clear. The flag must be
// on the online side of the send in both directions, not simply always after it.
func TestObservationsHandler_RecoveryStateIsReportedWhileAlreadyOnline(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("ChargerState").Return(chargepoint.StateCharging, now)
	cacheMock.On("SetChargerState", chargepoint.StateUnknown, now).Return(true)
	cacheMock.On("SetChargerState", chargepoint.StateSuspendedByEV, now).Return(true)

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()

	handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, cacheMock, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	// Drive it offline first, so the recovery below is a real transition. The sender publishes
	// off the drain, so wait for the flag rather than reading it straight after the handler.
	offline := make(chan struct{})
	srv.On("SendStateReport", false).Run(func(mock.Arguments) { close(offline) }).Return(true, nil).Once()
	cacheMock.On("SetRequestedOfferedCurrent", 0, mock.Anything).Return(true).Maybe()
	require.NoError(t, handler.HandleObservation(model.Observation{
		ID: model.ChargerOPState, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
		Timestamp: now, Value: strconv.Itoa(int(model.ChargerStateOffline)),
	}))
	<-offline
	require.Eventually(t, func() bool { return !handler.IsOnline() }, time.Second, time.Millisecond)

	online := make(chan bool, 1)
	srv.On("SendStateReport", false).Run(func(mock.Arguments) {
		online <- handler.IsOnline()
	}).Return(true, nil).Once()

	require.NoError(t, handler.HandleObservation(model.Observation{
		ID: model.ChargerOPState, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
		Timestamp: now, Value: strconv.Itoa(int(model.ChargerStateReadyToCharge)),
	}))

	assert.True(t, <-online, "the recovery report must be sent once the charger already counts as online")

	handler.Close()

	assert.True(t, handler.IsOnline())
}

// The two error grid types map onto a known grid with zero phases - TN400VNeutralOnWrongPin to
// (TN, 0), ITGroundConnectedToPin2Or3 to (IT, 0). Zero phases is a wiring fault, not a topology:
// republishing it deletes phases and sup_phase_modes, so cliffhanger rejects every
// cmd.phase_mode.set and energy guard drops the charger from phase balancing until the fault
// clears. The alarm above is the report for this case.
func TestObservationsHandler_ZeroPhaseFaultKeepsTheChargepointProps(t *testing.T) {
	t.Parallel()

	now := time.Now()

	for _, tc := range []struct {
		name     string
		gridType model.GridType
		cached   types.GridType
	}{
		{"TN neutral on wrong pin", model.GridTypeErrorTN400VNeutralOnWrongPin, types.GridTypeTN},
		{"IT ground on pin 2 or 3", model.GridTypeErrorITGroundConnectedToPin2Or3, types.GridTypeIT},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cacheMock := mockedcache.NewCache(t)
			cacheMock.On("GridType").Return(tc.cached, now)
			cacheMock.On("Phases").Return(3, now)

			props := map[string]interface{}{
				chargepoint.PropertyGridType:            tc.cached,
				chargepoint.PropertyPhases:              3,
				chargepoint.PropertySupportedPhaseModes: []types.PhaseMode{types.PhaseModeNL3, types.PhaseModeNL1L2L3},
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
				Value:     strconv.Itoa(int(tc.gridType)),
			}))

			assert.Equal(t, props, srv.Specification().Props, "a wiring fault must leave the advertised topology alone")
			assert.Zero(t, thing.inclusionCount(), "nothing to republish while the phase count is unknown")
		})
	}
}

// The fault-clear path still has to land: once a genuine topology arrives the props are
// republished. The strict cache mock is what proves the fault in between never reached it.
func TestObservationsHandler_TopologyAfterZeroPhaseFaultStillRepublishes(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(types.GridTypeTN, now)
	cacheMock.On("Phases").Return(3, now)
	cacheMock.On("SetInstallationParameters", types.GridTypeIT, 1, now).Return(true)

	props := map[string]interface{}{
		chargepoint.PropertyGridType:            types.GridTypeTN,
		chargepoint.PropertyPhases:              3,
		chargepoint.PropertySupportedPhaseModes: []types.PhaseMode{types.PhaseModeNL3, types.PhaseModeNL1L2L3},
	}

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("Specification").Return(&fimptype.Service{Props: props}).Maybe()

	thing := &phaseThing{srv: srv}

	handler, err := signalr.NewObservationsHandler(thing, cacheMock, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	for _, gridType := range []model.GridType{model.GridTypeErrorTN400VNeutralOnWrongPin, model.GridTypeIT1Phase} {
		require.NoError(t, handler.HandleObservation(model.Observation{
			ID:        model.DetectedPowerGridType,
			ChargerID: testChargerID,
			DataType:  model.ObservationDataTypeInteger,
			Timestamp: now,
			Value:     strconv.Itoa(int(gridType)),
		}))
	}

	flushReports(t, handler)

	assert.Equal(t, types.GridTypeIT, srv.Specification().Props[chargepoint.PropertyGridType])
	assert.Equal(t, 1, srv.Specification().Props[chargepoint.PropertyPhases])
	assert.Equal(t, 1, thing.inclusionCount(), "only the recovery republishes")
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
	assert.Zero(t, thing.inclusionCount(), "nothing to republish when the topology did not change")
}

// A charger repeats its grid type on every reconnect, so a lasting wiring warning is logged when
// it changes rather than on every observation.
func TestObservationsHandler_FaultyGridTypeWarnsOnChange(t *testing.T) {
	type obs struct {
		gridType model.GridType
		at       time.Duration
	}

	faulty := model.GridTypeWarningTN1PhaseNeutralOnPin3

	tests := []struct {
		name string
		obs  []obs
		want []int
	}{
		{
			name: "each change to a faulty or undocumented grid type",
			obs:  []obs{{faulty, 0}, {faulty, 0}, {model.GridTypeTN1Phase, 0}, {faulty, 0}, {faulty, 0}, {model.GridTypeUnknown, 0}, {6, 0}},
			want: []int{31, 31, 6},
		},
		{
			name: "a stale healthy grid type does not reset the warning",
			obs:  []obs{{faulty, 2 * time.Second}, {faulty, 4 * time.Second}, {model.GridTypeTN1Phase, 3 * time.Second}, {faulty, 5 * time.Second}},
			want: []int{31},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := logtest.NewLocal(log.StandardLogger())
			t.Cleanup(hook.Reset)

			now := time.Now()

			cacheMock := mockedcache.NewCache(t)
			cacheMock.On("GridType").Return(types.GridTypeTN, now)
			cacheMock.On("Phases").Return(1, now)

			handler, err := signalr.NewObservationsHandler(&phaseThing{srv: mockedchargepoint.NewService(t)}, cacheMock, nil, nil, testChargerID, nil)
			require.NoError(t, err)

			for _, o := range tt.obs {
				require.NoError(t, handler.HandleObservation(model.Observation{
					ID:        model.DetectedPowerGridType,
					ChargerID: testChargerID,
					DataType:  model.ObservationDataTypeInteger,
					Timestamp: now.Add(o.at),
					Value:     strconv.Itoa(int(o.gridType)),
				}))
			}

			flushReports(t, handler)

			var warnings []string

			for _, entry := range hook.AllEntries() {
				if strings.Contains(strings.ToLower(entry.Message), "faulty grid type") {
					warnings = append(warnings, entry.Message)
				}
			}

			require.Len(t, warnings, len(tt.want), "warnings: %v", warnings)

			for i, gridType := range tt.want {
				assert.Contains(t, warnings[i], "["+testChargerID+"] Faulty grid type "+strconv.Itoa(gridType)+" ")
			}
		})
	}
}

// Every report the handler sends takes the same service lock a chargepoint command holds while
// it waits up to CurrentWaitDuration for its echo observation - and that echo can only arrive on
// the single goroutine draining every charger's observations. Sending inline parks that drain
// behind the command, so the echo the command is waiting for is never read, the wait times out,
// and a start that did work is reported as failed. HandleObservation must therefore return
// while the service lock is still held.
func TestObservationsHandler_ReportsDoNotBlockTheDrain(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("ChargerState").Return(chargepoint.StateDisconnected, now)
	cacheMock.On("SetChargerState", chargepoint.StateCharging, now).Return(true)
	cacheMock.On("OfferedCurrent").Return(0, now)
	cacheMock.On("SetOfferedCurrent", 16, now).Return(true)

	// Stands in for the service lock a command holds across its echo wait.
	commandDone := make(chan struct{})
	releaseCommand := sync.OnceFunc(func() { close(commandDone) })

	defer releaseCommand()

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("SendStateReport", false).Run(func(mock.Arguments) { <-commandDone }).Return(true, nil).Maybe()
	srv.On("SendCurrentSessionReport", false).Return(true, nil).Maybe()

	handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, cacheMock, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	defer func() {
		releaseCommand()
		handler.Close()
	}()

	// The drain handles both observations in sequence, as the manager's single goroutine does.
	drained := make(chan error, 2)
	go func() {
		drained <- handler.HandleObservation(model.Observation{
			ID: model.ChargerOPState, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
			Timestamp: now, Value: strconv.Itoa(int(model.ChargerStateCharging)),
		})
		drained <- handler.HandleObservation(model.Observation{
			ID: model.DynamicChargerCurrent, ChargerID: testChargerID, DataType: model.ObservationDataTypeDouble,
			Timestamp: now, Value: "16",
		})
	}()

	// Both must come back while the report above is still parked on the service lock.
	for range 2 {
		select {
		case err := <-drained:
			require.NoError(t, err)
		case <-time.After(2 * time.Second):
			t.Fatal("observation drain is parked behind a report waiting on the service lock")
		}
	}
}

// A report queued for a handler that is then torn down must not leave its sender goroutine
// running, and Close must wait for a send that is still parked on the service lock.
func TestObservationsHandler_CloseWaitsForTheSender(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("ChargerState").Return(chargepoint.StateDisconnected, now)
	cacheMock.On("SetChargerState", chargepoint.StateCharging, now).Return(true)

	release := make(chan struct{})

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("SendStateReport", false).Run(func(mock.Arguments) { <-release }).Return(true, nil).Maybe()

	handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, cacheMock, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	require.NoError(t, handler.HandleObservation(model.Observation{
		ID: model.ChargerOPState, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
		Timestamp: now, Value: strconv.Itoa(int(model.ChargerStateCharging)),
	}))

	closed := make(chan struct{})
	go func() {
		handler.Close()
		close(closed)
	}()

	select {
	case <-closed:
		t.Fatal("Close returned while a send was still parked - the sender would outlive it")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)

	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return once the parked send completed")
	}
}

// The manager looks a handler up under its lock but dispatches the observation after releasing
// it, so an Unregister can close the handler while a dispatch is still on its way to enqueue.
// The send must not land on a closed channel.
func TestObservationsHandler_CloseRacingADispatch(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("ChargerState").Return(chargepoint.StateDisconnected, now).Maybe()
	cacheMock.On("SetChargerState", mock.Anything, now).Return(true).Maybe()
	cacheMock.On("SetRequestedOfferedCurrent", 0, mock.Anything).Return(true).Maybe()

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("SendStateReport", false).Return(true, nil).Maybe()

	handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, cacheMock, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		for range 50 {
			_ = handler.HandleObservation(model.Observation{
				ID: model.ChargerOPState, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
				Timestamp: now, Value: strconv.Itoa(int(model.ChargerStateCharging)),
			})
		}
	}()

	handler.Close()
	wg.Wait()
}

// enqueue is lossy once the sender is parked behind a stalled service and the queue is full. A
// dropped telemetry report is the next observation's problem; a dropped state transition is not,
// because the online flag it carries gates every later command and report. The flag has to move
// at the drop, and the stale transitions still queued ahead of it must not move it back when
// they finally drain.
func TestObservationsHandler_DroppedStateTransitionStillMovesTheFlag(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("ChargerState").Return(chargepoint.StateCharging, now)
	cacheMock.On("SetChargerState", mock.Anything, now).Return(true)
	cacheMock.On("SetRequestedOfferedCurrent", 0, mock.Anything).Return(true).Maybe()
	cacheMock.On("SetCableLocked", true, now).Return(true)

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()

	handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, cacheMock, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	observe := func(id model.ObservationID, dataType model.ObservationDataType, value string) {
		require.NoError(t, handler.HandleObservation(model.Observation{
			ID: id, ChargerID: testChargerID, DataType: dataType, Timestamp: now, Value: value,
		}))
	}

	state := func(s model.ChargerState) {
		observe(model.ChargerOPState, model.ObservationDataTypeInteger, strconv.Itoa(int(s)))
	}

	sent := make(chan struct{})
	srv.On("SendStateReport", false).Run(func(mock.Arguments) { close(sent) }).Return(true, nil).Once()
	state(model.ChargerStateOffline)
	<-sent
	require.Eventually(t, func() bool { return !handler.IsOnline() }, time.Second, time.Millisecond)

	// Park the sender on a report that does not touch the flag, then fill the queue behind it
	// with two transitions last: an online one, then the offline one that would win the drain.
	parked, release := make(chan struct{}), make(chan struct{})
	srv.On("SendCableLockReport", false).Run(func(mock.Arguments) { close(parked); <-release }).Return(true, nil).Once()
	srv.On("SendCableLockReport", false).Return(true, nil)
	srv.On("SendStateReport", false).Return(true, nil)

	observe(model.CableLocked, model.ObservationDataTypeBoolean, "true")
	<-parked

	for range 62 {
		observe(model.CableLocked, model.ObservationDataTypeBoolean, "true")
	}

	state(model.ChargerStateDisconnected)
	state(model.ChargerStateOffline)

	state(model.ChargerStateReadyToCharge)
	assert.True(t, handler.IsOnline(), "the dropped recovery must still bring the charger online")

	close(release)
	handler.Close()

	assert.True(t, handler.IsOnline(), "the offline transition queued before the drop must not win the drain")
}

// alarmThing exposes both a chargepoint and an alarm service, so a test can park one report on
// a service lock and watch whether the other path still reaches the drain.
type alarmThing struct {
	adapter.Thing

	cp    *mockedchargepoint.Service
	alarm *mockedalarm.Service

	// inclusionBlock, when set, parks SendInclusionReport - standing in for a stalled MQTT.
	inclusionBlock chan struct{}
}

func (t *alarmThing) Services(name fimptype.ServiceNameT) []adapter.Service {
	if name == alarm.AlarmSystem {
		return []adapter.Service{t.alarm}
	}

	return []adapter.Service{t.cp}
}

func (t *alarmThing) Update(...adapter.ThingUpdate) error { return nil }

func (t *alarmThing) SendInclusionReport(bool) (bool, error) {
	if t.inclusionBlock != nil {
		<-t.inclusionBlock
	}

	return true, nil
}

// #167: #158 moved the chargepoint and meter reports onto the queue but left the alarm path
// publishing inline, so an alarm blocked on the service lock parked the only drainer of the
// observation channel.
func TestObservationsHandler_AlarmReportsDoNotBlockTheDrain(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("AlarmActive", alarm.EventOtherChargeErr).Return(false).Maybe()
	cacheMock.On("SetAlarm", alarm.EventOtherChargeErr, true, now).Return(true)
	cacheMock.On("ChargerState").Return(chargepoint.StateDisconnected, now).Maybe()
	cacheMock.On("SetChargerState", chargepoint.StateCharging, now).Return(true).Maybe()

	// Stands in for the service lock a command holds across its echo wait.
	release := make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })

	defer releaseOnce()

	alarmSrv := mockedalarm.NewService(t)
	alarmSrv.On("Name").Return(alarm.AlarmSystem).Maybe()
	alarmSrv.On("SendAlarmReport", alarm.EventOtherChargeErr, false).
		Run(func(mock.Arguments) { <-release }).Return(true, nil).Maybe()

	cp := mockedchargepoint.NewService(t)
	cp.On("Name").Return(chargepoint.Chargepoint).Maybe()
	cp.On("SendStateReport", false).Return(true, nil).Maybe()

	handler, err := signalr.NewObservationsHandler(
		&alarmThing{cp: cp, alarm: alarmSrv}, cacheMock, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	defer func() {
		releaseOnce()
		handler.Close()
	}()

	drained := make(chan error, 2)
	go func() {
		drained <- handler.HandleObservation(model.Observation{
			ID: model.ErrorCode, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
			Timestamp: now, Value: "1",
		})
		drained <- handler.HandleObservation(model.Observation{
			ID: model.ChargerOPState, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
			Timestamp: now, Value: strconv.Itoa(int(model.ChargerStateCharging)),
		})
	}()

	for range 2 {
		select {
		case err := <-drained:
			require.NoError(t, err)
		case <-time.After(2 * time.Second):
			t.Fatal("observation drain is parked behind an alarm report waiting on the service lock")
		}
	}
}

// #167: the inclusion report republished for a topology change had the same problem - it runs
// on the drain and waits on MQTT.
func TestObservationsHandler_InclusionReportsDoNotBlockTheDrain(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(types.GridTypeTN, now).Maybe()
	cacheMock.On("Phases").Return(1, now).Maybe()
	cacheMock.On("SetInstallationParameters", types.GridTypeTN, 3, now).Return(true)
	cacheMock.On("SetAlarm", mock.Anything, mock.Anything, now).Return(false).Maybe()
	cacheMock.On("ChargerState").Return(chargepoint.StateDisconnected, now).Maybe()
	cacheMock.On("SetChargerState", chargepoint.StateCharging, now).Return(true).Maybe()

	release := make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })

	defer releaseOnce()

	alarmSrv := mockedalarm.NewService(t)
	alarmSrv.On("Name").Return(alarm.AlarmSystem).Maybe()
	alarmSrv.On("SendAlarmReport", mock.Anything, false).Return(true, nil).Maybe()

	cp := mockedchargepoint.NewService(t)
	cp.On("Name").Return(chargepoint.Chargepoint).Maybe()
	cp.On("Specification").Return(&fimptype.Service{Props: map[string]interface{}{}})
	cp.On("SendStateReport", false).Return(true, nil).Maybe()

	thing := &alarmThing{cp: cp, alarm: alarmSrv, inclusionBlock: release}

	handler, err := signalr.NewObservationsHandler(thing, cacheMock, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	defer func() {
		releaseOnce()
		handler.Close()
	}()

	drained := make(chan error, 2)
	go func() {
		drained <- handler.HandleObservation(model.Observation{
			ID: model.DetectedPowerGridType, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
			Timestamp: now, Value: strconv.Itoa(int(model.GridTypeTN3Phase)),
		})
		drained <- handler.HandleObservation(model.Observation{
			ID: model.ChargerOPState, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
			Timestamp: now, Value: strconv.Itoa(int(model.ChargerStateCharging)),
		})
	}()

	for range 2 {
		select {
		case err := <-drained:
			require.NoError(t, err)
		case <-time.After(2 * time.Second):
			t.Fatal("observation drain is parked behind an inclusion report waiting on MQTT")
		}
	}
}

// meterThing exposes a meter_elec service alongside the chargepoint, so the lifetime-energy
// goroutine has somewhere to publish.
type meterThing struct {
	adapter.Thing

	cp    *mockedchargepoint.Service
	meter *mockednumericmeter.Service
}

func (t *meterThing) Services(name fimptype.ServiceNameT) []adapter.Service {
	if name == numericmeter.MeterElec {
		return []adapter.Service{t.meter}
	}

	return []adapter.Service{t.cp}
}

func (t *meterThing) Update(...adapter.ThingUpdate) error    { return nil }
func (t *meterThing) SendInclusionReport(bool) (bool, error) { return true, nil }

// Easee's backplate is PE, N, L1, L2, L3 on TN and PE, L1, L2, L3 on IT/TT, so the lines sit on
// terminals T3-T5 on TN but T2-T4 on IT/TT. An IT hub showed L1L2L3 at 17 A as i1/i2 only.
func TestObservationsHandler_PhaseCurrentsFollowTheGridType(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		gridType types.GridType
		ignored  model.ObservationID
		want     [3]float64
	}{
		{"TN", types.GridTypeTN, model.InCurrentT2, [3]float64{3, 4, 5}},
		{"unknown", "", model.InCurrentT2, [3]float64{3, 4, 5}},
		{"IT", types.GridTypeIT, model.InCurrentT5, [3]float64{2, 3, 4}},
		{"TT", types.GridTypeTT, model.InCurrentT5, [3]float64{2, 3, 4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			now := time.Now()
			phases := cache.NewCache(testChargerID)

			cp := mockedchargepoint.NewService(t)
			cp.On("Name").Return(chargepoint.Chargepoint).Maybe()
			cp.On("Specification").Return(&fimptype.Service{Props: map[string]interface{}{
				chargepoint.PropertyGridType: tc.gridType,
			}}).Maybe()

			meter := mockednumericmeter.NewService(t)
			meter.On("Name").Return(numericmeter.MeterElec).Maybe()
			for _, v := range []numericmeter.Value{numericmeter.ValueCurrentPhase1, numericmeter.ValueCurrentPhase2, numericmeter.ValueCurrentPhase3} {
				meter.On("SendMeterExtendedReport", numericmeter.Values{v}, false).Return(true, nil).Once()
			}

			handler, err := signalr.NewObservationsHandler(
				&meterThing{cp: cp, meter: meter}, phases, nil, nil, testChargerID, nil)
			require.NoError(t, err)

			for i, id := range []model.ObservationID{model.InCurrentT2, model.InCurrentT3, model.InCurrentT4, model.InCurrentT5} {
				value := strconv.Itoa(i + 2)
				if id == tc.ignored {
					value = "x"
				}

				assert.True(t, id.Supported(), "the receiver drops %s", id.Str())
				assert.NoError(t, handler.HandleObservation(model.Observation{
					ID: id, ChargerID: testChargerID, DataType: model.ObservationDataTypeDouble,
					Timestamp: now, Value: value,
				}))
			}

			flushReports(t, handler)

			i1, _ := phases.Phase1Current()
			i2, _ := phases.Phase2Current()
			i3, _ := phases.Phase3Current()
			assert.Equal(t, tc.want, [3]float64{i1, i2, i3})
		})
	}
}

// On TN the neutral current streams as often as the line currents, so dropping it must not cost a
// service lookup per reading.
func TestObservationsHandler_NeutralCurrentReadsTheGridTypeOnce(t *testing.T) {
	t.Parallel()

	cp := mockedchargepoint.NewService(t)
	cp.On("Name").Return(chargepoint.Chargepoint).Maybe()
	cp.On("Specification").Return(&fimptype.Service{Props: map[string]interface{}{
		chargepoint.PropertyGridType: types.GridTypeTN,
	}}).Once()

	handler, err := signalr.NewObservationsHandler(&phaseThing{srv: cp}, mockedcache.NewCache(t), nil, nil, testChargerID, nil)
	require.NoError(t, err)

	for _, value := range []string{"1", "2"} {
		require.NoError(t, handler.HandleObservation(model.Observation{
			ID: model.InCurrentT2, ChargerID: testChargerID, DataType: model.ObservationDataTypeDouble,
			Timestamp: time.Now(), Value: value,
		}))
	}
}

// The grid type is kept after the first reading, so a topology change has to replace it.
func TestObservationsHandler_PhaseCurrentsFollowATopologyChange(t *testing.T) {
	t.Parallel()

	now := time.Now()
	phases := cache.NewCache(testChargerID)

	cp := mockedchargepoint.NewService(t)
	cp.On("Name").Return(chargepoint.Chargepoint).Maybe()
	cp.On("Specification").Return(&fimptype.Service{Props: map[string]interface{}{
		chargepoint.PropertyGridType: types.GridTypeTN,
	}}).Maybe()

	meter := mockednumericmeter.NewService(t)
	meter.On("Name").Return(numericmeter.MeterElec).Maybe()
	meter.On("SendMeterExtendedReport", numericmeter.Values{numericmeter.ValueCurrentPhase1}, false).Return(true, nil).Once()

	handler, err := signalr.NewObservationsHandler(&meterThing{cp: cp, meter: meter}, phases, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	for _, o := range []model.Observation{
		{ID: model.InCurrentT2, DataType: model.ObservationDataTypeDouble, Value: "7"},
		{ID: model.DetectedPowerGridType, DataType: model.ObservationDataTypeInteger, Value: strconv.Itoa(int(model.GridTypeIT3Phase))},
		{ID: model.InCurrentT2, DataType: model.ObservationDataTypeDouble, Value: "8"},
	} {
		o.ChargerID, o.Timestamp = testChargerID, now
		require.NoError(t, handler.HandleObservation(o))
	}

	flushReports(t, handler)

	i1, _ := phases.Phase1Current()
	assert.InDelta(t, 8, i1, 0)
}

// #166: Close closed the report queue and waited for runSender, but never stopped the
// lifetime-energy goroutine. Unregister calls Close, so after cmd.thing.delete that goroutine
// still woke at EnergyLifetimeInterval and published a meter report on a torn-down thing.
func TestObservationsHandler_CloseStopsTheEnergyGoroutine(t *testing.T) {
	t.Parallel()

	now := time.Now()

	storage := fakes.NewConfigStorage(t, &config.Config{
		PublicConfig: config.PublicConfig{EnergyLifetimeInterval: "150ms"},
	}, config.Factory)
	cfgSrv := config.NewService(storage)

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("LifetimeEnergy").Return(0.0, time.Time{})
	cacheMock.On("SetLifetimeEnergy", mock.Anything, mock.Anything).Return(true).Maybe()

	var published atomic.Bool

	meter := mockednumericmeter.NewService(t)
	meter.On("Name").Return(numericmeter.MeterElec).Maybe()
	meter.On("SendMeterReport", numericmeter.UnitKWh, false).
		Run(func(mock.Arguments) { published.Store(true) }).Return(true, nil).Maybe()
	meter.On("SendMeterExtendedReport", mock.Anything, false).
		Run(func(mock.Arguments) { published.Store(true) }).Return(true, nil).Maybe()

	cp := mockedchargepoint.NewService(t)
	cp.On("Name").Return(chargepoint.Chargepoint).Maybe()

	handler, err := signalr.NewObservationsHandler(
		&meterThing{cp: cp, meter: meter}, cacheMock, cfgSrv, nil, testChargerID, nil)
	require.NoError(t, err)

	// Starts the energy goroutine and its timer.
	require.NoError(t, handler.HandleObservation(model.Observation{
		ID: model.LifetimeEnergy, ChargerID: testChargerID, DataType: model.ObservationDataTypeDouble,
		Timestamp: now, Value: "123.4",
	}))

	handler.Close()

	// Past the interval the timer would have fired at.
	time.Sleep(400 * time.Millisecond)

	assert.False(t, published.Load(), "the energy goroutine published on a thing that was torn down")
}

// #176: Close waited for the queued report sender before stopping the lifetime-energy timer, so a
// report blocked on the service lock held that wait open while the timer kept running - and the
// energy goroutine published a meter report on a thing the adapter was already tearing down.
func TestObservationsHandler_CloseStopsTheEnergyGoroutineBeforeDrainingReports(t *testing.T) {
	t.Parallel()

	now := time.Now()

	storage := fakes.NewConfigStorage(t, &config.Config{
		PublicConfig: config.PublicConfig{EnergyLifetimeInterval: "150ms"},
	}, config.Factory)
	cfgSrv := config.NewService(storage)

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("LifetimeEnergy").Return(0.0, time.Time{})
	cacheMock.On("SetLifetimeEnergy", mock.Anything, mock.Anything).Return(true).Maybe()
	cacheMock.On("SetCableLocked", mock.Anything, mock.Anything).Return(true).Maybe()

	var published atomic.Bool

	// release unblocks the queued report, standing in for the service lock a command holds for
	// up to CurrentWaitDuration while the sender is parked on it.
	release := make(chan struct{})

	meter := mockednumericmeter.NewService(t)
	meter.On("Name").Return(numericmeter.MeterElec).Maybe()
	meter.On("SendMeterReport", numericmeter.UnitKWh, false).
		Run(func(mock.Arguments) { published.Store(true) }).Return(true, nil).Maybe()
	meter.On("SendMeterExtendedReport", mock.Anything, false).
		Run(func(mock.Arguments) { published.Store(true) }).Return(true, nil).Maybe()

	cp := mockedchargepoint.NewService(t)
	cp.On("Name").Return(chargepoint.Chargepoint).Maybe()
	cp.On("SendCableLockReport", false).
		Run(func(mock.Arguments) { <-release }).Return(true, nil).Maybe()

	handler, err := signalr.NewObservationsHandler(
		&meterThing{cp: cp, meter: meter}, cacheMock, cfgSrv, nil, testChargerID, nil)
	require.NoError(t, err)

	// Starts the energy goroutine and its timer.
	require.NoError(t, handler.HandleObservation(model.Observation{
		ID: model.LifetimeEnergy, ChargerID: testChargerID, DataType: model.ObservationDataTypeDouble,
		Timestamp: now, Value: "123.4",
	}))

	// Queues a report that parks the sender, so senderWG.Wait() cannot return on its own.
	require.NoError(t, handler.HandleObservation(model.Observation{
		ID: model.CableLocked, ChargerID: testChargerID, DataType: model.ObservationDataTypeBoolean,
		Timestamp: now, Value: "false",
	}))

	closed := make(chan struct{})

	go func() {
		handler.Close()
		close(closed)
	}()

	// Past the interval the energy timer would have fired at, while Close is still waiting.
	time.Sleep(400 * time.Millisecond)

	assert.False(t, published.Load(),
		"the energy goroutine published while Close was still draining the report queue")

	close(release)

	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return after the queued report was released")
	}
}

// #176: the topology was written to the cache before the inclusion report was enqueued, so a
// republish the sender dropped or failed left the cache already holding the new grid type - and
// the equality check turned every repeat observation away, leaving the hub advertising the old
// topology until the next restart.
func TestObservationsHandler_FailedTopologyRepublishIsRetried(t *testing.T) {
	t.Parallel()

	now := time.Now()

	// Mirrors the real cache: the setter commits, and the getters return what was committed, so
	// the equality check in the handler short-circuits exactly when production would.
	var (
		cacheMu  sync.Mutex
		gridType = types.GridTypeTN
		phases   = 3
	)

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(func() (types.GridType, time.Time) {
		cacheMu.Lock()
		defer cacheMu.Unlock()

		return gridType, now
	})
	cacheMock.On("Phases").Return(func() (int, time.Time) {
		cacheMu.Lock()
		defer cacheMu.Unlock()

		return phases, now
	})
	cacheMock.On("SetInstallationParameters", types.GridTypeIT, 1, now).
		Run(func(mock.Arguments) {
			cacheMu.Lock()
			defer cacheMu.Unlock()

			gridType, phases = types.GridTypeIT, 1
		}).Return(true).Maybe()

	props := map[string]interface{}{
		chargepoint.PropertyGridType:            types.GridTypeTN,
		chargepoint.PropertyPhases:              3,
		chargepoint.PropertySupportedPhaseModes: []types.PhaseMode{types.PhaseModeNL3, types.PhaseModeNL1L2L3},
	}

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("Specification").Return(&fimptype.Service{Props: props}).Maybe()

	// The first inclusion report fails; the second must still be attempted.
	thing := &phaseThing{srv: srv, fail: 1}

	handler, err := signalr.NewObservationsHandler(thing, cacheMock, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	obs := model.Observation{
		ID:        model.DetectedPowerGridType,
		ChargerID: testChargerID,
		DataType:  model.ObservationDataTypeInteger,
		Timestamp: now,
		Value:     strconv.Itoa(int(model.GridTypeIT1Phase)),
	}

	require.NoError(t, handler.HandleObservation(obs))

	waitFor(t, func() bool { return thing.inclusionCount() == 1 })

	// The repeat observation must not be short-circuited by a cache the failed republish wrote.
	require.NoError(t, handler.HandleObservation(obs))

	flushReports(t, handler)

	assert.Equal(t, 2, thing.inclusionCount(), "the dropped topology republish was never retried")
}

// Raw grid types 2 and 30 share TN with 2 phases but not their modes: 30 drives L1 and L3 and
// advertises none. A change between them must republish although the FIMP pair stays, and the
// raw type the thing was created with must keep a start on an unchanged charger quiet.
func TestObservationsHandler_TN2PhaseVariantRefreshesTheModes(t *testing.T) {
	t.Parallel()

	nominal := []types.PhaseMode{types.PhaseModeNL1, types.PhaseModeNL2, types.PhaseModeNL1L2}
	tn3 := []types.PhaseMode{types.PhaseModeNL1, types.PhaseModeNL2, types.PhaseModeNL3, types.PhaseModeNL1L2L3}

	tests := []struct {
		name       string
		created    model.GridType // raw type in the thing state; zero in one stored by an older adapter
		phases     int
		advertised []types.PhaseMode
		observed   model.GridType
		want       []types.PhaseMode
		inclusions int
	}{
		{"pins 2, 3, 5 appear", model.GridTypeTN2PhasePin23, 2, nominal, model.GridTypeWarningTN2PhasePin235, nil, 1},
		{"pins 2, 3, 5 cleared", model.GridTypeWarningTN2PhasePin235, 2, nil, model.GridTypeTN2PhasePin23, nominal, 1},
		{"a new topology on pins 2, 3, 5", model.GridTypeTN3Phase, 3, tn3, model.GridTypeWarningTN2PhasePin235, nil, 1},
		{"unchanged pins 2, 3, 5", model.GridTypeWarningTN2PhasePin235, 2, nil, model.GridTypeWarningTN2PhasePin235, nil, 0},
		{"unchanged nominal", model.GridTypeTN2PhasePin23, 2, nominal, model.GridTypeTN2PhasePin23, nominal, 0},
		{"warning on pins 2, 3, 4", model.GridTypeTN2PhasePin23, 2, nominal, model.GridTypeWarningTN2PhasePIN234, nominal, 0},
		{"ground fault", model.GridTypeTN2PhasePin23, 2, nominal, model.GridTypeWarningTN2PhaseGNDFault, nominal, 0},
		{"older adapter's state, pins 2, 3, 5", model.GridTypeNotYetDetected, 2, nominal, model.GridTypeWarningTN2PhasePin235, nil, 1},
		{"older adapter's state, nominal", model.GridTypeNotYetDetected, 2, nominal, model.GridTypeTN2PhasePin23, nominal, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			now := time.Now()

			cacheMock := mockedcache.NewCache(t)
			cacheMock.On("GridType").Return(types.GridTypeTN, time.Time{})
			cacheMock.On("Phases").Return(tt.phases, time.Time{})
			cacheMock.On("SetInstallationParameters", types.GridTypeTN, 2, now).Return(true).Maybe()

			props := map[string]interface{}{
				chargepoint.PropertyGridType: types.GridTypeTN,
				chargepoint.PropertyPhases:   tt.phases,
			}
			if tt.advertised != nil {
				props[chargepoint.PropertySupportedPhaseModes] = tt.advertised
			}

			srv := mockedchargepoint.NewService(t)
			srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
			srv.On("Specification").Return(&fimptype.Service{Props: props}).Maybe()

			thing := &phaseThing{srv: srv}

			handler, err := signalr.NewObservationsHandler(thing, cacheMock, nil, nil, testChargerID, &fakePhaseStore{gridType: tt.created})
			require.NoError(t, err)

			require.NoError(t, handler.HandleObservation(model.Observation{
				ID:        model.DetectedPowerGridType,
				ChargerID: testChargerID,
				DataType:  model.ObservationDataTypeInteger,
				Timestamp: now,
				Value:     strconv.Itoa(int(tt.observed)),
			}))

			flushReports(t, handler)

			got := srv.Specification().Props
			if tt.want == nil {
				assert.NotContains(t, got, chargepoint.PropertySupportedPhaseModes)
			} else {
				assert.Equal(t, tt.want, got[chargepoint.PropertySupportedPhaseModes])
			}

			assert.Equal(t, types.GridTypeTN, got[chargepoint.PropertyGridType])
			assert.Equal(t, tt.inclusions, thing.inclusionCount())
		})
	}
}

// The raw type is committed with the cache only once the inclusion report is out, so a dropped
// report is retried by the next observation and a landed one is not repeated.
func TestObservationsHandler_DroppedPin235RepublishIsRetried(t *testing.T) {
	t.Parallel()

	now := time.Now()
	published := make(chan struct{}, 1)

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(types.GridTypeTN, time.Time{})
	cacheMock.On("Phases").Return(2, time.Time{})
	cacheMock.On("SetInstallationParameters", types.GridTypeTN, 2, now).
		Run(func(mock.Arguments) { published <- struct{}{} }).Return(true).Once()

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("Specification").Return(&fimptype.Service{Props: map[string]interface{}{
		chargepoint.PropertyGridType:            types.GridTypeTN,
		chargepoint.PropertyPhases:              2,
		chargepoint.PropertySupportedPhaseModes: []types.PhaseMode{types.PhaseModeNL1, types.PhaseModeNL2, types.PhaseModeNL1L2},
	}}).Maybe()

	thing := &phaseThing{srv: srv, fail: 1}

	handler, err := signalr.NewObservationsHandler(thing, cacheMock, nil, nil, testChargerID, &fakePhaseStore{gridType: model.GridTypeTN2PhasePin23})
	require.NoError(t, err)

	obs := model.Observation{
		ID:        model.DetectedPowerGridType,
		ChargerID: testChargerID,
		DataType:  model.ObservationDataTypeInteger,
		Timestamp: now,
		Value:     strconv.Itoa(int(model.GridTypeWarningTN2PhasePin235)),
	}

	require.NoError(t, handler.HandleObservation(obs))
	waitFor(t, func() bool { return thing.inclusionCount() == 1 })

	require.NoError(t, handler.HandleObservation(obs))

	select {
	case <-published:
	case <-time.After(2 * time.Second):
		t.Fatal("the dropped republish was never retried")
	}

	require.NoError(t, handler.HandleObservation(obs))
	flushReports(t, handler)

	assert.Equal(t, 2, thing.inclusionCount(), "a landed republish must not be repeated")
}

func TestObservationsHandler_OutdatedTopologyKeepsProps(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(types.GridTypeIT, now)
	cacheMock.On("Phases").Return(1, now)

	props := map[string]interface{}{
		chargepoint.PropertyGridType:            types.GridTypeIT,
		chargepoint.PropertyPhases:              1,
		chargepoint.PropertySupportedPhaseModes: []types.PhaseMode{types.PhaseModeNL1},
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
		Timestamp: now.Add(-time.Minute),
		Value:     strconv.Itoa(int(model.GridTypeTN3Phase)),
	}))

	flushReports(t, handler)

	assert.Equal(t, types.GridTypeIT, srv.Specification().Props[chargepoint.PropertyGridType])
	assert.Equal(t, 0, thing.inclusionCount())
}

func TestObservationsHandler_TopologyOlderThanPendingKeepsProps(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(types.GridTypeIT, now.Add(-2*time.Minute))
	cacheMock.On("Phases").Return(1, now.Add(-2*time.Minute))
	cacheMock.On("SetInstallationParameters", mock.Anything, mock.Anything, mock.Anything).Return(true).Maybe()

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("Specification").Return(&fimptype.Service{Props: map[string]interface{}{}})

	block := make(chan struct{})
	thing := &phaseThing{srv: srv, block: block}

	handler, err := signalr.NewObservationsHandler(thing, cacheMock, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	for _, obs := range []struct {
		at time.Time
		gt model.GridType
	}{{now, model.GridTypeTN3Phase}, {now.Add(-time.Minute), model.GridTypeIT3Phase}} {
		require.NoError(t, handler.HandleObservation(model.Observation{
			ID:        model.DetectedPowerGridType,
			ChargerID: testChargerID,
			DataType:  model.ObservationDataTypeInteger,
			Timestamp: obs.at,
			Value:     strconv.Itoa(int(obs.gt)),
		}))
	}

	close(block)
	flushReports(t, handler)

	assert.Equal(t, types.GridTypeTN, srv.Specification().Props[chargepoint.PropertyGridType])
	assert.Equal(t, 1, thing.inclusionCount())
}

func TestObservationsHandler_TopologyOlderThanUnchangedKeepsProps(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(types.GridTypeIT, now.Add(-2*time.Minute))
	cacheMock.On("Phases").Return(3, now.Add(-2*time.Minute))

	srv := mockedchargepoint.NewService(t)
	srv.On("Specification").Return(&fimptype.Service{Props: map[string]interface{}{}}).Maybe()

	thing := &phaseThing{srv: srv}

	handler, err := signalr.NewObservationsHandler(thing, cacheMock, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	for _, obs := range []struct {
		at time.Time
		gt model.GridType
	}{{now, model.GridTypeIT3Phase}, {now.Add(-time.Minute), model.GridTypeTN3Phase}} {
		require.NoError(t, handler.HandleObservation(model.Observation{
			ID:        model.DetectedPowerGridType,
			ChargerID: testChargerID,
			DataType:  model.ObservationDataTypeInteger,
			Timestamp: obs.at,
			Value:     strconv.Itoa(int(obs.gt)),
		}))
	}

	flushReports(t, handler)

	assert.Nil(t, srv.Specification().Props[chargepoint.PropertyGridType])
	assert.Equal(t, 0, thing.inclusionCount())
}

func TestObservationsHandler_CloudConnectedFlipSendsConnectivityReport(t *testing.T) {
	t.Parallel()

	thing := &phaseThing{}

	handler, err := signalr.NewObservationsHandler(thing, mockedcache.NewCache(t), nil, nil, testChargerID, nil)
	require.NoError(t, err)

	thing.online = handler.IsOnline

	cloud := func(v string) {
		require.NoError(t, handler.HandleObservation(model.Observation{
			ID: model.CloudConnected, ChargerID: testChargerID, DataType: model.ObservationDataTypeBoolean, Value: v,
		}))
	}

	cloud("true")
	cloud("false")
	waitFor(t, func() bool { return len(thing.connectivityReports()) == 1 })
	cloud("false")
	cloud("true")
	flushReports(t, handler)

	assert.Equal(t, []bool{false, true}, thing.connectivityReports())
}

func TestObservationsHandler_StateOnlineFlipSendsConnectivityReport(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("ChargerState").Return(chargepoint.StateCharging, now)
	cacheMock.On("SetChargerState", mock.Anything, now).Return(true)
	cacheMock.On("SetRequestedOfferedCurrent", 0, mock.Anything).Return(true).Maybe()

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("SendStateReport", false).Return(true, nil)

	thing := &phaseThing{srv: srv}

	handler, err := signalr.NewObservationsHandler(thing, cacheMock, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	thing.online = handler.IsOnline

	state := func(s model.ChargerState) {
		require.NoError(t, handler.HandleObservation(model.Observation{
			ID: model.ChargerOPState, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
			Timestamp: now, Value: strconv.Itoa(int(s)),
		}))
	}

	state(model.ChargerStateCharging)
	state(model.ChargerStateOffline)
	waitFor(t, func() bool { return len(thing.connectivityReports()) == 1 })
	state(model.ChargerStateOffline)
	state(model.ChargerStateReadyToCharge)
	flushReports(t, handler)

	assert.Equal(t, []bool{false, true}, thing.connectivityReports())
}

func TestObservationsHandler_OutputPhaseUsesPendingTopology(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(types.GridType(""), time.Time{})
	cacheMock.On("Phases").Return(0, time.Time{})
	cacheMock.On("SetInstallationParameters", types.GridTypeTN, 3, now).Return(true).Once()
	cacheMock.On("SetOutputPhaseType", types.PhaseModeNL2, now).Return(true)

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("Specification").Return(&fimptype.Service{Props: map[string]interface{}{}})
	srv.On("IsPhaseModeAware").Return(true).Maybe()
	srv.On("SendPhaseModeReport", false).Return(true, nil).Maybe()

	block := make(chan struct{})
	thing := &phaseThing{srv: srv, block: block}
	store := &fakePhaseStore{}

	handler, err := signalr.NewObservationsHandler(thing, cacheMock, nil, nil, testChargerID, store)
	require.NoError(t, err)

	require.NoError(t, handler.HandleObservation(model.Observation{
		ID:        model.DetectedPowerGridType,
		ChargerID: testChargerID,
		DataType:  model.ObservationDataTypeInteger,
		Timestamp: now,
		Value:     strconv.Itoa(int(model.GridTypeTN3Phase)),
	}))
	require.NoError(t, handler.HandleObservation(model.Observation{
		ID:        model.OutputPhase,
		ChargerID: testChargerID,
		DataType:  model.ObservationDataTypeInteger,
		Timestamp: now,
		Value:     strconv.Itoa(int(model.P1T2T4TN)),
	}))

	assert.Equal(t, model.SettablePhaseModes(types.GridTypeTN, 3),
		srv.Specification().Props[chargepoint.PropertySupportedPhaseModes])

	close(block)
	flushReports(t, handler)

	assert.Equal(t, types.PhaseModeNL2, store.mode)
	assert.Equal(t, 1, thing.inclusionCount(), "only the topology republishes")
}

// The 0 W reading after a stop is the one reading that publishes ready_to_charge, so a failed
// meter report must not cost the hub the state report.
func TestObservationsHandler_FailedMeterReportStillSendsTheStateReport(t *testing.T) {
	hook := logtest.NewLocal(log.StandardLogger())
	t.Cleanup(hook.Reset)

	now := time.Now()
	c := cache.NewCache(testChargerID)
	c.SetChargerState(chargepoint.StateReadyToCharge, now)

	cp := mockedchargepoint.NewService(t)
	cp.On("Name").Return(chargepoint.Chargepoint).Maybe()
	cp.On("SendStateReport", false).Return(true, nil).Once()

	meter := mockednumericmeter.NewService(t)
	meter.On("Name").Return(numericmeter.MeterElec).Maybe()
	meter.On("SendMeterReport", numericmeter.UnitW, false).Return(false, assert.AnError).Once()
	meter.On("SendMeterExtendedReport", numericmeter.Values{numericmeter.ValuePowerImport}, false).Return(true, nil).Once()

	handler, err := signalr.NewObservationsHandler(&meterThing{cp: cp, meter: meter}, c, nil, nil, testChargerID, nil)
	require.NoError(t, err)

	require.NoError(t, handler.HandleObservation(model.Observation{
		ID: model.TotalPower, ChargerID: testChargerID, DataType: model.ObservationDataTypeDouble,
		Timestamp: now, Value: "0",
	}))

	flushReports(t, handler)

	var logged bool

	for _, entry := range hook.AllEntries() {
		logged = logged || strings.Contains(entry.Message, assert.AnError.Error())
	}

	assert.True(t, logged, "the meter report error is still logged")
}

// After a restart no charger state is cached yet, so a 0 W reading would publish an empty state.
func TestObservationsHandler_PowerBeforeAnyChargerStateSendsNoStateReport(t *testing.T) {
	t.Parallel()

	cp := mockedchargepoint.NewService(t)
	cp.On("Name").Return(chargepoint.Chargepoint).Maybe()

	meter := mockednumericmeter.NewService(t)
	meter.On("Name").Return(numericmeter.MeterElec).Maybe()
	meter.On("SendMeterReport", numericmeter.UnitW, false).Return(true, nil).Once()
	meter.On("SendMeterExtendedReport", numericmeter.Values{numericmeter.ValuePowerImport}, false).Return(true, nil).Once()

	handler, err := signalr.NewObservationsHandler(
		&meterThing{cp: cp, meter: meter}, cache.NewCache(testChargerID), nil, nil, testChargerID, nil)
	require.NoError(t, err)

	require.NoError(t, handler.HandleObservation(model.Observation{
		ID: model.TotalPower, ChargerID: testChargerID, DataType: model.ObservationDataTypeDouble,
		Timestamp: time.Now(), Value: "0",
	}))

	flushReports(t, handler)
}

func TestObservationsHandler_ChargerStateMovesTheThreePhaseLock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		lock     bool
		prev     chargepoint.State
		next     model.ChargerState
		internal int
		wantSet  bool
	}{
		{name: "a session start locks auto", lock: true, prev: chargepoint.StateDisconnected, next: model.ChargerStateAwaitingStart, internal: 2, wantSet: true},
		{name: "a disconnect releases the lock", lock: true, prev: chargepoint.StateCharging, next: model.ChargerStateDisconnected, internal: 3, wantSet: true},
		{name: "a completed session keeps the lock", lock: true, prev: chargepoint.StateCharging, next: model.ChargerStateCompleted, internal: 3},
		{name: "a disconnect after completed releases the lock", lock: true, prev: chargepoint.StateFinished, next: model.ChargerStateDisconnected, internal: 3, wantSet: true},
		{name: "a change inside a session leaves auto alone", lock: true, prev: chargepoint.StateReadyToCharge, next: model.ChargerStateCharging, internal: 2},
		{name: "single phase is left alone at a start", lock: true, prev: chargepoint.StateDisconnected, next: model.ChargerStateAwaitingStart, internal: 1},
		{name: "single phase is left alone at a disconnect", lock: true, prev: chargepoint.StateCharging, next: model.ChargerStateDisconnected, internal: 1},
		{name: "auto stays at a disconnect", lock: true, prev: chargepoint.StateCharging, next: model.ChargerStateDisconnected, internal: 2},
		{name: "plugging in locks auto", lock: true, prev: chargepoint.StateDisconnected, next: model.ChargerStateAwaitingAuthentication, internal: 2, wantSet: true},
		{name: "an error keeps the lock", lock: true, prev: chargepoint.StateCharging, next: model.ChargerStateError, internal: 3},
		{name: "awaiting authentication keeps the lock", lock: true, prev: chargepoint.StateCharging, next: model.ChargerStateAwaitingAuthentication, internal: 3},
		{name: "a disconnect after an error releases the lock", lock: true, prev: chargepoint.StateError, next: model.ChargerStateDisconnected, internal: 3, wantSet: true},
		{name: "the first state after a restart releases an idle lock", lock: true, next: model.ChargerStateDisconnected, internal: 3, wantSet: true},
		{name: "the lock stays while charging", lock: true, prev: chargepoint.StateReadyToCharge, next: model.ChargerStateCharging, internal: 3},
		{name: "setting off", prev: chargepoint.StateDisconnected, next: model.ChargerStateAwaitingStart, internal: 2},
		{name: "offline triggers nothing", lock: true, prev: chargepoint.StateCharging, next: model.ChargerStateOffline, internal: 3},
		{name: "an unchanged state triggers nothing", lock: true, prev: chargepoint.StateCharging, next: model.ChargerStateCharging, internal: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			observed := time.Now().Add(-time.Hour)

			c := cache.NewCache(testChargerID)
			c.SetInstallationParameters(types.GridTypeTN, 3, observed)
			c.SetPhaseMode(tt.internal, observed)
			c.SetChargerState(tt.prev, observed)

			srv := mockedchargepoint.NewService(t)
			srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
			srv.On("SendStateReport", false).Return(true, nil).Maybe()

			if tt.wantSet {
				srv.On("SetPhaseMode", types.PhaseModeNL1L2L3).Return(nil).Run(func(mock.Arguments) {
					state, _ := c.ChargerState()
					assert.Equal(t, tt.next.ToFimpState(), state, "the queued set must see the new state")
				}).Once()
			}

			cfg := &config.Config{}
			cfg.UseLocked3Phases = tt.lock
			confSrv := config.NewService(fakes.NewConfigStorage(t, cfg, config.Factory))

			handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, c, confSrv, nil, testChargerID, nil)
			require.NoError(t, err)

			require.NoError(t, handler.HandleObservation(model.Observation{
				ID: model.ChargerOPState, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
				Timestamp: time.Now(), Value: strconv.Itoa(int(tt.next)),
			}))
			flushReports(t, handler)

			if !tt.wantSet {
				srv.AssertNotCalled(t, "SetPhaseMode", mock.Anything)
			}
		})
	}
}

func TestObservationsHandler_SettingTurnedOffReleasesTheLockAtTheSessionEnd(t *testing.T) {
	t.Parallel()

	observed := time.Now().Add(-time.Hour)

	c := cache.NewCache(testChargerID)
	c.SetInstallationParameters(types.GridTypeTN, 3, observed)
	c.SetPhaseMode(3, observed)
	c.SetChargerState(chargepoint.StateDisconnected, observed)

	// One sender runs the reports in order, so the calls show which state change released the lock.
	var calls []string

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("SendStateReport", false).Return(true, nil).Run(func(mock.Arguments) { calls = append(calls, "state") })
	srv.On("SetPhaseMode", types.PhaseModeNL1L2L3).Return(nil).Run(func(mock.Arguments) { calls = append(calls, "set") })

	cfg := &config.Config{}
	cfg.UseLocked3Phases = true
	confSrv := config.NewService(fakes.NewConfigStorage(t, cfg, config.Factory))

	handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, c, confSrv, nil, testChargerID, nil)
	require.NoError(t, err)

	observe := func(state model.ChargerState) {
		require.NoError(t, handler.HandleObservation(model.Observation{
			ID: model.ChargerOPState, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
			Timestamp: time.Now(), Value: strconv.Itoa(int(state)),
		}))
	}

	observe(model.ChargerStateCharging)
	require.NoError(t, confSrv.SetUseLocked3Phases(false))
	observe(model.ChargerStateReadyToCharge)
	// Offline and back is still the same session.
	observe(model.ChargerStateOffline)
	observe(model.ChargerStateCharging)
	observe(model.ChargerStateDisconnected)
	// The mock leaves the cache on mode 3: from here on it counts as set elsewhere.
	observe(model.ChargerStateAwaitingStart)
	observe(model.ChargerStateDisconnected)
	flushReports(t, handler)

	assert.Equal(t, []string{"state", "state", "state", "state", "state", "set", "state", "state"}, calls,
		"released once, after the state report of the first disconnect")
}

func TestObservationsHandler_LockPlacedAfterTheSessionStartIsReleasedAtItsEnd(t *testing.T) {
	t.Parallel()

	observed := time.Now().Add(-time.Hour)

	c := cache.NewCache(testChargerID)
	c.SetInstallationParameters(types.GridTypeTN, 3, observed)
	c.SetPhaseMode(2, observed)
	c.SetChargerState(chargepoint.StateDisconnected, observed)

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("SendStateReport", false).Return(true, nil).Maybe()
	srv.On("IsPhaseModeAware").Return(true).Maybe()
	srv.On("SendPhaseModeReport", false).Return(true, nil).Maybe()
	srv.On("SetPhaseMode", types.PhaseModeNL1L2L3).Return(nil).Once()

	confSrv := config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory))

	handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, c, confSrv, nil, testChargerID, nil)
	require.NoError(t, err)

	observe := func(id model.ObservationID, value int) {
		require.NoError(t, handler.HandleObservation(model.Observation{
			ID: id, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
			Timestamp: time.Now(), Value: strconv.Itoa(value),
		}))
	}

	// The session starts with the setting off; a hub request locks it while the setting is on.
	observe(model.ChargerOPState, int(model.ChargerStateCharging))
	require.NoError(t, confSrv.SetUseLocked3Phases(true))
	observe(model.PhaseMode, model.EaseePhaseModeLock3)
	require.NoError(t, confSrv.SetUseLocked3Phases(false))
	observe(model.ChargerOPState, int(model.ChargerStateDisconnected))
	flushReports(t, handler)
}

func TestObservationsHandler_ModeThreeSetElsewhereAfterAReleaseIsLeftAlone(t *testing.T) {
	t.Parallel()

	observed := time.Now().Add(-time.Hour)

	c := cache.NewCache(testChargerID)
	c.SetInstallationParameters(types.GridTypeTN, 3, observed)
	c.SetPhaseMode(3, observed)
	c.SetChargerState(chargepoint.StateCharging, observed)

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("SendStateReport", false).Return(true, nil).Maybe()
	srv.On("IsPhaseModeAware").Return(true).Maybe()
	srv.On("SendPhaseModeReport", false).Return(true, nil).Maybe()
	srv.On("SetPhaseMode", types.PhaseModeNL1L2L3).Return(nil).Maybe()

	cfg := &config.Config{}
	cfg.UseLocked3Phases = true
	confSrv := config.NewService(fakes.NewConfigStorage(t, cfg, config.Factory))

	handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, c, confSrv, nil, testChargerID, nil)
	require.NoError(t, err)

	observe := func(id model.ObservationID, value int) {
		require.NoError(t, handler.HandleObservation(model.Observation{
			ID: id, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
			Timestamp: time.Now(), Value: strconv.Itoa(value),
		}))
	}

	observe(model.ChargerOPState, int(model.ChargerStateDisconnected))
	observe(model.PhaseMode, model.EaseePhaseModeAuto)
	require.NoError(t, confSrv.SetUseLocked3Phases(false))
	observe(model.PhaseMode, model.EaseePhaseModeLock3)
	observe(model.ChargerOPState, int(model.ChargerStateAwaitingStart))
	flushReports(t, handler)

	srv.AssertNumberOfCalls(t, "SetPhaseMode", 1)
}

func TestObservationsHandler_FailedReleaseIsAskedAgain(t *testing.T) {
	t.Parallel()

	observed := time.Now().Add(-time.Hour)

	c := cache.NewCache(testChargerID)
	c.SetInstallationParameters(types.GridTypeTN, 3, observed)
	c.SetPhaseMode(3, observed)
	c.SetChargerState(chargepoint.StateDisconnected, observed)

	reported := make(chan struct{}, 8)

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("SendStateReport", false).Return(true, nil).Run(func(mock.Arguments) { reported <- struct{}{} })
	srv.On("SetPhaseMode", types.PhaseModeNL1L2L3).Return(assert.AnError).Once()
	srv.On("SetPhaseMode", types.PhaseModeNL1L2L3).Return(nil).Once()

	cfg := &config.Config{}
	cfg.UseLocked3Phases = true
	confSrv := config.NewService(fakes.NewConfigStorage(t, cfg, config.Factory))

	handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, c, confSrv, nil, testChargerID, nil)
	require.NoError(t, err)

	observe := func(state model.ChargerState) {
		require.NoError(t, handler.HandleObservation(model.Observation{
			ID: model.ChargerOPState, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
			Timestamp: time.Now(), Value: strconv.Itoa(int(state)),
		}))
	}

	observe(model.ChargerStateCharging)
	require.NoError(t, confSrv.SetUseLocked3Phases(false))
	observe(model.ChargerStateDisconnected)
	// One sender: the third state report goes out only after the failed release.
	observe(model.ChargerStateDisconnected)

	for range 3 {
		<-reported
	}

	observe(model.ChargerStateAwaitingStart)
	flushReports(t, handler)
}

func TestObservationsHandler_ReleaseSkippedOnAOneModeTopologyIsAskedAgain(t *testing.T) {
	t.Parallel()

	observed := time.Now().Add(-time.Hour)

	c := cache.NewCache(testChargerID)
	c.SetInstallationParameters(types.GridTypeTN, 3, observed)
	c.SetPhaseMode(3, observed)
	c.SetChargerState(chargepoint.StateDisconnected, observed)

	srv := mockedchargepoint.NewService(t)
	srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
	srv.On("SendStateReport", false).Return(true, nil).Maybe()
	srv.On("SetPhaseMode", types.PhaseModeNL1L2L3).Return(nil).Once()

	cfg := &config.Config{}
	cfg.UseLocked3Phases = true
	confSrv := config.NewService(fakes.NewConfigStorage(t, cfg, config.Factory))

	handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, c, confSrv, nil, testChargerID, nil)
	require.NoError(t, err)

	observe := func(state model.ChargerState) {
		require.NoError(t, handler.HandleObservation(model.Observation{
			ID: model.ChargerOPState, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
			Timestamp: time.Now(), Value: strconv.Itoa(int(state)),
		}))
	}

	observe(model.ChargerStateCharging)
	require.NoError(t, confSrv.SetUseLocked3Phases(false))

	c.SetInstallationParameters(types.GridTypeTN, 1, time.Now())
	observe(model.ChargerStateDisconnected)
	srv.AssertNotCalled(t, "SetPhaseMode", mock.Anything)

	c.SetInstallationParameters(types.GridTypeTN, 3, time.Now())
	observe(model.ChargerStateAwaitingStart)
	flushReports(t, handler)
}

func TestObservationsHandler_SinglePhaseSetWhileTheLockWaitsIsLeftAlone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		prev     chargepoint.State
		next     model.ChargerState
		internal int
	}{
		{name: "a session start on auto", prev: chargepoint.StateDisconnected, next: model.ChargerStateAwaitingStart, internal: 2},
		{name: "a disconnect on the lock", prev: chargepoint.StateCharging, next: model.ChargerStateDisconnected, internal: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			observed := time.Now().Add(-time.Hour)

			c := cache.NewCache(testChargerID)
			c.SetInstallationParameters(types.GridTypeTN, 3, observed)
			c.SetPhaseMode(tt.internal, observed)
			c.SetChargerState(tt.prev, observed)

			release := make(chan struct{})

			srv := mockedchargepoint.NewService(t)
			srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
			srv.On("SendStateReport", false).Return(true, nil).Run(func(mock.Arguments) { <-release }).Once()
			srv.On("IsPhaseModeAware").Return(true).Maybe()
			srv.On("SendPhaseModeReport", false).Return(true, nil).Maybe()

			cfg := &config.Config{}
			cfg.UseLocked3Phases = true
			confSrv := config.NewService(fakes.NewConfigStorage(t, cfg, config.Factory))

			handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, c, confSrv, nil, testChargerID, nil)
			require.NoError(t, err)

			observe := func(id model.ObservationID, value int) {
				require.NoError(t, handler.HandleObservation(model.Observation{
					ID: id, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
					Timestamp: time.Now(), Value: strconv.Itoa(value),
				}))
			}

			// The sender holds the queued set behind the state report while single phase is set.
			observe(model.ChargerOPState, int(tt.next))
			observe(model.PhaseMode, 1)
			close(release)
			flushReports(t, handler)

			srv.AssertNotCalled(t, "SetPhaseMode", mock.Anything)
		})
	}
}

func TestObservationsHandler_OnlyAnAcceptedModeThreeIsTheAdapters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cached  int
		seenAt  time.Duration
		wantSet int
	}{
		{name: "the echo of a lock the controller already cached", cached: 3, seenAt: time.Minute, wantSet: 1},
		{name: "a mode 3 older than the cached mode", cached: 2, seenAt: -time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			observed := time.Now().Add(-time.Hour)

			c := cache.NewCache(testChargerID)
			c.SetInstallationParameters(types.GridTypeTN, 3, observed)
			c.SetPhaseMode(tt.cached, observed)
			c.SetChargerState(chargepoint.StateCharging, observed)

			srv := mockedchargepoint.NewService(t)
			srv.On("Name").Return(chargepoint.Chargepoint).Maybe()
			srv.On("SendStateReport", false).Return(true, nil).Maybe()
			srv.On("IsPhaseModeAware").Return(true).Maybe()
			srv.On("SendPhaseModeReport", false).Return(true, nil).Maybe()
			srv.On("SetPhaseMode", types.PhaseModeNL1L2L3).Return(nil).Maybe()

			cfg := &config.Config{}
			cfg.UseLocked3Phases = true
			confSrv := config.NewService(fakes.NewConfigStorage(t, cfg, config.Factory))

			handler, err := signalr.NewObservationsHandler(&phaseThing{srv: srv}, c, confSrv, nil, testChargerID, nil)
			require.NoError(t, err)

			observe := func(id model.ObservationID, value int, at time.Time) {
				require.NoError(t, handler.HandleObservation(model.Observation{
					ID: id, ChargerID: testChargerID, DataType: model.ObservationDataTypeInteger,
					Timestamp: at, Value: strconv.Itoa(value),
				}))
			}

			observe(model.PhaseMode, model.EaseePhaseModeLock3, observed.Add(tt.seenAt))
			require.NoError(t, confSrv.SetUseLocked3Phases(false))
			// Set in the Easee app with the setting off.
			observe(model.PhaseMode, model.EaseePhaseModeLock3, time.Now())
			observe(model.ChargerOPState, int(model.ChargerStateDisconnected), time.Now())
			flushReports(t, handler)

			srv.AssertNumberOfCalls(t, "SetPhaseMode", tt.wantSet)
		})
	}
}
