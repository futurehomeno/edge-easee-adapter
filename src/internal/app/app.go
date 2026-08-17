package app

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
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
	// The hook is attached to the global logger, so it has to be attached once however many
	// applications are built: a second hook records every warning into the diag report twice.
	errorHookOnce.Do(func() {
		errorHook = formatters.NewErrorHook()
		log.AddHook(errorHook)
	})

	return &application{
		ad:            ad,
		mfLoader:      mfLoader,
		lifecycle:     lc,
		cfgService:    cfgService,
		client:        client,
		auth:          auth,
		signalRClient: signalRClient,
		credentials:   credentials,
		errorHook:     errorHook,
	}
}

var (
	errorHookOnce sync.Once
	errorHook     *formatters.ErrorHook
)

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

	// An absent selection includes every charger and must not silently drop live ones down
	// to the auto-cap: keep what is already seeded, minus chargers Easee no longer lists -
	// persisting a stale ID makes every later boot burn its missing-selected retries before
	// it gives up on them. An empty one is the user deselecting everything, and is obeyed.
	selected := cfg.SelectedDevices
	if selected.IncludeAll() {
		if owned := a.ownedListedChargers(chargers); len(owned) > 0 {
			selected = owned
		}
	}

	selected, err = a.applyChargers(chargers, effectiveSelection(chargers, selected))
	if err != nil {
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

	// Every step runs even if an earlier one fails: credentials live in their own secrets
	// store, so a failed config reset must not leave the Easee tokens on the hub.
	var errs error

	if err := a.ad.DestroyAllThings(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("destroy all things: %w", err))
	}

	if err := a.cfgService.Reset(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("reset configuration: %w", err))
	}

	if err := a.credentials.ClearCredentials(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("clear credentials: %w", err))
	}

	if errs != nil {
		return errs
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
func (a *application) configureChargers(selected selection.Selection) error {
	chargers, err := a.client.Chargers()
	if err != nil {
		return fmt.Errorf("fetch available chargers: %w", err)
	}

	// A hub upgraded while logged out never reached the boot-time adoption, because
	// Initialize returns before it when the secrets store is empty. Catch up here.
	// Adopt every owned charger, not just the listed ones: a selection filtered against
	// this response is a subset of it by construction, so the retry budget below would
	// never see a partial list and a transiently absent charger would lose its thing.
	// Nothing owned being listed is a different Easee account rather than a short list,
	// and adopting there would leave the hub selecting devices it can never see.
	if selected.IncludeAll() {
		if len(a.ownedListedChargers(chargers)) > 0 {
			selected = a.ownedDeviceIDs()

			if err := a.cfgService.SetSelectedDevices(selected); err != nil {
				return fmt.Errorf("persist adopted selection: %w", err)
			}
		}
	}

	// An empty response leaves nothing for the adoption above to work off, so nothing below
	// would flag the owned chargers as missing and the sync would destroy every thing on the
	// hub. Put them through the same retry budget instead: a successful fetch listing nothing
	// is no more trustworthy than one listing too little.
	if len(chargers) == 0 && selected.IncludeAll() {
		if owned := a.ownedDeviceIDs(); len(owned) > 0 {
			selected = owned
		}
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
		// every later login. Clone rather than slices.Clone: cleaning out the last entry
		// must leave an empty selection ("no chargers"), not a nil one ("every charger").
		log.Infof("[app] Remove vanished chargers %v from the saved selection", missing)

		selected = slices.DeleteFunc(selected.Clone(), func(id string) bool {
			return slices.Contains(missing, id)
		})

		if err := a.cfgService.SetSelectedDevices(selected); err != nil {
			return fmt.Errorf("persist cleaned selection: %w", err)
		}
	}

	a.missingRetries = 0
	a.lastMissing = ""

	// Persist an auto-selection so the manifest and the seeds work off the same list
	// instead of each re-deriving it.
	if capped := effectiveSelection(chargers, selected); len(capped) > 0 && selected.IncludeAll() {
		selected = capped

		if err := a.cfgService.SetSelectedDevices(selected); err != nil {
			return fmt.Errorf("persist auto-selected devices: %w", err)
		}
	}

	synced, err := a.applyChargers(chargers, selected)
	if err != nil {
		return err
	}

	if slices.Equal(synced, selected) {
		return nil
	}

	if err := a.cfgService.SetSelectedDevices(synced); err != nil {
		return fmt.Errorf("persist selection after sync: %w", err)
	}

	return nil
}

// applyChargers seeds the selected chargers and returns the selection the sync leaves behind.
// Details are fetched up front because a seed function cannot fail, and the fetch must be the
// sole authority on what exists.
func (a *application) applyChargers(chargers []model.Charger, selected selection.Selection) (selection.Selection, error) {
	products := make(map[string]string, len(chargers))

	for _, charger := range chargers {
		if !selected.Contains(charger.ID) {
			continue
		}

		details, err := a.client.ChargerDetails(charger.ID)
		if err != nil {
			return nil, fmt.Errorf("fetch charger details: %w", err)
		}

		products[charger.ID] = details.Product
	}

	seeds, excluded, err := adapter.SyncThings(
		a.ad,
		func() ([]model.Charger, error) { return chargers, nil },
		selected,
		func(charger model.Charger) *adapter.ThingSeed {
			return &adapter.ThingSeed{
				ID:   charger.ID,
				Info: easee.Info{ChargerID: charger.ID, Product: products[charger.ID]},
			}
		},
	)
	if err != nil {
		return nil, fmt.Errorf("sync things: %w", err)
	}

	if len(seeds) > 0 {
		a.signalRClient.Start()
	}

	if len(excluded) == 0 {
		return selected, nil
	}

	// The sync announced an exclusion for each of these, so keeping them selected would
	// re-announce the same vanished charger on every later sync.
	return slices.DeleteFunc(selected.Clone(), func(id string) bool {
		return slices.Contains(excluded, id)
	}), nil
}

// adoptSeededSelection makes config catch up with reality on upgrades from versions without
// a selection: things were seeded but selected_devices is absent, so a later re-seed would
// fall back to "all" (or to the auto-cap) instead of this hub's own chargers.
func (a *application) adoptSeededSelection() {
	if !a.cfgService.SelectedDevices().IncludeAll() {
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

// ownedListedChargers returns the chargers this hub holds as things and the account still
// lists, so a stale ID cannot be carried forward into the selection.
func (a *application) ownedListedChargers(chargers []model.Charger) selection.Selection {
	owned := a.ownedDeviceIDs()
	stale := missingSelected(chargers, owned)

	return slices.DeleteFunc(owned, func(id string) bool { return slices.Contains(stale, id) })
}

// ownedDeviceIDs returns the chargers the adapter currently holds as things.
func (a *application) ownedDeviceIDs() selection.Selection {
	things := a.ad.Things()

	ids := make(selection.Selection, 0, len(things))
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

// effectiveSelection turns an unconfigured (nil) selection into the concrete charger list,
// capped at maxAutoSelected. Materialising it is what makes cmd.thing.delete stick: "every
// charger" cannot express an exclusion, so the next sync would recreate the deleted thing.
// The cap keeps installer accounts, which expose hundreds of chargers, from flooding the hub
// with things nobody asked for.
func effectiveSelection(chargers []model.Charger, selected selection.Selection) selection.Selection {
	if !selected.IncludeAll() {
		return selected
	}

	auto := make(selection.Selection, 0, min(len(chargers), maxAutoSelected))

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
