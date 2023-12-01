package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/thoas/go-funk"
)

const (
	loginURI        = "/api/accounts/login"
	tokenRefreshURI = "/api/accounts/refresh_token" //nolint:gosec
	chargersURI     = "/api/chargers"
	healthURI       = "/health"

	chargerConfigURITemplate   = "/api/chargers/%s/config"
	chargerSiteURITemplate     = "/api/chargers/%s/site"
	chargerSettingsURITemplate = "/api/chargers/%s/settings"
	chargerStartURITemplate    = "/api/chargers/%s/commands/resume_charging"
	chargerStopURITemplate     = "/api/chargers/%s/commands/pause_charging"
	cableLockURITemplate       = "/api/chargers/%s/commands/lock_state"
	chargerSessionsURITemplate = "/api/sessions/charger/%s/sessions/descending?limit=2"

	authorizationHeader = "Authorization"
	contentTypeHeader   = "Content-Type"

	jsonContentType = "application/*+json"

	pauseChargingCurrent = 0.
)

// HTTPClient represents Easee HTTP API Client.
type HTTPClient interface {
	// UpdateMaxCurrent updates max charger current.
	UpdateMaxCurrent(accessToken, chargerID string, current float64) error
	// UpdateDynamicCurrent updates dynamic charger current, dynamic current is used as offered current.
	UpdateDynamicCurrent(accessToken, chargerID string, current float64) error
	// Login logs the user in the Easee API and retrieves credentials.
	Login(userName, password string) (*Credentials, error)
	// RefreshToken retrieves new credentials based on an access token and a refresh token.
	RefreshToken(accessToken, refreshToken string) (*Credentials, error)
	// StartCharging starts charging session for the selected charger.
	StartCharging(accessToken, chargerID string) error
	// StopCharging stops charging session for the selected charger.
	StopCharging(accessToken, chargerID string) error
	// SetCableLock locks/unlocks the cable for the selected charger.
	SetCableLock(accessToken, chargerID string, locked bool) error
	// ChargerConfig retrieves charger config.
	ChargerConfig(accessToken, chargerID string) (*ChargerConfig, error)
	// ChargerSiteInfo retrieves charger rated current, rated current is used as supported max current.
	ChargerSiteInfo(accessToken, chargerID string) (*ChargerSiteInfo, error)
	// ChargerSessions retrieves at most two latest charging sessions including current if present.
	ChargerSessions(accessToken, chargerID string) (ChargeSessions, error)
	// Chargers returns all available chargers.
	Chargers(accessToken string) ([]Charger, error)
	// Ping checks if an external service is available.
	Ping(accessToken string) error
}

type httpClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewHTTPClient returns a new instance of Easee HTTPClient.
func NewHTTPClient(http *http.Client, baseURL string) HTTPClient {
	return &httpClient{
		httpClient: http,
		baseURL:    baseURL,
	}
}

func (c *httpClient) Login(userName, password string) (*Credentials, error) {
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

	resp, err := c.performRequest(req, http.StatusOK)
	if err != nil {
		return nil, errors.Wrap(err, "login request failed")
	}

	defer resp.Body.Close()

	credentials := &Credentials{}

	err = c.readResponseBody(resp, credentials)
	if err != nil {
		return nil, errors.Wrap(err, "could not read response body")
	}

	return credentials, nil
}

func (c *httpClient) RefreshToken(accessToken, refreshToken string) (*Credentials, error) {
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

	resp, err := c.performRequest(req, http.StatusOK)
	if err != nil {
		if resp == nil {
			return nil, err
		}

		return nil, HTTPError{err: errors.Wrap(err, "failed to perform token refresh api call"), Status: resp.StatusCode, Body: resp.Body}
	}

	defer resp.Body.Close()

	loginData := &Credentials{}

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

	resp, err := c.performRequest(req, http.StatusAccepted)
	if err != nil {
		return errors.Wrap(err, "update max current request failed")
	}

	defer resp.Body.Close()

	return nil
}

func (c *httpClient) UpdateDynamicCurrent(accessToken, chargerID string, current float64) error {
	u := c.buildURL(chargerSettingsURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodPost, u).
		withBody(dynamicCurrentBody{DynamicChargerCurrent: current}).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		addHeader(contentTypeHeader, jsonContentType).
		build()
	if err != nil {
		return errors.Wrap(err, "failed to create dynamic current request")
	}

	resp, err := c.performRequest(req, http.StatusAccepted)
	if err != nil {
		return errors.Wrap(err, "update dynamic current request failed")
	}

	defer resp.Body.Close()

	return nil
}

func (c *httpClient) StartCharging(accessToken, chargerID string) error {
	u := c.buildURL(chargerStartURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodPost, u).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		build()
	if err != nil {
		return errors.Wrap(err, "failed to create start charging request")
	}

	resp, err := c.performRequest(req, http.StatusAccepted)
	if err != nil {
		return errors.Wrap(err, "start charging request failed")
	}

	defer resp.Body.Close()

	return nil
}

func (c *httpClient) StopCharging(accessToken, chargerID string) error {
	u := c.buildURL(chargerStopURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodPost, u).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		build()
	if err != nil {
		return errors.Wrap(err, "failed to create stop charging request")
	}

	resp, err := c.performRequest(req, http.StatusAccepted)
	if err != nil {
		return errors.Wrap(err, "stop charging request failed")
	}

	defer resp.Body.Close()

	return nil
}

func (c *httpClient) SetCableLock(accessToken, chargerID string, locked bool) error {
	u := c.buildURL(cableLockURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodPost, u).
		withBody(cableLockBody{State: locked}).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		addHeader(contentTypeHeader, jsonContentType).
		build()
	if err != nil {
		return errors.Wrap(err, "failed to create cable lock request")
	}

	resp, err := c.performRequest(req, http.StatusAccepted)
	if err != nil {
		return errors.Wrap(err, "could not perform cable lock api call")
	}

	defer resp.Body.Close()

	return nil
}

func (c *httpClient) ChargerConfig(accessToken, chargerID string) (*ChargerConfig, error) {
	u := c.buildURL(chargerConfigURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodGet, u).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create charger state request")
	}

	resp, err := c.performRequest(req, http.StatusOK)
	if err != nil {
		return nil, errors.Wrap(err, "could not perform charger state api call")
	}

	defer resp.Body.Close()

	state := &ChargerConfig{}

	err = c.readResponseBody(resp, state)
	if err != nil {
		return nil, errors.Wrap(err, "could not read charger state response body")
	}

	return state, nil
}

func (c *httpClient) ChargerSiteInfo(accessToken, chargerID string) (*ChargerSiteInfo, error) {
	u := c.buildURL(chargerSiteURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodGet, u).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create charger state request")
	}

	resp, err := c.performRequest(req, http.StatusOK)
	if err != nil {
		return nil, errors.Wrap(err, "could not perform charger state api call")
	}

	defer resp.Body.Close()

	state := &ChargerSiteInfo{}

	err = c.readResponseBody(resp, state)
	if err != nil {
		return nil, errors.Wrap(err, "could not read charger state response body")
	}

	return state, nil
}

func (c *httpClient) ChargerSessions(accessToken, chargerID string) (ChargeSessions, error) {
	u := c.buildURL(chargerSessionsURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodGet, u).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create charger state request")
	}

	resp, err := c.performRequest(req, http.StatusOK)
	if err != nil {
		return nil, errors.Wrap(err, "could not perform charger state api call")
	}

	defer resp.Body.Close()

	sessions := ChargeSessions{}

	err = c.readResponseBody(resp, &sessions)
	if err != nil {
		return nil, errors.Wrap(err, "could not read charger state response body")
	}

	return sessions, nil
}

func (c *httpClient) Chargers(accessToken string) ([]Charger, error) {
	req, err := newRequestBuilder(http.MethodGet, c.buildURL(chargersURI)).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chargers request")
	}

	resp, err := c.performRequest(req, http.StatusOK)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch chargers from api")
	}

	defer resp.Body.Close()

	var chargers []Charger

	if err := c.readResponseBody(resp, &chargers); err != nil {
		return nil, errors.Wrap(err, "failed to read request body")
	}

	return chargers, nil
}

func (c *httpClient) Ping(accessToken string) error {
	req, err := newRequestBuilder(http.MethodGet, c.buildURL(healthURI)).
		addHeader(authorizationHeader, c.bearerTokenHeader(accessToken)).
		build()
	if err != nil {
		return errors.Wrap(err, "failed to create ping request")
	}

	resp, err := c.performRequest(req, http.StatusOK)
	if err != nil {
		return errors.Wrap(err, "failed to perform ping request")
	}

	defer resp.Body.Close()

	return nil
}

func (c *httpClient) buildURL(path string, args ...interface{}) string {
	return c.baseURL + fmt.Sprintf(path, args...)
}

func (c *httpClient) performRequest(req *http.Request, wantResponseCode int) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "could not perform http call")
	}

	if resp.StatusCode != wantResponseCode {
		return resp, errors.Errorf("expected response code to be %d, but got %d instead", wantResponseCode, resp.StatusCode)
	}

	return resp, nil
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
