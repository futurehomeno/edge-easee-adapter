package easee

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/thoas/go-funk"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

const (
	loginURI        = "/api/accounts/login"
	tokenRefreshURI = "/api/accounts/refresh_token" //nolint:gosec
	chargersURI     = "/api/chargers"
	healthURI       = "/health"

	chargerConfigURITemplate   = "/api/chargers/%s/config"
	chargerSettingsURITemplate = "/api/chargers/%s/settings"
	cableLockURITemplate       = "/api/chargers/%s/commands/lock_state"
	observationsURITemplate    = "/api/chargers/%s/observations/%d/%s/%s"

	authorizationHeader = "Authorization"
	contentTypeHeader   = "Content-Type"

	jsonContentType = "application/*+json"

	pauseChargingCurrent = 0.0
)

// Client represents Easee API client.
type Client interface {
	// Login logs the user in the Easee API and retrieves credentials.
	Login(userName, password string) (*Credentials, error)
	// StartCharging starts charging session for the selected charger.
	StartCharging(chargerID string, current float64) error
	// StopCharging stops charging session for the selected charger.
	StopCharging(chargerID string) error
	// SetCableLock locks/unlocks the cable for the selected charger.
	SetCableLock(chargerID string, locked bool) error
	// ChargerConfig retrieves charger config.
	ChargerConfig(chargerID string) (*ChargerConfig, error)
	// Observations retrieves observations (e.g. energy measurements, charger states) for the selected charger based on observation ID and time.
	Observations(chargerID string, obID ObservationID, from, to time.Time) ([]Observation, error)
	// Chargers returns all available chargers.
	Chargers() ([]Charger, error)
	// Ping checks if an external service is available.
	Ping() error
}

type client struct {
	httpClient *http.Client
	cfgSvc     *config.Service
	baseURL    string
}

// NewClient returns a new instance of Client.
func NewClient(
	httpClient *http.Client,
	cfgSvc *config.Service,
	baseURL string,
) Client {
	return &client{
		httpClient: httpClient,
		cfgSvc:     cfgSvc,
		baseURL:    baseURL,
	}
}

func (c *client) Login(userName, password string) (*Credentials, error) {
	body := loginBody{
		Username: userName,
		Password: password,
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

func (c *client) StartCharging(chargerID string, current float64) error {
	token, err := c.accessToken()
	if err != nil {
		return errors.Wrap(err, "failed to get access token")
	}

	u := c.buildURL(chargerSettingsURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodPost, u).
		withBody(chargerCurrentBody{DynamicChargerCurrent: current}).
		addHeader(authorizationHeader, c.bearerTokenHeader(token)).
		addHeader(contentTypeHeader, jsonContentType).
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

func (c *client) StopCharging(chargerID string) error {
	token, err := c.accessToken()
	if err != nil {
		return errors.Wrap(err, "failed to get access token")
	}

	u := c.buildURL(chargerSettingsURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodPost, u).
		withBody(chargerCurrentBody{DynamicChargerCurrent: pauseChargingCurrent}).
		addHeader(authorizationHeader, c.bearerTokenHeader(token)).
		addHeader(contentTypeHeader, jsonContentType).
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

func (c *client) SetCableLock(chargerID string, locked bool) error {
	token, err := c.accessToken()
	if err != nil {
		return errors.Wrap(err, "failed to get access token")
	}

	u := c.buildURL(cableLockURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodGet, u).
		withBody(cableLockBody{State: locked}).
		addHeader(authorizationHeader, c.bearerTokenHeader(token)).
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

func (c *client) ChargerConfig(chargerID string) (*ChargerConfig, error) {
	token, err := c.accessToken()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get access token")
	}

	u := c.buildURL(chargerConfigURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodGet, u).
		addHeader(authorizationHeader, c.bearerTokenHeader(token)).
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

func (c *client) Chargers() ([]Charger, error) {
	token, err := c.accessToken()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get access token")
	}

	req, err := newRequestBuilder(http.MethodGet, c.buildURL(chargersURI)).
		addHeader(authorizationHeader, c.bearerTokenHeader(token)).
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

func (c *client) Ping() error {
	token, err := c.accessToken()
	if err != nil {
		return errors.Wrap(err, "failed to get access token")
	}

	req, err := newRequestBuilder(http.MethodGet, c.buildURL(healthURI)).
		addHeader(authorizationHeader, c.bearerTokenHeader(token)).
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

func (c *client) Observations(chargerID string, obID ObservationID, from, to time.Time) ([]Observation, error) {
	if from.After(to) {
		return nil, errors.New("'from' date must be before 'to' date")
	}

	token, err := c.accessToken()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get access token")
	}

	u := c.buildURL(
		observationsURITemplate,
		chargerID,
		obID,
		url.QueryEscape(from.Format(time.RFC3339)),
		url.QueryEscape(to.Format(time.RFC3339)),
	)

	req, err := newRequestBuilder(http.MethodGet, u).
		addHeader(authorizationHeader, c.bearerTokenHeader(token)).
		addHeader(contentTypeHeader, jsonContentType).
		build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create charger state request")
	}

	resp, err := c.performRequest(req, http.StatusOK)
	if err != nil {
		return nil, errors.Wrap(err, "could not perform charger state api call")
	}

	defer resp.Body.Close()

	var observations []Observation

	err = c.readResponseBody(resp, &observations)
	if err != nil {
		return nil, errors.Wrap(err, "could not read charger state response body")
	}

	return observations, nil
}

func (c *client) buildURL(path string, args ...interface{}) string {
	return c.baseURL + fmt.Sprintf(path, args...)
}

func (c *client) accessToken() (string, error) {
	creds := c.cfgSvc.GetCredentials()
	if creds.Empty() {
		return "", errors.New("credentials are empty - login first")
	}

	if creds.Expired() {
		if err := c.refreshAccessToken(); err != nil {
			return "", errors.Wrap(err, "failed to refresh access token")
		}

		creds = c.cfgSvc.GetCredentials()
	}

	return creds.AccessToken, nil
}

func (c *client) refreshAccessToken() error {
	creds := c.cfgSvc.GetCredentials()
	body := refreshBody{
		AccessToken:  creds.AccessToken,
		RefreshToken: creds.RefreshToken,
	}

	req, err := newRequestBuilder(http.MethodPost, c.buildURL(tokenRefreshURI)).
		withBody(body).
		addHeader(contentTypeHeader, jsonContentType).
		build()
	if err != nil {
		return errors.Wrap(err, "failed to create token refresh request")
	}

	resp, err := c.performRequest(req, http.StatusOK)
	if err != nil {
		return errors.Wrap(err, "failed to perform token refresh api call")
	}

	defer resp.Body.Close()

	var loginData Credentials

	err = c.readResponseBody(resp, &loginData)
	if err != nil {
		return errors.Wrap(err, "could not read token refresh response body")
	}

	err = c.cfgSvc.SetCredentials(loginData.AccessToken, loginData.RefreshToken, loginData.ExpiresIn)
	if err != nil {
		return errors.Wrap(err, "could not save refreshed credentials in config")
	}

	return nil
}

func (c *client) performRequest(req *http.Request, wantResponseCode int) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "could not perform http call")
	}

	if resp.StatusCode != wantResponseCode {
		return nil, errors.Errorf("expected response code to be %d, but got %d instead", wantResponseCode, resp.StatusCode)
	}

	return resp, nil
}

func (c *client) readResponseBody(r *http.Response, body interface{}) error {
	err := json.NewDecoder(r.Body).Decode(body)
	if err != nil {
		return errors.Wrap(err, "could not decode response body")
	}

	if funk.IsEmpty(body) {
		return errors.New("response body does not contain expected data")
	}

	return nil
}

func (c *client) bearerTokenHeader(authToken string) string {
	return "Bearer " + authToken
}

type requestBuilder struct {
	method  string
	url     string
	body    interface{}
	headers map[string]string
}

func newRequestBuilder(method, url string) *requestBuilder {
	return &requestBuilder{
		method:  method,
		url:     url,
		headers: make(map[string]string),
	}
}

func (r *requestBuilder) withBody(body interface{}) *requestBuilder {
	r.body = body

	return r
}

func (r *requestBuilder) addHeader(key, value string) *requestBuilder {
	r.headers[key] = value

	return r
}

func (r *requestBuilder) build() (*http.Request, error) {
	var body io.Reader

	if r.body != nil {
		b, err := json.Marshal(r.body)
		if err != nil {
			return nil, err
		}

		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(r.method, r.url, body) //nolint:noctx
	if err != nil {
		return nil, err
	}

	for key, value := range r.headers {
		req.Header.Add(key, value)
	}

	return req, nil
}
