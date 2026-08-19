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
				cacheMock.On("SetRequestedPhaseMode", tt.mode, mock.AnythingOfType("time.Time")).Return(true)
			}

			if !tt.wantErr && !tt.expectNoCall {
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

	ctrl := newTestController(t, managerMock, cacheMock, mockapi.NewClient(t), mockeddb.NewChargingSessionStorage(t), nil)

	mode, err := ctrl.ChargepointPhaseModeReport()
	assert.NoError(t, err)
	assert.Equal(t, types.PhaseModeNL3, mode)
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
