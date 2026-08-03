package config_test

import (
	"encoding/json"
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
