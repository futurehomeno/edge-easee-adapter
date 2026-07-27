package config

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/storage"
	"github.com/michalkurzeja/go-clock"
	log "github.com/sirupsen/logrus"
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
		uint32OrDefault(b.InitialFailureCount, 1),
		uint32OrDefault(b.RepeatedFailureCount, 1),
	)
}

func uint32OrDefault(v, def uint32) uint32 {
	if v == 0 {
		return def
	}

	return v
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
		// Corrupt legacy data must not block migration progress - if we returned the error
		// the version stays at 2 and the next startup retries with the same broken bytes.
		// Drop the legacy field and let defaults seed the new shape.
		log.Warnf("[config] drop corrupt legacy authenticatorBackoff: %v", err)
		c.LegacyAuthenticatorBackoff = nil

		return nil
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

// DefaultStore exposes the config as a cliffhanger DefaultStore whose Save runs
// under cs.lock, so debug-route log mutations serialize with credential writes on
// the shared config instead of racing under the store's separate mutex.
func (cs *Service) DefaultStore() *config.DefaultStore {
	return config.NewDefaultStore(
		func() *config.Default { return &cs.Model().Default },
		func() error {
			cs.lock.Lock()
			defer cs.lock.Unlock()

			return cs.Save()
		},
	)
}

func (cs *Service) PublicConfig() PublicConfig {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.Model().PublicConfig
}

// PublicModel implements cliffhanger's app.PublicModeler so
// cmd.config.get_extended_report and the manifest expose only the FIMP-safe
// PublicConfig. Without it publicConfigState falls back to the full model,
// leaking config.Default (log level/format, MQTT credentials, telemetry, ...)
// and the access/refresh tokens.
func (cs *Service) PublicModel() any {
	return cs.PublicConfig()
}

func (cs *Service) EaseeBaseURL() string {
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

func (cs *Service) EnergyLifetimeInterval() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	duration, err := time.ParseDuration(cs.Model().EnergyLifetimeInterval)
	if err != nil {
		return 10 * time.Second
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

func (cs *Service) Credentials() Credentials {
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

func (cs *Service) PollingInterval() time.Duration {
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

func (cs *Service) CurrentWaitDuration() time.Duration {
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

func (cs *Service) SlowChargingCurrentInAmperes() float64 {
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

func (cs *Service) HTTPTimeout() time.Duration {
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

func (cs *Service) SignalRBaseURL() string {
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

func (cs *Service) SignalRConnCreationTimeout() time.Duration {
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

func (cs *Service) SignalRKeepAliveInterval() time.Duration {
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

func (cs *Service) SignalRTimeoutInterval() time.Duration {
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

func (cs *Service) SignalRInitialBackoff() time.Duration {
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

func (cs *Service) SignalRRepeatedBackoff() time.Duration {
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

func (cs *Service) SignalRFinalBackoff() time.Duration {
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

func (cs *Service) SignalRInitialFailureCount() uint32 {
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

func (cs *Service) SignalRRepeatedFailureCount() uint32 {
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

func (cs *Service) SignalRInvokeTimeout() time.Duration {
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

func (cs *Service) OfferedCurrentWaitTime() time.Duration {
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

func (cs *Service) TokenRefreshInterval() time.Duration {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	interval, err := time.ParseDuration(cs.Model().TokenRefreshInterval)
	if err != nil {
		return 30 * time.Minute
	}

	return interval
}

func (cs *Service) SetTokenRefreshInterval(interval time.Duration) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.Model().ConfiguredAt = time.Now().Format(time.RFC3339)
	cs.Model().TokenRefreshInterval = interval.String()

	return cs.Save()
}
