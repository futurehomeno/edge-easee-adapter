package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	cliffCfg "github.com/futurehomeno/cliffhanger/config"
	cliffStorage "github.com/futurehomeno/cliffhanger/storage"
	"github.com/futurehomeno/cliffhanger/telemetry"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

func TestTelemetry_freshInstallIsEnabled(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	body, err := os.ReadFile("../../package/debian/usr/share/futurehome/easee/defaults/config.json")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(workDir, "defaults"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "defaults", "config.json"), body, 0o600)) //nolint:gosec

	cfgSvc := config.NewService(cliffStorage.New(config.New(workDir), workDir, "config.json"))
	require.NoError(t, cfgSvc.Load())
	require.NoError(t, migrateConfig(cfgSvc, config.NewCredentialsStore(workDir)))

	assert.True(t, newTestTelemetry(t, cfgSvc.DefaultStore()).IsEnabled())
}

func TestMigrateConfig_enablesTheOldDisabledTelemetrySeedOnce(t *testing.T) {
	t.Parallel()

	svc, dir := newMigrationConfigServiceFrom(t,
		`{"config_version":7,"telemetry":{"enabled_at":"0001-01-01T00:00:00Z","validity":3600000000000}}`)

	require.NoError(t, migrateConfig(svc, config.NewCredentialsStore(dir)))

	tel, err := svc.DefaultStore().Telemetry()
	require.NoError(t, err)
	assert.True(t, tel.Enabled)
	assert.WithinDuration(t, time.Now(), tel.EnabledAt, time.Minute)
	assert.Equal(t, time.Hour, tel.Validity)
	assert.Equal(t, 8, svc.Model().ConfigVersion)
	assert.True(t, newTestTelemetry(t, svc.DefaultStore()).IsEnabled())

	reloaded := config.New(dir)
	require.NoError(t, cliffStorage.New(reloaded, dir, "config.json").Load())
	assert.Equal(t, 8, reloaded.ConfigVersion)
	assert.True(t, reloaded.Telemetry.Enabled)
}

func TestMigrateConfig_leavesDisabledTelemetryAloneOnceMigrated(t *testing.T) {
	t.Parallel()

	svc, dir := newMigrationConfigServiceFrom(t, `{"config_version":8,"telemetry":{"enabled_at":"0001-01-01T00:00:00Z"}}`)

	require.NoError(t, migrateConfig(svc, config.NewCredentialsStore(dir)))

	tel, err := svc.DefaultStore().Telemetry()
	require.NoError(t, err)
	assert.False(t, tel.Enabled)
}

func newTestTelemetry(t *testing.T, store *cliffCfg.DefaultStore) telemetry.Telemetry {
	t.Helper()

	mqtt := fimpgo.NewMqttTransport("tcp://localhost:1883", "easee-test", "", "", true, 1, 1, func(error) {})

	tel, err := telemetry.New(mqtt, fimptype.EaseeRn, store, "1.0.0")
	require.NoError(t, err)

	return tel
}
