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
		require.NoError(t, store.RefreshCredentials(fresh, "old-refresh"))

		assert.True(t, store.Credentials().Empty(), "logout must not be undone by an in-flight refresh")
	})

	// Logging back in between the exchange and its result leaves a non-empty store, so
	// emptiness alone would let the old session overwrite the new one - with another
	// account's tokens, if the user switched.
	t.Run("a refresh from a replaced session is dropped", func(t *testing.T) {
		t.Parallel()

		store := newCredentialsStore(t, config.Credentials{AccessToken: "a", RefreshToken: "a-refresh"})
		session := config.Credentials{AccessToken: "b", RefreshToken: "b-refresh"}

		require.NoError(t, store.ClearCredentials())
		require.NoError(t, store.SetCredentials(session))

		require.NoError(t, store.RefreshCredentials(fresh, "a-refresh"))

		assert.Equal(t, session, store.Credentials(), "the newer session must survive")
	})

	t.Run("a refresh during a live session is stored", func(t *testing.T) {
		t.Parallel()

		store := newCredentialsStore(t, config.Credentials{AccessToken: "old", RefreshToken: "old-refresh"})

		require.NoError(t, store.RefreshCredentials(fresh, "old-refresh"))

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

// A legacy install predates the persisted expiry times, so they are read back from the JWTs.
// A token whose claim will not parse must not strand the migration: it can never parse on a
// retry, so failing here would block the version bump for good and force a re-login.
func TestMigrateCredentials_MigratesTokensWithoutExpiryClaim(t *testing.T) {
	t.Parallel()

	unparseable, alsoUnparseable := "not-a-jwt", "not-a-jwt-either"

	legacy := config.Credentials{
		AccessToken:  unparseable,
		RefreshToken: alsoUnparseable,
	}

	store := newCredentialsStore(t, config.Credentials{})
	cfg := &config.Config{Credentials: legacy}

	require.NoError(t, config.MigrateCredentials(cfg, store))

	assert.Equal(t, legacy, store.Credentials(), "both tokens migrate with the expiries left zero")
	assert.True(t, cfg.Empty(), "the config copy is dropped, so the version can advance")
}
