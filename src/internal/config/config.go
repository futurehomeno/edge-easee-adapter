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
	CurrentWaitDuration          string     `json:"currentWaitDuration"`
	SlowChargingCurrentInAmperes float64    `json:"slowChargingCurrentInAmperes"`
	HTTPTimeout                  string     `json:"httpTimeout"`
	SignalR                      SignalR    `json:"signalR"`
	AuthenticatorBackoff         backoffCfg `json:"authenticatorBackoff"`
	OfferedCurrentWaitTime       string     `json:"offered_current_wait_time"`
	EnergyLifetimeInterval       string     `json:"energyLifetimeInterval"`
}

// New creates new instance of a configuration object.
func New(workDir string) *Config {
	return &Config{
		Default: config.NewDefault(workDir),
	}
}

// NewConfigServiceWithStorage creates a new configuration service.
func NewConfigServiceWithStorage(storage storage.Storage[*Config]) *Service {
	return &Service{
		Storage: storage,
		lock:    &sync.RWMutex{},
	}
}

// Factory is a factory method returning the configuration object without default settings.
func Factory() *Config {
	return &Config{}
}

// Credentials represent Easee API credentials.
type Credentials struct {
	AccessToken           string    `json:"accessToken"`
	RefreshToken          string    `json:"refreshToken"`
	AccessTokenExpiresAt  time.Time `json:"expiresAt,omitzero"`
	RefreshTokenExpiresAt time.Time `json:"refreshTokenExpiresAt,omitzero"`
}

// Empty checks if credentials are empty.
func (c Credentials) Empty() bool {
	return c == Credentials{}
}

// AccessTokenExpired checks if credentials are expired.
func (c Credentials) AccessTokenExpired() bool {
	return clock.Now().After(c.AccessTokenExpiresAt)
}

// RefreshTokenExpired checks if credentials are expired.
func (c Credentials) RefreshTokenExpired() bool {
	return clock.Now().After(c.RefreshTokenExpiresAt)
}

// SignalR represents SignalR configuration settings.
type SignalR struct {
	BaseURL              string `json:"baseURL"`
	ConnCreationTimeout  string `json:"connCreationTimeout"`
	KeepAliveInterval    string `json:"keepAliveInterval2"`
	TimeoutInterval      string `json:"timeoutInterval2"`
	InitialBackoff       string `json:"initialBackoff"`
	RepeatedBackoff      string `json:"repeatedBackoff"`
	FinalBackoff         string `json:"finalBackoff"`
	InitialFailureCount  uint32 `json:"initialFailureCount"`
	RepeatedFailureCount uint32 `json:"repeatedFailureCount"`
	InvokeTimeout        string `json:"invokeTimeout"`
}

// Service is a configuration service responsible for:
// - providing concurrency safe access to settings
// - persistence of settings.
type Service struct {
	storage.Storage[*Config]
	lock *sync.RWMutex
}

// backoffCfg represents a file storage representation of BackoffCfg.
type backoffCfg struct {
	InitialBackoff       string `json:"initialBackoff"`
	RepeatedBackoff      string `json:"repeatedBackoff"`
	FinalBackoff         string `json:"finalBackoff"`
	InitialFailureCount  uint32 `json:"initialFailureCount"`
	RepeatedFailureCount uint32 `json:"repeatedFailureCount"`
}

// BackoffCfg represents values used to configure backoff.
type BackoffCfg struct {
	InitialBackoff       time.Duration
	RepeatedBackoff      time.Duration
	FinalBackoff         time.Duration
	InitialFailureCount  uint32
	RepeatedFailureCount uint32
}

// NewService creates a new configuration service.
func NewService(storage storage.Storage[*Config]) *Service {
	return &Service{
		Storage: storage,
		lock:    &sync.RWMutex{},
	}
}

// GetWorkDir allows to safely access a configuration setting.
func (cs *Service) GetWorkDir() string {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().WorkDir
}

// GetEaseeBaseURL allows to safely access a configuration setting.
func (cs *Service) GetEaseeBaseURL() string {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().EaseeBaseURL
}

// SetEaseeBaseURL allows to safely set and persist configuration settings.
func (cs *Service) SetEaseeBaseURL(url string) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().EaseeBaseURL = url

	return cs.Save()
}

// GetEnergyLifetimeInterval allows to safely access a configuration setting.
func (cs *Service) GetEnergyLifetimeInterval() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	duration, err := time.ParseDuration(cs.Model().EnergyLifetimeInterval)
	if err != nil {
		return 15 * time.Second
	}

	return duration
}

// SetEnergyLifetimeInterval allows to safely set and persist configuration settings.
func (cs *Service) SetEnergyLifetimeInterval(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().EnergyLifetimeInterval = interval.String()

	return cs.Save()
}

// GetLogLevel allows to safely access a configuration setting.
func (cs *Service) GetLogLevel() string {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().LogLevel
}

// SetLogLevel allows to safely set and persist configuration settings.
func (cs *Service) SetLogLevel(logLevel string) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().LogLevel = logLevel

	return cs.Save()
}

// GetCredentials allows to safely access a configuration setting.
func (cs *Service) GetCredentials() Credentials {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().Credentials
}

// SetCredentials allows to safely set and persist configuration settings.
func (cs *Service) SetCredentials(credentials Credentials) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().Credentials = credentials

	return cs.Save()
}

// ClearCredentials resets credentials to empty.
func (cs *Service) ClearCredentials() error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().Credentials = Credentials{}

	return cs.Save()
}

// GetPollingInterval allows to safely access a configuration setting.
func (cs *Service) GetPollingInterval() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	duration, err := time.ParseDuration(cs.Model().PollingInterval)
	if err != nil {
		return 30 * time.Second
	}

	return duration
}

// SetPollingInterval allows to safely set and persist configuration settings.
func (cs *Service) SetPollingInterval(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().PollingInterval = interval.String()

	return cs.Save()
}

// GetCurrentWaitDuration allows to safely access a configuration setting.
func (cs *Service) GetCurrentWaitDuration() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	duration, err := time.ParseDuration(cs.Model().CurrentWaitDuration)
	if err != nil {
		return 3 * time.Second
	}

	return duration
}

// SetCurrentWaitDuration allows to safely set and persist configuration settings.
func (cs *Service) SetCurrentWaitDuration(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().CurrentWaitDuration = interval.String()

	return cs.Save()
}

// GetSlowChargingCurrentInAmperes allows to safely access a configuration setting.
func (cs *Service) GetSlowChargingCurrentInAmperes() float64 {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().SlowChargingCurrentInAmperes
}

// SetSlowChargingCurrentInAmperes allows to safely set and persist configuration settings.
func (cs *Service) SetSlowChargingCurrentInAmperes(current float64) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SlowChargingCurrentInAmperes = current

	return cs.Save()
}

// GetHTTPTimeout allows to safely access a configuration setting.
func (cs *Service) GetHTTPTimeout() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	timeout, err := time.ParseDuration(cs.Model().HTTPTimeout)
	if err != nil {
		return 30 * time.Second
	}

	return timeout
}

// SetHTTPTimeout allows to safely set and persist configuration settings.
func (cs *Service) SetHTTPTimeout(timeout time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().HTTPTimeout = timeout.String()

	return cs.Save()
}

// GetSignalRBaseURL allows to safely access a configuration setting.
func (cs *Service) GetSignalRBaseURL() string {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().SignalR.BaseURL
}

// SetSignalRBaseURL allows to safely set and persist configuration settings.
func (cs *Service) SetSignalRBaseURL(url string) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SignalR.BaseURL = url

	return cs.Save()
}

// GetSignalRConnCreationTimeout allows to safely access a configuration setting.
func (cs *Service) GetSignalRConnCreationTimeout() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	timeout, err := time.ParseDuration(cs.Model().SignalR.ConnCreationTimeout)
	if err != nil {
		return 30 * time.Second
	}

	return timeout
}

// SetSignalRConnCreationTimeout allows to safely set and persist configuration settings.
func (cs *Service) SetSignalRConnCreationTimeout(timeout time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SignalR.ConnCreationTimeout = timeout.String()

	return cs.Save()
}

// GetSignalRKeepAliveInterval allows to safely access a configuration setting.
func (cs *Service) GetSignalRKeepAliveInterval() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	interval, err := time.ParseDuration(cs.Model().SignalR.KeepAliveInterval)
	if err != nil {
		return 30 * time.Second
	}

	return interval
}

// SetSignalRKeepAliveInterval allows to safely set and persist configuration settings.
func (cs *Service) SetSignalRKeepAliveInterval(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SignalR.KeepAliveInterval = interval.String()

	return cs.Save()
}

// GetSignalRTimeoutInterval allows to safely access a configuration setting.
func (cs *Service) GetSignalRTimeoutInterval() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	interval, err := time.ParseDuration(cs.Model().SignalR.TimeoutInterval)
	if err != nil {
		return 1 * time.Minute
	}

	return interval
}

// SetSignalRTimeoutInterval allows to safely set and persist configuration settings.
func (cs *Service) SetSignalRTimeoutInterval(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SignalR.TimeoutInterval = interval.String()

	return cs.Save()
}

// GetSignalRInitialBackoff allows to safely access a configuration setting.
func (cs *Service) GetSignalRInitialBackoff() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	interval, err := time.ParseDuration(cs.Model().SignalR.InitialBackoff)
	if err != nil {
		return 5 * time.Second
	}

	return interval
}

// SetSignalRInitialBackoff allows to safely set and persist configuration settings.
func (cs *Service) SetSignalRInitialBackoff(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SignalR.InitialBackoff = interval.String()

	return cs.Save()
}

// GetSignalRRepeatedBackoff allows to safely access a configuration setting.
func (cs *Service) GetSignalRRepeatedBackoff() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	interval, err := time.ParseDuration(cs.Model().SignalR.RepeatedBackoff)
	if err != nil {
		return 30 * time.Second
	}

	return interval
}

// SetSignalRRepeatedBackoff allows to safely set and persist configuration settings.
func (cs *Service) SetSignalRRepeatedBackoff(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SignalR.RepeatedBackoff = interval.String()

	return cs.Save()
}

// GetSignalRFinalBackoff allows to safely access a configuration setting.
func (cs *Service) GetSignalRFinalBackoff() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	interval, err := time.ParseDuration(cs.Model().SignalR.FinalBackoff)
	if err != nil {
		return 2 * time.Minute
	}

	return interval
}

// SetSignalRFinalBackoff allows to safely set and persist configuration settings.
func (cs *Service) SetSignalRFinalBackoff(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SignalR.FinalBackoff = interval.String()

	return cs.Save()
}

// GetSignalRInitialFailureCount allows to safely access signalr initial failure count.
func (cs *Service) GetSignalRInitialFailureCount() uint32 {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().SignalR.InitialFailureCount
}

// SetSignalRInitialFailureCount allows to safely alter signalr initial failure count.
func (cs *Service) SetSignalRInitialFailureCount(n uint32) error {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	cs.Model().SignalR.InitialFailureCount = n

	return cs.Save()
}

// GetSignalRRepeatedFailureCount allows to safely access repeated failure count.
func (cs *Service) GetSignalRRepeatedFailureCount() uint32 {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().SignalR.RepeatedFailureCount
}

// SetSignalRRepeatedFailureCount allows to safely alter repeated failure count.
func (cs *Service) SetSignalRRepeatedFailureCount(n uint32) error {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	cs.Model().SignalR.RepeatedFailureCount = n

	return cs.Save()
}

// GetSignalRInvokeTimeout allows to safely access a configuration setting.
func (cs *Service) GetSignalRInvokeTimeout() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	timeout, err := time.ParseDuration(cs.Model().SignalR.InvokeTimeout)
	if err != nil {
		return 10 * time.Second
	}

	return timeout
}

// SetSignalRInvokeTimeout allows to safely set and persist configuration settings.
func (cs *Service) SetSignalRInvokeTimeout(timeout time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SignalR.InvokeTimeout = timeout.String()

	return cs.Save()
}

// GetOfferedCurrentWaitTime allows to safely access a configuration setting.
func (cs *Service) GetOfferedCurrentWaitTime() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	duration, err := time.ParseDuration(cs.Model().OfferedCurrentWaitTime)
	if err != nil {
		return 30 * time.Second
	}

	return duration
}

// SetOfferedCurrentWaitTime allows to safely set and persist a configuration setting.
func (cs *Service) SetOfferedCurrentWaitTime(duration time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().OfferedCurrentWaitTime = duration.String()

	return cs.Save()
}

// GetAuthenticatorBackoffCfg allows to safely access api backoff settings.
func (cs *Service) GetAuthenticatorBackoffCfg() BackoffCfg {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	initial, err := time.ParseDuration(cs.Model().AuthenticatorBackoff.InitialBackoff)
	if err != nil {
		initial = 1 * time.Minute
	}

	repeated, err := time.ParseDuration(cs.Model().AuthenticatorBackoff.RepeatedBackoff)
	if err != nil {
		repeated = 5 * time.Minute
	}

	final, err := time.ParseDuration(cs.Model().AuthenticatorBackoff.FinalBackoff)
	if err != nil {
		final = 10 * time.Minute
	}

	return BackoffCfg{
		InitialBackoff:       initial,
		RepeatedBackoff:      repeated,
		FinalBackoff:         final,
		InitialFailureCount:  cs.Model().AuthenticatorBackoff.InitialFailureCount,
		RepeatedFailureCount: cs.Model().AuthenticatorBackoff.RepeatedFailureCount,
	}
}

// SetAuthenticatorBackoffCfg allows to safely alter repeated failure count.
func (cs *Service) SetAuthenticatorBackoffCfg(cfg BackoffCfg) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().AuthenticatorBackoff = backoffCfg{
		InitialBackoff:       cfg.InitialBackoff.String(),
		RepeatedBackoff:      cfg.RepeatedBackoff.String(),
		FinalBackoff:         cfg.FinalBackoff.String(),
		InitialFailureCount:  cfg.InitialFailureCount,
		RepeatedFailureCount: cfg.RepeatedFailureCount,
	}

	return cs.Save()
}
