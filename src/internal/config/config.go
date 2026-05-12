package config

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/storage"
	"github.com/michalkurzeja/go-clock"
)

type PublicConfig struct {
	EaseeBaseURL                 string  `json:"easeeBaseURL2"`
	PollingInterval              string  `json:"pollingInterval"`
	TokenRefreshInterval         string  `json:"token_refresh_interval"`
	CurrentWaitDuration          string  `json:"currentWaitDuration"`
	SlowChargingCurrentInAmperes float64 `json:"slowChargingCurrentInAmperes"`
	HTTPTimeout                  string  `json:"httpTimeout"`
	SignalR                      SignalR `json:"signalR"`
	OfferedCurrentWaitTime       string  `json:"offered_current_wait_time"`
	EnergyLifetimeInterval       string  `json:"energyLifetimeInterval"`

	AuthBackoff         backoffSettings `json:"auth_backoff"`
	AuthMaxUnauthorized string          `json:"auth_max_unauthorized,omitempty"`

	LegacyAuthenticatorBackoff json.RawMessage `json:"authenticatorBackoff,omitempty"`
}

type Config struct {
	config.Default
	PublicConfig
	Credentials
}

func New(workDir string) *Config {
	return &Config{
		Default: config.NewDefault(workDir),
	}
}

func Factory() *Config {
	return &Config{}
}

type Credentials struct {
	AccessToken           string    `json:"accessToken"`
	RefreshToken          string    `json:"refreshToken"`
	AccessTokenExpiresAt  time.Time `json:"expiresAt,omitzero"`
	RefreshTokenExpiresAt time.Time `json:"refreshTokenExpiresAt,omitzero"`
}

func (c Credentials) Empty() bool {
	return c == Credentials{}
}

func (c Credentials) AccessTokenExpired() bool {
	return clock.Now().After(c.AccessTokenExpiresAt)
}

func (c Credentials) RefreshTokenExpired() bool {
	return clock.Now().After(c.RefreshTokenExpiresAt)
}

type backoffSettings struct {
	InitialBackoff       string `json:"initialBackoff"`
	RepeatedBackoff      string `json:"repeatedBackoff"`
	FinalBackoff         string `json:"finalBackoff"`
	InitialFailureCount  uint32 `json:"initialFailureCount"`
	RepeatedFailureCount uint32 `json:"repeatedFailureCount"`
}

func (b backoffSettings) stateful(initial, repeated, final time.Duration) backoff.Stateful {
	return backoff.NewStateful(
		parseDuration(b.InitialBackoff, initial),
		parseDuration(b.RepeatedBackoff, repeated),
		parseDuration(b.FinalBackoff, final),
		b.InitialFailureCount,
		b.RepeatedFailureCount,
	)
}

type SignalR struct {
	BaseURL             string `json:"baseURL"`
	ConnCreationTimeout string `json:"connCreationTimeout"`
	KeepAliveInterval   string `json:"keepAliveInterval2"`
	TimeoutInterval     string `json:"timeoutInterval2"`
	backoffSettings            // anonymous embed - fields promoted to top level of "signalR" JSON object; wire-compatible
	InvokeTimeout       string `json:"invokeTimeout"`
}

// Service is a configuration service responsible for:
// - providing concurrency safe access to settings
// - persistence of settings.
type Service struct {
	storage.Storage[*Config]
	lock *sync.RWMutex
}

func parseDuration(s string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}

	return d
}

func (c *Config) MigrateAuthBackoff() error {
	if len(c.LegacyAuthenticatorBackoff) == 0 {
		return nil
	}

	var legacy struct {
		InitialBackoff          string `json:"initialBackoff"`
		RepeatedBackoff         string `json:"repeatedBackoff"`
		FinalBackoff            string `json:"finalBackoff"`
		InitialFailureCount     uint32 `json:"initialFailureCount"`
		RepeatedFailureCount    uint32 `json:"repeatedFailureCount"`
		MaxUnauthorizedDuration string `json:"maxUnauthorizedDuration"`
	}

	if err := json.Unmarshal(c.LegacyAuthenticatorBackoff, &legacy); err != nil {
		return err
	}

	c.AuthBackoff = backoffSettings{
		InitialBackoff:       legacy.InitialBackoff,
		RepeatedBackoff:      legacy.RepeatedBackoff,
		FinalBackoff:         legacy.FinalBackoff,
		InitialFailureCount:  legacy.InitialFailureCount,
		RepeatedFailureCount: legacy.RepeatedFailureCount,
	}
	c.AuthMaxUnauthorized = legacy.MaxUnauthorizedDuration
	c.LegacyAuthenticatorBackoff = nil

	return nil
}

func NewService(storage storage.Storage[*Config]) *Service {
	return &Service{
		Storage: storage,
		lock:    &sync.RWMutex{},
	}
}

func (cs *Service) GetPublicConfig() PublicConfig {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().PublicConfig
}

func (cs *Service) GetEaseeBaseURL() string {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().EaseeBaseURL
}

func (cs *Service) SetEaseeBaseURL(url string) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().EaseeBaseURL = url

	return cs.Save()
}

func (cs *Service) GetEnergyLifetimeInterval() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	duration, err := time.ParseDuration(cs.Model().EnergyLifetimeInterval)
	if err != nil {
		return 5 * time.Minute
	}

	return duration
}

func (cs *Service) SetEnergyLifetimeInterval(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().EnergyLifetimeInterval = interval.String()

	return cs.Save()
}

func (cs *Service) GetCredentials() Credentials {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().Credentials
}

func (cs *Service) SetCredentials(credentials Credentials) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().Credentials = credentials

	return cs.Save()
}

func (cs *Service) ClearCredentials() error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().Credentials = Credentials{}

	return cs.Save()
}

func (cs *Service) GetPollingInterval() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	duration, err := time.ParseDuration(cs.Model().PollingInterval)
	if err != nil {
		return 10 * time.Minute
	}

	return duration
}

func (cs *Service) SetPollingInterval(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().PollingInterval = interval.String()

	return cs.Save()
}

func (cs *Service) GetCurrentWaitDuration() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	duration, err := time.ParseDuration(cs.Model().CurrentWaitDuration)
	if err != nil {
		return 3 * time.Second
	}

	return duration
}

func (cs *Service) SetCurrentWaitDuration(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().CurrentWaitDuration = interval.String()

	return cs.Save()
}

func (cs *Service) GetSlowChargingCurrentInAmperes() float64 {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().SlowChargingCurrentInAmperes
}

func (cs *Service) SetSlowChargingCurrentInAmperes(current float64) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SlowChargingCurrentInAmperes = current

	return cs.Save()
}

func (cs *Service) GetHTTPTimeout() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	timeout, err := time.ParseDuration(cs.Model().HTTPTimeout)
	if err != nil {
		return 30 * time.Second
	}

	return timeout
}

func (cs *Service) SetHTTPTimeout(timeout time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().HTTPTimeout = timeout.String()

	return cs.Save()
}

func (cs *Service) GetSignalRBaseURL() string {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().SignalR.BaseURL
}

func (cs *Service) SetSignalRBaseURL(url string) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SignalR.BaseURL = url

	return cs.Save()
}

func (cs *Service) GetSignalRConnCreationTimeout() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	timeout, err := time.ParseDuration(cs.Model().SignalR.ConnCreationTimeout)
	if err != nil {
		return 30 * time.Second
	}

	return timeout
}

func (cs *Service) SetSignalRConnCreationTimeout(timeout time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SignalR.ConnCreationTimeout = timeout.String()

	return cs.Save()
}

func (cs *Service) GetSignalRKeepAliveInterval() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	interval, err := time.ParseDuration(cs.Model().SignalR.KeepAliveInterval)
	if err != nil {
		return 30 * time.Second
	}

	return interval
}

func (cs *Service) SetSignalRKeepAliveInterval(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SignalR.KeepAliveInterval = interval.String()

	return cs.Save()
}

func (cs *Service) GetSignalRTimeoutInterval() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	interval, err := time.ParseDuration(cs.Model().SignalR.TimeoutInterval)
	if err != nil {
		return 1 * time.Minute
	}

	return interval
}

func (cs *Service) SetSignalRTimeoutInterval(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SignalR.TimeoutInterval = interval.String()

	return cs.Save()
}

func (cs *Service) GetSignalRInitialBackoff() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	interval, err := time.ParseDuration(cs.Model().SignalR.InitialBackoff)
	if err != nil {
		return 5 * time.Second
	}

	return interval
}

func (cs *Service) SetSignalRInitialBackoff(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SignalR.InitialBackoff = interval.String()

	return cs.Save()
}

func (cs *Service) GetSignalRRepeatedBackoff() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	interval, err := time.ParseDuration(cs.Model().SignalR.RepeatedBackoff)
	if err != nil {
		return 30 * time.Second
	}

	return interval
}

func (cs *Service) SetSignalRRepeatedBackoff(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SignalR.RepeatedBackoff = interval.String()

	return cs.Save()
}

func (cs *Service) GetSignalRFinalBackoff() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	interval, err := time.ParseDuration(cs.Model().SignalR.FinalBackoff)
	if err != nil {
		return 2 * time.Minute
	}

	return interval
}

func (cs *Service) SetSignalRFinalBackoff(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SignalR.FinalBackoff = interval.String()

	return cs.Save()
}

func (cs *Service) GetSignalRInitialFailureCount() uint32 {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().SignalR.InitialFailureCount
}

func (cs *Service) SetSignalRInitialFailureCount(n uint32) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().SignalR.InitialFailureCount = n

	return cs.Save()
}

func (cs *Service) GetSignalRRepeatedFailureCount() uint32 {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().SignalR.RepeatedFailureCount
}

func (cs *Service) SetSignalRRepeatedFailureCount(n uint32) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().SignalR.RepeatedFailureCount = n

	return cs.Save()
}

func (cs *Service) GetSignalRInvokeTimeout() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	timeout, err := time.ParseDuration(cs.Model().SignalR.InvokeTimeout)
	if err != nil {
		return 10 * time.Second
	}

	return timeout
}

func (cs *Service) SetSignalRInvokeTimeout(timeout time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().SignalR.InvokeTimeout = timeout.String()

	return cs.Save()
}

func (cs *Service) GetOfferedCurrentWaitTime() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	duration, err := time.ParseDuration(cs.Model().OfferedCurrentWaitTime)
	if err != nil {
		return 15 * time.Second
	}

	return duration
}

func (cs *Service) SetOfferedCurrentWaitTime(duration time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().OfferedCurrentWaitTime = duration.String()

	return cs.Save()
}

func (cs *Service) AuthenticatorBackoffStateful() backoff.Stateful {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().AuthBackoff.stateful(time.Minute, 5*time.Minute, 10*time.Minute)
}

func (cs *Service) SignalRBackoffStateful() backoff.Stateful {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().SignalR.stateful(5*time.Second, 30*time.Second, 2*time.Minute)
}

func (cs *Service) AuthenticatorMaxUnauthorized() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return parseDuration(cs.Model().AuthMaxUnauthorized, 2*time.Hour)
}

func (cs *Service) SetAuthenticatorBackoff(
	initial, repeated, final time.Duration,
	initialFailureCount, repeatedFailureCount uint32,
	maxUnauthorized time.Duration,
) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	c := cs.Model()
	c.ConfiguredAt = time.Now().Format(time.RFC3339)
	c.AuthBackoff = backoffSettings{
		InitialBackoff:       initial.String(),
		RepeatedBackoff:      repeated.String(),
		FinalBackoff:         final.String(),
		InitialFailureCount:  initialFailureCount,
		RepeatedFailureCount: repeatedFailureCount,
	}
	c.AuthMaxUnauthorized = maxUnauthorized.String()

	return cs.Save()
}

func (cs *Service) GetTokenRefreshInterval() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	interval, err := time.ParseDuration(cs.Storage.Model().TokenRefreshInterval)
	if err != nil {
		return 30 * time.Minute
	}

	return interval
}

func (cs *Service) SetTokenRefreshInterval(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Storage.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Storage.Model().TokenRefreshInterval = interval.String()

	return cs.Save()
}
