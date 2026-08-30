package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	mockedstorage "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/storage"
)

// newMigrationFixture builds a config sitting at version 5 with tokens still in config.json,
// which is exactly the state the 5->6 credentials migration exists to move.
func newMigrationFixture(secretsSaveErr error) (*config.Service, *config.CredentialsStore, *config.Config) {
	cfg := config.Factory()
	cfg.ConfigVersion = 5
	cfg.Credentials = config.Credentials{AccessToken: "access", RefreshToken: "refresh"}

	cfgStorage := &mockedstorage.Storage[*config.Config]{}
	cfgStorage.On("Model").Return(cfg)
	cfgStorage.On("Save").Return(nil)

	secrets := &mockedstorage.Storage[*config.Credentials]{}
	secrets.On("Model").Return(&config.Credentials{})
	secrets.On("Save").Return(secretsSaveErr)

	return config.NewService(cfgStorage), config.NewCredentialsStoreWithStorage(secrets), cfg
}

// TestMigrateConfig_failedCredentialsMigrationSurfaces pins that a failed 5->6 migration is
// reported to the caller. Swallowing it booted the adapter logged out - the credentials store
// stays empty while the tokens are still sitting in config.json - and the version bump never
// lands, so only a hard failure gets the migration retried on the next start.
func TestMigrateConfig_failedCredentialsMigrationSurfaces(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("secrets write failed")
	cfgSvc, credentials, cfg := newMigrationFixture(wantErr)

	err := migrateConfig(cfgSvc, credentials)

	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)

	// The version must not advance past a migration that did not apply, so the next start
	// retries it. SetCredentials updates the in-memory model before the failing write, so
	// only the on-disk secrets are missing - which is why continuing the boot is not safe.
	assert.Equal(t, 5, cfg.ConfigVersion)
}

// TestMigrateConfig_credentialsMigrationSucceeds is the counterpart: the happy path still
// moves the tokens and advances the version.
func TestMigrateConfig_credentialsMigrationSucceeds(t *testing.T) {
	t.Parallel()

	cfgSvc, credentials, cfg := newMigrationFixture(nil)

	require.NoError(t, migrateConfig(cfgSvc, credentials))

	assert.Equal(t, 6, cfg.ConfigVersion)
	assert.Equal(t, "access", credentials.Credentials().AccessToken)
	assert.Equal(t, "refresh", credentials.Credentials().RefreshToken)
	assert.True(t, cfg.Empty())
}
