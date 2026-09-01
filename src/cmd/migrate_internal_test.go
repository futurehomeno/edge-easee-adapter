package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	cliffStorage "github.com/futurehomeno/cliffhanger/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// A failed credentials migration must surface, not be swallowed: the app reads the secrets
// store alone to decide it is configured, so continuing past a failed write brings an
// authenticated install up as logged out while the tokens still sit in config.json.
func TestMigrateConfig_failedCredentialsMigrationSurfaces(t *testing.T) {
	t.Parallel()

	svc, _ := newMigrationConfigService(t, 5)

	store := config.NewCredentialsStoreWithStorage(failingSecrets{})

	err := migrateConfig(svc, store)

	require.Error(t, err, "a failed credentials migration must be returned")
	assert.Contains(t, err.Error(), "migrate config")
}

func TestMigrateConfig_credentialsMigrationSucceeds(t *testing.T) {
	t.Parallel()

	svc, dir := newMigrationConfigService(t, 5)

	require.NoError(t, migrateConfig(svc, config.NewCredentialsStore(dir)))
}

// failingSecrets stands in for a secrets file that cannot be written - a full disk or a
// permission fault during the upgrade.
type failingSecrets struct {
	credentials config.Credentials
}

func (f failingSecrets) Load() error                { return nil }
func (f failingSecrets) Save() error                { return errors.New("write secrets: disk full") }
func (f failingSecrets) Reset() error               { return nil }
func (f failingSecrets) Model() *config.Credentials { return &f.credentials }

func newMigrationConfigService(t *testing.T, version int) (*config.Service, string) {
	t.Helper()

	dir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "data"), 0o750))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "data", "config.json"),
		[]byte(`{"config_version":`+strconv.Itoa(version)+`,"accessToken":"a","refreshToken":"r"}`),
		0o600,
	))

	cfg := config.New(dir)
	svc := config.NewService(cliffStorage.New(cfg, dir, "config.json"))

	require.NoError(t, svc.Load())

	return svc, dir
}
