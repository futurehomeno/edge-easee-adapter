package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	cliffCfg "github.com/futurehomeno/cliffhanger/config"
	cliffStorage "github.com/futurehomeno/cliffhanger/storage"
	"github.com/futurehomeno/cliffhanger/telemetry"
	telemetryTypes "github.com/futurehomeno/cliffhanger/telemetry/types"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// newTelemetryStore builds a config store from the packaged defaults, which is what a fresh
// install and an upgrade from a config predating telemetry both start from.
func newTelemetryStore(t *testing.T) *cliffCfg.DefaultStore {
	t.Helper()

	workDir := t.TempDir()
	body, err := os.ReadFile("../../package/debian/usr/share/futurehome/easee/defaults/config.json")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(workDir, "defaults"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "defaults", "config.json"), body, 0o600)) //nolint:gosec

	cfg := config.New(workDir)
	cfgSvc := config.NewService(cliffStorage.New(cfg, workDir, "config.json"))
	require.NoError(t, cfgSvc.Load())

	return cfgSvc.DefaultStore()
}

// TestSeedTelemetryDisabled_keepsTelemetryOffUntilTheCloudEnablesIt pins the opt-in contract.
// cliffhanger's telemetry.New seeds a missing telemetry block as enabled for 30 days, so without
// the seed a fresh install would start reporting panics before the cloud ever asked for them.
func TestSeedTelemetryDisabled_keepsTelemetryOffUntilTheCloudEnablesIt(t *testing.T) {
	t.Parallel()

	store := newTelemetryStore(t)

	// The packaged defaults ship no telemetry block - this is the state that triggers the seed.
	_, err := store.Telemetry()
	require.Error(t, err)

	require.NoError(t, seedTelemetryDisabled(store))

	cfg, err := store.Telemetry()
	require.NoError(t, err)
	assert.False(t, cfg.Enabled)

	// The regression guard: constructing the reporter must not flip it on.
	mqtt := fimpgo.NewMqttTransport("tcp://localhost:1883", "easee-test", "", "", true, 1, 1, func(error) {})

	reporter, err := telemetry.New(mqtt, fimptype.EaseeRn, store, "1.0.0")
	require.NoError(t, err)

	assert.False(t, reporter.IsEnabled())
}

// TestSeedTelemetryDisabled_leavesAnExistingBlockAlone makes sure the seed never overwrites a
// choice the cloud already pushed.
func TestSeedTelemetryDisabled_leavesAnExistingBlockAlone(t *testing.T) {
	t.Parallel()

	store := newTelemetryStore(t)

	enabled := &telemetryTypes.TelemetryConfig{Enabled: true, EnabledAt: time.Now(), Validity: time.Hour}
	require.NoError(t, store.SetTelemetry(enabled))

	require.NoError(t, seedTelemetryDisabled(store))

	cfg, err := store.Telemetry()
	require.NoError(t, err)
	assert.True(t, cfg.Enabled)
}
