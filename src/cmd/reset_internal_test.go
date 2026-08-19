package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	cliffStorage "github.com/futurehomeno/cliffhanger/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// TestUninstall_resetAndReMigrate pins what the Uninstall reset leaves behind. Since
// cliffhanger v1.3.4 storage.Reset zeroes the model before reloading the defaults, so every
// config field the packaged defaults do not carry is genuinely cleared - selected_devices,
// auth_backoff, auth_max_unauthorized and config_version. The dropped version makes the next
// boot re-run migrations 0->6, which must be harmless.
func TestUninstall_resetAndReMigrate(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	body, err := os.ReadFile("../../package/debian/usr/share/futurehome/easee/defaults/config.json")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(workDir, "defaults"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "defaults", "config.json"), body, 0o600)) //nolint:gosec

	cfg := config.New(workDir)
	cfgSvc := config.NewService(cliffStorage.New(cfg, workDir, "config.json"))
	require.NoError(t, cfgSvc.Load())

	credentials := config.NewCredentialsStore(workDir)
	require.NoError(t, credentials.Load())

	migrateConfig(cfgSvc, credentials)
	require.Equal(t, 7, cfg.ConfigVersion)

	require.NoError(t, cfgSvc.SetSelectedDevices([]string{"charger-1"}))
	require.NoError(t, cfgSvc.SetAuthenticatorBackoff(time.Second, 2*time.Second, 3*time.Second, 1, 2, time.Hour))
	require.NoError(t, credentials.SetCredentials(config.Credentials{AccessToken: "access", RefreshToken: "refresh"}))

	// The configuration half of Uninstall.
	require.NoError(t, cfgSvc.Reset())
	require.NoError(t, credentials.ClearCredentials())

	// Absent from the packaged defaults: blanked. Losing the device selection on an uninstall
	// is accepted - the user re-runs the Configure step.
	assert.Nil(t, cfgSvc.SelectedDevices())
	assert.Zero(t, cfg.ConfigVersion)
	assert.Empty(t, cfg.AuthBackoff.InitialBackoff)
	assert.Empty(t, cfg.AuthMaxUnauthorized)

	// Present in the packaged defaults: restored, not blanked. WorkDir is json:"-" and comes
	// back from config.Service.Reset.
	assert.Equal(t, workDir, cfg.WorkDir)
	assert.Equal(t, "https://api.easee.com", cfgSvc.EaseeBaseURL())
	assert.Equal(t, 20*time.Second, cfgSvc.OfferedCurrentWaitTime())
	assert.Equal(t, 10*time.Second, cfgSvc.EnergyLifetimeInterval())
	assert.True(t, credentials.Credentials().Empty())

	// The next boot re-runs every migration. It only restores what the reset produced: the
	// authenticator backoff is rebuilt from the legacy block the defaults still ship, the two
	// value migrations find the current defaults and do nothing, and the credential migration
	// has nothing to move.
	migrateConfig(cfgSvc, credentials)

	assert.Equal(t, 7, cfg.ConfigVersion)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "budzik", cfg.LogFormat)
	assert.Equal(t, "1m", cfg.AuthBackoff.InitialBackoff)
	assert.Equal(t, "5m", cfg.AuthBackoff.RepeatedBackoff)
	assert.Equal(t, "10m", cfg.AuthBackoff.FinalBackoff)
	assert.Equal(t, 2*time.Hour, cfgSvc.AuthenticatorMaxUnauthorized())
	assert.Equal(t, 20*time.Second, cfgSvc.OfferedCurrentWaitTime())
	assert.Equal(t, 10*time.Minute, cfgSvc.SignalRFinalBackoff())
	assert.True(t, credentials.Credentials().Empty())
	assert.Nil(t, cfgSvc.SelectedDevices())
}
