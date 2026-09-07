package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/futurehomeno/cliffhanger/auth"
	"github.com/futurehomeno/cliffhanger/storage"
	log "github.com/sirupsen/logrus"
)

const (
	credentialsFileName = "secrets.json"
	configFileName      = "config.json"
	backupExtension     = ".bak"
)

// CredentialsStore persists the Easee tokens outside the world-readable config.json.
// storage.Model() hands out the live model unlocked, so every access is guarded here:
// login, logout and the token refresh run on different goroutines.
type CredentialsStore struct {
	lock    sync.RWMutex
	storage storage.Storage[*Credentials]
}

func NewCredentialsStore(workDir string) *CredentialsStore {
	return NewCredentialsStoreWithStorage(storage.NewSecrets(&Credentials{}, workDir, credentialsFileName))
}

func NewCredentialsStoreWithStorage(s storage.Storage[*Credentials]) *CredentialsStore {
	return &CredentialsStore{storage: s}
}

func (s *CredentialsStore) Load() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.storage.Load()
}

func (s *CredentialsStore) Credentials() Credentials {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return *s.storage.Model()
}

// SetCredentials deliberately keeps the new credentials in memory when the write fails.
// Easee rotates the refresh token on every exchange, so rolling back would leave the
// process holding a token the API has already retired - a transient disk error would
// then cost the session. Disk catches up on the next successful save.
func (s *CredentialsStore) SetCredentials(credentials Credentials) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	*s.storage.Model() = credentials

	return s.storage.Save()
}

// RefreshCredentials stores tokens produced by a background refresh, provided the session it
// started from is still the one on disk. expected is the refresh token the exchange began
// with: a logout empties the store and a new login replaces it, and in both cases the refresh
// is finishing on behalf of a session that no longer exists. Writing it would resurrect a
// logged-out hub, or overwrite the newer session - with another account's tokens, if the user
// logged back in elsewhere. The comparison shares this store's lock with SetCredentials and
// ClearCredentials, so the three cannot interleave.
func (s *CredentialsStore) RefreshCredentials(credentials Credentials, expected string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if current := s.storage.Model(); current.Empty() || current.RefreshToken != expected {
		log.Info("[auth] Skip refreshed credentials, the session changed meanwhile")

		return nil
	}

	// Written to the model only once the save succeeded. The framework retries a refused
	// rotation through persistUnsaved, which reads this store back to decide what is still
	// outstanding - so a model updated ahead of a failed write reports a token the disk does
	// not have, and the retry concludes there is nothing left to persist. SetCredentials keeps
	// the opposite behaviour deliberately: a login's session stays usable until the restart.
	previous := *s.storage.Model()
	*s.storage.Model() = credentials

	if err := s.storage.Save(); err != nil {
		*s.storage.Model() = previous

		return err
	}

	return nil
}

func (s *CredentialsStore) ClearCredentials() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.storage.Reset()
}

// MigrateCredentials moves the tokens out of config.json into the secrets store, backfilling
// the expiry times that installs predating them never persisted. A failed write is returned so
// the config version does not advance and the next startup retries with the tokens still in place.
func MigrateCredentials(cfg *Config, store *CredentialsStore) error {
	if cfg.Empty() {
		return nil
	}

	// An earlier run may have written the secrets and then failed to save the version bump,
	// leaving this migration to run again. Its tokens have been refreshed since, and Easee
	// retires a refresh token as soon as it is exchanged, so restoring the pair the config
	// still carries would cost the session.
	if !store.Credentials().Empty() {
		cfg.Credentials = Credentials{}

		log.Info("[config] Credentials already in the secrets storage, drop the config copy")

		return nil
	}

	credentials := cfg.Credentials

	// An unreadable expiry claim is left zero rather than failing the migration: retrying it
	// can never parse a token that did not parse before, and zero is the value the framework
	// already treats as "refresh now" for the access token and "no local check" for the
	// refresh one - so the session recovers on the first exchange instead of needing a login.
	if credentials.AccessTokenExpiresAt.IsZero() {
		credentials.AccessTokenExpiresAt, _ = auth.TokenExpirationDate(credentials.AccessToken)
	}

	if credentials.RefreshTokenExpiresAt.IsZero() {
		credentials.RefreshTokenExpiresAt, _ = auth.TokenExpirationDate(credentials.RefreshToken)
	}

	if err := store.SetCredentials(credentials); err != nil {
		return fmt.Errorf("store credentials err: %w", err)
	}

	cfg.Credentials = Credentials{}

	log.Info("[config] Move credentials to the secrets storage")

	return nil
}

// DropConfigBackup removes the config backup cliffhanger's Save() leaves behind. The rename in
// makeBackup carries the pre-migration config over, tokens and all, and chmods it to the config
// store's world-readable 0644 - so the credentials this migration just moved into the 0640
// secrets file stay legible to any local user in data/config.json.bak. Best-effort: the backup
// only serves corruption recovery, and a stale one is worth less than the leak.
func DropConfigBackup(workDir string) {
	path := filepath.Join(workDir, "data", configFileName+backupExtension)

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Warnf("[config] Remove %s. err: %v", path, err)
	}
}
