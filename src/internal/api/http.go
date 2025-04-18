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

	lock              sync.RWMutex
	lastMaxCurrentSet map[string]time.Time
}

// NewHTTPClient returns a new instance of Easee HTTPClient.
func NewHTTPClient(cfgSrv *config.Service, http *http.Client, baseURL string) HTTPClient {
	return &httpClient{
		httpClient:        http,
		baseURL:           baseURL,
		lastMaxCurrentSet: make(map[string]time.Time),
		cfgSrv:            cfgSrv,
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
		return nil, errors.Wrap(err, "login request failed")
	}

	defer resp.Body.Close()

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
		return nil, errors.Wrap(err, "token refresh request failed")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logFailedResponse(resp)

		return nil, c.handleFailedResponse(resp, "token refresh request failed: unexpected status code")
	}

	loginData := &model.Credentials{}

	err = c.readResponseBody(resp, loginData)
	if err != nil {
		return nil, errors.Wrap(err, "could not read token refresh response body")
	}

	return loginData, nil
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
		return errors.Wrap(err, "update max current request failed")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		c.logFailedResponse(resp)

		return c.handleFailedResponse(resp, "update max current request failed: unexpected status code")
	}

	return nil
}

func (c *httpClient) UpdateDynamicCurrent(accessToken, chargerID string, current float64) error {
	if c.shouldBackoffWithMaxCurrentChange(chargerID) {
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
		return errors.Wrap(err, "update dynamic current request failed")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		c.logFailedResponse(resp)

		return c.handleFailedResponse(resp, "update dynamic current request failed: unexpected status code")
	}

	c.registerMaxCurrentChange(chargerID)

	return nil
}

func (c *httpClient) StopCharging(accessToken, chargerID string) error {
	// When stop charging command is sent, Easee sets dynamic current to 0.
	// That's why a protection against changing offered current more often than once in 30 seconds is needed.
	if c.shouldBackoffWithMaxCurrentChange(chargerID) {
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
		return errors.Wrap(err, "stop charging request failed")
	}

	defer resp.Body.Close()

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
		return errors.Wrap(err, "could not perform cable lock api call")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		c.logFailedResponse(resp)

		return c.handleFailedResponse(resp, "cable lock request failed: unexpected status code")
	}

	return nil
}

func (c *httpClient) ChargerConfig(accessToken, chargerID string) (*model.ChargerConfig, error) {
	u := c.buildURL(chargerConfigURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodGet, u).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create charger state request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "could not perform charger state api call")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logFailedResponse(resp)

		return nil, c.handleFailedResponse(resp, "charger state request failed: unexpected status code")
	}

	state := &model.ChargerConfig{}

	err = c.readResponseBody(resp, state)
	if err != nil {
		return nil, errors.Wrap(err, "could not read charger state response body")
	}

	return state, nil
}

func (c *httpClient) ChargerSiteInfo(accessToken, chargerID string) (*model.ChargerSiteInfo, error) {
	u := c.buildURL(chargerSiteURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodGet, u).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create charger site info request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "could not perform charger site info api call")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logFailedResponse(resp)

		return nil, c.handleFailedResponse(resp, "charger site info request failed: unexpected status code")
	}

	state := &model.ChargerSiteInfo{}

	err = c.readResponseBody(resp, state)
	if err != nil {
		return nil, errors.Wrap(err, "could not read charger site info response body")
	}

	return state, nil
}

func (c *httpClient) Chargers(accessToken string) ([]model.Charger, error) {
	req, err := newRequestBuilder(http.MethodGet, c.buildURL(chargersURI)).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chargers request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch chargers from api")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logFailedResponse(resp)

		return nil, c.handleFailedResponse(resp, "chargers request failed: unexpected status code")
	}

	var chargers []model.Charger

	if err := c.readResponseBody(resp, &chargers); err != nil {
		return nil, errors.Wrap(err, "failed to read request body")
	}

	return chargers, nil
}

func (c *httpClient) ChargerDetails(accessToken string, chargerID string) (model.ChargerDetails, error) {
	u := c.buildURL(chargerDetailsURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodGet, u).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		build()
	if err != nil {
		return model.ChargerDetails{}, errors.Wrap(err, "failed to create charger details state request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return model.ChargerDetails{}, errors.Wrap(err, "could not perform charger details api call")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logFailedResponse(resp)

		return model.ChargerDetails{}, c.handleFailedResponse(resp, "charger details request failed: unexpected status code")
	}

	chargerDetails := model.ChargerDetails{}

	err = c.readResponseBody(resp, &chargerDetails)
	if err != nil {
		return model.ChargerDetails{}, errors.Wrap(err, "could not read charger details response body")
	}

	return chargerDetails, nil
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
		return errors.Wrap(err, "failed to perform ping request")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logFailedResponse(resp)

		return c.handleFailedResponse(resp, "ping request failed: unexpected status code")
	}

	return nil
}

func (c *httpClient) buildURL(path string, args ...interface{}) string {
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

func (c *httpClient) readResponseBody(r *http.Response, body interface{}) error {
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

func (c *httpClient) shouldBackoffWithMaxCurrentChange(chargerID string) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	lastMaxCurrentSet, ok := c.lastMaxCurrentSet[chargerID]
	if !ok {
		return false
	}

	if clock.Now().Sub(lastMaxCurrentSet) >= c.cfgSrv.GetOfferedCurrentWaitTime() {
		return false
	}

	return true
}

func (c *httpClient) registerMaxCurrentChange(chargerID string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.lastMaxCurrentSet[chargerID] = clock.Now()
}
