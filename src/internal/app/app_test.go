package app_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	cliffApp "github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/manifest"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	"github.com/michalkurzeja/go-clock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/edge-easee-adapter/internal/app"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/test"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/fakes"
	mockapi "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/api"
	mockedmanifest "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/manifest"
	mocksignalr "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/signalr"
)

//nolint:godox
// TODO: Move as much test cases as possible to component tests to avoid excessive mocking.

func TestApplication_GetManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mockLoader func(l *mockedmanifest.Loader)
		want       *manifest.Manifest
		wantErr    bool
	}{
		{
			name: "manifest is loaded successfully",
			mockLoader: func(l *mockedmanifest.Loader) {
				l.On("Load").Return(test.LoadManifest(t), nil)
			},
			want: test.LoadManifest(t),
		},
		{
			name: "manifest loading fails",
			mockLoader: func(l *mockedmanifest.Loader) {
				l.On("Load").Return(nil, errors.New("failed to load manifest"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loaderMock := mockedmanifest.NewLoader(t)
			if tt.mockLoader != nil {
				tt.mockLoader(loaderMock)
			}

			a := app.New(nil, nil, lifecycle.New(nil), loaderMock, nil, nil, nil, nil)

			got, err := a.GetManifest()

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplication_Configure_RejectsWrongModelType(t *testing.T) {
	t.Parallel()

	a := app.New(nil, nil, nil, nil, nil, nil, nil, nil)

	assert.Error(t, a.Configure("anything"))
}

func TestApplication_Uninstall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		cfg                 *config.Config
		credentials         config.Credentials
		setLifecycle        func(lc *lifecycle.Lifecycle)
		mockAdapter         func(a *mockedadapter.Adapter)
		wantErr             bool
		lifecycleAssertions func(lc *lifecycle.Lifecycle)
		configAssertions    func(c *config.Config)
	}{
		{
			name: "successful config, lifecycle and adapter reset",
			cfg:  &config.Config{},
			credentials: config.Credentials{
				AccessToken:          "access-token",
				RefreshToken:         "refresh-token",
				AccessTokenExpiresAt: time.Date(2022, time.September, 10, 8, 0, 12, 0, time.UTC),
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("DestroyAllThings").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth())
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
			cfg:  &config.Config{},
			credentials: config.Credentials{
				AccessToken:          "access-token",
				RefreshToken:         "refresh-token",
				AccessTokenExpiresAt: time.Date(2022, time.September, 10, 8, 0, 12, 0, time.UTC),
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("DestroyAllThings").Return(errors.New("test error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lc := lifecycle.New(nil)
			if tt.setLifecycle != nil {
				tt.setLifecycle(lc)
			}

			adapterMock := new(mockedadapter.Adapter)
			if tt.mockAdapter != nil {
				tt.mockAdapter(adapterMock)
			}

			defer adapterMock.AssertExpectations(t)

			storage := fakes.NewConfigStorage(t, tt.cfg, config.Factory)
			cfgService := config.NewService(storage)

			credentials := newCredentialsStore(t, tt.credentials)
			application := app.New(adapterMock, cfgService, lc, nil, nil, nil, nil, credentials)

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
				tt.configAssertions(cfgService.Model())
			}
		})
	}
}

func TestApplication_Login(t *testing.T) { //nolint:paralleltest
	clock.Mock(time.Date(2022, time.September, 10, 8, 0, 12, 0, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name                string
		loginData           *cliffApp.LoginCredentials
		setLifecycle        func(lc *lifecycle.Lifecycle)
		mockAdapter         func(a *mockedadapter.Adapter)
		mockClient          func(c *mockapi.Client)
		mockAuthenticator   func(a *mockapi.Authenticator)
		mockSignalRClient   func(c *mocksignalr.Client)
		wantErr             bool
		lifecycleAssertions func(lc *lifecycle.Lifecycle)
	}{
		{
			name: "if login was successful, credentials and lifecycle should be set up",
			loginData: &cliffApp.LoginCredentials{
				Username: "test-user",
				Password: "test-password",
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAuthenticator: func(a *mockapi.Authenticator) {
				a.On("Login", "test-user", "test-password").Return(nil)
			},
			mockClient: func(c *mockapi.Client) {
				c.On("Chargers").Return([]model.Charger{
					{ID: "123"},
					{ID: "456"},
				}, nil)
				c.On("ChargerDetails", "123").Return(model.ChargerDetails{Product: "xd"}, nil)
				c.On("ChargerDetails", "456").Return(model.ChargerDetails{Product: "edi"}, nil)
				c.On("Ping").Return(nil)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("EnsureThings", adapter.ThingSeeds{
					&adapter.ThingSeed{
						ID: "123",
						Info: easee.Info{
							ChargerID: "123",
							Product:   "xd",
						},
					},
					&adapter.ThingSeed{
						ID: "456",
						Info: easee.Info{
							ChargerID: "456",
							Product:   "edi",
						},
					},
				}).Return(nil)
			},
			mockSignalRClient: func(c *mocksignalr.Client) {
				c.On("Start")
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthRunning, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateConfigured, lc.ConfigState())
			},
		},
		{
			name: "if Easee API returned an error, login procedure should be skipped with no side effects on config",
			loginData: &cliffApp.LoginCredentials{
				Username: "test-user",
				Password: "test-password",
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockAuthenticator: func(a *mockapi.Authenticator) {
				a.
					On("Login", "test-user", "test-password").
					Return(errors.New("oops"))
			},
			wantErr: true,
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
		},
		{
			name: "successful login, but ping failed for some reason",
			loginData: &cliffApp.LoginCredentials{
				Username: "test-user",
				Password: "test-password",
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAuthenticator: func(a *mockapi.Authenticator) {
				a.On("Login", "test-user", "test-password").Return(nil)
			},
			mockClient: func(c *mockapi.Client) {
				c.On("Chargers").Return([]model.Charger{
					{ID: "123"},
					{ID: "456"},
				}, nil)
				c.On("ChargerDetails", "123").Return(model.ChargerDetails{Product: "xd"}, nil)
				c.On("ChargerDetails", "456").Return(model.ChargerDetails{Product: "edi"}, nil)
				c.On("Ping").Return(errors.New("oops"))
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("EnsureThings", adapter.ThingSeeds{
					&adapter.ThingSeed{
						ID: "123",
						Info: easee.Info{
							ChargerID: "123",
							Product:   "xd",
						},
					},
					&adapter.ThingSeed{
						ID: "456",
						Info: easee.Info{
							ChargerID: "456",
							Product:   "edi",
						},
					},
				}).Return(nil)
			},
			mockSignalRClient: func(c *mocksignalr.Client) {
				c.On("Start")
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthRunning, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateConfigured, lc.ConfigState())
			},
		},
		{
			name: "failed to register all things",
			loginData: &cliffApp.LoginCredentials{
				Username: "test-user",
				Password: "test-password",
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAuthenticator: func(a *mockapi.Authenticator) {
				a.On("Login", "test-user", "test-password").Return(nil)
			},
			mockClient: func(c *mockapi.Client) {
				c.On("Chargers").Return([]model.Charger{
					{ID: "123"},
					{ID: "456"},
				}, nil)
				c.On("ChargerDetails", "123").Return(model.ChargerDetails{Product: "xd"}, nil)
				c.On("ChargerDetails", "456").Return(model.ChargerDetails{Product: "edi"}, nil)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("EnsureThings", adapter.ThingSeeds{
					&adapter.ThingSeed{
						ID: "123",
						Info: easee.Info{
							ChargerID: "123",
							Product:   "xd",
						},
					},
					&adapter.ThingSeed{
						ID: "456",
						Info: easee.Info{
							ChargerID: "456",
							Product:   "edi",
						},
					},
				}).Return(errors.New("oops"))
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			lc := lifecycle.New(nil)
			if tt.setLifecycle != nil {
				tt.setLifecycle(lc)
			}

			adapterMock := mockedadapter.NewAdapter(t)
			if tt.mockAdapter != nil {
				tt.mockAdapter(adapterMock)
			}

			clientMock := mockapi.NewClient(t)
			if tt.mockClient != nil {
				tt.mockClient(clientMock)
			}

			authMock := mockapi.NewAuthenticator(t)
			if tt.mockAuthenticator != nil {
				tt.mockAuthenticator(authMock)
			}

			signalRClientMock := mocksignalr.NewClient(t)
			if tt.mockSignalRClient != nil {
				tt.mockSignalRClient(signalRClientMock)
			}

			cfgService := config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory))
			application := app.New(adapterMock, cfgService, lc, nil, clientMock, authMock, signalRClientMock, newCredentialsStore(t, config.Credentials{}))

			err := application.Login(tt.loginData)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.lifecycleAssertions != nil {
				tt.lifecycleAssertions(lc)
			}
		})
	}
}

func TestApplication_Logout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		setLifecycle        func(lc *lifecycle.Lifecycle)
		authLogoutError     error
		wantErr             bool
		lifecycleAssertions func(lc *lifecycle.Lifecycle)
		mockSignalRClient   func(c *mocksignalr.Client)
	}{
		{
			name: "successful config, lifecycle and adapter reset",
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
				assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
			},
			mockSignalRClient: func(c *mocksignalr.Client) {
				c.On("Close").Return(nil)
			},
		},
		{
			name: "adapter error on destroying all things",
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			authLogoutError: errors.New("error"),
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthError, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
			mockSignalRClient: func(c *mocksignalr.Client) {
				c.On("Close").Return(nil)
			},

			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lc := lifecycle.New(nil)
			tt.setLifecycle(lc)

			clientMock := new(mockapi.Client)
			clientMock.On("Ping").Return(errors.New("oops"))

			authMock := &mockapi.Authenticator{}
			authMock.On("Logout").Return(tt.authLogoutError)

			signalRClientMock := mocksignalr.NewClient(t)
			if tt.mockSignalRClient != nil {
				tt.mockSignalRClient(signalRClientMock)
			}

			cfgService := config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory))
			application := app.New(nil, cfgService, lc, nil, clientMock, authMock, signalRClientMock, newCredentialsStore(t, config.Credentials{}))
			err := application.Logout()

			assert.Equal(t, tt.wantErr, err != nil, "failed error expectation")
			tt.lifecycleAssertions(lc)
		})
	}
}

func TestApplication_Initialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		cfg                 *config.Config
		credentials         config.Credentials
		setLifecycle        func(lc *lifecycle.Lifecycle)
		mockAdapter         func(a *mockedadapter.Adapter)
		mockClient          func(c *mockapi.Client)
		wantErr             bool
		lifecycleAssertions func(lc *lifecycle.Lifecycle)
	}{
		{
			name: "successful thing initialization",
			cfg:  &config.Config{},
			credentials: config.Credentials{
				AccessToken:          "access-token",
				RefreshToken:         "refresh-token",
				AccessTokenExpiresAt: time.Date(2022, time.September, 10, 8, 0, 12, 0, time.UTC),
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("InitializeThings").Return(nil)
			},
			mockClient: func(c *mockapi.Client) {
				c.On("Ping").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthRunning, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateConfigured, lc.ConfigState())
			},
		},
		{
			name: "empty credentials - unconfigure lifecycle",
			cfg:  &config.Config{},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("InitializeThings").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
		},
		{
			name: "error on thing initialization",
			cfg:  &config.Config{},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("InitializeThings").Return(errors.New("oops"))
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
			wantErr: true,
		},
		{
			name: "successful thing initialization, but ping failed",
			cfg:  &config.Config{},
			credentials: config.Credentials{
				AccessToken:          "access-token",
				RefreshToken:         "refresh-token",
				AccessTokenExpiresAt: time.Date(2022, time.September, 10, 8, 0, 12, 0, time.UTC),
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("InitializeThings").Return(nil)
			},
			mockClient: func(c *mockapi.Client) {
				c.On("Ping").Return(errors.New("oops"))
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthRunning, lc.AppHealth())
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

			lc := lifecycle.New(nil)
			if tt.setLifecycle != nil {
				tt.setLifecycle(lc)
			}

			adapterMock := mockedadapter.NewAdapter(t)
			if tt.mockAdapter != nil {
				tt.mockAdapter(adapterMock)
			}

			// Read by the boot-time selection adoption.
			adapterMock.On("Things").Return([]adapter.Thing{}).Maybe()

			clientMock := new(mockapi.Client)
			if tt.mockClient != nil {
				tt.mockClient(clientMock)
			}

			defer func() {
				adapterMock.AssertExpectations(t)
				clientMock.AssertExpectations(t)
			}()

			storage := fakes.NewConfigStorage(t, tt.cfg, config.Factory)
			cfgService := config.NewService(storage)

			credentials := newCredentialsStore(t, tt.credentials)
			application := app.New(adapterMock, cfgService, lc, nil, clientMock, nil, nil, credentials)

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

func newCredentialsStore(t *testing.T, credentials config.Credentials) *config.CredentialsStore {
	t.Helper()

	return config.NewCredentialsStoreWithStorage(
		fakes.NewConfigStorage(t, &credentials, func() *config.Credentials { return &config.Credentials{} }),
	)
}

func TestApplication_Configure_Selection(t *testing.T) {
	t.Parallel()

	chargers := []model.Charger{{ID: "123"}, {ID: "456"}}

	tests := []struct {
		name         string
		selected     []string
		owned        []adapter.Thing
		wantErr      string
		wantSeeded   []string
		wantSelected []string
	}{
		{
			name:         "valid subset is seeded and persisted",
			selected:     []string{"456"},
			wantSeeded:   []string{"456"},
			wantSelected: []string{"456"},
		},
		{
			name:     "unknown device id is rejected before anything is mutated",
			selected: []string{"unknown"},
			wantErr:  "unknown device IDs",
		},
		{
			name:         "empty selection includes every charger",
			wantSeeded:   []string{"123", "456"},
			wantSelected: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clientMock := mockapi.NewClient(t)
			clientMock.On("Chargers").Return(chargers, nil)

			adapterMock := mockedadapter.NewAdapter(t)
			adapterMock.On("Things").Return(tt.owned).Maybe()

			var seeded []string

			for _, id := range tt.wantSeeded {
				clientMock.On("ChargerDetails", id).Return(model.ChargerDetails{}, nil)
			}

			if tt.wantErr == "" {
				adapterMock.On("EnsureThings", mock.Anything).Run(func(args mock.Arguments) {
					for _, seed := range args.Get(0).(adapter.ThingSeeds) { //nolint:forcetypeassert
						seeded = append(seeded, seed.ID)
					}
				}).Return(nil)
			}

			cfgService := config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory))
			signalRClientMock := mocksignalr.NewClient(t)
			signalRClientMock.On("Start").Maybe()

			application := app.New(adapterMock, cfgService, lifecycle.New(nil), nil, clientMock, nil, signalRClientMock,
				newCredentialsStore(t, config.Credentials{}))

			err := application.Configure(&config.Config{
				PublicConfig: config.PublicConfig{SelectedDevices: tt.selected},
			})

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, cfgService.SelectedDevices(), "a rejected configure must not persist a selection")

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantSeeded, seeded)
			assert.Equal(t, tt.wantSelected, cfgService.SelectedDevices())
		})
	}
}
