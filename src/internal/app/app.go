package app

import (
	"fmt"

	"github.com/futurehomeno/cliffhanger/adapter"
	cliffApp "github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/manifest"
	"github.com/pkg/errors"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
)

// Application is an interface representing a service responsible for preparing an application manifest and configuring app.
type Application interface {
	cliffApp.App
	cliffApp.LogginableApp
	cliffApp.CheckableApp
	cliffApp.InitializableApp
}

// New creates new instance of an Application.
func New(ad adapter.ExtendedAdapter, cfgService *config.Service, lc *lifecycle.Lifecycle, mf *manifest.Manifest, client easee.Client) Application {
	return &application{
		ad:         ad,
		mf:         mf,
		lifecycle:  lc,
		cfgService: cfgService,
		client:     client,
	}
}

type application struct {
	ad         adapter.ExtendedAdapter
	cfgService *config.Service
	lifecycle  *lifecycle.Lifecycle
	mf         *manifest.Manifest
	client     easee.Client
}

func (a *application) GetManifest() (*manifest.Manifest, error) {
	return a.mf, nil
}

func (a *application) Configure(_ interface{}) error {
	return nil
}

func (a *application) Uninstall() error {
	err := a.ad.DestroyAllThings()
	if err != nil {
		return errors.New("failed to destroy all things")
	}

	err = a.cfgService.Reset()
	if err != nil {
		return errors.New("failed to reset configuration")
	}

	a.lifecycle.SetAppState(lifecycle.AppStateNotConfigured, nil)
	a.lifecycle.SetConfigState(lifecycle.ConfigStateNotConfigured)
	a.lifecycle.SetConnectionState(lifecycle.ConnStateDisconnected)
	a.lifecycle.SetAuthState(lifecycle.AuthStateNotAuthenticated)

	return nil
}

func (a *application) Login(credentials *cliffApp.LoginCredentials) error {
	defer a.Check() //nolint:errcheck

	if err := a.login(credentials); err != nil {
		a.lifecycle.SetAppState(lifecycle.AppStateNotConfigured, nil)
		a.lifecycle.SetAuthState(lifecycle.AuthStateNotAuthenticated)
		a.lifecycle.SetConfigState(lifecycle.ConfigStateNotConfigured)

		return errors.Wrap(err, "failed to login")
	}

	if err := a.registerChargers(); err != nil {
		a.lifecycle.SetAppState(lifecycle.AppStateNotConfigured, nil)
		a.lifecycle.SetAuthState(lifecycle.AuthStateNotAuthenticated)
		a.lifecycle.SetConfigState(lifecycle.ConfigStateNotConfigured)

		return errors.Wrap(err, "failed to register chargers on login")
	}

	a.lifecycle.SetAppState(lifecycle.AppStateRunning, nil)
	a.lifecycle.SetAuthState(lifecycle.AuthStateAuthenticated)
	a.lifecycle.SetConfigState(lifecycle.ConfigStateConfigured)

	return nil
}

func (a *application) Check() error {
	if err := a.client.Ping(); err != nil {
		a.lifecycle.SetConnectionState(lifecycle.ConnStateDisconnected)

		return nil //nolint:nilerr
	}

	a.lifecycle.SetConnectionState(lifecycle.ConnStateConnected)

	return nil
}

func (a *application) Initialize() error {
	defer a.Check() //nolint:errcheck

	if err := a.ad.InitializeThings(); err != nil {
		return errors.Wrap(err, "failed to initialize things")
	}

	if a.cfgService.GetCredentials().Empty() {
		a.lifecycle.SetAppState(lifecycle.AppStateNotConfigured, nil)
		a.lifecycle.SetConfigState(lifecycle.ConfigStateNotConfigured)
		a.lifecycle.SetAuthState(lifecycle.AuthStateNotAuthenticated)

		return nil
	}

	a.lifecycle.SetAppState(lifecycle.AppStateRunning, nil)
	a.lifecycle.SetConfigState(lifecycle.ConfigStateConfigured)
	a.lifecycle.SetAuthState(lifecycle.AuthStateAuthenticated)

	return nil
}

func (a *application) login(credentials *cliffApp.LoginCredentials) error {
	loginData, err := a.client.Login(credentials.Username, credentials.Password)
	if err != nil {
		return errors.Wrap(err, "failed to authenticate the user in Easee API")
	}

	err = a.cfgService.SetCredentials(loginData.AccessToken, loginData.RefreshToken, loginData.ExpiresIn)
	if err != nil {
		return errors.Wrap(err, "failed to save credentials in config")
	}

	return nil
}

func (a *application) Logout() error {
	return a.Uninstall()
}

func (a *application) registerChargers() error {
	chargers, err := a.client.Chargers()
	if err != nil {
		return errors.Wrap(err, "failed to fetch available chargers from Easee API")
	}

	for _, charger := range chargers {
		if err := a.ad.CreateThing(charger.ID, easee.Info{ChargerID: charger.ID}); err != nil {
			return fmt.Errorf("failed to register charger ID %s: %w", charger.ID, err)
		}
	}

	return nil
}
