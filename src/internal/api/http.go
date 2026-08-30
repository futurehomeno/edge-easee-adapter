package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/auth"
	"github.com/futurehomeno/cliffhanger/httpclient"
	"github.com/michalkurzeja/go-clock"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

var (
	// ErrNotLoggedIn is returned when no credentials are stored locally so callers can downgrade their logging.
	ErrNotLoggedIn = auth.ErrNotLoggedIn
	// ErrRefreshBackoff is returned while the authenticator is in backoff after refresh-token failures, to avoid hammering the API.
	ErrRefreshBackoff = errors.New("too many requests: backoff")
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
	phaseModeURITemplate       = "/api/chargers/%s/commands/set_phase_mode"
	chargerSessionsURITemplate = "/api/sessions/charger/%s/sessions/descending?limit=2"
	chargerDetailsURITemplate  = "/api/chargers/%s/details?alwaysGetChargerAccessLevel=false"

	authorizationHeader = "Authorization"
	contentTypeHeader   = "Content-Type"

	jsonContentType = "application/*+json"
)

type HTTPClient interface {
	UpdateMaxCurrent(accessToken, chargerID string, current float64) error
	// UpdateDynamicCurrent sets the dynamic current, which Easee uses as the offered current.
	UpdateDynamicCurrent(accessToken, chargerID string, current float64) error
	Login(userName, password string) (*model.Credentials, error)
	// RefreshToken exchanges an access/refresh token pair for a new one.
	RefreshToken(accessToken, refreshToken string) (*model.Credentials, error)
	StopCharging(accessToken, chargerID string) error
	ChargerConfig(accessToken, chargerID string) (*model.ChargerConfig, error)
	// ChargerSiteInfo retrieves the rated current, used as the supported max current.
	ChargerSiteInfo(accessToken, chargerID string) (*model.ChargerSiteInfo, error)
	Chargers(accessToken string) ([]model.Charger, error)
	// ChargerDetails carries the product name.
	ChargerDetails(accessToken string, chargerID string) (model.ChargerDetails, error)
	SetCableAlwaysLocked(accessToken string, chargerID string, locked bool) error
	SetPhaseMode(accessToken string, chargerID string, phaseMode int) error
	Ping(accessToken string) error
}

type httpClient struct {
	httpClient *http.Client
	baseURL    string
	cfgSrv     *config.Service

	lock              sync.RWMutex
	lastMaxCurrentSet map[string]time.Time
}

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

	req, err := httpclient.NewJSONRequest(context.Background(), http.MethodPost, c.baseURL+loginURI, body,
		map[string]string{contentTypeHeader: jsonContentType})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create login request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("transport error: %w", err)
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

	req, err := httpclient.NewJSONRequest(context.Background(), http.MethodPost, c.baseURL+tokenRefreshURI, body,
		map[string]string{contentTypeHeader: jsonContentType})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create token refresh request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("transport error: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		c.logFailedResponse(resp)

		return nil, c.handleFailedResponse(resp, "token refresh request failed")
	}

	loginData := &model.Credentials{}

	if err = c.readResponseBody(resp, loginData); err != nil {
		return nil, errors.Wrap(err, "could not read token refresh response body")
	}

	return loginData, nil
}

func (c *httpClient) UpdateMaxCurrent(accessToken, chargerID string, current float64) error {
	u := c.buildURL(chargerSettingsURITemplate, chargerID)

	req, err := httpclient.NewJSONRequest(context.Background(), http.MethodPost, u, maxCurrentBody{MaxChargerCurrent: current},
		map[string]string{authorizationHeader: c.bearerTokenHeader(accessToken), contentTypeHeader: jsonContentType})
	if err != nil {
		return errors.Wrap(err, "failed to create max current request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("transport error: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

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

	req, err := httpclient.NewJSONRequest(context.Background(), http.MethodPost, u, dynamicCurrentBody{DynamicChargerCurrent: current},
		map[string]string{authorizationHeader: c.bearerTokenHeader(accessToken), contentTypeHeader: jsonContentType})
	if err != nil {
		return errors.Wrap(err, "failed to create dynamic current request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("transport error: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		c.logFailedResponse(resp)

		return c.handleFailedResponse(resp, "update dynamic current request failed: unexpected status code")
	}

	c.registerMaxCurrentChange(chargerID)

	return nil
}

func (c *httpClient) StopCharging(accessToken, chargerID string) error {
	// Easee zeroes the dynamic current on stop, so offered-current changes are rate-limited
	// against it (OfferedCurrentWaitTime).
	if c.shouldBackoffWithMaxCurrentChange(chargerID) {
		return errors.New("client: failed to stop charging: too many requests to the charger")
	}

	u := c.buildURL(chargerStopURITemplate, chargerID)

	req, err := httpclient.NewJSONRequest(context.Background(), http.MethodPost, u, nil,
		map[string]string{authorizationHeader: c.bearerTokenHeader(accessToken)})
	if err != nil {
		return errors.Wrap(err, "failed to create stop charging request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("transport error: %w", err)
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

	req, err := httpclient.NewJSONRequest(context.Background(), http.MethodPost, u, cableLockStateBody{State: locked},
		map[string]string{authorizationHeader: c.bearerTokenHeader(accessToken), contentTypeHeader: jsonContentType})
	if err != nil {
		return errors.Wrap(err, "failed to create cable lock request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("transport error: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		c.logFailedResponse(resp)

		return c.handleFailedResponse(resp, "cable lock request failed: unexpected status code")
	}

	return nil
}

func (c *httpClient) SetPhaseMode(accessToken, chargerID string, phaseMode int) error {
	u := c.buildURL(phaseModeURITemplate, chargerID)

	req, err := httpclient.NewJSONRequest(context.Background(), http.MethodPost, u, phaseModeBody{PhaseMode: phaseMode},
		map[string]string{authorizationHeader: c.bearerTokenHeader(accessToken), contentTypeHeader: jsonContentType})
	if err != nil {
		return errors.Wrap(err, "failed to create phase mode request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("transport error: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	// Unlike the other command endpoints this one is documented to answer 200, but Easee
	// has been observed answering 202 as well.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		c.logFailedResponse(resp)

		return c.handleFailedResponse(resp, "phase mode request failed: unexpected status code")
	}

	return nil
}

func (c *httpClient) ChargerConfig(accessToken, chargerID string) (*model.ChargerConfig, error) {
	var chargerConfig model.ChargerConfig
	if err := c.getResponse(&chargerConfig, c.buildURL(chargerConfigURITemplate, chargerID), accessToken); err != nil {
		return nil, err
	}

	return &chargerConfig, nil
}

func (c *httpClient) ChargerSiteInfo(accessToken, chargerID string) (*model.ChargerSiteInfo, error) {
	var siteInfo model.ChargerSiteInfo
	if err := c.getResponse(&siteInfo, c.buildURL(chargerSiteURITemplate, chargerID), accessToken); err != nil {
		return nil, err
	}

	return &siteInfo, nil
}

func (c *httpClient) Chargers(accessToken string) ([]model.Charger, error) {
	var chargers []model.Charger
	if err := c.getResponse(&chargers, c.baseURL+chargersURI, accessToken); err != nil {
		return nil, err
	}

	return chargers, nil
}

func (c *httpClient) ChargerDetails(accessToken string, chargerID string) (model.ChargerDetails, error) {
	var details model.ChargerDetails
	if err := c.getResponse(&details, c.buildURL(chargerDetailsURITemplate, chargerID), accessToken); err != nil {
		return model.ChargerDetails{}, err
	}

	return details, nil
}

func (c *httpClient) Ping(accessToken string) error {
	req, err := httpclient.NewJSONRequest(context.Background(), http.MethodGet, c.baseURL+healthURI, nil,
		map[string]string{authorizationHeader: c.bearerTokenHeader(accessToken)})
	if err != nil {
		return errors.Wrap(err, "failed to create ping request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("transport error: %w", err)
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
	// ErrorFromResponse is nil below 300, and the command endpoints treat anything but 202 as a
	// failure - Easee answers some of them 200. Wrapping nil renders as %!w(<nil>).
	if err := httpclient.ErrorFromResponse(resp); err != nil {
		return fmt.Errorf("%s, status code: %d: %w", message, resp.StatusCode, err)
	}

	return fmt.Errorf("%s, status code: %d", message, resp.StatusCode)
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

func (c *httpClient) shouldBackoffWithMaxCurrentChange(chargerID string) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	lastMaxCurrentSet, ok := c.lastMaxCurrentSet[chargerID]
	if !ok {
		return false
	}

	if clock.Now().Sub(lastMaxCurrentSet) >= c.cfgSrv.OfferedCurrentWaitTime() {
		return false
	}

	return true
}

func (c *httpClient) registerMaxCurrentChange(chargerID string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.lastMaxCurrentSet[chargerID] = clock.Now()
}

func (c *httpClient) getResponse(state any, url, accessToken string) error {
	req, err := httpclient.NewJSONRequest(context.Background(), http.MethodGet, url, nil,
		map[string]string{authorizationHeader: c.bearerTokenHeader(accessToken)})
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("transport error: %w", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.WithError(err).Error("failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		c.logFailedResponse(resp)
		return c.handleFailedResponse(resp, "unexpected status code")
	}

	if err = c.readResponseBody(resp, state); err != nil {
		return errors.Wrap(err, "could not read response body")
	}

	return nil
}
