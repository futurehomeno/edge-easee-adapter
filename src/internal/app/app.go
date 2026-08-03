package app

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	cliffApp "github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/debug/formatters"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/manifest"
	"github.com/futurehomeno/cliffhanger/selection"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
)

const (
	// maxMissingSelectedRetries bounds how long a selected charger missing from the
	// chargers list blocks a re-seed before it is dropped.
	maxMissingSelectedRetries = 3
	// maxAutoSelected caps how many chargers an unconfigured install auto-selects.
	maxAutoSelected = 10

	blockConfiguration    = "configuration"
	configSelectedDevices = "selected_devices"
)

type ApplicationWithToken interface {
	Application
	RefreshToken()
}

// Application is an interface representing a service responsible for preparing an application manifest and configuring app.
type Application interface {
	cliffApp.App
	cliffApp.LogginableApp
	cliffApp.CheckableApp
	cliffApp.InitializableApp
}

// New creates new instance of an Application.
func New(
	ad adapter.Adapter,
	cfgService *config.Service,
	lc *lifecycle.Lifecycle,
	mfLoader manifest.Loader,
	client api.Client,
	auth api.Authenticator,
	signalRClient signalr.Client,
	credentials *config.CredentialsStore,
) ApplicationWithToken {
	hook := formatters.NewErrorHook()
	log.AddHook(hook)

	return &application{
		ad:            ad,
		mfLoader:      mfLoader,
		lifecycle:     lc,
		cfgService:    cfgService,
		client:        client,
		auth:          auth,
		signalRClient: signalRClient,
		credentials:   credentials,
		errorHook:     hook,
	}
}

type application struct {
	ad            adapter.Adapter
	cfgService    *config.Service
	lifecycle     *lifecycle.Lifecycle
	mfLoader      manifest.Loader
	client        api.Client
	auth          api.Authenticator
	signalRClient signalr.Client
	credentials   *config.CredentialsStore
	errorHook     *formatters.ErrorHook

	missingRetries int
	lastMissing    string
}

func (a *application) ErrorsReport() ([]string, error) {
	return a.errorHook.ErrorsReport()
}

func (a *application) GetManifest() (*manifest.Manifest, error) {
	mf, err := a.mfLoader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	ready := a.lifecycle.ConnectionState() == lifecycle.ConnStateConnected &&
		a.lifecycle.AuthState() == lifecycle.AuthStateAuthenticated

	err = selection.PrepareManifest(mf, selection.Block{
		Block:    blockConfiguration,
		Config:   configSelectedDevices,
		NotReady: "Warning! You're currently not connected to Easee, please return to the previous page and log in.",
		Failed:   "Warning! Failed to retrieve any chargers from Easee, please try again later.",
	}, ready, func() ([]model.Charger, error) { return a.client.Chargers() }, func(charger model.Charger) manifest.SelectOption {
		return manifest.SelectOption{Val: charger.ID, Label: map[string]string{"en": chargerLabel(charger)}}
	})
	if err != nil {
		log.Errorf("[app] Prepare device selector. err: %v", err)
	}

	return mf, nil
}

func chargerLabel(charger model.Charger) string {
	if charger.Name == "" {
		return charger.ID
	}

	return charger.Name + " (" + charger.ID + ")"
}

func (a *application) Configure(model any) error {
	cfg, ok := model.(*config.Config)
	if !ok {
		return fmt.Errorf("configure: invalid config model type %T", model)
	}

	chargers, err := a.client.Chargers()
	if err != nil {
		return fmt.Errorf("configure: fetch available chargers: %w", err)
	}

	if err := validateSelectedDevices(chargers, cfg.SelectedDevices); err != nil {
		return err
	}

	// An empty selection from the UI must not silently drop live chargers down to
	// the auto-cap: keep what is already seeded, minus chargers Easee no longer
	// lists - persisting a stale ID makes every later boot burn its missing-selected
	// retries before it gives up on them.
	selected := cfg.SelectedDevices
	if len(selected) == 0 {
		owned := a.ownedDeviceIDs()
		stale := missingSelected(chargers, owned)
		selected = slices.DeleteFunc(owned, func(id string) bool { return slices.Contains(stale, id) })
	}

	selected = effectiveSelection(chargers, selected)

	if err := a.applyChargers(chargers, selected); err != nil {
		return err
	}

	if err := a.cfgService.SetSelectedDevices(selected); err != nil {
		return fmt.Errorf("configure: persist selected_devices: %w", err)
	}

	return nil
}

func (a *application) Check() error {
	return nil
}

func (a *application) CheckInterval() time.Duration {
	return 0
}

func (a *application) RefreshToken() {
	prevConnState := a.lifecycle.ConnectionState()

	if err := a.client.Ping(); err != nil {
		switch {
		case errors.Is(err, api.ErrNotLoggedIn):
			log.Debugf("[auth] Refresh token skipped: %v", err)
		case errors.Is(err, api.ErrRefreshBackoff):
			log.Debugf("[auth] Refresh token skipped (backoff): %v", err)
		default:
			log.Warnf("[auth] Refresh token failed API client disconnected err: %v", err)
		}
		a.lifecycle.SetConnState(lifecycle.ConnStateDisconnected)
		return
	}

	if prevConnState == lifecycle.ConnStateDisconnected {
		log.Info("[auth] API client reconnected")
	}

	if prevConnState != lifecycle.ConnStateConnected {
		a.lifecycle.SetConnState(lifecycle.ConnStateConnected)
	}
}

func (a *application) Uninstall() error {
	log.Info("[app] Uninstall requested: destroying things and resetting config")

	if err := a.ad.DestroyAllThings(); err != nil {
		return fmt.Errorf("destroy all things: %w", err)
	}

	if err := a.cfgService.Reset(); err != nil {
		return fmt.Errorf("reset configuration: %w", err)
	}

	// Credentials live in their own secrets store, so resetting the config no longer clears them.
	if err := a.credentials.ClearCredentials(); err != nil {
		return fmt.Errorf("clear credentials: %w", err)
	}

	a.lifecycle.MarkNotConfigured()

	return nil
}

func (a *application) Login(credentials *cliffApp.LoginCredentials) error {
	defer func() { _ = a.Check() }()

	if err := a.auth.Login(credentials.Username, credentials.Password); err != nil {
		a.lifecycle.MarkNotConfigured()

		return fmt.Errorf("failed to login as '%s': %w", credentials.Username, err)
	}

	// A hub upgraded while logged out never reached the boot-time adoption, because
	// Initialize returns before it when the secrets store is empty. Without this the
	// selection would still be empty here and the cap below could drop chargers this
	// hub already had.
	a.adoptSeededSelection()

	if err := a.configureChargers(a.cfgService.SelectedDevices()); err != nil {
		a.lifecycle.MarkNotConfigured()

		return fmt.Errorf("failed to register chargers on login: %w", err)
	}

	a.lifecycle.MarkRunning()

	a.RefreshToken()

	return nil
}

func (a *application) Initialize() error {
	if err := a.ad.InitializeThings(); err != nil {
		return fmt.Errorf("failed to initialize things: %w", err)
	}

	if err := a.cfgService.Save(); err != nil {
		return fmt.Errorf("failed to save configs at application initialization: %w", err)
	}

	if a.credentials.Credentials().Empty() {
		a.lifecycle.MarkNotConfigured()

		return nil
	}

	a.adoptSeededSelection()

	a.lifecycle.SetAppHealth(lifecycle.AppHealthRunning, nil)
	a.lifecycle.SetConfigState(lifecycle.ConfigStateConfigured)
	a.lifecycle.SetAuthState(lifecycle.AuthStateAuthenticated)

	a.RefreshToken()

	return nil
}

func (a *application) Logout() error {
	log.Info("[app] Logout requested via cmd.auth.logout")

	if err := a.signalRClient.Close(); err != nil {
		log.Warnf("[app] Disconnect signalR client. err: %v", err)
	}

	if err := a.auth.Logout(); err != nil {
		a.lifecycle.SetAppHealth(lifecycle.AppHealthError, nil)
		a.lifecycle.SetAuthState(lifecycle.AuthStateNotAuthenticated)
		a.lifecycle.SetConfigState(lifecycle.ConfigStateNotConfigured)

		return err
	}

	a.lifecycle.MarkNotConfigured()

	return nil
}

// configureChargers reconciles things with the selected chargers, fetching the list first
// so the selection policy below can reason about what Easee currently reports.
func (a *application) configureChargers(selected []string) error {
	chargers, err := a.client.Chargers()
	if err != nil {
		return fmt.Errorf("fetch available chargers: %w", err)
	}

	// A partial /chargers response must not drop selected chargers: the sync destroys
	// every persisted thing absent from the seeds. Retry a few times, then seed without
	// them so a legitimately removed charger cannot block startup.
	if missing := missingSelected(chargers, selected); len(missing) > 0 {
		// Reset the budget when the missing set changes, so a newly missing charger
		// gets full retries instead of inheriting an exhausted counter. Sorted, or
		// re-ticking the same devices in another order would look like a new set.
		slices.Sort(missing)

		if key := strings.Join(missing, ","); key != a.lastMissing {
			a.lastMissing = key
			a.missingRetries = 0
		}

		if a.missingRetries < maxMissingSelectedRetries {
			a.missingRetries++

			return fmt.Errorf("selected devices %v not found in chargers list; refusing partial re-seed (%d/%d)",
				missing, a.missingRetries, maxMissingSelectedRetries)
		}

		log.Warnf("[app] Selected devices %v still missing after %d retries, seed without them", missing, maxMissingSelectedRetries)

		// Drop them from the persisted selection too, or the budget is spent again on
		// every later login. Never down to an empty selection: empty means "every
		// charger", so cleaning the last entry would widen a selection the user
		// narrowed. That case keeps the stale ID and pays the retries again.
		remaining := slices.DeleteFunc(slices.Clone(selected), func(id string) bool {
			return slices.Contains(missing, id)
		})

		if len(remaining) > 0 {
			log.Infof("[app] Remove vanished chargers %v from the saved selection", missing)

			selected = remaining

			if err := a.cfgService.SetSelectedDevices(selected); err != nil {
				return fmt.Errorf("persist cleaned selection: %w", err)
			}
		}
	}

	a.missingRetries = 0
	a.lastMissing = ""

	// Persist an auto-selection so the manifest and the seeds work off the same list
	// instead of each re-deriving it.
	if capped := effectiveSelection(chargers, selected); len(capped) > 0 && len(selected) == 0 {
		selected = capped

		if err := a.cfgService.SetSelectedDevices(selected); err != nil {
			return fmt.Errorf("persist auto-selected devices: %w", err)
		}
	}

	return a.applyChargers(chargers, selected)
}

// applyChargers seeds the selected chargers. Details are fetched up front because a seed
// function cannot fail, and the fetch must be the sole authority on what exists.
func (a *application) applyChargers(chargers []model.Charger, selected []string) error {
	products := make(map[string]string, len(chargers))

	for _, charger := range chargers {
		if len(selected) > 0 && !slices.Contains(selected, charger.ID) {
			continue
		}

		details, err := a.client.ChargerDetails(charger.ID)
		if err != nil {
			return fmt.Errorf("fetch charger details: %w", err)
		}

		products[charger.ID] = details.Product
	}

	// SyncThings reads an empty selection as "no devices" and a nil one as "all", while
	// the persisted config uses empty for "all" - translate rather than migrate the field.
	var selection []string
	if len(selected) > 0 {
		selection = selected
	}

	seeds, err := adapter.SyncThings(
		a.ad,
		func() ([]model.Charger, error) { return chargers, nil },
		selection,
		func(charger model.Charger) *adapter.ThingSeed {
			return &adapter.ThingSeed{
				ID:   charger.ID,
				Info: easee.Info{ChargerID: charger.ID, Product: products[charger.ID]},
			}
		},
	)
	if err != nil {
		return fmt.Errorf("sync things: %w", err)
	}

	if len(seeds) > 0 {
		a.signalRClient.Start()
	}

	return nil
}

// adoptSeededSelection makes config catch up with reality on upgrades from versions without
// a selection: things were seeded but selected_devices is empty, so a later re-seed would
// fall back to "all" (or to the auto-cap) instead of this hub's own chargers.
func (a *application) adoptSeededSelection() {
	if len(a.cfgService.SelectedDevices()) > 0 {
		return
	}

	owned := a.ownedDeviceIDs()
	if len(owned) == 0 {
		return
	}

	if err := a.cfgService.SetSelectedDevices(owned); err != nil {
		log.Warnf("[app] Adopt seeded chargers as selection. err: %v", err)
	}
}

// ownedDeviceIDs returns the chargers the adapter currently holds as things.
func (a *application) ownedDeviceIDs() []string {
	things := a.ad.Things()

	ids := make([]string, 0, len(things))
	for _, t := range things {
		ids = append(ids, t.InclusionReport().DeviceId)
	}

	return ids
}

// missingSelected returns persisted selections that no longer appear in the chargers list.
func missingSelected(chargers []model.Charger, selected []string) []string {
	present := make(map[string]struct{}, len(chargers))
	for _, charger := range chargers {
		present[charger.ID] = struct{}{}
	}

	var missing []string

	for _, id := range selected {
		if _, ok := present[id]; !ok {
			missing = append(missing, id)
		}
	}

	return missing
}

// effectiveSelection turns an unconfigured (empty) selection into the concrete charger
// list, capped at maxAutoSelected. Materialising it is what makes cmd.thing.delete stick:
// an empty selection means "every charger" and cannot express an exclusion, so the next
// sync would recreate the deleted thing. The cap keeps installer accounts, which expose
// hundreds of chargers, from flooding the hub with things nobody asked for.
func effectiveSelection(chargers []model.Charger, selected []string) []string {
	if len(selected) > 0 {
		return selected
	}

	auto := make([]string, 0, min(len(chargers), maxAutoSelected))

	for _, charger := range chargers {
		if charger.ID == "" {
			continue
		}

		auto = append(auto, charger.ID)

		if len(auto) == maxAutoSelected {
			break
		}
	}

	if len(chargers) > maxAutoSelected {
		log.Warnf("[app] No devices selected out of %d available, auto-select first %d: %v", len(chargers), len(auto), auto)
	}

	return auto
}

// validateSelectedDevices rejects selections referencing IDs absent from the freshly fetched
// chargers list, so Configure fails fast before mutating state.
func validateSelectedDevices(chargers []model.Charger, selected []string) error {
	if len(selected) == 0 {
		return nil
	}

	if unknown := missingSelected(chargers, selected); len(unknown) > 0 {
		return fmt.Errorf("configure: unknown device IDs: %v", unknown)
	}

	return nil
}
