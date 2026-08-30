package easee_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

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
) easee.Controller {
	t.Helper()

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
	)
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
		})
	}
}

// A charging session keeps its phase count until it is bounced, so the setter has to
// restart it for the new mode to take effect right away.
func TestController_SetChargepointPhaseMode_RestartsActiveSession(t *testing.T) {
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
	cacheMock.On("MaxCurrent").Return(16, time.Time{})
	cacheMock.On("RequestedOfferedCurrent").Return(16, time.Time{})
	cacheMock.On("SetRequestedOfferedCurrent", 16, mock.AnythingOfType("time.Time")).Return(true)
	cacheMock.On("WaitForOfferedCurrent", 16, mock.AnythingOfType("time.Duration")).Return(true)

	clientMock := mockapi.NewClient(t)
	clientMock.On("SetPhaseMode", "test-charger", 1).Return(nil).Once()
	clientMock.On("StopCharging", "test-charger").Return(nil).Once()
	clientMock.On("UpdateDynamicCurrent", "test-charger", float64(16)).Return(nil).Once()

	ctrl := newTestController(t, managerMock, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

	assert.NoError(t, ctrl.SetChargepointPhaseMode(types.PhaseModeNL1))
	clientMock.AssertCalled(t, "StopCharging", "test-charger")
	clientMock.AssertCalled(t, "UpdateDynamicCurrent", "test-charger", float64(16))
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

	ctrl := newTestController(t, managerMock, cacheMock, mockapi.NewClient(t), mockeddb.NewChargingSessionStorage(t), nil)

	assert.NoError(t, ctrl.SetChargepointPhaseMode(types.PhaseModeNL2))
	cacheMock.AssertNotCalled(t, "SetRequestedPhaseMode", mock.Anything, mock.Anything)
}

// Bouncing a session to apply a phase mode must put back the current that was running.
// Resuming through a normal-mode start floored it to initial_charging_current, silently
// pushing a 6A slow session to 16A - and nothing records the mode, so it cannot be restored.
func TestController_SetChargepointPhaseMode_ResumesAtTheSessionCurrent(t *testing.T) {
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
	cacheMock.On("MaxCurrent").Return(32, time.Time{})
	cacheMock.On("RequestedOfferedCurrent").Return(6, time.Time{})
	cacheMock.On("SetRequestedOfferedCurrent", 6, mock.AnythingOfType("time.Time")).Return(true)
	cacheMock.On("WaitForOfferedCurrent", 6, mock.AnythingOfType("time.Duration")).Return(true)

	clientMock := mockapi.NewClient(t)
	clientMock.On("SetPhaseMode", "test-charger", 1).Return(nil).Once()
	clientMock.On("StopCharging", "test-charger").Return(nil).Once()
	clientMock.On("UpdateDynamicCurrent", "test-charger", float64(6)).Return(nil).Once()

	ctrl := newTestController(t, managerMock, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), &config.Config{
		PublicConfig: config.PublicConfig{InitialChargingCurrent: 16},
	})

	assert.NoError(t, ctrl.SetChargepointPhaseMode(types.PhaseModeNL1))
}

// Pausing then failing to resume leaves the car not charging, so the caller has to hear
// about it even though the phase mode itself was stored successfully.
func TestController_SetChargepointPhaseMode_ReportsFailedResume(t *testing.T) {
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
	cacheMock.On("MaxCurrent").Return(16, time.Time{})
	cacheMock.On("RequestedOfferedCurrent").Return(16, time.Time{})

	boom := errors.New("cloud rejected the resume")

	clientMock := mockapi.NewClient(t)
	clientMock.On("SetPhaseMode", "test-charger", 1).Return(nil).Once()
	clientMock.On("StopCharging", "test-charger").Return(nil).Once()
	clientMock.On("UpdateDynamicCurrent", "test-charger", float64(16)).Return(boom).Once()

	ctrl := newTestController(t, managerMock, cacheMock, clientMock, mockeddb.NewChargingSessionStorage(t), nil)

	err := ctrl.SetChargepointPhaseMode(types.PhaseModeNL1)
	assert.ErrorIs(t, err, boom)
	assert.Contains(t, err.Error(), "left stopped")
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
