package easee

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/thoas/go-funk"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

const (
	loginURI                  = "/api/accounts/token"
	tokenRefreshURI           = "/api/accounts/refresh_token"
	chargersURI               = "/api/chargers"
	healthURI                 = "/health"
	chargerControlURITemplate = "/api/chargers/%s/commands/%s"
	chargerStateURITemplate   = "/api/chargers/%s/state"
	cableLockURITemplate      = "/api/chargers/%s/commands/lock_state"

	startChargingCommand = "resume_charging"
	stopChargingCommand  = "pause_charging"

	authorizationHeader = "Authorization"
	contentTypeHeader   = "Content-Type"

	jsonContentType = "application/*+json"
)

type Client interface {
	Login(userName string, password string) (*LoginData, error)
	StartCharging(chargerID string) error
	StopCharging(chargerID string) error
	SetCableLock(chargerID string, locked bool) error
	ChargerState(chargerID string) (*ChargerState, error)
	Chargers() ([]Charger, error)
	Ping() error
}

type client struct {
	httpClient *http.Client
	cfgSvc     *config.Service
	baseURL    string
}

func NewClient(httpClient *http.Client, cfgSvc *config.Service, baseURL string) Client {
	return &client{
		httpClient: httpClient,
		cfgSvc:     cfgSvc,
		baseURL:    baseURL,
	}
}

func (c *client) Login(userName string, password string) (*LoginData, error) {
	body := loginBody{
		Username: userName,
		Password: password,
	}

	req, err := newRequestBuilder(http.MethodPost, c.url(loginURI)).
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

	loginData := &LoginData{}

	err = c.readResponseBody(resp, loginData)
	if err != nil {
		return nil, errors.Wrap(err, "could not read response body")
	}

	return loginData, nil
}

func (c *client) StartCharging(chargerID string) error {
	token, err := c.accessToken()
	if err != nil {
		return errors.Wrap(err, "failed to get access token")
	}

	uri := fmt.Sprintf(chargerControlURITemplate, chargerID, startChargingCommand)

	req, err := newRequestBuilder(http.MethodPost, c.url(uri)).
		addHeader(authorizationHeader, c.bearerTokenHeader(token)).
		build()
	if err != nil {
		return errors.Wrap(err, "failed to create start charging request")
	}

	resp, err := c.performRequest(req, http.StatusOK)
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

	uri := fmt.Sprintf(chargerControlURITemplate, chargerID, stopChargingCommand)

	req, err := newRequestBuilder(http.MethodPost, c.url(uri)).
		addHeader(authorizationHeader, c.bearerTokenHeader(token)).
		build()
	if err != nil {
		return errors.Wrap(err, "failed to create stop charging request")
	}

	resp, err := c.performRequest(req, http.StatusOK)
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

	uri := fmt.Sprintf(cableLockURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodGet, c.url(uri)).
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

func (c *client) ChargerState(chargerID string) (*ChargerState, error) {
	token, err := c.accessToken()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get access token")
	}

	uri := fmt.Sprintf(chargerStateURITemplate, chargerID)

	req, err := newRequestBuilder(http.MethodGet, c.url(uri)).
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

	state := &ChargerState{}

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

	req, err := newRequestBuilder(http.MethodGet, c.url(chargersURI)).
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

	req, err := newRequestBuilder(http.MethodGet, c.url(healthURI)).
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

	req, err := newRequestBuilder(http.MethodPost, c.url(tokenRefreshURI)).
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

	var loginData LoginData

	err = c.readResponseBody(resp, &loginData)
	if err != nil {
		return errors.Wrap(err, "could not read token refresh response body")
	}

	err = c.cfgSvc.SetCredentials(loginData.AccessToken, loginData.RefreshToken, int(loginData.ExpiresIn))
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

func (c *client) url(uri string) string {
	return c.baseURL + uri
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
