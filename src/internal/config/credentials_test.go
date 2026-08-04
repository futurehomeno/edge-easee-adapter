package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/fakes"
)

func newCredentialsStore(t *testing.T, credentials config.Credentials) *config.CredentialsStore {
	t.Helper()

	return config.NewCredentialsStoreWithStorage(
		fakes.NewConfigStorage(t, &credentials, func() *config.Credentials { return &config.Credentials{} }),
	)
}

// A refresh that started before an explicit logout completes after it. Writing its tokens
// would leave a hub that reports itself logged out but authenticates again on next boot.
func TestCredentialsStore_RefreshDoesNotResurrectClearedCredentials(t *testing.T) {
	t.Parallel()

	fresh := config.Credentials{AccessToken: "new-access", RefreshToken: "new-refresh"}

	t.Run("a refresh after logout is dropped", func(t *testing.T) {
		t.Parallel()

		store := newCredentialsStore(t, config.Credentials{AccessToken: "old", RefreshToken: "old-refresh"})

		require.NoError(t, store.ClearCredentials())
		require.NoError(t, store.RefreshCredentials(fresh))

		assert.True(t, store.Credentials().Empty(), "logout must not be undone by an in-flight refresh")
	})

	t.Run("a refresh during a live session is stored", func(t *testing.T) {
		t.Parallel()

		store := newCredentialsStore(t, config.Credentials{AccessToken: "old", RefreshToken: "old-refresh"})

		require.NoError(t, store.RefreshCredentials(fresh))

		assert.Equal(t, fresh, store.Credentials())
	})

	t.Run("login still writes unconditionally", func(t *testing.T) {
		t.Parallel()

		store := newCredentialsStore(t, config.Credentials{})

		require.NoError(t, store.SetCredentials(fresh))

		assert.Equal(t, fresh, store.Credentials())
	})
}

// If the 5->6 migration wrote the secrets but the version bump failed to save, it runs again
// on the next boot - and must not restore the stale tokens the configuration still carries.
func TestMigrateCredentials_KeepsAlreadyMigratedSecrets(t *testing.T) {
	t.Parallel()

	stale := config.Credentials{
		AccessToken:           "stale-access",
		RefreshToken:          "stale-refresh",
		AccessTokenExpiresAt:  time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		RefreshTokenExpiresAt: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
	}
	refreshed := config.Credentials{
		AccessToken:           "refreshed-access",
		RefreshToken:          "refreshed-refresh",
		AccessTokenExpiresAt:  time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC),
		RefreshTokenExpiresAt: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
	}

	t.Run("already migrated secrets survive a re-run", func(t *testing.T) {
		t.Parallel()

		store := newCredentialsStore(t, refreshed)
		cfg := &config.Config{Credentials: stale}

		require.NoError(t, config.MigrateCredentials(cfg, store))

		assert.Equal(t, refreshed, store.Credentials(), "a re-run must not roll the session back")
		assert.True(t, cfg.Empty(), "the config copy is dropped either way")
	})

	t.Run("empty secrets are migrated", func(t *testing.T) {
		t.Parallel()

		store := newCredentialsStore(t, config.Credentials{})
		cfg := &config.Config{Credentials: stale}

		require.NoError(t, config.MigrateCredentials(cfg, store))

		assert.Equal(t, stale, store.Credentials())
		assert.True(t, cfg.Empty())
	})
}
