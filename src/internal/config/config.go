package config

import (
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/storage"
	"github.com/michalkurzeja/go-clock"
)

// Config is a model containing all application configuration settings.
type Config struct {
	config.Default
	Credentials

	EaseeBaseURL    string `json:"easeeBaseURL"`
	PollingInterval string `json:"pollingInterval"`
}

// New creates new instance of a configuration object.
func New(workDir string) *Config {
	return &Config{
		Default: config.NewDefault(workDir),
	}
}

// Factory is a factory method returning the configuration object without default settings.
func Factory() interface{} {
	return &Config{}
}

// Credentials represent Easee API credentials.
type Credentials struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

// Empty checks if credentials are empty.
func (c Credentials) Empty() bool {
	return c == Credentials{}
}

// Expired checks if credentials are expired.
func (c Credentials) Expired() bool {
	return clock.Now().UTC().After(c.ExpiresAt)
}

// Service is a configuration service responsible for:
// - providing concurrency safe access to settings
// - persistence of settings
type Service struct {
	storage.Storage
	lock *sync.RWMutex
}

// NewService creates a new configuration service.
func NewService(storage storage.Storage) *Service {
	return &Service{
		Storage: storage,
		lock:    &sync.RWMutex{},
	}
}

// GetEaseeBaseURL allows to safely access a configuration setting.
func (cs *Service) GetEaseeBaseURL() string {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Storage.Model().(*Config).EaseeBaseURL
}

// SetLogLevel allows to safely set and persist configuration settings.
func (cs *Service) SetLogLevel(logLevel string) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Storage.Model().(*Config).ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Storage.Model().(*Config).LogLevel = logLevel

	return cs.Storage.Save()
}

// GetCredentials allows to safely access a configuration setting.
func (cs *Service) GetCredentials() Credentials {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Storage.Model().(*Config).Credentials
}

// SetCredentials allows to safely set and persist configuration settings.
func (cs *Service) SetCredentials(accessToken, refreshToken string, expirationInSeconds int) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Storage.Model().(*Config).ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Storage.Model().(*Config).Credentials = Credentials{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    clock.Now().UTC().Add(time.Duration(expirationInSeconds) * time.Second),
	}

	return cs.Storage.Save()
}

// GetPollingInterval allows to safely access a configuration setting.
func (cs *Service) GetPollingInterval() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	duration, err := time.ParseDuration(cs.Storage.Model().(*Config).PollingInterval)
	if err != nil {
		return 30 * time.Second
	}

	return duration
}
