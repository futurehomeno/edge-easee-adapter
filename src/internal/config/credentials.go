package config

import (
	"fmt"
	"sync"

	"github.com/futurehomeno/cliffhanger/auth"
	"github.com/futurehomeno/cliffhanger/storage"
	log "github.com/sirupsen/logrus"
)

const credentialsFileName = "secrets.json"

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

func (s *CredentialsStore) SetCredentials(credentials Credentials) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	*s.storage.Model() = credentials

	return s.storage.Save()
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

	credentials := cfg.Credentials

	if credentials.AccessTokenExpiresAt.IsZero() || credentials.RefreshTokenExpiresAt.IsZero() {
		accessExpiresAt, err := auth.TokenExpirationDate(credentials.AccessToken)
		if err != nil {
			return fmt.Errorf("access token expiration time err: %w", err)
		}

		refreshExpiresAt, err := auth.TokenExpirationDate(credentials.RefreshToken)
		if err != nil {
			return fmt.Errorf("refresh token expiration time err: %w", err)
		}

		credentials.AccessTokenExpiresAt = accessExpiresAt
		credentials.RefreshTokenExpiresAt = refreshExpiresAt
	}

	if err := store.SetCredentials(credentials); err != nil {
		return fmt.Errorf("store credentials err: %w", err)
	}

	cfg.Credentials = Credentials{}

	log.Info("[config] Move credentials to the secrets storage")

	return nil
}
