package easee_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/service/alarm"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	"github.com/futurehomeno/cliffhanger/types"
	"github.com/michalkurzeja/go-clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/db"
	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/fakes"
	mockapi "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/api"
	mockedcache "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/cache"
	mockeddb "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/db"
	mockedsignalr "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/signalr"
)

func newTestController(
	t *testing.T,
	manager *mockedsignalr.Manager,
	cacheMock *mockedcache.Cache,
	clientMock *mockapi.Client,
	sessionStorage *mockeddb.ChargingSessionStorage,
	cfg *config.Config,
	persisted ...types.PhaseMode,
) easee.Controller {
	t.Helper()

	var persistedPhase types.PhaseMode
	if len(persisted) > 0 {
		persistedPhase = persisted[0]
	}

	if cfg == nil {
		cfg = &config.Config{}
	}

	storage := fakes.NewConfigStorage(t, cfg, config.Factory)
	cfgService := config.NewService(storage)

	return easee.NewController(
		manager,
		clientMock,
		"test-charger",
		cacheMock,
		cfgService,
		sessionStorage,
		func() types.PhaseMode { return persistedPhase },
	)
}

// The channel tests drive the global clock, so they cannot run in parallel. A fired deadline
// runs on its own goroutine; each test awaits it through done, signalled from the last mock
// call of that path.
func awaitDeferred(t *testing.T, done <-chan struct{}) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("the deferred send did not run")
	}
}

func signal(done chan<- struct{}) func(mock.Arguments) {
	return func(mock.Arguments) { done <- struct{}{} }
}

// Replays the customer log: a write every 5s. One send opens the window, everything inside it
// is stored, and the deadline sends the latest value once - three writes on the wire for ten
// commands, none closer than the wait time.
func TestController_OfferedCurrentChannel_PacesWrites(t *testing.T) {
	clk := clock.Mock(time.Date(2026, time.September, 20, 1, 0, 0, 0, time.UTC))
	t.Cleanup(clock.Restore)

	done := make(chan struct{}, 1)

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("MaxCurrent").Return(32, time.Time{})
	cacheMock.On("RequestedOfferedCurrent").Return(0, time.Time{})
	cacheMock.On("SetRequestedOfferedCurrent", mock.Anything, mock.AnythingOfType("time.Time")).Return(true)
	cacheMock.On("WaitForOfferedCurrent", 10, mock.Anything).Return(true).Once()
	cacheMock.On("WaitForOfferedCurrent", 18, mock.Anything).Return(true).Run(signal(done)).Once()
	cacheMock.On("WaitForOfferedCurrent", 19, mock.Anything).Return(true).Run(signal(done)).Once()

	clientMock := mockapi.NewClient(t)
	clientMock.On("UpdateDynamicCurrent", "test-charger", float64(10)).Return(nil).Once()
	clientMock.On("UpdateDynamicCurrent", "test-charger", float64(18)).Return(nil).Once()
	clientMock.On("UpdateDynamicCurrent", "test-charger", float64(19)).Return(nil).Once()

	ctrl := newTestController(t, nil, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

	require.NoError(t, ctrl.SetChargepointOfferedCurrent(10))
	clientMock.AssertNumberOfCalls(t, "UpdateDynamicCurrent", 1)

	for current := 11; current <= 18; current++ {
		clk.Add(5 * time.Second)
		require.NoError(t, ctrl.SetChargepointOfferedCurrent(current))
	}

	clk.Add(4 * time.Second) // t0+44s
	clientMock.AssertNumberOfCalls(t, "UpdateDynamicCurrent", 1)

	clk.Add(time.Second) // t0+45s: the deadline the first send opened
	awaitDeferred(t, done)
	clientMock.AssertNumberOfCalls(t, "UpdateDynamicCurrent", 2)
	clientMock.AssertCalled(t, "UpdateDynamicCurrent", "test-charger", float64(18))

	clk.Add(5 * time.Second) // t0+50s: inside the window the deferred send opened
	require.NoError(t, ctrl.SetChargepointOfferedCurrent(19))
	clientMock.AssertNumberOfCalls(t, "UpdateDynamicCurrent", 2)

	clk.Add(40 * time.Second) // t0+90s
	awaitDeferred(t, done)
	clientMock.AssertNumberOfCalls(t, "UpdateDynamicCurrent", 3)
}

// A write exactly offered_current_wait_time after the last send is on a quiet channel and goes
// out synchronously; one inside the window that send opened is stored.
func TestController_OfferedCurrentChannel_SendsAfterTheWindow(t *testing.T) {
	clk := clock.Mock(time.Date(2026, time.September, 20, 1, 0, 0, 0, time.UTC))
	t.Cleanup(clock.Restore)

	done := make(chan struct{}, 1)

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("MaxCurrent").Return(32, time.Time{})
	cacheMock.On("RequestedOfferedCurrent").Return(0, time.Time{})
	cacheMock.On("SetRequestedOfferedCurrent", mock.Anything, mock.AnythingOfType("time.Time")).Return(true)
	cacheMock.On("WaitForOfferedCurrent", 10, mock.Anything).Return(true).Once()
	cacheMock.On("WaitForOfferedCurrent", 11, mock.Anything).Return(true).Once()
	cacheMock.On("WaitForOfferedCurrent", 12, mock.Anything).Return(true).Run(signal(done)).Once()

	clientMock := mockapi.NewClient(t)
	clientMock.On("UpdateDynamicCurrent", "test-charger", mock.Anything).Return(nil)

	ctrl := newTestController(t, nil, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

	require.NoError(t, ctrl.SetChargepointOfferedCurrent(10))

	clk.Add(20 * time.Second)
	require.NoError(t, ctrl.SetChargepointOfferedCurrent(11))
	clientMock.AssertNumberOfCalls(t, "UpdateDynamicCurrent", 2)

	clk.Add(19 * time.Second)
	require.NoError(t, ctrl.SetChargepointOfferedCurrent(12))
	clientMock.AssertNumberOfCalls(t, "UpdateDynamicCurrent", 2)

	clk.Add(26 * time.Second) // 45s after the second send
	awaitDeferred(t, done)
	clientMock.AssertNumberOfCalls(t, "UpdateDynamicCurrent", 3)
}

// Easee zeroes the dynamic current on a stop and ignores a write that follows it too closely,
// so a stop opens the window like any other send.
func TestController_StopChargepointCharging_StampsTheChannel(t *testing.T) {
	clk := clock.Mock(time.Date(2026, time.September, 20, 1, 0, 0, 0, time.UTC))
	t.Cleanup(clock.Restore)

	done := make(chan struct{}, 1)

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("MaxCurrent").Return(32, time.Time{})
	cacheMock.On("RequestedOfferedCurrent").Return(0, time.Time{})
	cacheMock.On("SetRequestedOfferedCurrent", 10, mock.AnythingOfType("time.Time")).Return(true)
	cacheMock.On("WaitForOfferedCurrent", 10, mock.Anything).Return(true).Run(signal(done)).Once()

	clientMock := mockapi.NewClient(t)
	clientMock.On("StopCharging", "test-charger").Return(nil).Once()
	clientMock.On("UpdateDynamicCurrent", "test-charger", float64(10)).Return(nil).Once()

	ctrl := newTestController(t, nil, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

	require.NoError(t, ctrl.StopChargepointCharging())
	clientMock.AssertCalled(t, "StopCharging", "test-charger")

	clk.Add(time.Second)
	require.NoError(t, ctrl.SetChargepointOfferedCurrent(10))
	clientMock.AssertNotCalled(t, "UpdateDynamicCurrent", mock.Anything, mock.Anything)

	clk.Add(44 * time.Second)
	awaitDeferred(t, done)
	clientMock.AssertCalled(t, "UpdateDynamicCurrent", "test-charger", float64(10))
}

// Latest wins across kinds: a start behind a stored stop replaces it, so the charger is never
// paused, and the start reports success although its write goes out at the deadline.
func TestController_StopChargepointCharging_IsReplacedByAStart(t *testing.T) {
	clk := clock.Mock(time.Date(2026, time.September, 20, 1, 0, 0, 0, time.UTC))
	t.Cleanup(clock.Restore)

	done := make(chan struct{}, 1)

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("MaxCurrent").Return(16, time.Time{})
	cacheMock.On("RequestedOfferedCurrent").Return(0, time.Time{})
	cacheMock.On("SetRequestedOfferedCurrent", mock.Anything, mock.AnythingOfType("time.Time")).Return(true)
	cacheMock.On("WaitForOfferedCurrent", 10, mock.Anything).Return(true).Once()
	cacheMock.On("WaitForOfferedCurrent", 16, mock.Anything).Return(true).Run(signal(done)).Once()

	clientMock := mockapi.NewClient(t)
	clientMock.On("UpdateDynamicCurrent", "test-charger", float64(10)).Return(nil).Once()
	clientMock.On("UpdateDynamicCurrent", "test-charger", float64(16)).Return(nil).Once()

	ctrl := newTestController(t, nil, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

	require.NoError(t, ctrl.SetChargepointOfferedCurrent(10))

	clk.Add(5 * time.Second)
	require.NoError(t, ctrl.StopChargepointCharging())
	clientMock.AssertNotCalled(t, "StopCharging", mock.Anything)

	clk.Add(5 * time.Second)
	require.NoError(t, ctrl.StartChargepointCharging(&chargepoint.ChargingSettings{Mode: model.ChargingModeNormal}))

	clk.Add(35 * time.Second)
	awaitDeferred(t, done)
	clientMock.AssertNumberOfCalls(t, "UpdateDynamicCurrent", 2)
	clientMock.AssertNotCalled(t, "StopCharging", mock.Anything)
}

func TestController_SetChargepointOfferedCurrent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		inputCurrent    int
		maxCurrent      int // 0 means cache is empty/unknown
		expectedCurrent float64
	}{
		{
			name:            "current within limit is sent as-is",
			inputCurrent:    16,
			maxCurrent:      32,
			expectedCurrent: 16,
		},
		{
			name:            "current equal to limit is sent as-is",
			inputCurrent:    32,
			maxCurrent:      32,
			expectedCurrent: 32,
		},
		{
			name:            "current exceeding known max is clamped to max",
			inputCurrent:    36,
			maxCurrent:      32,
			expectedCurrent: 32,
		},
		{
			name:            "current exceeding lower charger max is clamped",
			inputCurrent:    20,
			maxCurrent:      16,
			expectedCurrent: 16,
		},
		{
			name:            "current exceeding hard limit when cache empty is clamped to 32",
			inputCurrent:    40,
			maxCurrent:      0,
			expectedCurrent: 32,
		},
		{
			name:            "current within hard limit when cache empty is sent as-is",
			inputCurrent:    20,
			maxCurrent:      0,
			expectedCurrent: 20,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cacheMock := mockedcache.NewCache(t)
			clientMock := mockapi.NewClient(t)

			cacheMock.On("MaxCurrent").Return(tt.maxCurrent, time.Time{})
			cacheMock.On("RequestedOfferedCurrent").Return(0, time.Time{})
			clientMock.On("UpdateDynamicCurrent", "test-charger", tt.expectedCurrent).Return(nil)
			cacheMock.On("SetRequestedOfferedCurrent", int(tt.expectedCurrent), mock.AnythingOfType("time.Time")).Return(true)
			cacheMock.On("WaitForOfferedCurrent", int(tt.expectedCurrent), mock.AnythingOfType("time.Duration")).Return(true)

			ctrl := newTestController(t, nil, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

			err := ctrl.SetChargepointOfferedCurrent(tt.inputCurrent)

			assert.NoError(t, err)
			clientMock.AssertCalled(t, "UpdateDynamicCurrent", "test-charger", tt.expectedCurrent)
		})
	}
}

func TestController_StartChargepointCharging(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		settings         *chargepoint.ChargingSettings
		maxCurrent       int
		requestedCurrent int
		slowCurrent      float64
		startCurrent     int
		expectedCurrent  float64
		wantErr          bool
	}{
		{
			name:             "uses maxCurrent when no requested offered current",
			settings:         &chargepoint.ChargingSettings{Mode: model.ChargingModeNormal},
			maxCurrent:       32,
			requestedCurrent: 0,
			expectedCurrent:  32,
		},
		{
			name:             "raises a low requested offered current to the start current",
			settings:         &chargepoint.ChargingSettings{Mode: model.ChargingModeNormal},
			maxCurrent:       32,
			requestedCurrent: 8,
			expectedCurrent:  16,
		},
		{
			name:             "start current is capped by maxCurrent",
			settings:         &chargepoint.ChargingSettings{Mode: model.ChargingModeNormal},
			maxCurrent:       10,
			requestedCurrent: 8,
			expectedCurrent:  10,
		},
		{
			name:             "configured start current overrides the default",
			settings:         &chargepoint.ChargingSettings{Mode: model.ChargingModeNormal},
			maxCurrent:       32,
			requestedCurrent: 8,
			startCurrent:     20,
			expectedCurrent:  20,
		},
		{
			name:             "uses requestedOfferedCurrent over maxCurrent",
			settings:         &chargepoint.ChargingSettings{Mode: model.ChargingModeNormal},
			maxCurrent:       32,
			requestedCurrent: 20,
			expectedCurrent:  20,
		},
		{
			name:             "slow mode uses configured slow current",
			settings:         &chargepoint.ChargingSettings{Mode: model.ChargingModeSlow},
			maxCurrent:       32,
			requestedCurrent: 20,
			slowCurrent:      8,
			expectedCurrent:  8,
		},
		{
			name:             "slow mode falls back to requestedOfferedCurrent when slow current not configured",
			settings:         &chargepoint.ChargingSettings{Mode: model.ChargingModeSlow},
			maxCurrent:       32,
			requestedCurrent: 20,
			slowCurrent:      0,
			expectedCurrent:  20,
		},
		{
			name:             "slow mode is not raised to the start current when slow current not configured",
			settings:         &chargepoint.ChargingSettings{Mode: model.ChargingModeSlow},
			maxCurrent:       32,
			requestedCurrent: 8,
			slowCurrent:      0,
			expectedCurrent:  8,
		},
		{
			name:             "configured slow current wins over the start current",
			settings:         &chargepoint.ChargingSettings{Mode: model.ChargingModeSlow},
			maxCurrent:       32,
			requestedCurrent: 8,
			slowCurrent:      10,
			expectedCurrent:  10,
		},
		{
			name:             "returns error when all current sources are zero",
			settings:         &chargepoint.ChargingSettings{Mode: model.ChargingModeNormal},
			maxCurrent:       0,
			requestedCurrent: 0,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cacheMock := mockedcache.NewCache(t)
			clientMock := mockapi.NewClient(t)

			cacheMock.On("MaxCurrent").Return(tt.maxCurrent, time.Time{})
			cacheMock.On("RequestedOfferedCurrent").Return(tt.requestedCurrent, time.Time{})

			if !tt.wantErr {
				clientMock.On("UpdateDynamicCurrent", "test-charger", tt.expectedCurrent).Return(nil)
				cacheMock.On("SetRequestedOfferedCurrent", int(tt.expectedCurrent), mock.AnythingOfType("time.Time")).Return(true)
				cacheMock.On("WaitForOfferedCurrent", int(tt.expectedCurrent), mock.AnythingOfType("time.Duration")).Return(true)
			}

			cfg := &config.Config{}
			cfg.SlowChargingCurrentInAmperes = tt.slowCurrent
			cfg.InitialChargingCurrent = tt.startCurrent

			ctrl := newTestController(t, nil, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), cfg)

			err := ctrl.StartChargepointCharging(tt.settings)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			clientMock.AssertCalled(t, "UpdateDynamicCurrent", "test-charger", tt.expectedCurrent)
		})
	}
}

// Regression: pressing Start within OfferedCurrentWaitTime of a previous identical request
// must NOT be silently suppressed by the dedup gate. The cache is only cleared async by the
// SignalR session-finished observation; if a user re-presses Start before that lands, dedup
// would otherwise drop the UpdateDynamicCurrent and leave the charger stopped.
func TestController_StartChargepointCharging_BypassesOfferedCurrentDedup(t *testing.T) {
	t.Parallel()

	cacheMock := mockedcache.NewCache(t)
	clientMock := mockapi.NewClient(t)

	// Cache reports a recent identical offered-current request - exactly the state where
	// the buggy dedup would fire (lastValue == startCurrent && time.Since(lastSet) < wait).
	cacheMock.On("MaxCurrent").Return(16, time.Time{})
	cacheMock.On("RequestedOfferedCurrent").Return(16, time.Now())
	clientMock.On("UpdateDynamicCurrent", "test-charger", float64(16)).Return(nil).Once()
	cacheMock.On("SetRequestedOfferedCurrent", 16, mock.AnythingOfType("time.Time")).Return(true)
	cacheMock.On("WaitForOfferedCurrent", 16, mock.AnythingOfType("time.Duration")).Return(true)

	cfg := &config.Config{PublicConfig: config.PublicConfig{OfferedCurrentWaitTime: "15s"}}

	ctrl := newTestController(t, nil, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), cfg)

	err := ctrl.StartChargepointCharging(&chargepoint.ChargingSettings{Mode: model.ChargingModeNormal})
	assert.NoError(t, err)
	clientMock.AssertCalled(t, "UpdateDynamicCurrent", "test-charger", float64(16))
}

// SetChargepointOfferedCurrent (the non-Start path) must keep the existing dedup so that
// rapid in-progress current adjustments to the same value don't hammer the API.
func TestController_SetChargepointOfferedCurrent_KeepsDedup(t *testing.T) {
	t.Parallel()

	cacheMock := mockedcache.NewCache(t)
	clientMock := mockapi.NewClient(t)

	cacheMock.On("MaxCurrent").Return(16, time.Time{})
	cacheMock.On("RequestedOfferedCurrent").Return(16, time.Now())

	cfg := &config.Config{PublicConfig: config.PublicConfig{OfferedCurrentWaitTime: "15s"}}

	ctrl := newTestController(t, nil, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), cfg)

	err := ctrl.SetChargepointOfferedCurrent(16)
	assert.NoError(t, err)
	clientMock.AssertNotCalled(t, "UpdateDynamicCurrent", mock.Anything, mock.Anything)
}

func TestController_SetChargepointPhaseMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mode          types.PhaseMode
		cachedMode    int
		chargerState  chargepoint.State
		expectedEasee int
		expectNoCall  bool
		expectRecord  bool
		wantErr       bool
	}{
		{
			name:          "single phase locks the charger to one phase",
			mode:          types.PhaseModeNL1,
			cachedMode:    2,
			chargerState:  chargepoint.StateReadyToCharge,
			expectedEasee: 1,
		},
		{
			name:          "three phase maps to auto, so an EV that cannot go 1->3 is not stranded",
			mode:          types.PhaseModeNL1L2L3,
			cachedMode:    1,
			chargerState:  chargepoint.StateReadyToCharge,
			expectedEasee: 2,
		},
		{
			name:         "already in the target mode, no API call and no session interruption",
			mode:         types.PhaseModeNL1,
			cachedMode:   1,
			expectNoCall: true,
			expectRecord: true,
		},
		{
			name: "another leg is the same Easee mode 1, so it is recorded rather than dropped",
			// Easee mode 1 means "one phase", not a chosen leg: without recording the request
			// the report that follows republishes the old leg and the UI reverts the choice.
			mode:         types.PhaseModeNL2,
			cachedMode:   1,
			expectNoCall: true,
			expectRecord: true,
		},
		{
			name:         "auto is already the target, and no leg observation is pending",
			mode:         types.PhaseModeNL1L2L3,
			cachedMode:   2,
			expectRecord: true,
			expectNoCall: true,
		},
		{
			name:       "mode outside the grid's capabilities",
			mode:       types.PhaseModeL1L2,
			cachedMode: 2,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			managerMock := mockedsignalr.NewManager(t)
			managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

			cacheMock := mockedcache.NewCache(t)
			cacheMock.On("GridType").Return(types.GridTypeTN, time.Time{})
			cacheMock.On("Phases").Return(3, time.Time{})

			clientMock := mockapi.NewClient(t)

			if !tt.wantErr {
				cacheMock.On("PhaseMode").Return(tt.cachedMode, time.Time{})
			}

			if tt.expectRecord {
				// No live observation of the leg in use, so the request is free to stand in.
				cacheMock.On("OutputPhaseType").Return(types.PhaseMode(""), time.Time{})
				cacheMock.On("SetRequestedPhaseMode", tt.mode, mock.AnythingOfType("time.Time")).Return(true)
			}

			if !tt.wantErr && !tt.expectNoCall {
				cacheMock.On("OutputPhaseType").Return(types.PhaseMode(""), time.Time{})
				cacheMock.On("SetRequestedPhaseMode", tt.mode, mock.AnythingOfType("time.Time")).Return(true)
				clientMock.On("SetPhaseMode", "test-charger", tt.expectedEasee).Return(nil).Once()
				cacheMock.On("TotalPower").Return(float64(0), time.Time{})
				cacheMock.On("ChargerState").Return(tt.chargerState, time.Time{})
			}

			ctrl := newTestController(t, managerMock, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

			err := ctrl.SetChargepointPhaseMode(tt.mode)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)

			if tt.expectNoCall {
				clientMock.AssertNotCalled(t, "SetPhaseMode", mock.Anything, mock.Anything)
			}

			if tt.expectNoCall && !tt.expectRecord {
				cacheMock.AssertNotCalled(t, "SetRequestedPhaseMode", mock.Anything, mock.Anything)
			}
		})
	}
}

// Resuming needs a current to resume at. With both cached values empty - the adapter
// restarted mid-session and the state observation landed before the current ones - the
// session must not be "resumed" at 0A, which reads as success while the charger stays paused.
func TestController_SetChargepointPhaseMode_FailsWhenNoResumeCurrentIsKnown(t *testing.T) {
	t.Parallel()

	managerMock := mockedsignalr.NewManager(t)
	managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(types.GridTypeTN, time.Time{})
	cacheMock.On("Phases").Return(3, time.Time{})
	cacheMock.On("PhaseMode").Return(2, time.Time{})
	cacheMock.On("OutputPhaseType").Return(types.PhaseMode(""), time.Time{})
	cacheMock.On("SetRequestedPhaseMode", types.PhaseModeNL1, mock.AnythingOfType("time.Time")).Return(true)
	cacheMock.On("TotalPower").Return(3000.0, time.Time{})
	cacheMock.On("MaxCurrent").Return(0, time.Time{})
	cacheMock.On("RequestedOfferedCurrent").Return(0, time.Time{})
	cacheMock.On("OfferedCurrent").Return(0, time.Time{})

	clientMock := mockapi.NewClient(t)
	clientMock.On("SetPhaseMode", "test-charger", 1).Return(nil).Once()

	ctrl := newTestController(t, managerMock, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

	assert.Error(t, ctrl.SetChargepointPhaseMode(types.PhaseModeNL1))
	clientMock.AssertNotCalled(t, "StopCharging", "test-charger")
	clientMock.AssertNotCalled(t, "UpdateDynamicCurrent", mock.Anything, mock.Anything)
}

// outputPhase survives as a stale echo of the previous session, so a mode we just
// requested has to win until the charger reports a newer one.
func TestController_ChargepointPhaseModeReport_PrefersRequestedMode(t *testing.T) {
	t.Parallel()

	managerMock := mockedsignalr.NewManager(t)
	managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("OutputPhaseType").Return(types.PhaseModeNL3, time.Now().Add(-time.Hour))
	cacheMock.On("RequestedPhaseMode").Return(types.PhaseModeNL1L2L3, time.Now())
	cacheMock.On("PhaseMode").Return(2, time.Time{})

	ctrl := newTestController(t, managerMock, cacheMock, mockapi.NewClient(t), mockeddb.NewChargingSessionStorage(t), nil)

	mode, err := ctrl.ChargepointPhaseModeReport()
	assert.NoError(t, err)
	assert.Equal(t, types.PhaseModeNL1L2L3, mode)
}

// Once the charger reports an output phase of its own, that observation is the truth.
func TestController_ChargepointPhaseModeReport_OutputPhaseWinsWhenNewer(t *testing.T) {
	t.Parallel()

	managerMock := mockedsignalr.NewManager(t)
	managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("OutputPhaseType").Return(types.PhaseModeNL3, time.Now())
	cacheMock.On("RequestedPhaseMode").Return(types.PhaseModeNL1L2L3, time.Now().Add(-time.Hour))
	cacheMock.On("PhaseMode").Return(2, time.Now().Add(-time.Hour))

	ctrl := newTestController(t, managerMock, cacheMock, mockapi.NewClient(t), mockeddb.NewChargingSessionStorage(t), nil)

	mode, err := ctrl.ChargepointPhaseModeReport()
	assert.NoError(t, err)
	assert.Equal(t, types.PhaseModeNL3, mode)
}

// Nothing ever clears a requested mode, so an internal phase mode changed elsewhere - in the
// Easee app - has to void it, or the adapter keeps publishing a leg the charger left behind.
// The charger also echoes back the mode we just set, always newer than the request, and that
// confirmation must not be mistaken for such a change.
func TestController_ChargepointPhaseModeReport_RequestedModeVsInternalMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		internal int
		want     types.PhaseMode
	}{
		{name: "external change voids the request", internal: 2, want: types.PhaseModeNL3},
		{name: "our own echo keeps it", internal: 1, want: types.PhaseModeNL1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			managerMock := mockedsignalr.NewManager(t)
			managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

			cacheMock := mockedcache.NewCache(t)
			cacheMock.On("OutputPhaseType").Return(types.PhaseModeNL3, time.Now().Add(-time.Hour))
			cacheMock.On("RequestedPhaseMode").Return(types.PhaseModeNL1, time.Now().Add(-30*time.Minute))
			cacheMock.On("PhaseMode").Return(tt.internal, time.Now())
			cacheMock.On("GridType").Return(types.GridTypeTN, time.Time{})
			cacheMock.On("Phases").Return(3, time.Time{})

			ctrl := newTestController(t, managerMock, cacheMock, mockapi.NewClient(t), mockeddb.NewChargingSessionStorage(t), nil)

			mode, err := ctrl.ChargepointPhaseModeReport()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, mode)
		})
	}
}

// Nothing ever clears outputPhase either, so an internal mode changed elsewhere while the
// charger sits idle has to void the leg of the previous session - observation 38 now triggers
// a report, so a stale leg would be actively republished rather than merely served on request.
func TestController_ChargepointPhaseModeReport_VoidsStaleOutputPhase(t *testing.T) {
	t.Parallel()

	managerMock := mockedsignalr.NewManager(t)
	managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("OutputPhaseType").Return(types.PhaseModeNL1, time.Now().Add(-time.Hour))
	cacheMock.On("RequestedPhaseMode").Return(types.PhaseMode(""), time.Time{})
	cacheMock.On("PhaseMode").Return(3, time.Now())
	cacheMock.On("GridType").Return(types.GridTypeTN, time.Time{})
	cacheMock.On("Phases").Return(3, time.Time{})
	cacheMock.On("TotalPower").Return(float64(0), time.Time{})
	cacheMock.On("ChargerState").Return(chargepoint.StateReadyToCharge, time.Now())

	clientMock := mockapi.NewClient(t)
	clientMock.On("ChargerConfig", "test-charger").
		Return(&model.ChargerConfig{DetectedPowerGridType: model.GridTypeTN3Phase, PhaseMode: 3}, nil)
	clientMock.On("ChargerSiteInfo", "test-charger").Return(&model.ChargerSiteInfo{RatedCurrent: 32}, nil)

	ctrl := newTestController(t, managerMock, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

	mode, err := ctrl.ChargepointPhaseModeReport()
	assert.NoError(t, err)
	assert.Equal(t, types.PhaseModeNL1L2L3, mode)
}

// After a restart the cache holds no output phase, so the report falls back to the cloud
// config. In single mode that used to mean modes[0] (NL1) while the inclusion report already
// advertised the persisted phase, leaving the guard with a mode outside sup_phase_modes.
func TestController_ChargepointPhaseModeReport_UsesPersistedPhaseAfterRestart(t *testing.T) {
	t.Parallel()

	managerMock := mockedsignalr.NewManager(t)
	managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("OutputPhaseType").Return(types.PhaseMode(""), time.Time{})
	cacheMock.On("RequestedPhaseMode").Return(types.PhaseMode(""), time.Time{})

	clientMock := mockapi.NewClient(t)
	clientMock.On("ChargerConfig", "test-charger").
		Return(&model.ChargerConfig{DetectedPowerGridType: model.GridTypeTN3Phase, PhaseMode: 1}, nil)
	clientMock.On("ChargerSiteInfo", "test-charger").Return(&model.ChargerSiteInfo{RatedCurrent: 32}, nil)

	ctrl := newTestController(t, managerMock, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil, types.PhaseModeNL3)

	mode, err := ctrl.ChargepointPhaseModeReport()
	assert.NoError(t, err)
	assert.Equal(t, types.PhaseModeNL3, mode)
}

// The charger applies a new phase mode only at a session boundary, so a mode changed mid-session
// must not pre-empt the leg the charger is actually drawing on.
func TestController_ChargepointPhaseModeReport_KeepsLiveOutputPhaseWhileCharging(t *testing.T) {
	t.Parallel()

	managerMock := mockedsignalr.NewManager(t)
	managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("OutputPhaseType").Return(types.PhaseModeNL1, time.Now().Add(-time.Hour))
	cacheMock.On("RequestedPhaseMode").Return(types.PhaseMode(""), time.Time{})
	cacheMock.On("PhaseMode").Return(3, time.Now())
	cacheMock.On("GridType").Return(types.GridTypeTN, time.Time{})
	cacheMock.On("Phases").Return(3, time.Time{})
	cacheMock.On("TotalPower").Return(float64(7000), time.Now())

	ctrl := newTestController(t, managerMock, cacheMock, mockapi.NewClient(t), mockeddb.NewChargingSessionStorage(t), nil)

	mode, err := ctrl.ChargepointPhaseModeReport()
	assert.NoError(t, err)
	assert.Equal(t, types.PhaseModeNL1, mode)
}

// A no-op request must not stamp the cache: doing so would outrank a live outputPhase
// observation and report a leg the charger is not actually on.
func TestController_SetChargepointPhaseMode_NoOpDoesNotMaskOutputPhase(t *testing.T) {
	t.Parallel()

	managerMock := mockedsignalr.NewManager(t)
	managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(types.GridTypeTN, time.Time{})
	cacheMock.On("Phases").Return(3, time.Time{})
	// Already locked to a single phase, so asking for another leg changes nothing.
	cacheMock.On("PhaseMode").Return(1, time.Time{})
	// A live observation names the leg actually in use, and the charger - not the request -
	// picks it, so recording NL2 here would report a leg the charger is not on. Only a
	// charging charger is on a leg at all, which is what makes the observation live.
	cacheMock.On("OutputPhaseType").Return(types.PhaseModeNL1, time.Now())
	cacheMock.On("TotalPower").Return(7000.0, time.Time{})

	ctrl := newTestController(t, managerMock, cacheMock, mockapi.NewClient(t), mockeddb.NewChargingSessionStorage(t), nil)

	assert.NoError(t, ctrl.SetChargepointPhaseMode(types.PhaseModeNL2))
	cacheMock.AssertNotCalled(t, "SetRequestedPhaseMode", mock.Anything, mock.Anything)
}

// An idle charger holds the leg of a finished session, and no internal mode change has
// happened since - so outputPhaseStale still calls it fresh. Dropping the request there left
// NL1 -> NL2 unrecorded, and the report republished NL1 until the next session started.
func TestController_SetChargepointPhaseMode_IdleChargerRecordsRequestOverStaleLeg(t *testing.T) {
	t.Parallel()

	managerMock := mockedsignalr.NewManager(t)
	managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

	ended := time.Now().Add(-2 * time.Hour)

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(types.GridTypeTN, time.Time{})
	cacheMock.On("Phases").Return(3, time.Time{})
	// Locked to a single phase, and the leg was cached when the session ended - nothing has
	// moved the internal mode since, so the staleness check alone keeps NL1.
	cacheMock.On("PhaseMode").Return(1, ended)
	cacheMock.On("OutputPhaseType").Return(types.PhaseModeNL1, ended)
	cacheMock.On("TotalPower").Return(0.0, time.Time{})
	cacheMock.On("ChargerState").Return(chargepoint.StateReadyToCharge, time.Time{})
	cacheMock.On("SetRequestedPhaseMode", types.PhaseModeNL2, mock.AnythingOfType("time.Time")).Return(true).Once()

	ctrl := newTestController(t, managerMock, cacheMock, mockapi.NewClient(t), mockeddb.NewChargingSessionStorage(t), nil)

	assert.NoError(t, ctrl.SetChargepointPhaseMode(types.PhaseModeNL2))
}

// chargingCharger mocks a charging charger on a three-phase TN grid in auto mode, the state
// every bounce test starts from, with the three currents the resume can be read from.
func chargingCharger(t *testing.T, requested, offered, maxCurrent int) (*mockedsignalr.Manager, *mockedcache.Cache) {
	t.Helper()

	managerMock := mockedsignalr.NewManager(t)
	managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(types.GridTypeTN, time.Time{})
	cacheMock.On("Phases").Return(3, time.Time{})
	cacheMock.On("PhaseMode").Return(2, time.Time{})
	cacheMock.On("OutputPhaseType").Return(types.PhaseMode(""), time.Time{})
	cacheMock.On("SetRequestedPhaseMode", types.PhaseModeNL1, mock.AnythingOfType("time.Time")).Return(true)
	cacheMock.On("TotalPower").Return(3000.0, time.Time{})
	cacheMock.On("MaxCurrent").Return(maxCurrent, time.Time{})
	cacheMock.On("RequestedOfferedCurrent").Return(requested, time.Time{})
	cacheMock.On("OfferedCurrent").Return(offered, time.Time{}).Maybe()

	return managerMock, cacheMock
}

// The defect from the customer log: the resume used to follow the pause 10 ms later and Easee
// ignored it, parking the charger at 0A while the adapter reported success. The pause opens
// the window, so the resume is stored and sent at the deadline; the command succeeds once
// the pause is accepted.
func TestController_SetChargepointPhaseMode_DefersTheResume(t *testing.T) {
	t0 := time.Date(2026, time.September, 20, 1, 0, 0, 0, time.UTC)
	clk := clock.Mock(t0)
	t.Cleanup(clock.Restore)

	done := make(chan struct{}, 1)

	managerMock, cacheMock := chargingCharger(t, 16, 0, 16)
	cacheMock.On("SetRequestedOfferedCurrent", 16, t0.Add(45*time.Second)).Return(true).Once()
	cacheMock.On("WaitForOfferedCurrent", 16, mock.Anything).Return(true).Run(signal(done)).Once()

	clientMock := mockapi.NewClient(t)
	clientMock.On("SetPhaseMode", "test-charger", 1).Return(nil).Once()
	clientMock.On("StopCharging", "test-charger").Return(nil).Once()
	clientMock.On("UpdateDynamicCurrent", "test-charger", float64(16)).Return(nil).Once()

	ctrl := newTestController(t, managerMock, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

	require.NoError(t, ctrl.SetChargepointPhaseMode(types.PhaseModeNL1))
	clientMock.AssertCalled(t, "StopCharging", "test-charger")
	clientMock.AssertNotCalled(t, "UpdateDynamicCurrent", mock.Anything, mock.Anything)

	clk.Add(45 * time.Second)
	awaitDeferred(t, done)
	clientMock.AssertNumberOfCalls(t, "UpdateDynamicCurrent", 1)
	cacheMock.AssertCalled(t, "SetRequestedOfferedCurrent", 16, t0.Add(45*time.Second))
}

// Bouncing a session must put back the current that was running. Resuming through a
// normal-mode start floored it to initial_charging_current, silently pushing a 6A slow session
// to 16A; and after an adapter restart only the charger's own observation still describes the
// session, so falling through to the user's maximum would raise it to 32A.
func TestController_SetChargepointPhaseMode_ResumesAtTheSessionCurrent(t *testing.T) {
	tests := []struct {
		name      string
		requested int
		offered   int
	}{
		{name: "the requested current", requested: 6},
		{name: "the charger's own current after a restart", offered: 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clk := clock.Mock(time.Date(2026, time.September, 20, 1, 0, 0, 0, time.UTC))
			t.Cleanup(clock.Restore)

			done := make(chan struct{}, 1)

			managerMock, cacheMock := chargingCharger(t, tt.requested, tt.offered, 32)
			cacheMock.On("SetRequestedOfferedCurrent", 6, mock.AnythingOfType("time.Time")).Return(true).Once()
			cacheMock.On("WaitForOfferedCurrent", 6, mock.Anything).Return(true).Run(signal(done)).Once()

			clientMock := mockapi.NewClient(t)
			clientMock.On("SetPhaseMode", "test-charger", 1).Return(nil).Once()
			clientMock.On("StopCharging", "test-charger").Return(nil).Once()
			clientMock.On("UpdateDynamicCurrent", "test-charger", float64(6)).Return(nil).Once()

			ctrl := newTestController(t, managerMock, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), &config.Config{
				PublicConfig: config.PublicConfig{InitialChargingCurrent: 16},
			})

			require.NoError(t, ctrl.SetChargepointPhaseMode(types.PhaseModeNL1))

			clk.Add(45 * time.Second)
			awaitDeferred(t, done)
			clientMock.AssertCalled(t, "UpdateDynamicCurrent", "test-charger", float64(6))
			clientMock.AssertNotCalled(t, "UpdateDynamicCurrent", "test-charger", float64(16))
			clientMock.AssertNotCalled(t, "UpdateDynamicCurrent", "test-charger", float64(32))
		})
	}
}

// The command answered when the resume was stored, so whatever the deferred send meets -
// Easee refusing it, or the charger never echoing it back - can only be logged.
func TestController_SetChargepointPhaseMode_DeferredResumeOutcomesAreLogOnly(t *testing.T) {
	tests := []struct {
		name    string
		refused error
		echoed  bool
	}{
		{name: "refused", refused: errors.New("cloud rejected the resume")},
		{name: "unconfirmed", echoed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clk := clock.Mock(time.Date(2026, time.September, 20, 1, 0, 0, 0, time.UTC))
			t.Cleanup(clock.Restore)

			done := make(chan struct{}, 1)

			managerMock, cacheMock := chargingCharger(t, 16, 0, 16)

			clientMock := mockapi.NewClient(t)
			clientMock.On("SetPhaseMode", "test-charger", 1).Return(nil).Once()
			clientMock.On("StopCharging", "test-charger").Return(nil).Once()

			if tt.refused != nil {
				clientMock.On("UpdateDynamicCurrent", "test-charger", float64(16)).Return(tt.refused).Run(signal(done)).Once()
			} else {
				clientMock.On("UpdateDynamicCurrent", "test-charger", float64(16)).Return(nil).Once()
				cacheMock.On("SetRequestedOfferedCurrent", 16, mock.AnythingOfType("time.Time")).Return(true).Once()
				cacheMock.On("WaitForOfferedCurrent", 16, mock.Anything).Return(tt.echoed).Run(signal(done)).Once()
			}

			ctrl := newTestController(t, managerMock, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

			require.NoError(t, ctrl.SetChargepointPhaseMode(types.PhaseModeNL1))

			clk.Add(45 * time.Second)
			awaitDeferred(t, done)
			clientMock.AssertNumberOfCalls(t, "UpdateDynamicCurrent", 1)
		})
	}
}

// Failing to pause changes nothing on the charger, so it stays a warning.
func TestController_SetChargepointPhaseMode_TolerateFailedPause(t *testing.T) {
	t.Parallel()

	managerMock := mockedsignalr.NewManager(t)
	managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(types.GridTypeTN, time.Time{})
	cacheMock.On("Phases").Return(3, time.Time{})
	cacheMock.On("PhaseMode").Return(2, time.Time{})
	cacheMock.On("OutputPhaseType").Return(types.PhaseMode(""), time.Time{})
	cacheMock.On("SetRequestedPhaseMode", types.PhaseModeNL1, mock.AnythingOfType("time.Time")).Return(true)
	cacheMock.On("TotalPower").Return(3000.0, time.Time{})
	// Read before the pause is attempted, so it is consulted even when the pause fails.
	cacheMock.On("RequestedOfferedCurrent").Return(16, time.Time{})

	clientMock := mockapi.NewClient(t)
	clientMock.On("SetPhaseMode", "test-charger", 1).Return(nil).Once()
	clientMock.On("StopCharging", "test-charger").Return(errors.New("too many requests")).Once()

	ctrl := newTestController(t, managerMock, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

	assert.NoError(t, ctrl.SetChargepointPhaseMode(types.PhaseModeNL1))
}

func TestController_UpdateState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		initialState *easee.State
		mockClient   func(c *mockapi.Client)
		wantState    *easee.State
		wantErr      bool
	}{
		{
			name:         "rated current above 32 is clamped to 32",
			initialState: &easee.State{},
			mockClient: func(c *mockapi.Client) {
				c.On("ChargerConfig", "test-charger").Return(&model.ChargerConfig{}, nil)
				c.On("ChargerSiteInfo", "test-charger").Return(&model.ChargerSiteInfo{RatedCurrent: 40}, nil)
			},
			wantState: &easee.State{SupportedMaxCurrent: 32},
		},
		{
			name:         "rated current below 32 is kept as-is",
			initialState: &easee.State{},
			mockClient: func(c *mockapi.Client) {
				c.On("ChargerConfig", "test-charger").Return(&model.ChargerConfig{}, nil)
				c.On("ChargerSiteInfo", "test-charger").Return(&model.ChargerSiteInfo{RatedCurrent: 20}, nil)
			},
			wantState: &easee.State{SupportedMaxCurrent: 20},
		},
		{
			name:         "rated current exactly 32 stays at 32",
			initialState: &easee.State{},
			mockClient: func(c *mockapi.Client) {
				c.On("ChargerConfig", "test-charger").Return(&model.ChargerConfig{}, nil)
				c.On("ChargerSiteInfo", "test-charger").Return(&model.ChargerSiteInfo{RatedCurrent: 32}, nil)
			},
			wantState: &easee.State{SupportedMaxCurrent: 32},
		},
		{
			name:         "site info error when state needs update returns error",
			initialState: &easee.State{},
			mockClient: func(c *mockapi.Client) {
				c.On("ChargerConfig", "test-charger").Return(&model.ChargerConfig{}, nil)
				c.On("ChargerSiteInfo", "test-charger").Return((*model.ChargerSiteInfo)(nil), errors.New("api error"))
			},
			wantErr: true,
		},
		{
			name:         "config error when state needs update returns error",
			initialState: &easee.State{},
			mockClient: func(c *mockapi.Client) {
				c.On("ChargerConfig", "test-charger").Return((*model.ChargerConfig)(nil), errors.New("api error"))
				c.On("ChargerSiteInfo", "test-charger").Return(&model.ChargerSiteInfo{RatedCurrent: 20}, nil)
			},
			wantErr: true,
		},
		{
			name:         "config error is ignored when state is already populated",
			initialState: &easee.State{SupportedMaxCurrent: 20, Phases: 3, GridType: "TN"},
			mockClient: func(c *mockapi.Client) {
				c.On("ChargerConfig", "test-charger").Return((*model.ChargerConfig)(nil), errors.New("api error"))
				c.On("ChargerSiteInfo", "test-charger").Return(&model.ChargerSiteInfo{RatedCurrent: 20}, nil)
			},
			wantState: &easee.State{SupportedMaxCurrent: 20, Phases: 3, GridType: "TN"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clientMock := mockapi.NewClient(t)
			tt.mockClient(clientMock)

			ctrl := newTestController(t, nil, mockedcache.NewCache(t), clientMock, mockeddb.NewChargingSessionStorage(t), nil)

			err := ctrl.UpdateState("test-charger", tt.initialState)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantState, tt.initialState)
		})
	}
}

func TestController_ChargepointCurrentSessionReport(t *testing.T) {
	t.Parallel()

	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		sessions   db.ChargingSessions
		energy     float64
		maxCurrent int
		offered    int
		wantReport *chargepoint.SessionReport
	}{
		{
			name:       "no sessions returns empty report",
			sessions:   db.ChargingSessions{},
			energy:     1.5,
			wantReport: &chargepoint.SessionReport{SessionEnergy: 1.5},
		},
		{
			name: "active session includes offered current",
			sessions: db.ChargingSessions{
				{Start: now},
			},
			energy:     2.0,
			maxCurrent: 32,
			offered:    20,
			wantReport: &chargepoint.SessionReport{
				SessionEnergy:  2.0,
				StartedAt:      now,
				OfferedCurrent: 20,
			},
		},
		{
			name: "active session clamps offered current to maxCurrent",
			sessions: db.ChargingSessions{
				{Start: now},
			},
			energy:     2.0,
			maxCurrent: 16,
			offered:    20,
			wantReport: &chargepoint.SessionReport{
				SessionEnergy:  2.0,
				StartedAt:      now,
				OfferedCurrent: 16,
			},
		},
		{
			name: "finished session has no offered current",
			sessions: db.ChargingSessions{
				{Start: now, Stop: now.Add(time.Hour), Energy: 5},
			},
			energy: 5.0,
			wantReport: &chargepoint.SessionReport{
				SessionEnergy: 5.0,
				StartedAt:     now,
				FinishedAt:    now.Add(time.Hour),
			},
		},
		{
			name: "previous session energy is included",
			sessions: db.ChargingSessions{
				{Start: now},
				{Start: now.Add(-2 * time.Hour), Stop: now.Add(-time.Hour), Energy: 7},
			},
			energy:     2.0,
			maxCurrent: 32,
			offered:    16,
			wantReport: &chargepoint.SessionReport{
				SessionEnergy:         2.0,
				StartedAt:             now,
				OfferedCurrent:        16,
				PreviousSessionEnergy: 7,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cacheMock := mockedcache.NewCache(t)
			clientMock := mockapi.NewClient(t)
			managerMock := mockedsignalr.NewManager(t)
			sessionMock := mockeddb.NewChargingSessionStorage(t)

			managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))
			cacheMock.On("EnergySession").Return(tt.energy, time.Now())
			sessionMock.On("LatestSessionsByChargerID", "test-charger").Return(tt.sessions, nil)

			// only called when there's an active session
			if len(tt.sessions) > 0 && tt.sessions[0].Stop.IsZero() {
				cacheMock.On("OfferedCurrent").Return(tt.offered, time.Now())
				cacheMock.On("MaxCurrent").Return(tt.maxCurrent, time.Now())
			}

			ctrl := newTestController(t, managerMock, cacheMock, clientMock, sessionMock, nil)

			report, err := ctrl.ChargepointCurrentSessionReport()

			assert.NoError(t, err)
			assert.Equal(t, tt.wantReport, report)
		})
	}
}

// A grid with one settable mode has nothing to switch between: the command must not flip
// Easee's internal mode and bounce a session for a change nothing can observe.
func TestController_SetChargepointPhaseMode_SingleSettableModeIsNoOp(t *testing.T) {
	t.Parallel()

	managerMock := mockedsignalr.NewManager(t)
	managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(types.GridTypeTN, time.Time{})
	cacheMock.On("Phases").Return(1, time.Time{})

	clientMock := mockapi.NewClient(t)

	ctrl := newTestController(t, managerMock, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

	assert.NoError(t, ctrl.SetChargepointPhaseMode(types.PhaseModeNL1))
	clientMock.AssertNotCalled(t, "SetPhaseMode", mock.Anything, mock.Anything)
	clientMock.AssertNotCalled(t, "StopCharging", mock.Anything)
	cacheMock.AssertNotCalled(t, "SetRequestedPhaseMode", mock.Anything, mock.Anything)
}

// Observation timestamps come from Easee's feed, so a hub clock behind it would let a live
// observation outrank a request just issued. The request is stamped in the observation clock
// instead - it is only ever compared against those - so it wins whatever the hub clock says.
func TestController_SetChargepointPhaseMode_OutranksObservationsUnderClockSkew(t *testing.T) {
	t.Parallel()

	outputPhaseSet := time.Now().Add(time.Hour)
	internalAt := outputPhaseSet.Add(-30 * time.Minute)

	managerMock := mockedsignalr.NewManager(t)
	managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

	var requestedAt time.Time

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("GridType").Return(types.GridTypeTN, time.Time{})
	cacheMock.On("Phases").Return(3, time.Time{})
	cacheMock.On("PhaseMode").Return(2, internalAt)
	cacheMock.On("OutputPhaseType").Return(types.PhaseModeNL3, outputPhaseSet)
	cacheMock.On("SetRequestedPhaseMode", types.PhaseModeNL1, mock.AnythingOfType("time.Time")).
		Run(func(args mock.Arguments) { requestedAt, _ = args.Get(1).(time.Time) }).Return(true)
	cacheMock.On("TotalPower").Return(float64(0), time.Time{})
	cacheMock.On("ChargerState").Return(chargepoint.StateReadyToCharge, time.Time{})

	clientMock := mockapi.NewClient(t)
	clientMock.On("SetPhaseMode", "test-charger", 1).Return(nil).Once()

	ctrl := newTestController(t, managerMock, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

	assert.NoError(t, ctrl.SetChargepointPhaseMode(types.PhaseModeNL1))
	assert.True(t, requestedAt.After(outputPhaseSet), "the request has to outrank the newest observation")

	cacheMock.On("RequestedPhaseMode").Return(types.PhaseModeNL1, requestedAt)

	mode, err := ctrl.ChargepointPhaseModeReport()
	assert.NoError(t, err)
	assert.Equal(t, types.PhaseModeNL1, mode)
}

// Auto is not pinned to a leg, and the setter maps a three-phase request onto it. Falling back to
// the first supported mode would answer that request with a single leg once the cache is empty.
func TestController_ChargepointPhaseModeReport_AutoFallbackPrefersMultiPhase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		gridType model.GridType
		internal int
		want     types.PhaseMode
	}{
		{"auto on a 3-phase grid", model.GridTypeTN3Phase, model.EaseePhaseModeAuto, types.PhaseModeNL1L2L3},
		{"locked to a single phase", model.GridTypeTN3Phase, 1, types.PhaseModeNL1},
		{"auto on a 1-phase grid", model.GridTypeTN1Phase, model.EaseePhaseModeAuto, types.PhaseModeNL1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			managerMock := mockedsignalr.NewManager(t)
			managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

			cacheMock := mockedcache.NewCache(t)
			cacheMock.On("OutputPhaseType").Return(types.PhaseMode(""), time.Time{})
			cacheMock.On("RequestedPhaseMode").Return(types.PhaseMode(""), time.Time{})

			clientMock := mockapi.NewClient(t)
			clientMock.On("ChargerConfig", "test-charger").
				Return(&model.ChargerConfig{DetectedPowerGridType: tt.gridType, PhaseMode: tt.internal}, nil)
			clientMock.On("ChargerSiteInfo", "test-charger").Return(&model.ChargerSiteInfo{RatedCurrent: 32}, nil)

			ctrl := newTestController(t, managerMock, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

			mode, err := ctrl.ChargepointPhaseModeReport()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, mode)
		})
	}
}

// Observations carry Easee's clock while the seed is written by the hub, so seeding at
// time.Now() lets a hub running ahead of the server suppress the authoritative value in the
// cache's timestamp guard. The zero time keeps the optimistic echo without ever outranking
// an observation.
func TestController_SetParameter_SeedsCableLock(t *testing.T) {
	t.Parallel()

	clientMock := mockapi.NewClient(t)
	clientMock.On("SetCableAlwaysLocked", "test-charger", true).Return(nil).Once()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("SeedCableAlwaysLocked", true).Once()

	ctrl := newTestController(t, mockedsignalr.NewManager(t), cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

	require.NoError(t, ctrl.SetParameter(&parameters.Parameter{
		ID:        model.CableAlwaysLockedParameter,
		ValueType: parameters.ValueTypeBool,
		Value:     json.RawMessage("true"),
	}))
}

// Easee accepting the start is not the charger acting on it. Without the observation echoing
// the current back the session may well still be paused, so cmd.charge.start must not report
// success - the same contract the phase-mode resume already holds itself to.
func TestController_StartChargepointCharging_ReportsUnconfirmedStart(t *testing.T) {
	t.Parallel()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("MaxCurrent").Return(16, time.Time{})
	cacheMock.On("RequestedOfferedCurrent").Return(0, time.Time{})
	cacheMock.On("SetRequestedOfferedCurrent", 16, mock.AnythingOfType("time.Time")).Return(true)
	// The charger never echoed the current back.
	cacheMock.On("WaitForOfferedCurrent", 16, mock.AnythingOfType("time.Duration")).Return(false)

	clientMock := mockapi.NewClient(t)
	clientMock.On("UpdateDynamicCurrent", "test-charger", float64(16)).Return(nil).Once()

	ctrl := newTestController(t, nil, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

	err := ctrl.StartChargepointCharging(&chargepoint.ChargingSettings{Mode: model.ChargingModeNormal})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "did not resume")
}

// A start the channel stores has to report success at once: the echo it would wait for
// cannot arrive before the deadline, and failing cmd.charge.start would leave the user
// re-pressing Start against a write that is already on its way.
func TestController_StartChargepointCharging_DeferredStartReportsSuccess(t *testing.T) {
	clk := clock.Mock(time.Date(2026, time.September, 20, 1, 0, 0, 0, time.UTC))
	t.Cleanup(clock.Restore)

	done := make(chan struct{}, 1)

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("MaxCurrent").Return(16, time.Time{})
	cacheMock.On("RequestedOfferedCurrent").Return(0, time.Time{})
	cacheMock.On("SetRequestedOfferedCurrent", mock.Anything, mock.AnythingOfType("time.Time")).Return(true)
	cacheMock.On("WaitForOfferedCurrent", 10, mock.Anything).Return(true).Once()
	cacheMock.On("WaitForOfferedCurrent", 16, mock.Anything).Return(true).Run(signal(done)).Once()

	clientMock := mockapi.NewClient(t)
	clientMock.On("UpdateDynamicCurrent", "test-charger", float64(10)).Return(nil).Once()
	clientMock.On("UpdateDynamicCurrent", "test-charger", float64(16)).Return(nil).Once()

	ctrl := newTestController(t, nil, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

	require.NoError(t, ctrl.SetChargepointOfferedCurrent(10))

	clk.Add(5 * time.Second)
	require.NoError(t, ctrl.StartChargepointCharging(&chargepoint.ChargingSettings{Mode: model.ChargingModeNormal}))
	cacheMock.AssertNotCalled(t, "WaitForOfferedCurrent", 16, mock.Anything)

	clk.Add(40 * time.Second)
	awaitDeferred(t, done)
	clientMock.AssertCalled(t, "UpdateDynamicCurrent", "test-charger", float64(16))
}

// Every report goes through the connection gate; a disconnected charger answers with the reason.
func TestController_Reports_RequireAConnectedCharger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(c easee.Controller) error
	}{
		{"cable lock", func(c easee.Controller) error { _, err := c.ChargepointCableLockReport(); return err }},
		{"phase mode", func(c easee.Controller) error { _, err := c.ChargepointPhaseModeReport(); return err }},
		{"max current", func(c easee.Controller) error { _, err := c.ChargepointMaxCurrentReport(); return err }},
		{"session", func(c easee.Controller) error { _, err := c.ChargepointCurrentSessionReport(); return err }},
		{"alarm", func(c easee.Controller) error { _, err := c.AlarmReport("evt"); return err }},
		{"state", func(c easee.Controller) error { _, err := c.ChargepointStateReport(); return err }},
		{"meter", func(c easee.Controller) error { _, err := c.MeterReport(numericmeter.UnitW); return err }},
		{"extended meter", func(c easee.Controller) error { _, err := c.MeterExtendedReport(nil); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			managerMock := mockedsignalr.NewManager(t)
			managerMock.On("Connected", "test-charger").Return(false, signalr.DisconnectionReason("offline"))

			ctrl := newTestController(t, managerMock, mockedcache.NewCache(t), mockapi.NewClient(t), mockeddb.NewChargingSessionStorage(t), nil)

			err := tt.call(ctrl)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "offline")
		})
	}
}

func TestController_ChargepointCableLockReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		locked      bool
		cable       int
		cableAt     time.Time
		wantCurrent *int
	}{
		{name: "unlocked reports zero", locked: false, cable: 32, cableAt: time.Now(), wantCurrent: new(int)},
		{name: "locked with an observed rating", locked: true, cable: 20, cableAt: time.Now(), wantCurrent: ptr(20)},
		{name: "locked with no rating observed", locked: true, cable: 20},
		{name: "locked with a negative rating", locked: true, cable: -1, cableAt: time.Now()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			managerMock := mockedsignalr.NewManager(t)
			managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

			cacheMock := mockedcache.NewCache(t)
			cacheMock.On("CableLocked").Return(tt.locked, time.Time{})
			cacheMock.On("CableCurrent").Return(tt.cable, tt.cableAt).Maybe()

			ctrl := newTestController(t, managerMock, cacheMock, mockapi.NewClient(t), mockeddb.NewChargingSessionStorage(t), nil)

			report, err := ctrl.ChargepointCableLockReport()
			require.NoError(t, err)
			assert.Equal(t, tt.locked, report.CableLock)
			assert.Equal(t, tt.wantCurrent, report.CableCurrent)
		})
	}
}

func ptr(v int) *int { return &v }

func TestController_MaxCurrent(t *testing.T) {
	t.Parallel()

	managerMock := mockedsignalr.NewManager(t)
	managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("WaitForMaxCurrent", 20, mock.AnythingOfType("time.Duration")).Return(true).Once()
	cacheMock.On("MaxCurrent").Return(20, time.Time{})

	boom := errors.New("cloud down")

	clientMock := mockapi.NewClient(t)
	clientMock.On("UpdateMaxCurrent", "test-charger", float64(20)).Return(nil).Once()
	clientMock.On("UpdateMaxCurrent", "test-charger", float64(25)).Return(boom).Once()

	ctrl := newTestController(t, managerMock, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

	require.NoError(t, ctrl.SetChargepointMaxCurrent(20))
	assert.ErrorIs(t, ctrl.SetChargepointMaxCurrent(25), boom)
	cacheMock.AssertNumberOfCalls(t, "WaitForMaxCurrent", 1)

	current, err := ctrl.ChargepointMaxCurrentReport()
	require.NoError(t, err)
	assert.Equal(t, 20, current)
}

func TestController_AlarmReport(t *testing.T) {
	t.Parallel()

	managerMock := mockedsignalr.NewManager(t)
	managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("AlarmActive", "burglar").Return(true)
	cacheMock.On("AlarmActive", "smoke").Return(false)

	ctrl := newTestController(t, managerMock, cacheMock, mockapi.NewClient(t), mockeddb.NewChargingSessionStorage(t), nil)

	report, err := ctrl.AlarmReport("burglar")
	require.NoError(t, err)
	assert.Equal(t, &alarm.Report{Event: "burglar", Status: alarm.StatusActivate}, report)

	report, err = ctrl.AlarmReport("smoke")
	require.NoError(t, err)
	assert.Equal(t, &alarm.Report{Event: "smoke", Status: alarm.StatusDeactivate}, report)
}

func TestController_MeterReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		unit     numericmeter.Unit
		energyAt time.Time
		want     float64
		wantErr  string
	}{
		{name: "power", unit: numericmeter.UnitW, want: 2300},
		{name: "energy", unit: numericmeter.UnitKWh, energyAt: time.Now(), want: 12.5},
		{name: "energy never observed", unit: numericmeter.UnitKWh, wantErr: "energy value not updated"},
		{name: "unsupported unit", unit: numericmeter.UnitA, wantErr: "unsupported unit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			managerMock := mockedsignalr.NewManager(t)
			managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

			cacheMock := mockedcache.NewCache(t)
			cacheMock.On("TotalPower").Return(2300.0, time.Time{}).Maybe()
			cacheMock.On("LifetimeEnergy").Return(12.5, tt.energyAt).Maybe()

			ctrl := newTestController(t, managerMock, cacheMock, mockapi.NewClient(t), mockeddb.NewChargingSessionStorage(t), nil)

			got, err := ctrl.MeterReport(tt.unit)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.InDelta(t, tt.want, got, 0)
		})
	}
}

// Lifetime energy has no meaningful zero, so it is the one value left out until observed;
// unknown values are skipped rather than failing the report.
func TestController_MeterExtendedReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		energyAt time.Time
		want     numericmeter.ValuesReport
	}{
		{
			name:     "energy observed",
			energyAt: time.Now(),
			want:     numericmeter.ValuesReport{numericmeter.ValueCurrentPhase1: 10, numericmeter.ValuePowerImport: 2300, numericmeter.ValueEnergyImport: 12.5},
		},
		{
			name: "energy never observed",
			want: numericmeter.ValuesReport{numericmeter.ValueCurrentPhase1: 10, numericmeter.ValuePowerImport: 2300},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			managerMock := mockedsignalr.NewManager(t)
			managerMock.On("Connected", "test-charger").Return(true, signalr.DisconnectionReason(""))

			cacheMock := mockedcache.NewCache(t)
			cacheMock.On("Phase1Current").Return(10.0, time.Time{})
			cacheMock.On("TotalPower").Return(2300.0, time.Time{})
			cacheMock.On("LifetimeEnergy").Return(12.5, tt.energyAt)

			ctrl := newTestController(t, managerMock, cacheMock, mockapi.NewClient(t), mockeddb.NewChargingSessionStorage(t), nil)

			got, err := ctrl.MeterExtendedReport(numericmeter.Values{
				numericmeter.ValueCurrentPhase1, numericmeter.ValuePowerImport, numericmeter.ValueEnergyImport, numericmeter.Value("unknown"),
			})
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestController_Parameters(t *testing.T) {
	t.Parallel()

	cacheMock := mockedcache.NewCache(t)
	cacheMock.On("CableAlwaysLocked").Return(true, time.Time{})

	ctrl := newTestController(t, nil, cacheMock, mockapi.NewClient(t), mockeddb.NewChargingSessionStorage(t), nil)

	specs, err := ctrl.GetParameterSpecifications()
	require.NoError(t, err)
	require.Len(t, specs, 1)
	assert.Equal(t, model.CableAlwaysLockedParameter, specs[0].ID)

	param, err := ctrl.GetParameter(model.CableAlwaysLockedParameter)
	require.NoError(t, err)
	assert.Equal(t, parameters.NewBoolParameter(model.CableAlwaysLockedParameter, true), param)

	_, err = ctrl.GetParameter("other")
	assert.ErrorContains(t, err, "parameter: other not supported")

	err = ctrl.SetParameter(parameters.NewBoolParameter("other", true))
	assert.ErrorContains(t, err, "parameter: other not supported")

	err = ctrl.SetParameter(parameters.NewStringParameter(model.CableAlwaysLockedParameter, "yes"))
	assert.Error(t, err, "a non-boolean value is rejected before anything is sent")
}
