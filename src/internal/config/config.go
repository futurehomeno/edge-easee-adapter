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

	EaseeBaseURL                 string  `json:"easeeBaseURL"`
	EaseeBackoff                 string  `json:"easeeBackoff"`
	PollingInterval              string  `json:"pollingInterval"`
	SlowChargingCurrentInAmperes float64 `json:"slowChargingCurrentInAmperes"`
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
// - persistence of settings.
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

	return cs.Storage.Model().(*Config).EaseeBaseURL //nolint:forcetypeassert
}

// SetLogLevel allows to safely set and persist configuration settings.
func (cs *Service) SetLogLevel(logLevel string) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Storage.Model().(*Config).ConfiguredAt = time.Now().Format(time.RFC3339) //nolint:forcetypeassert
	cs.Storage.Model().(*Config).LogLevel = logLevel                            //nolint:forcetypeassert

	return cs.Storage.Save()
}

// GetCredentials allows to safely access a configuration setting.
func (cs *Service) GetCredentials() Credentials {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Storage.Model().(*Config).Credentials //nolint:forcetypeassert
}

// SetCredentials allows to safely set and persist configuration settings.
func (cs *Service) SetCredentials(accessToken, refreshToken string, expirationInSeconds int) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Storage.Model().(*Config).ConfiguredAt = time.Now().Format(time.RFC3339) //nolint:forcetypeassert
	cs.Storage.Model().(*Config).Credentials = Credentials{                     //nolint:forcetypeassert
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

// SetPollingInterval allows to safely set and persist configuration settings.
func (cs *Service) SetPollingInterval(interval time.Duration) error {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	cs.Storage.Model().(*Config).ConfiguredAt = time.Now().Format(time.RFC3339) //nolint:forcetypeassert
	cs.Storage.Model().(*Config).PollingInterval = interval.String()            //nolint:forcetypeassert

	return cs.Storage.Save()
}

// GetEaseeBackoff allows to safely access a configuration setting.
func (cs *Service) GetEaseeBackoff() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	duration, err := time.ParseDuration(cs.Storage.Model().(*Config).EaseeBackoff)
	if err != nil {
		return 4 * time.Second
	}

	return duration
}

// SetEaseeBackoff allows to safely set and persist configuration settings.
func (cs *Service) SetEaseeBackoff(backoff time.Duration) error {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	cs.Storage.Model().(*Config).ConfiguredAt = time.Now().Format(time.RFC3339) //nolint:forcetypeassert
	cs.Storage.Model().(*Config).EaseeBackoff = backoff.String()                //nolint:forcetypeassert

	return cs.Storage.Save()
}

// GetSlowChargingCurrentInAmperes allows to safely access a configuration setting.
func (cs *Service) GetSlowChargingCurrentInAmperes() float64 {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Storage.Model().(*Config).SlowChargingCurrentInAmperes //nolint:forcetypeassert
}

// SetSlowChargingCurrentInAmperes allows to safely set and persist configuration settings.
func (cs *Service) SetSlowChargingCurrentInAmperes(current float64) error {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	cs.Storage.Model().(*Config).ConfiguredAt = time.Now().Format(time.RFC3339) //nolint:forcetypeassert
	cs.Storage.Model().(*Config).SlowChargingCurrentInAmperes = current         //nolint:forcetypeassert

	return cs.Storage.Save()
}
