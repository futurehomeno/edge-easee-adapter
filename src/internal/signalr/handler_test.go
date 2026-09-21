package signalr_test

import (
	"strconv"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

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

// phaseThing exposes a single chargepoint service and counts inclusion reports.
type phaseThing struct {
	adapter.Thing

	srv *mockedchargepoint.Service

	// The inclusion report is published on the sender goroutine, so the counter and the failure
	// budget are both touched off the test goroutine.
	mu        sync.Mutex
	inclusion int
	fail      int
}

func (t *phaseThing) Services(fimptype.ServiceNameT) []adapter.Service {
	return []adapter.Service{t.srv}
}
func (t *phaseThing) Update(...adapter.ThingUpdate) error { return nil }

func (t *phaseThing) SendInclusionReport(bool) (bool, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.inclusion++

	if t.fail > 0 {
		t.fail--

		return false, assert.AnError
	}

	return true, nil
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

	// The props assignment is synchronous; the report and the phase-store write that follows it
	// run on the sender goroutine, so both need the queue flushed first.
	assert.Equal(t, []types.PhaseMode{types.PhaseModeNL3, types.PhaseModeNL1L2L3},
		srv.Specification().Props[chargepoint.PropertySupportedPhaseModes])

	require.NoError(t, handler.HandleObservation(observation))

	flushReports(t, handler)

	assert.Equal(t, types.PhaseModeNL3, store.mode)
	assert.Equal(t, 1, thing.inclusionCount(), "an unchanged phase must not republish the inclusion report")
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

	// The failure now surfaces on the sender goroutine rather than from HandleObservation, so
	// the drain sees no error; what must still hold is that the phase is not persisted, which is
	// what makes the next observation retry the republish.
	require.NoError(t, handler.HandleObservation(observation))

	waitFor(t, func() bool { return thing.inclusionCount() == 1 })
	assert.Equal(t, types.PhaseModeUnknown, store.mode, "a failed republish must not be persisted")

	require.NoError(t, handler.HandleObservation(observation))

	flushReports(t, handler)

	assert.Equal(t, types.PhaseModeNL3, store.mode)
	assert.Equal(t, 2, thing.inclusionCount())
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
