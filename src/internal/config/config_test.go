package config_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/selection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/fakes"
	mockedstorage "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/storage"
)

// TestService_PublicModel_NoSecrets locks the FIMP contract: cmd.config.get_extended_report
// and the manifest must expose only PublicConfig, never config.Default or the tokens.
func TestService_PublicModel_NoSecrets(t *testing.T) {
	cfg := &config.Config{}
	cfg.EaseeBaseURL = "https://api.easee.test"
	cfg.AccessToken = "secret-access"
	cfg.RefreshToken = "secret-refresh"
	cfg.MQTTUsername = "mqtt-user"
	cfg.MQTTPassword = "mqtt-pass"
	cfg.LogLevel = "debug"

	st := &mockedstorage.Storage[*config.Config]{}
	st.On("Model").Return(cfg)
	cs := config.NewService(st)

	public, ok := cs.PublicModel().(config.PublicConfig)
	require.True(t, ok)
	assert.Equal(t, "https://api.easee.test", public.EaseeBaseURL)

	body, err := json.Marshal(public)
	require.NoError(t, err)
	assert.NotContains(t, string(body), "secret-access")
	assert.NotContains(t, string(body), "secret-refresh")
	assert.NotContains(t, string(body), "mqtt-user")
	assert.NotContains(t, string(body), "mqtt-pass")
	assert.NotContains(t, string(body), "log_level")
	assert.NotContains(t, string(body), "telemetry")
}

// TestService_SignalRFinalBackoff_MatchesStatefulDefault locks the getter's unset-config
// default to the final backoff SignalRBackoffStateful actually applies, so cmd.config.get
// signalr_final_backoff can't silently drift from runtime behavior again.
func TestService_SignalRFinalBackoff_MatchesStatefulDefault(t *testing.T) {
	cfg := &config.Config{}

	st := &mockedstorage.Storage[*config.Config]{}
	st.On("Model").Return(cfg)
	cs := config.NewService(st)

	assert.Equal(t, 10*time.Minute, cs.SignalRFinalBackoff())
}

// TestService_SignalRFinalBackoff_MatchesPackagedDefault ties the ceiling to the value the
// packaged config actually ships. Without it the getter/stateful fallbacks can be raised while
// every real install keeps reading the old value from disk, leaving the bump a no-op.
func TestService_SignalRFinalBackoff_MatchesPackagedDefault(t *testing.T) {
	body, err := os.ReadFile("../../../package/debian/usr/share/futurehome/easee/defaults/config.json")
	require.NoError(t, err)

	cfg := &config.Config{}
	require.NoError(t, json.Unmarshal(body, cfg))

	st := &mockedstorage.Storage[*config.Config]{}
	st.On("Model").Return(cfg)

	assert.Equal(t, 10*time.Minute, config.NewService(st).SignalRFinalBackoff())
}

// TestService_OfferedCurrentWaitTime_MatchesPackagedDefault ties the getter's unset-config
// fallback to the value the packaged config actually ships, so bumping one without the other
// can't silently leave every real install on the old wait time again.
func TestService_OfferedCurrentWaitTime_MatchesPackagedDefault(t *testing.T) {
	body, err := os.ReadFile("../../../package/debian/usr/share/futurehome/easee/defaults/config.json")
	require.NoError(t, err)

	packaged := &config.Config{}
	require.NoError(t, json.Unmarshal(body, packaged))

	for name, cfg := range map[string]*config.Config{"packaged": packaged, "unset": {}} {
		t.Run(name, func(t *testing.T) {
			st := &mockedstorage.Storage[*config.Config]{}
			st.On("Model").Return(cfg)

			assert.Equal(t, 20*time.Second, config.NewService(st).OfferedCurrentWaitTime())
		})
	}
}

func TestService_OfferedCurrentDeferralTime_MatchesPackagedDefault(t *testing.T) {
	body, err := os.ReadFile("../../../package/debian/usr/share/futurehome/easee/defaults/config.json")
	require.NoError(t, err)

	packaged := &config.Config{}
	require.NoError(t, json.Unmarshal(body, packaged))

	for name, cfg := range map[string]*config.Config{"packaged": packaged, "unset": {}} {
		t.Run(name, func(t *testing.T) {
			st := &mockedstorage.Storage[*config.Config]{}
			st.On("Model").Return(cfg)

			assert.Equal(t, 45*time.Second, config.NewService(st).OfferedCurrentDeferralTime())
		})
	}
}

func TestService_InitialChargingCurrent_MatchesPackagedDefault(t *testing.T) {
	body, err := os.ReadFile("../../../package/debian/usr/share/futurehome/easee/defaults/config.json")
	require.NoError(t, err)

	packaged := &config.Config{}
	require.NoError(t, json.Unmarshal(body, packaged))

	// Asserted on the raw field, not through the getter: a packaged key that stopped binding
	// (renamed struct tag, typo) reads as 0 and the getter would still answer 16.
	assert.Equal(t, 16, packaged.InitialChargingCurrent, "packaged JSON key no longer binds to the config field")

	for name, cfg := range map[string]*config.Config{"packaged": packaged, "unset": {}} {
		t.Run(name, func(t *testing.T) {
			st := &mockedstorage.Storage[*config.Config]{}
			st.On("Model").Return(cfg)

			assert.Equal(t, 16, config.NewService(st).InitialChargingCurrent())
		})
	}
}

func TestConfig_MigrateSignalRFinalBackoff(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		expected string
	}{
		{name: "superseded packaged default is lifted", current: "2m", expected: "10m"},
		{name: "tuned value is preserved", current: "5m", expected: "5m"},
		{name: "unset value is left to the getter fallback", current: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.SignalR.FinalBackoff = tt.current

			require.NoError(t, cfg.MigrateSignalRFinalBackoff())
			assert.Equal(t, tt.expected, cfg.SignalR.FinalBackoff)
		})
	}
}

func TestConfig_MigrateOfferedCurrentWaitTime(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		expected string
	}{
		{name: "superseded packaged default is lifted", current: "15s", expected: "20s"},
		{name: "tuned value is preserved", current: "45s", expected: "45s"},
		{name: "unset value is left to the getter fallback", current: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.OfferedCurrentWaitTime = tt.current

			require.NoError(t, cfg.MigrateOfferedCurrentWaitTime())
			assert.Equal(t, tt.expected, cfg.OfferedCurrentWaitTime)
		})
	}
}

// TestConfig_SelectedDevices_NilVsEmpty locks the distinction the selection rests on: a
// configuration written before v3.0 carries no selected_devices key and must read as "every
// charger", while an explicit empty list from the UI must read as "no charger". Both survive
// a round trip through the stored JSON.
func TestConfig_SelectedDevices_NilVsEmpty(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		wantIncludeAll bool
		wantJSON       string
	}{
		{
			name:           "a pre-v3 configuration has no key and includes every charger",
			body:           `{"easeeBaseURL2":"https://api.easee.test"}`,
			wantIncludeAll: true,
			wantJSON:       `"selected_devices":null`,
		},
		{
			name:     "an explicit empty list includes no charger",
			body:     `{"selected_devices":[]}`,
			wantJSON: `"selected_devices":[]`,
		},
		{
			name:     "a chosen subset is kept as is",
			body:     `{"selected_devices":["EH123"]}`,
			wantJSON: `"selected_devices":["EH123"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Factory()
			require.NoError(t, json.Unmarshal([]byte(tt.body), cfg))

			assert.Equal(t, tt.wantIncludeAll, cfg.SelectedDevices.IncludeAll())

			body, err := json.Marshal(cfg.PublicConfig)
			require.NoError(t, err)
			assert.Contains(t, string(body), tt.wantJSON)
		})
	}
}

// Every setting the service exposes reads back what was written, and reads its documented
// default from an empty configuration.
func TestService_SettingsRoundTrip(t *testing.T) {
	t.Parallel()

	empty := config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory))
	set := config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory))

	durations := []struct {
		name string
		def  time.Duration
		get  func(*config.Service) time.Duration
		set  func(*config.Service, time.Duration) error
	}{
		{"polling interval", 10 * time.Minute, (*config.Service).PollingInterval, (*config.Service).SetPollingInterval},
		{"current wait", 3 * time.Second, (*config.Service).CurrentWaitDuration, (*config.Service).SetCurrentWaitDuration},
		{"http timeout", 30 * time.Second, (*config.Service).HTTPTimeout, (*config.Service).SetHTTPTimeout},
		{"signalr conn creation", 30 * time.Second, (*config.Service).SignalRConnCreationTimeout, (*config.Service).SetSignalRConnCreationTimeout},
		{"signalr keepalive", 30 * time.Second, (*config.Service).SignalRKeepAliveInterval, (*config.Service).SetSignalRKeepAliveInterval},
		{"signalr timeout", time.Minute, (*config.Service).SignalRTimeoutInterval, (*config.Service).SetSignalRTimeoutInterval},
		{"signalr initial backoff", 5 * time.Second, (*config.Service).SignalRInitialBackoff, (*config.Service).SetSignalRInitialBackoff},
		{"signalr repeated backoff", 30 * time.Second, (*config.Service).SignalRRepeatedBackoff, (*config.Service).SetSignalRRepeatedBackoff},
		{"signalr final backoff", 10 * time.Minute, (*config.Service).SignalRFinalBackoff, (*config.Service).SetSignalRFinalBackoff},
		{"signalr invoke timeout", 10 * time.Second, (*config.Service).SignalRInvokeTimeout, (*config.Service).SetSignalRInvokeTimeout},
	}

	for _, tt := range durations {
		assert.Equal(t, tt.def, tt.get(empty), tt.name)
		require.NoError(t, tt.set(set, 7*time.Second), tt.name)
		assert.Equal(t, 7*time.Second, tt.get(set), tt.name)
	}

	assert.Equal(t, 10*time.Second, empty.EnergyLifetimeInterval())
	assert.Equal(t, 30*time.Minute, empty.TokenRefreshInterval())
	assert.Equal(t, 2*time.Hour, empty.AuthenticatorMaxUnauthorized())
	assert.Equal(t, uint32(0), empty.SignalRInitialFailureCount())
	assert.Equal(t, uint32(0), empty.SignalRRepeatedFailureCount())
	assert.Nil(t, empty.SelectedDevices())
	assert.Empty(t, empty.EaseeBaseURL())
	assert.Empty(t, empty.SignalRBaseURL())
	assert.InDelta(t, 0, empty.SlowChargingCurrentInAmperes(), 0)

	require.NoError(t, set.SetEaseeBaseURL("https://api"))
	require.NoError(t, set.SetSignalRBaseURL("https://streams"))
	require.NoError(t, set.SetSlowChargingCurrentInAmperes(6))
	require.NoError(t, set.SetSignalRInitialFailureCount(2))
	require.NoError(t, set.SetSignalRRepeatedFailureCount(3))
	require.NoError(t, set.SetSelectedDevices(selection.Selection{"EH1"}))
	require.NoError(t, set.SetAuthenticatorBackoff(time.Minute, 2*time.Minute, 3*time.Minute, 4, 5, time.Hour))

	assert.Equal(t, "https://api", set.EaseeBaseURL())
	assert.Equal(t, "https://streams", set.SignalRBaseURL())
	assert.InDelta(t, 6, set.SlowChargingCurrentInAmperes(), 0)
	assert.Equal(t, uint32(2), set.SignalRInitialFailureCount())
	assert.Equal(t, uint32(3), set.SignalRRepeatedFailureCount())
	assert.Equal(t, selection.Selection{"EH1"}, set.SelectedDevices())
	assert.Equal(t, time.Hour, set.AuthenticatorMaxUnauthorized())
	assert.Equal(t, "1m0s", set.PublicConfig().AuthBackoff.InitialBackoff)
	assert.NotNil(t, set.AuthenticatorBackoffStateful())
	assert.NotNil(t, set.SignalRBackoffStateful())
}

func TestConfig_MigrateAuthBackoff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		legacy string
		want   string
	}{
		{name: "legacy object present", legacy: `{"initialBackoff":"2m","maxUnauthorizedDuration":"3h"}`, want: "2m"},
		{name: "corrupt legacy object is dropped", legacy: `{"initialBackoff":`},
		{name: "no legacy object"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{}
			if tt.legacy != "" {
				cfg.LegacyAuthenticatorBackoff = json.RawMessage(tt.legacy)
			}

			require.NoError(t, cfg.MigrateAuthBackoff())
			assert.Nil(t, cfg.LegacyAuthenticatorBackoff)
			assert.Equal(t, tt.want, cfg.AuthBackoff.InitialBackoff)
		})
	}
}

func TestCredentials_AccessTokenExpired(t *testing.T) {
	t.Parallel()

	assert.True(t, config.Credentials{AccessTokenExpiresAt: time.Now().Add(-time.Minute)}.AccessTokenExpired())
	assert.False(t, config.Credentials{AccessTokenExpiresAt: time.Now().Add(time.Minute)}.AccessTokenExpired())
	assert.True(t, config.Credentials{}.Empty())
	assert.Equal(t, "/tmp/work", config.New("/tmp/work").WorkDir)
}
