package signalr_test

import (
	"strconv"
	"sync"
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

// The error grid types keep a known grid type with zero phases - TN400VNeutralOnWrongPin maps
// to (TN, 0) and ITGroundConnectedToPin2Or3 to (IT, 0). Discarding those on the phase count
// left grid_type advertising the previous topology; the update has to land, with the
// phase-dependent props cleared rather than the whole write skipped.
func TestObservationsHandler_ZeroPhaseFaultStillUpdatesGridType(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(types.GridTypeIT, now)
	cacheMock.On("Phases").Return(1, now)
	cacheMock.On("SetInstallationParameters", types.GridTypeTN, 0, now).Return(true)

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
		Timestamp: now,
		Value:     strconv.Itoa(int(model.GridTypeErrorTN400VNeutralOnWrongPin)),
	}))

	assert.Equal(t, types.GridTypeTN, srv.Specification().Props[chargepoint.PropertyGridType],
		"the known grid type of the fault must be advertised")
	assert.NotContains(t, srv.Specification().Props, chargepoint.PropertyPhases,
		"zero phases clears the phase count rather than advertising a stale one")
	assert.NotContains(t, srv.Specification().Props, chargepoint.PropertySupportedPhaseModes)
	assert.Equal(t, 1, thing.inclusion)
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
