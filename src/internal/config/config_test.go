package config_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
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

// TestService_OfferedCurrentWaitTime_MatchesPackagedDefault ties the getter's unset-config
// fallback to the value the packaged config actually ships, so bumping one without the other
// can't silently leave every real install on the old wait time again.
func TestService_OfferedCurrentWaitTime_MatchesPackagedDefault(t *testing.T) {
	body, err := os.ReadFile("../../../package/debian/opt/thingsplex/easee/defaults/config.json")
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
