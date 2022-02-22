package app_test

import (
	"testing"
	"time"

	cliffApp "github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/michalkurzeja/go-clock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	adapterMocks "github.com/futurehomeno/cliffhanger/mocks/adapter"
	storageMocks "github.com/futurehomeno/cliffhanger/mocks/storage"

	"github.com/futurehomeno/edge-easee-adapter/internal/app"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
	easeeMocks "github.com/futurehomeno/edge-easee-adapter/internal/easee/mocks"
	"github.com/futurehomeno/edge-easee-adapter/internal/test"
)

func TestApplication_GetManifest(t *testing.T) {
	t.Parallel()

	mf := test.LoadManifest(t)
	a := app.New(nil, nil, nil, mf, nil)

	got, err := a.GetManifest()

	assert.NoError(t, err)
	assert.Equal(t, mf, got)
}

func TestApplication_Configure_NOOP(t *testing.T) {
	t.Parallel()

	a := app.New(nil, nil, nil, nil, nil)
	err := a.Configure("anything")

	assert.NoError(t, err)
}

func TestApplication_Uninstall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		cfg                 *config.Config
		setLifecycle        func(lc *lifecycle.Lifecycle)
		mockAdapter         func(a *adapterMocks.ExtendedAdapter)
		mockConfigStorage   func(s *storageMocks.Storage, cfg *config.Config)
		wantErr             bool
		lifecycleAssertions func(lc *lifecycle.Lifecycle)
		configAssertions    func(c *config.Config)
	}{
		{
			name: "successful config, lifecycle and adapter reset",
			cfg: &config.Config{
				Credentials: config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC),
				},
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockAdapter: func(a *adapterMocks.ExtendedAdapter) {
				a.On("DestroyAllThings").Return(nil)
			},
			mockConfigStorage: func(s *storageMocks.Storage, cfg *config.Config) {
				*cfg = config.Config{}

				s.On("Reset").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateNotConfigured, lc.AppState())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
			configAssertions: func(c *config.Config) {
				assert.Equal(t, &config.Config{}, c)
			},
		},
		{
			name: "adapter error on destroying all things",
			cfg: &config.Config{
				Credentials: config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC),
				},
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockAdapter: func(a *adapterMocks.ExtendedAdapter) {
				a.On("DestroyAllThings").Return(errors.New("test error"))
			},
			wantErr: true,
		},
		{
			name: "config service error on reset",
			cfg: &config.Config{
				Credentials: config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC),
				},
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockAdapter: func(a *adapterMocks.ExtendedAdapter) {
				a.On("DestroyAllThings").Return(nil)
			},
			mockConfigStorage: func(s *storageMocks.Storage, cfg *config.Config) {
				s.On("Reset").Return(errors.New("test error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lc := lifecycle.New()
			if tt.setLifecycle != nil {
				tt.setLifecycle(lc)
			}

			adapterMock := new(adapterMocks.ExtendedAdapter)
			if tt.mockAdapter != nil {
				tt.mockAdapter(adapterMock)
			}

			cfgStorageMock := new(storageMocks.Storage)
			if tt.mockConfigStorage != nil {
				tt.mockConfigStorage(cfgStorageMock, tt.cfg)
			}

			defer func() {
				adapterMock.AssertExpectations(t)
				cfgStorageMock.AssertExpectations(t)
			}()

			cfgService := config.NewService(cfgStorageMock)

			application := app.New(adapterMock, cfgService, lc, nil, nil)

			err := application.Uninstall()

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)

			if tt.lifecycleAssertions != nil {
				tt.lifecycleAssertions(lc)
			}

			if tt.configAssertions != nil {
				tt.configAssertions(tt.cfg)
			}
		})
	}
}

func TestApplication_Login(t *testing.T) { //nolint:paralleltest
	clock.Mock(time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name                string
		loginData           *cliffApp.LoginCredentials
		cfg                 *config.Config
		setLifecycle        func(lc *lifecycle.Lifecycle)
		mockAdapter         func(a *adapterMocks.ExtendedAdapter)
		mockClient          func(c *easeeMocks.Client)
		mockConfigStorage   func(s *storageMocks.Storage, cfg *config.Config)
		wantErr             bool
		lifecycleAssertions func(lc *lifecycle.Lifecycle)
		configAssertions    func(c *config.Config)
	}{
		{
			name: "if login was successful, credentials and lifecycle should be set up",
			loginData: &cliffApp.LoginCredentials{
				Username: "test-user",
				Password: "test-password",
			},
			cfg: &config.Config{},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockClient: func(c *easeeMocks.Client) {
				c.
					On("Login", "test-user", "test-password").
					Return(&easee.LoginData{
						AccessToken:  "access-token",
						ExpiresIn:    86400,
						AccessClaims: []string{"User"},
						TokenType:    "Bearer",
						RefreshToken: "refresh-token",
					}, nil)
				c.On("Chargers").Return([]easee.Charger{
					{ID: "123"},
					{ID: "456"},
				}, nil)
				c.On("Ping").Return(nil)
			},
			mockAdapter: func(a *adapterMocks.ExtendedAdapter) {
				a.On("CreateThing", "123", easee.Info{ChargerID: "123"}).Return(nil)
				a.On("CreateThing", "456", easee.Info{ChargerID: "456"}).Return(nil)
			},
			mockConfigStorage: func(s *storageMocks.Storage, cfg *config.Config) {
				s.On("Model").Return(cfg)
				s.On("Save").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateRunning, lc.AppState())
				assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateConfigured, lc.ConfigState())
			},
			configAssertions: func(c *config.Config) {
				assert.Equal(t, config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 11, 8, 00, 12, 00, time.UTC),
				}, c.Credentials)
			},
		},
		{
			name: "if Easee API returned an error, login procedure should be skipped with no side effects on config",
			loginData: &cliffApp.LoginCredentials{
				Username: "test-user",
				Password: "test-password",
			},
			cfg: &config.Config{},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockClient: func(c *easeeMocks.Client) {
				c.
					On("Login", "test-user", "test-password").
					Return(nil, errors.New("oops"))
				c.On("Ping").Return(nil)
			},
			mockConfigStorage: func(s *storageMocks.Storage, _ *config.Config) {
				s.AssertNotCalled(t, "Model")
				s.AssertNotCalled(t, "Save")
			},
			wantErr: true,
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateNotConfigured, lc.AppState())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
			configAssertions: func(c *config.Config) {
				assert.Equal(t, config.Credentials{}, c.Credentials)
			},
		},
		{
			name: "if config storage returned an error when saving credentials, lifecycle should not be configured",
			loginData: &cliffApp.LoginCredentials{
				Username: "test-user",
				Password: "test-password",
			},
			cfg: &config.Config{},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockClient: func(c *easeeMocks.Client) {
				c.
					On("Login", "test-user", "test-password").
					Return(&easee.LoginData{
						AccessToken:  "access-token",
						ExpiresIn:    86400,
						AccessClaims: []string{"User"},
						TokenType:    "Bearer",
						RefreshToken: "refresh-token",
					}, nil)
				c.On("Ping").Return(nil)
			},
			mockConfigStorage: func(s *storageMocks.Storage, cfg *config.Config) {
				s.On("Model").Return(cfg)
				s.On("Save").Return(errors.New("oops"))
			},
			wantErr: true,
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateNotConfigured, lc.AppState())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
			configAssertions: func(c *config.Config) {
				assert.Equal(t, config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 11, 8, 00, 12, 00, time.UTC),
				}, c.Credentials)
			},
		},
		{
			name: "successful login, but ping failed for some reason",
			loginData: &cliffApp.LoginCredentials{
				Username: "test-user",
				Password: "test-password",
			},
			cfg: &config.Config{},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockClient: func(c *easeeMocks.Client) {
				c.
					On("Login", "test-user", "test-password").
					Return(&easee.LoginData{
						AccessToken:  "access-token",
						ExpiresIn:    86400,
						AccessClaims: []string{"User"},
						TokenType:    "Bearer",
						RefreshToken: "refresh-token",
					}, nil)
				c.On("Chargers").Return([]easee.Charger{
					{ID: "123"},
					{ID: "456"},
				}, nil)
				c.On("Ping").Return(errors.New("oops"))
			},
			mockAdapter: func(a *adapterMocks.ExtendedAdapter) {
				a.On("CreateThing", "123", easee.Info{ChargerID: "123"}).Return(nil)
				a.On("CreateThing", "456", easee.Info{ChargerID: "456"}).Return(nil)
			},
			mockConfigStorage: func(s *storageMocks.Storage, cfg *config.Config) {
				s.On("Model").Return(cfg)
				s.On("Save").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateRunning, lc.AppState())
				assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateConfigured, lc.ConfigState())
			},
			configAssertions: func(c *config.Config) {
				assert.Equal(t, config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 11, 8, 00, 12, 00, time.UTC),
				}, c.Credentials)
			},
		},
		{
			name: "failed to register all things",
			loginData: &cliffApp.LoginCredentials{
				Username: "test-user",
				Password: "test-password",
			},
			cfg: &config.Config{},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockClient: func(c *easeeMocks.Client) {
				c.
					On("Login", "test-user", "test-password").
					Return(&easee.LoginData{
						AccessToken:  "access-token",
						ExpiresIn:    86400,
						AccessClaims: []string{"User"},
						TokenType:    "Bearer",
						RefreshToken: "refresh-token",
					}, nil)
				c.On("Chargers").Return([]easee.Charger{
					{ID: "123"},
					{ID: "456"},
				}, nil)
				c.On("Ping").Return(nil)
			},
			mockAdapter: func(a *adapterMocks.ExtendedAdapter) {
				a.On("CreateThing", "123", easee.Info{ChargerID: "123"}).Return(nil)
				a.On("CreateThing", "456", easee.Info{ChargerID: "456"}).Return(errors.New("oops"))
			},
			mockConfigStorage: func(s *storageMocks.Storage, cfg *config.Config) {
				s.On("Model").Return(cfg)
				s.On("Save").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateNotConfigured, lc.AppState())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
			configAssertions: func(c *config.Config) {
				assert.Equal(t, config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 11, 8, 00, 12, 00, time.UTC),
				}, c.Credentials)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			lc := lifecycle.New()
			if tt.setLifecycle != nil {
				tt.setLifecycle(lc)
			}

			adapterMock := new(adapterMocks.ExtendedAdapter)
			if tt.mockAdapter != nil {
				tt.mockAdapter(adapterMock)
			}

			cfgStorageMock := new(storageMocks.Storage)
			if tt.mockConfigStorage != nil {
				tt.mockConfigStorage(cfgStorageMock, tt.cfg)
			}

			clientMock := new(easeeMocks.Client)
			if tt.mockClient != nil {
				tt.mockClient(clientMock)
			}

			defer func() {
				adapterMock.AssertExpectations(t)
				cfgStorageMock.AssertExpectations(t)
				clientMock.AssertExpectations(t)
			}()

			cfgService := config.NewService(cfgStorageMock)

			application := app.New(adapterMock, cfgService, lc, nil, clientMock)

			err := application.Login(tt.loginData)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.lifecycleAssertions != nil {
				tt.lifecycleAssertions(lc)
			}

			if tt.configAssertions != nil {
				tt.configAssertions(tt.cfg)
			}
		})
	}
}

func TestApplication_Logout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		cfg                 *config.Config
		setLifecycle        func(lc *lifecycle.Lifecycle)
		mockAdapter         func(a *adapterMocks.ExtendedAdapter)
		mockConfigStorage   func(s *storageMocks.Storage, cfg *config.Config)
		wantErr             bool
		lifecycleAssertions func(lc *lifecycle.Lifecycle)
		configAssertions    func(c *config.Config)
	}{
		{
			name: "successful config, lifecycle and adapter reset",
			cfg: &config.Config{
				Credentials: config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC),
				},
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockAdapter: func(a *adapterMocks.ExtendedAdapter) {
				a.On("DestroyAllThings").Return(nil)
			},
			mockConfigStorage: func(s *storageMocks.Storage, cfg *config.Config) {
				*cfg = config.Config{}

				s.On("Reset").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateNotConfigured, lc.AppState())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
			configAssertions: func(c *config.Config) {
				assert.Equal(t, &config.Config{}, c)
			},
		},
		{
			name: "adapter error on destroying all things",
			cfg: &config.Config{
				Credentials: config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC),
				},
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockAdapter: func(a *adapterMocks.ExtendedAdapter) {
				a.On("DestroyAllThings").Return(errors.New("test error"))
			},
			wantErr: true,
		},
		{
			name: "config service error on reset",
			cfg: &config.Config{
				Credentials: config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC),
				},
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockAdapter: func(a *adapterMocks.ExtendedAdapter) {
				a.On("DestroyAllThings").Return(nil)
			},
			mockConfigStorage: func(s *storageMocks.Storage, cfg *config.Config) {
				s.On("Reset").Return(errors.New("test error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lc := lifecycle.New()
			if tt.setLifecycle != nil {
				tt.setLifecycle(lc)
			}

			adapterMock := new(adapterMocks.ExtendedAdapter)
			if tt.mockAdapter != nil {
				tt.mockAdapter(adapterMock)
			}

			cfgStorageMock := new(storageMocks.Storage)
			if tt.mockConfigStorage != nil {
				tt.mockConfigStorage(cfgStorageMock, tt.cfg)
			}

			defer func() {
				adapterMock.AssertExpectations(t)
				cfgStorageMock.AssertExpectations(t)
			}()

			cfgService := config.NewService(cfgStorageMock)

			application := app.New(adapterMock, cfgService, lc, nil, nil)

			err := application.Logout()

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			if tt.lifecycleAssertions != nil {
				tt.lifecycleAssertions(lc)
			}

			if tt.configAssertions != nil {
				tt.configAssertions(tt.cfg)
			}
		})
	}
}

func TestApplication_Initialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		cfg                 *config.Config
		setLifecycle        func(lc *lifecycle.Lifecycle)
		mockAdapter         func(a *adapterMocks.ExtendedAdapter)
		mockClient          func(c *easeeMocks.Client)
		mockConfigStorage   func(s *storageMocks.Storage, cfg *config.Config)
		wantErr             bool
		lifecycleAssertions func(lc *lifecycle.Lifecycle)
	}{
		{
			name: "successful thing initialization",
			cfg: &config.Config{
				Credentials: config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC),
				},
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAdapter: func(a *adapterMocks.ExtendedAdapter) {
				a.On("InitializeThings").Return(nil)
			},
			mockClient: func(c *easeeMocks.Client) {
				c.On("Ping").Return(nil)
			},
			mockConfigStorage: func(s *storageMocks.Storage, cfg *config.Config) {
				s.On("Model").Return(cfg)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateRunning, lc.AppState())
				assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateConfigured, lc.ConfigState())
			},
		},
		{
			name: "empty credentials - unconfigure lifecycle",
			cfg:  &config.Config{},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAdapter: func(a *adapterMocks.ExtendedAdapter) {
				a.On("InitializeThings").Return(nil)
			},
			mockClient: func(c *easeeMocks.Client) {
				c.On("Ping").Return(nil)
			},
			mockConfigStorage: func(s *storageMocks.Storage, cfg *config.Config) {
				s.On("Model").Return(cfg)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateNotConfigured, lc.AppState())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
		},
		{
			name: "error on thing initialization",
			cfg:  &config.Config{},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAdapter: func(a *adapterMocks.ExtendedAdapter) {
				a.On("InitializeThings").Return(errors.New("oops"))
			},
			mockClient: func(c *easeeMocks.Client) {
				c.On("Ping").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateNotConfigured, lc.AppState())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
			wantErr: true,
		},
		{
			name: "successful thing initialization, but ping failed",
			cfg: &config.Config{
				Credentials: config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC),
				},
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAdapter: func(a *adapterMocks.ExtendedAdapter) {
				a.On("InitializeThings").Return(nil)
			},
			mockClient: func(c *easeeMocks.Client) {
				c.On("Ping").Return(errors.New("oops"))
			},
			mockConfigStorage: func(s *storageMocks.Storage, cfg *config.Config) {
				s.On("Model").Return(cfg)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateRunning, lc.AppState())
				assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateConfigured, lc.ConfigState())
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lc := lifecycle.New()
			if tt.setLifecycle != nil {
				tt.setLifecycle(lc)
			}

			adapterMock := new(adapterMocks.ExtendedAdapter)
			if tt.mockAdapter != nil {
				tt.mockAdapter(adapterMock)
			}

			cfgStorageMock := new(storageMocks.Storage)
			if tt.mockConfigStorage != nil {
				tt.mockConfigStorage(cfgStorageMock, tt.cfg)
			}

			clientMock := new(easeeMocks.Client)
			if tt.mockClient != nil {
				tt.mockClient(clientMock)
			}

			defer func() {
				adapterMock.AssertExpectations(t)
				cfgStorageMock.AssertExpectations(t)
				clientMock.AssertExpectations(t)
			}()

			cfgService := config.NewService(cfgStorageMock)

			application := app.New(adapterMock, cfgService, lc, nil, clientMock)

			err := application.Initialize()

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			if tt.lifecycleAssertions != nil {
				tt.lifecycleAssertions(lc)
			}
		})
	}
}
