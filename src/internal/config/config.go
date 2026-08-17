package config

import (
	"encoding/json"
	"time"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/selection"
	"github.com/futurehomeno/cliffhanger/storage"
	"github.com/michalkurzeja/go-clock"
	log "github.com/sirupsen/logrus"
)

const defaultStartChargingCurrent = 16

type PublicConfig struct {
	EaseeBaseURL                 string  `json:"easeeBaseURL2"`
	PollingInterval              string  `json:"pollingInterval"`
	TokenRefreshInterval         string  `json:"token_refresh_interval"`
	CurrentWaitDuration          string  `json:"currentWaitDuration"`
	SlowChargingCurrentInAmperes float64 `json:"slowChargingCurrentInAmperes"`
	InitialChargingCurrent       int     `json:"initial_charging_current"`
	HTTPTimeout                  string  `json:"httpTimeout"`
	SignalR                      SignalR `json:"signalR"`
	OfferedCurrentWaitTime       string  `json:"offered_current_wait_time"`
	EnergyLifetimeInterval       string  `json:"energyLifetimeInterval"`

	AuthBackoff         backoffSettings `json:"auth_backoff"`
	AuthMaxUnauthorized string          `json:"auth_max_unauthorized,omitempty"`

	// SelectedDevices is the user's chosen subset of charger IDs from the Configure step.
	// A nil selection includes every charger, an empty one includes none - the distinction
	// tells an install that was never configured from one where the user deselected
	// everything. Configurations written before v3.0 carry no such key, so they read as nil.
	SelectedDevices selection.Selection `json:"selected_devices"`

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
	*config.Service[*Config]
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

// MigrateOfferedCurrentWaitTime lifts installs still carrying the superseded packaged
// default ("15s") onto the rate-limit-safe wait time. Any other value is left alone; a
// "15s" chosen deliberately is indistinguishable from the old default and is rewritten too.
func (c *Config) MigrateOfferedCurrentWaitTime() error {
	if c.OfferedCurrentWaitTime == "15s" {
		c.OfferedCurrentWaitTime = "20s"
	}

	return nil
}

// MigrateSignalRFinalBackoff lifts installs still carrying the superseded packaged
// default ("2m") onto the calmer ceiling. Any other value is left alone; a "2m" chosen
// deliberately is indistinguishable from the old default and is rewritten too.
func (c *Config) MigrateSignalRFinalBackoff() error {
	if c.SignalR.FinalBackoff == "2m" {
		c.SignalR.FinalBackoff = "10m"
	}

	return nil
}

func NewService(storage storage.Storage[*Config]) *Service {
	return &Service{
		// The redact function must return a copy: PublicModel would otherwise alias the
		// live model and race a concurrent Update while the report is marshalled.
		config.NewService(
			storage,
			func(c *Config) *config.Default { return &c.Default },
			func(c *Config) any { return c.PublicConfig },
		),
	}
}

func (cs *Service) PublicConfig() PublicConfig {
	return config.Get(cs.Service, func(c *Config) PublicConfig { return c.PublicConfig })
}

func (cs *Service) EaseeBaseURL() string {
	return config.Get(cs.Service, func(c *Config) string { return c.EaseeBaseURL })
}

func (cs *Service) SetEaseeBaseURL(url string) error {
	return cs.Update(func(c *Config) { c.EaseeBaseURL = url })
}

func (cs *Service) EnergyLifetimeInterval() time.Duration {
	return config.GetDuration(cs.Service, func(c *Config) string { return c.EnergyLifetimeInterval }, 10*time.Second)
}

func (cs *Service) SetEnergyLifetimeInterval(interval time.Duration) error {
	return cs.Update(func(c *Config) { c.EnergyLifetimeInterval = interval.String() })
}

// SelectedDevices returns a copy that preserves the nil/empty distinction - the idiomatic
// append([]string(nil), ...) and slices.Clone both collapse empty to nil, turning "no chargers"
// into "every charger".
func (cs *Service) SelectedDevices() selection.Selection {
	return config.Get(cs.Service, func(c *Config) selection.Selection {
		return c.SelectedDevices.Clone()
	})
}

func (cs *Service) SetSelectedDevices(devices selection.Selection) error {
	return cs.Update(func(c *Config) { c.SelectedDevices = devices.Clone() })
}

func (cs *Service) PollingInterval() time.Duration {
	return config.GetDuration(cs.Service, func(c *Config) string { return c.PollingInterval }, 10*time.Minute)
}

func (cs *Service) SetPollingInterval(interval time.Duration) error {
	return cs.Update(func(c *Config) { c.PollingInterval = interval.String() })
}

func (cs *Service) CurrentWaitDuration() time.Duration {
	return config.GetDuration(cs.Service, func(c *Config) string { return c.CurrentWaitDuration }, 3*time.Second)
}

func (cs *Service) SetCurrentWaitDuration(interval time.Duration) error {
	return cs.Update(func(c *Config) { c.CurrentWaitDuration = interval.String() })
}

func (cs *Service) SlowChargingCurrentInAmperes() float64 {
	return config.Get(cs.Service, func(c *Config) float64 { return c.SlowChargingCurrentInAmperes })
}

func (cs *Service) SetSlowChargingCurrentInAmperes(current float64) error {
	return cs.Update(func(c *Config) { c.SlowChargingCurrentInAmperes = current })
}

func (cs *Service) InitialChargingCurrent() int {
	if current := config.Get(cs.Service, func(c *Config) int { return c.InitialChargingCurrent }); current > 0 {
		return current
	}

	return defaultStartChargingCurrent
}

func (cs *Service) HTTPTimeout() time.Duration {
	return config.GetDuration(cs.Service, func(c *Config) string { return c.HTTPTimeout }, 30*time.Second)
}

func (cs *Service) SetHTTPTimeout(timeout time.Duration) error {
	return cs.Update(func(c *Config) { c.HTTPTimeout = timeout.String() })
}

func (cs *Service) SignalRBaseURL() string {
	return config.Get(cs.Service, func(c *Config) string { return c.SignalR.BaseURL })
}

func (cs *Service) SetSignalRBaseURL(url string) error {
	return cs.Update(func(c *Config) { c.SignalR.BaseURL = url })
}

func (cs *Service) SignalRConnCreationTimeout() time.Duration {
	return config.GetDuration(cs.Service, func(c *Config) string { return c.SignalR.ConnCreationTimeout }, 30*time.Second)
}

func (cs *Service) SetSignalRConnCreationTimeout(timeout time.Duration) error {
	return cs.Update(func(c *Config) { c.SignalR.ConnCreationTimeout = timeout.String() })
}

func (cs *Service) SignalRKeepAliveInterval() time.Duration {
	return config.GetDuration(cs.Service, func(c *Config) string { return c.SignalR.KeepAliveInterval }, 30*time.Second)
}

func (cs *Service) SetSignalRKeepAliveInterval(interval time.Duration) error {
	return cs.Update(func(c *Config) { c.SignalR.KeepAliveInterval = interval.String() })
}

func (cs *Service) SignalRTimeoutInterval() time.Duration {
	return config.GetDuration(cs.Service, func(c *Config) string { return c.SignalR.TimeoutInterval }, time.Minute)
}

func (cs *Service) SetSignalRTimeoutInterval(interval time.Duration) error {
	return cs.Update(func(c *Config) { c.SignalR.TimeoutInterval = interval.String() })
}

func (cs *Service) SignalRInitialBackoff() time.Duration {
	return config.GetDuration(cs.Service, func(c *Config) string { return c.SignalR.InitialBackoff }, 5*time.Second)
}

func (cs *Service) SetSignalRInitialBackoff(interval time.Duration) error {
	return cs.Update(func(c *Config) { c.SignalR.InitialBackoff = interval.String() })
}

func (cs *Service) SignalRRepeatedBackoff() time.Duration {
	return config.GetDuration(cs.Service, func(c *Config) string { return c.SignalR.RepeatedBackoff }, 30*time.Second)
}

func (cs *Service) SetSignalRRepeatedBackoff(interval time.Duration) error {
	return cs.Update(func(c *Config) { c.SignalR.RepeatedBackoff = interval.String() })
}

func (cs *Service) SignalRFinalBackoff() time.Duration {
	return config.GetDuration(cs.Service, func(c *Config) string { return c.SignalR.FinalBackoff }, 10*time.Minute)
}

func (cs *Service) SetSignalRFinalBackoff(interval time.Duration) error {
	return cs.Update(func(c *Config) { c.SignalR.FinalBackoff = interval.String() })
}

func (cs *Service) SignalRInitialFailureCount() uint32 {
	return config.Get(cs.Service, func(c *Config) uint32 { return c.SignalR.InitialFailureCount })
}

func (cs *Service) SetSignalRInitialFailureCount(n uint32) error {
	return cs.Persist(func(c *Config) { c.SignalR.InitialFailureCount = n })
}

func (cs *Service) SignalRRepeatedFailureCount() uint32 {
	return config.Get(cs.Service, func(c *Config) uint32 { return c.SignalR.RepeatedFailureCount })
}

func (cs *Service) SetSignalRRepeatedFailureCount(n uint32) error {
	return cs.Persist(func(c *Config) { c.SignalR.RepeatedFailureCount = n })
}

func (cs *Service) SignalRInvokeTimeout() time.Duration {
	return config.GetDuration(cs.Service, func(c *Config) string { return c.SignalR.InvokeTimeout }, 10*time.Second)
}

func (cs *Service) SetSignalRInvokeTimeout(timeout time.Duration) error {
	return cs.Update(func(c *Config) { c.SignalR.InvokeTimeout = timeout.String() })
}

func (cs *Service) OfferedCurrentWaitTime() time.Duration {
	return config.GetDuration(cs.Service, func(c *Config) string { return c.OfferedCurrentWaitTime }, 20*time.Second)
}

func (cs *Service) SetOfferedCurrentWaitTime(duration time.Duration) error {
	return cs.Update(func(c *Config) { c.OfferedCurrentWaitTime = duration.String() })
}

func (cs *Service) AuthenticatorBackoffStateful() backoff.Stateful {
	return config.Get(cs.Service, func(c *Config) backoff.Stateful {
		return c.AuthBackoff.stateful(time.Minute, 5*time.Minute, 10*time.Minute)
	})
}

func (cs *Service) SignalRBackoffStateful() backoff.Stateful {
	return config.Get(cs.Service, func(c *Config) backoff.Stateful {
		return c.SignalR.stateful(5*time.Second, 30*time.Second, 10*time.Minute)
	})
}

func (cs *Service) AuthenticatorMaxUnauthorized() time.Duration {
	return config.GetDuration(cs.Service, func(c *Config) string { return c.AuthMaxUnauthorized }, 2*time.Hour)
}

func (cs *Service) SetAuthenticatorBackoff(
	initial, repeated, final time.Duration,
	initialFailureCount, repeatedFailureCount uint32,
	maxUnauthorized time.Duration,
) error {
	return cs.Update(func(c *Config) {
		c.AuthBackoff = backoffSettings{
			InitialBackoff:       initial.String(),
			RepeatedBackoff:      repeated.String(),
			FinalBackoff:         final.String(),
			InitialFailureCount:  initialFailureCount,
			RepeatedFailureCount: repeatedFailureCount,
		}
		c.AuthMaxUnauthorized = maxUnauthorized.String()
	})
}

func (cs *Service) TokenRefreshInterval() time.Duration {
	return config.GetDuration(cs.Service, func(c *Config) string { return c.TokenRefreshInterval }, 30*time.Minute)
}

func (cs *Service) SetTokenRefreshInterval(interval time.Duration) error {
	return cs.Update(func(c *Config) { c.TokenRefreshInterval = interval.String() })
}
