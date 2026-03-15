package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/michalkurzeja/go-clock"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrTimeout      = errors.New("timeout")
	ErrServer       = errors.New("server_error")
	ErrUnexpected   = errors.New("unexpected")
	ErrTransport    = errors.New("transport_error")
)

const (
	loginURI        = "/api/accounts/login"
	tokenRefreshURI = "/api/accounts/refresh_token" //nolint:gosec
	chargersURI     = "/api/chargers"
	healthURI       = "/health"

	chargerConfigURITemplate   = "/api/chargers/%s/config"
	chargerSiteURITemplate     = "/api/chargers/%s/site"
	chargerSettingsURITemplate = "/api/chargers/%s/settings"
	chargerStopURITemplate     = "/api/chargers/%s/commands/pause_charging"
	cableLockURITemplate       = "/api/chargers/%s/commands/lock_state"
	chargerSessionsURITemplate = "/api/sessions/charger/%s/sessions/descending?limit=2"
	chargerDetailsURITemplate  = "/api/chargers/%s/details?alwaysGetChargerAccessLevel=false"

	authorizationHeader = "Authorization"
	contentTypeHeader   = "Content-Type"

	jsonContentType = "application/*+json"
)

// HTTPClient represents Easee HTTP API Client.
type HTTPClient interface {
	// UpdateMaxCurrent updates max charger current.
	UpdateMaxCurrent(accessToken, chargerID string, current float64) error
	// UpdateDynamicCurrent updates dynamic charger current, dynamic current is used as offered current.
	UpdateDynamicCurrent(accessToken, chargerID string, current float64) error
	UpdateEaseePhaseMode(accessToken, chargerID string, phaseMode model.EaseePhaseModeT) error
	SetActivePhases(accessToken, chargerID string, phase1, phase2, phase3 bool, defCurrent float64) error
	// Login logs the user in the Easee API and retrieves credentials.
	Login(userName, password string) (*model.Credentials, error)
	// RefreshToken retrieves new credentials based on an access token and a refresh token.
	RefreshToken(accessToken, refreshToken string) (*model.Credentials, error)
	// StopCharging stops charging session for the selected charger.
	StopCharging(accessToken, chargerID string) error
	// ChargerConfig retrieves charger config.
	ChargerConfig(accessToken, chargerID string) (*model.ChargerConfig, error)
	// ChargerSiteInfo retrieves charger rated current, rated current is used as supported max current.
	ChargerSiteInfo(accessToken, chargerID string) (*model.ChargerSiteInfo, error)
	// Chargers returns all available chargers.
	Chargers(accessToken string) ([]model.Charger, error)
	// ChargerDetails returns product's name.
	ChargerDetails(accessToken string, chargerID string) (model.ChargerDetails, error)
	// SetCableAlwaysLocked sets cable always lock state.
	SetCableAlwaysLocked(accessToken string, chargerID string, locked bool) error
	// Ping checks if an external service is available.
	Ping(accessToken string) error
}

type httpClient struct {
	httpClient *http.Client
	baseURL    string
	cfgSrv     *config.Service

	lock                  sync.RWMutex
	lastStop              map[string]time.Time
	lastDynamicCurrentSet map[string]time.Time
	lastDynamicCurrentVal map[string]float64
	lastPhaseModeChange   map[string]time.Time
	lastEaseePhaseMode    map[string]model.EaseePhaseModeT
}

// NewHTTPClient returns a new instance of Easee HTTPClient.
func NewHTTPClient(cfgSrv *config.Service, http *http.Client, baseURL string) HTTPClient {
	return &httpClient{
		httpClient:            http,
		baseURL:               baseURL,
		lastStop:              make(map[string]time.Time),
		lastDynamicCurrentSet: make(map[string]time.Time),
		lastDynamicCurrentVal: make(map[string]float64),
		lastPhaseModeChange:   make(map[string]time.Time),
		lastEaseePhaseMode:    make(map[string]model.EaseePhaseModeT),
		cfgSrv:                cfgSrv,
	}
}

func (c *httpClient) Login(userName, password string) (*model.Credentials, error) {
	body := loginBody{
		Username: strings.TrimSpace(userName),
		Password: strings.TrimSpace(password),
	}

	req, err := newRequestBuilder(http.MethodPost, c.buildURL(loginURI)).
		withBody(body).
		addHeader(contentTypeHeader, jsonContentType).
		build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create login request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTransport, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		c.logFailedResponse(resp)

		return nil, c.handleFailedResponse(resp, "login request failed: unexpected status code")
	}

	credentials := &model.Credentials{}

	err = c.readResponseBody(resp, credentials)
	if err != nil {
		return nil, errors.Wrap(err, "could not read response body")
	}

	return credentials, nil
}

func (c *httpClient) RefreshToken(accessToken, refreshToken string) (*model.Credentials, error) {
	body := refreshBody{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	req, err := newRequestBuilder(http.MethodPost, c.buildURL(tokenRefreshURI)).
		withBody(body).
		addHeader(contentTypeHeader, jsonContentType).
		build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create token refresh request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTransport, err)
	}

	defer func() { _ = resp.Body.Close() }()
	var reason error

	switch resp.StatusCode {
	case http.StatusOK:
		loginData := &model.Credentials{}

		err = c.readResponseBody(resp, loginData)
		if err != nil {
			return nil, errors.Wrap(err, "could not read token refresh response body")
		}

		return loginData, nil

	case http.StatusUnauthorized:
		reason = ErrUnauthorized

	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		reason = ErrTimeout

	case http.StatusInternalServerError:
		reason = ErrServer

	default:
		reason = ErrUnexpected
	}

	c.logFailedResponse(resp)
	return nil, fmt.Errorf("%w status code=%d", reason, resp.StatusCode)
}

func (c *httpClient) UpdateMaxCurrent(accessToken, chargerID string, current float64) error {
	u := c.buildURL(chargerSettingsURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodPost, u).
		withBody(maxCurrentBody{MaxChargerCurrent: current}).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		addHeader(contentTypeHeader, jsonContentType).
		build()
	if err != nil {
		return errors.Wrap(err, "failed to create max current request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransport, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		c.logFailedResponse(resp)
		return c.handleFailedResponse(resp, "update max current request failed: unexpected status code")
	}

	return nil
}

func (c *httpClient) UpdateDynamicCurrent(accessToken, chargerID string, current float64) error {
	if c.shouldBackoffWithDynamicCurrentChange(chargerID) {
		return errors.New("client: failed to update dynamic current: too many requests")
	}

	u := c.buildURL(chargerSettingsURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodPost, u).
		withBody(dynamicCurrentBody{DynamicChargerCurrent: current}).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		addHeader(contentTypeHeader, jsonContentType).
		build()
	if err != nil {
		return errors.Wrap(err, "failed to create dynamic current request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransport, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		c.logFailedResponse(resp)
		return c.handleFailedResponse(resp, "update dynamic current request failed: unexpected status code")
	}

	c.registerDynamicCurrentChange(chargerID, current)

	return nil
}

func (c *httpClient) UpdateEaseePhaseMode(accessToken, chargerID string, easeePhaseMode model.EaseePhaseModeT) error {
	if c.shouldBackoffWithPhaseModeChange(chargerID) {
		return errors.New("client: failed to update phase mode: too many requests")
	}

	u := c.buildURL(chargerSettingsURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodPost, u).
		withBody(setPhaseModeBody{SetPhaseMode: int(easeePhaseMode)}).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		addHeader(contentTypeHeader, jsonContentType).
		build()
	if err != nil {
		return errors.Wrap(err, "failed to create phase mode request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransport, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		c.logFailedResponse(resp)
		return c.handleFailedResponse(resp, "update phase mode request failed: unexpected status code")
	}

	c.registerPhaseModeChange(chargerID)

	return nil
}

func (c *httpClient) SetActivePhases(accessToken, chargerID string, phase1, phase2, phase3 bool, defCurrent float64) error {
	if c.shouldBackoffWithPhaseModeChange(chargerID) {
		return errors.New("client: failed to update phase mode: too many requests")
	}

	if easeePhaseMode, ok := c.lastEaseePhaseMode[chargerID]; !ok || easeePhaseMode != model.EaseePhaseModeAuto {
		if err := c.UpdateEaseePhaseMode(accessToken, chargerID, model.EaseePhaseModeAuto); err != nil {
			log.Warnf("UpdateEaseePhaseMode err: %v", err)
		}
	}

	u := c.buildURL(chargerSettingsURITemplate, chargerID)

	dynamicCurrentVal := defCurrent

	if val, ok := c.lastDynamicCurrentVal[chargerID]; ok {
		dynamicCurrentVal = val
	}

	const timeToLiveMin = 120
	payload := setPhaseCurrents{TimeToLive: timeToLiveMin}

	if phase1 {
		payload.SetPhase1 = defCurrent
	}

	if phase2 {
		payload.SetPhase2 = dynamicCurrentVal
	}

	if phase3 {
		payload.SetPhase3 = dynamicCurrentVal
	}

	req, err := newRequestBuilder(http.MethodPost, u).
		withBody(payload).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		addHeader(contentTypeHeader, jsonContentType).
		build()
	if err != nil {
		return errors.Wrap(err, "failed to create phase currents request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransport, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		c.logFailedResponse(resp)
		return c.handleFailedResponse(resp, "update phase currents request failed: unexpected status code")
	}

	return nil
}

func (c *httpClient) StopCharging(accessToken, chargerID string) error {
	// When stop charging command is sent, Easee sets dynamic current to 0.
	// That's why a protection against changing offered current more often than once in 30 seconds is needed.
	if c.shouldBackoffWithStop(chargerID) {
		return errors.New("client: failed to stop charging: too many requests to the charger")
	}

	u := c.buildURL(chargerStopURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodPost, u).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		build()
	if err != nil {
		return errors.Wrap(err, "failed to create stop charging request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransport, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		c.logFailedResponse(resp)

		return c.handleFailedResponse(resp, "stop charging request failed: unexpected status code")
	}

	return nil
}

func (c *httpClient) SetCableAlwaysLocked(accessToken, chargerID string, locked bool) error {
	u := c.buildURL(cableLockURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodPost, u).
		withBody(cableLockStateBody{State: locked}).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		addHeader(contentTypeHeader, jsonContentType).
		build()
	if err != nil {
		return errors.Wrap(err, "failed to create cable lock request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransport, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		c.logFailedResponse(resp)

		return c.handleFailedResponse(resp, "cable lock request failed: unexpected status code")
	}

	return nil
}

func (c *httpClient) ChargerConfig(accessToken, chargerID string) (*model.ChargerConfig, error) {
	var chargerConfig model.ChargerConfig
	rsp, err := c.getResponse(&chargerConfig, c.buildURL(chargerConfigURITemplate, chargerID), accessToken)
	if err != nil {
		return nil, err
	}

	ret, ok := rsp.(*model.ChargerConfig)
	if !ok {
		return nil, fmt.Errorf("failed to cast response to charger config (%T)", ret)
	}

	return ret, nil
}

func (c *httpClient) ChargerSiteInfo(accessToken, chargerID string) (*model.ChargerSiteInfo, error) {
	var siteInfo model.ChargerSiteInfo
	rsp, err := c.getResponse(&siteInfo, c.buildURL(chargerSiteURITemplate, chargerID), accessToken)
	if err != nil {
		return nil, err
	}

	ret, ok := rsp.(*model.ChargerSiteInfo)
	if !ok {
		return nil, errors.New("failed to cast response to charger site info")
	}

	return ret, nil
}

func (c *httpClient) Chargers(accessToken string) ([]model.Charger, error) {
	var chargers []model.Charger
	rsp, err := c.getResponse(&chargers, c.buildURL(chargersURI), accessToken)
	if err != nil {
		return nil, err
	}

	ret, ok := rsp.(*[]model.Charger)
	if !ok {
		return nil, errors.New("failed to cast response to chargers slice")
	}

	return *ret, nil
}

func (c *httpClient) ChargerDetails(accessToken string, chargerID string) (model.ChargerDetails, error) {
	var details model.ChargerDetails
	ret, err := c.getResponse(&details, c.buildURL(chargerDetailsURITemplate, chargerID), accessToken)
	if err != nil {
		return model.ChargerDetails{}, err
	}

	result, ok := ret.(*model.ChargerDetails)
	if !ok {
		return model.ChargerDetails{}, errors.New("failed to cast response to charger details")
	}

	return *result, nil
}

func (c *httpClient) Ping(accessToken string) error {
	req, err := newRequestBuilder(http.MethodGet, c.buildURL(healthURI)).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		build()
	if err != nil {
		return errors.Wrap(err, "failed to create ping request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransport, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		c.logFailedResponse(resp)

		return c.handleFailedResponse(resp, "ping request failed: unexpected status code")
	}

	return nil
}

func (c *httpClient) buildURL(path string, args ...any) string {
	return c.baseURL + fmt.Sprintf(path, args...)
}

func (c *httpClient) handleFailedResponse(resp *http.Response, message string) error {
	e := HTTPError{Message: message}

	if resp != nil {
		e.StatusCode = resp.StatusCode
	}

	return e
}

func (c *httpClient) logFailedResponse(resp *http.Response) {
	if resp == nil {
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).
			Errorf("%s %s %s: failed to read response body", resp.Request.Method, resp.Request.URL.String(), resp.Status)

		return
	}

	log.WithField("body", string(body)).
		Errorf("%s %s resulted in %s", resp.Request.Method, resp.Request.URL.String(), resp.Status)
}

func (c *httpClient) readResponseBody(r *http.Response, body any) error {
	err := json.NewDecoder(r.Body).Decode(body)
	if err != nil {
		return errors.Wrap(err, "could not decode response body")
	}

	if funk.IsEmpty(body) {
		return errors.New("response body does not contain expected data")
	}

	return nil
}

func (c *httpClient) bearerTokenHeader(authToken string) string {
	return "Bearer " + authToken
}

func (c *httpClient) shouldBackoffWithDynamicCurrentChange(chargerID string) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	lastDynamicCurrentSet, ok := c.lastDynamicCurrentSet[chargerID]
	if !ok {
		return false
	}

	if clock.Now().Sub(lastDynamicCurrentSet) >= c.cfgSrv.GetOfferedCurrentWaitTime() {
		return false
	}

	return true
}

func (c *httpClient) shouldBackoffWithStop(chargerID string) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	lastStop, ok := c.lastStop[chargerID]
	if !ok {
		return false
	}

	if clock.Now().Sub(lastStop) >= c.cfgSrv.GetOfferedCurrentWaitTime()/2 {
		return false
	}

	return true
}

func (c *httpClient) shouldBackoffWithPhaseModeChange(chargerID string) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	lastPhaseModeChange, ok := c.lastPhaseModeChange[chargerID]
	if !ok {
		return false
	}

	if clock.Now().Sub(lastPhaseModeChange) >= c.cfgSrv.GePhaseModeSwitchWaitTime() {
		return false
	}

	return true
}

func (c *httpClient) registerDynamicCurrentChange(chargerID string, current float64) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.lastDynamicCurrentVal[chargerID] = current
	c.lastDynamicCurrentSet[chargerID] = clock.Now()
}

func (c *httpClient) registerPhaseModeChange(chargerID string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.lastPhaseModeChange[chargerID] = clock.Now()
}

func (c *httpClient) getResponse(state any, url, accessToken string) (any, error) {
	req, err := newRequestBuilder(http.MethodGet, url).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTransport, err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.WithError(err).Error("failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		c.logFailedResponse(resp)
		return nil, c.handleFailedResponse(resp, "unexpected status code")
	}

	err = c.readResponseBody(resp, state)
	if err != nil {
		return nil, errors.Wrap(err, "could not read response body")
	}

	return state, nil
}
