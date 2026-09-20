package cmd

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

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

// legacyConfig280 is the data/config.json a v2.8.0 hub leaves behind: config_version 3 (its
// last migration), the packaged 2.8.0 defaults and the session tokens still inline.
const legacyConfig280 = `{
  "config_version": 3,
  "mqtt_server_uri": "tcp://localhost:1883",
  "mqtt_client_id_prefix": "easee",
  "work_dir": "/opt/thingsplex/easee",
  "log_file": "/var/log/thingsplex/easee/easee.log",
  "log_level": "info",
  "log_format": "budzik",
  "accessToken": %q,
  "refreshToken": %q,
  "easeeBaseURL2": "https://api.easee.com",
  "pollingInterval": "10m",
  "energyLifetimeInterval": "10s",
  "offered_current_wait_time": "15s",
  "httpTimeout": "30s",
  "signalR": {
    "baseURL": "https://streams.easee.com",
    "initialBackoff": "10s",
    "repeatedBackoff": "30s",
    "finalBackoff": "2m",
    "initialFailureCount": 2,
    "repeatedFailureCount": 3
  },
  "auth_backoff": {"initialBackoff": "1m", "repeatedBackoff": "5m", "finalBackoff": "10m"}
}`

// The real upgrade path: a v2.8.0 config runs 3->6 in one boot and the result must be what the
// service reads back from disk, not only what the in-memory model shows.
func TestMigrateConfig_upgradesFrom280(t *testing.T) {
	t.Parallel()

	accessExp := time.Date(2026, time.September, 8, 12, 0, 0, 0, time.UTC)
	refreshExp := accessExp.AddDate(0, 3, 0)
	access, refresh := jwtWithExp(accessExp), jwtWithExp(refreshExp)

	svc, dir := newMigrationConfigServiceFrom(t, fmt.Sprintf(legacyConfig280, access, refresh))
	credentials := config.NewCredentialsStore(dir)
	require.NoError(t, credentials.Load())

	require.NoError(t, migrateConfig(svc, credentials))

	want := config.Credentials{
		AccessToken: access, RefreshToken: refresh,
		AccessTokenExpiresAt: accessExp, RefreshTokenExpiresAt: refreshExp,
	}

	assert.Equal(t, 6, svc.Model().ConfigVersion)
	assert.Equal(t, 20*time.Second, svc.OfferedCurrentWaitTime())
	assert.Equal(t, 10*time.Minute, svc.SignalRFinalBackoff())
	assert.Equal(t, 2*time.Hour, svc.AuthenticatorMaxUnauthorized(), "untouched 2.8.0 settings survive")
	assert.True(t, svc.Model().Empty(), "the config copy of the tokens is dropped")
	assert.Equal(t, want, credentials.Credentials(), "expiries are backfilled from the JWT claims")

	// What the next boot loads.
	reloaded := config.New(dir)
	require.NoError(t, cliffStorage.New(reloaded, dir, "config.json").Load())
	assert.Equal(t, 6, reloaded.ConfigVersion)
	assert.True(t, reloaded.Empty())

	reloadedSecrets := config.NewCredentialsStore(dir)
	require.NoError(t, reloadedSecrets.Load())
	assert.Equal(t, want, reloadedSecrets.Credentials())

	info, err := os.Stat(filepath.Join(dir, "data", "secrets.json"))
	require.NoError(t, err)
	assert.Zero(t, info.Mode().Perm()&0o007, "secrets.json must be closed to others")

	require.NoError(t, migrateConfig(svc, credentials), "a second boot is a no-op")
	assert.Equal(t, want, credentials.Credentials())
}

// jwtWithExp builds an unsigned JWT carrying only an exp claim - all TokenExpirationDate reads.
func jwtWithExp(exp time.Time) string {
	seg := func(v string) string { return base64.RawURLEncoding.EncodeToString([]byte(v)) }

	return seg(`{"alg":"none","typ":"JWT"}`) + "." + seg(`{"exp":`+strconv.FormatInt(exp.Unix(), 10)+`}`) + ".sig"
}

func newMigrationConfigService(t *testing.T, version int) (*config.Service, string) {
	t.Helper()

	return newMigrationConfigServiceFrom(t, `{"config_version":`+strconv.Itoa(version)+`,"accessToken":"a","refreshToken":"r"}`)
}

func newMigrationConfigServiceFrom(t *testing.T, body string) (*config.Service, string) {
	t.Helper()

	dir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "data"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data", "config.json"), []byte(body), 0o600))

	cfg := config.New(dir)
	svc := config.NewService(cliffStorage.New(cfg, dir, "config.json"))

	require.NoError(t, svc.Load())

	return svc, dir
}

// The crash window: the v5->v6 step's Save renames the token-bearing config.json to a
// world-readable data/config.json.bak, and the process dies before the backup is dropped. The
// next boot finds config_version 6, applies nothing, and must still clear the leftover.
func TestMigrateConfig_dropsBackupLeftByAnInterruptedMigration(t *testing.T) {
	t.Parallel()

	svc, dir := newMigrationConfigServiceFrom(t, `{"config_version":6}`)

	backup := filepath.Join(dir, "data", "config.json.bak")
	// 0644 is the mode cliffhanger's makeBackup chmods the backup to; reproducing the leak needs it.
	require.NoError(t, os.WriteFile(backup, []byte(`{"config_version":5,"accessToken":"leaked"}`), 0o644)) //nolint:gosec

	require.NoError(t, migrateConfig(svc, config.NewCredentialsStore(dir)))

	_, err := os.Stat(backup)
	assert.True(t, os.IsNotExist(err), "a backup surviving an interrupted migration still leaks the tokens")
}

// Steady state: config.json corrupted by an interrupted runtime Save, a good backup beside it, and
// no migration step left to run. Load() recovers the backup in memory only, so the drop has to
// leave that copy on disk or the next boot has nothing to load.
func TestMigrateConfig_restoresTheBackupOverACorruptConfig(t *testing.T) {
	t.Parallel()

	svc, dir := newMigrationConfigService(t, 5)
	require.NoError(t, migrateConfig(svc, config.NewCredentialsStore(dir)))

	configPath := filepath.Join(dir, "data", "config.json")
	backup := filepath.Join(dir, "data", "config.json.bak")

	require.NoError(t, os.Rename(configPath, backup))
	good, err := os.ReadFile(backup) //nolint:gosec
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, []byte(`{tru`), 0o600))

	recovered := config.NewService(cliffStorage.New(config.New(dir), dir, "config.json"))
	require.NoError(t, recovered.Load(), "the corrupt config must be recovered from the backup")
	require.NoError(t, migrateConfig(recovered, config.NewCredentialsStore(dir)))

	assert.NoFileExists(t, backup)

	restored, err := os.ReadFile(configPath) //nolint:gosec
	require.NoError(t, err)
	assert.Equal(t, string(good), string(restored), "the backup is moved into place, not rewritten from memory")

	reloaded := config.New(dir)
	require.NoError(t, cliffStorage.New(reloaded, dir, "config.json").Load(), "the next boot must still load the config")
	assert.Equal(t, recovered.Model().ConfigVersion, reloaded.ConfigVersion)
}
