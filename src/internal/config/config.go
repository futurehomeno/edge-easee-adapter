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

	EaseeBaseURL                 string     `json:"easeeBaseURL2"`
	PollingInterval              string     `json:"pollingInterval"`
	SlowChargingCurrentInAmperes float64    `json:"slowChargingCurrentInAmperes"`
	HTTPTimeout                  string     `json:"httpTimeout"`
	SignalR                      SignalR    `json:"signalR"`
	Backoff                      BackoffCfg `json:"backoff"`
}

// New creates new instance of a configuration object.
func New(workDir string) *Config {
	return &Config{
		Default: config.NewDefault(workDir),
	}
}

// Factory is a factory method returning the configuration object without default settings.
func Factory() *Config {
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

// SignalR represents SignalR configuration settings.
type SignalR struct {
	BaseURL             string `json:"baseURL"`
	ConnCreationTimeout string `json:"connCreationTimeout"`
	KeepAliveInterval   string `json:"keepAliveInterval2"`
	TimeoutInterval     string `json:"timeoutInterval2"`
	InvokeTimeout       string `json:"invokeTimeout"`
}

// Service is a configuration service responsible for:
// - providing concurrency safe access to settings
// - persistence of settings.
type Service struct {
	storage.Storage[*Config]
	lock *sync.RWMutex
}

// BackoffCfg represents values used to configure
// reconnecting hook when http errors occur.
type BackoffCfg struct {
	Length      string `json:"length"`
	MaxAttempts int    `json:"maxAttempts"`
}

// NewService creates a new configuration service.
func NewService(storage storage.Storage[*Config]) *Service {
	return &Service{
		Storage: storage,
		lock:    &sync.RWMutex{},
	}
}

// GetBackoffCfg allows to safely access backoff settings.
func (cs *Service) GetBackoffCfg() BackoffCfg {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Storage.Model().Backoff
}

// GetWorkDir allows to safely access a configuration setting.
func (cs *Service) GetWorkDir() string {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Storage.Model().WorkDir
}

// GetEaseeBaseURL allows to safely access a configuration setting.
func (cs *Service) GetEaseeBaseURL() string {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Storage.Model().EaseeBaseURL
}

// SetEaseeBaseURL allows to safely set and persist configuration settings.
func (cs *Service) SetEaseeBaseURL(url string) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Storage.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Storage.Model().EaseeBaseURL = url

	return cs.Storage.Save()
}

// GetLogLevel allows to safely access a configuration setting.
func (cs *Service) GetLogLevel() string {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Storage.Model().LogLevel
}

// SetLogLevel allows to safely set and persist configuration settings.
func (cs *Service) SetLogLevel(logLevel string) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Storage.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Storage.Model().LogLevel = logLevel

	return cs.Storage.Save()
}

// GetCredentials allows to safely access a configuration setting.
func (cs *Service) GetCredentials() Credentials {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Storage.Model().Credentials
}

// SetCredentials allows to safely set and persist configuration settings.
func (cs *Service) SetCredentials(accessToken, refreshToken string, expirationInSeconds int) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Storage.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Storage.Model().Credentials = Credentials{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    clock.Now().UTC().Add(time.Duration(expirationInSeconds) * time.Second),
	}

	return cs.Storage.Save()
}

// ClearCredentials resets credentials to empty.
func (cs *Service) ClearCredentials() error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Storage.Model().Credentials = Credentials{}

	return cs.Storage.Save()
}

// GetPollingInterval allows to safely access a configuration setting.
func (cs *Service) GetPollingInterval() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	duration, err := time.ParseDuration(cs.Storage.Model().PollingInterval)
	if err != nil {
		return 30 * time.Second
	}

	return duration
}

// SetPollingInterval allows to safely set and persist configuration settings.
func (cs *Service) SetPollingInterval(interval time.Duration) error {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	cs.Storage.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Storage.Model().PollingInterval = interval.String()

	return cs.Storage.Save()
}

// GetSlowChargingCurrentInAmperes allows to safely access a configuration setting.
func (cs *Service) GetSlowChargingCurrentInAmperes() float64 {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Storage.Model().SlowChargingCurrentInAmperes
}

// SetSlowChargingCurrentInAmperes allows to safely set and persist configuration settings.
func (cs *Service) SetSlowChargingCurrentInAmperes(current float64) error {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	cs.Storage.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Storage.Model().SlowChargingCurrentInAmperes = current

	return cs.Storage.Save()
}

// GetHTTPTimeout allows to safely access a configuration setting.
func (cs *Service) GetHTTPTimeout() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	timeout, err := time.ParseDuration(cs.Storage.Model().HTTPTimeout)
	if err != nil {
		return 30 * time.Second
	}

	return timeout
}

// SetHTTPTimeout allows to safely set and persist configuration settings.
func (cs *Service) SetHTTPTimeout(timeout time.Duration) error {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	cs.Storage.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Storage.Model().HTTPTimeout = timeout.String()

	return cs.Storage.Save()
}

// GetSignalRBaseURL allows to safely access a configuration setting.
func (cs *Service) GetSignalRBaseURL() string {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Storage.Model().SignalR.BaseURL
}

// SetSignalRBaseURL allows to safely set and persist configuration settings.
func (cs *Service) SetSignalRBaseURL(url string) error {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	cs.Storage.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Storage.Model().SignalR.BaseURL = url

	return cs.Storage.Save()
}

// GetSignalRConnCreationTimeout allows to safely access a configuration setting.
func (cs *Service) GetSignalRConnCreationTimeout() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	timeout, err := time.ParseDuration(cs.Storage.Model().SignalR.ConnCreationTimeout)
	if err != nil {
		return 30 * time.Second
	}

	return timeout
}

// SetSignalRConnCreationTimeout allows to safely set and persist configuration settings.
func (cs *Service) SetSignalRConnCreationTimeout(timeout time.Duration) error {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	cs.Storage.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Storage.Model().SignalR.ConnCreationTimeout = timeout.String()

	return cs.Storage.Save()
}

// GetSignalRKeepAliveInterval allows to safely access a configuration setting.
func (cs *Service) GetSignalRKeepAliveInterval() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	interval, err := time.ParseDuration(cs.Storage.Model().SignalR.KeepAliveInterval)
	if err != nil {
		return 30 * time.Second
	}

	return interval
}

// SetSignalRKeepAliveInterval allows to safely set and persist configuration settings.
func (cs *Service) SetSignalRKeepAliveInterval(interval time.Duration) error {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	cs.Storage.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Storage.Model().SignalR.KeepAliveInterval = interval.String()

	return cs.Storage.Save()
}

// GetSignalRTimeoutInterval allows to safely access a configuration setting.
func (cs *Service) GetSignalRTimeoutInterval() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	interval, err := time.ParseDuration(cs.Storage.Model().SignalR.TimeoutInterval)
	if err != nil {
		return 1 * time.Minute
	}

	return interval
}

// SetSignalRTimeoutInterval allows to safely set and persist configuration settings.
func (cs *Service) SetSignalRTimeoutInterval(interval time.Duration) error {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	cs.Storage.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Storage.Model().SignalR.TimeoutInterval = interval.String()

	return cs.Storage.Save()
}

// GetSignalRInvokeTimeout allows to safely access a configuration setting.
func (cs *Service) GetSignalRInvokeTimeout() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	timeout, err := time.ParseDuration(cs.Storage.Model().SignalR.InvokeTimeout)
	if err != nil {
		return 10 * time.Second
	}

	return timeout
}

// SetSignalRInvokeTimeout allows to safely set and persist configuration settings.
func (cs *Service) SetSignalRInvokeTimeout(timeout time.Duration) error {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	cs.Storage.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Storage.Model().SignalR.InvokeTimeout = timeout.String()

	return cs.Storage.Save()
}

// GetBackoffLength allows to safely access backoff duration.
func (cs *Service) GetBackoffLength() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	length, err := time.ParseDuration(cs.Storage.Model().Backoff.Length)
	if err != nil {
		return 5 * time.Minute
	}

	return length
}

// SetBackoffLength allows to safely alter backoff length.
func (cs *Service) SetBackoffLength(l time.Duration) error {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	cs.Storage.Model().Backoff.Length = l.String()

	return cs.Storage.Save()
}

// GetBackoffMaxAttempts allows to safely access backoff max attempts.
func (cs *Service) GetBackoffMaxAttempts() int {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Storage.Model().Backoff.MaxAttempts
}

// SetBackoffMaxAttempts allows to safely alter backoff max attempts.
func (cs *Service) SetBackoffMaxAttempts(n int) error {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	cs.Storage.Model().Backoff.MaxAttempts = n

	return cs.Storage.Save()
}
