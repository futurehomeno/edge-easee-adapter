package easee

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

// ControlType for easee control API
type ControlType string

func (c ControlType) String() string {
	return string(c)
}

const (
	// DefaultBaseURL is Easee api url
	DefaultBaseURL = "https://api.easee.cloud"
	// Start command to start charging
	Start ControlType = "start_charging"
	// Stop command to stop charging
	Stop ControlType = "stop_charging"
	// Pause command to pause charge session
	Pause ControlType = "pause_charging"
	// Resume command to start charge session
	Resume ControlType = "resume_charging"
)

// Login for username and password
type Login struct {
	Username string `json:"userName,omitempty"`
	Password string `json:"password,omitempty"`
}

type refreshBody struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// UserToken for Easee token
type UserToken struct {
	AccessToken  string   `json:"accessToken"`
	ExpiresIn    float64  `json:"expiresIn"`
	AccessClaims []string `json:"accessClaims"`
	TokenType    string   `json:"tokenType"`
	RefreshToken string   `json:"refreshToken"`
}

// Client for easee api calls
type Client struct {
	userToken *UserToken
	BaseURL   *url.URL

	httpClient   *http.Client
	httpResponse *http.Response
}

// Control for start/stop/pause/resume commands
type Control struct {
	Device    string `json:"device"`
	CommandID int    `json:"commandId"`
	Ticks     int64  `json:"ticks"`
}

// NewClient configures new http client with timeout
func NewClient(userToken *UserToken) (*Client, error) {
	url, err := url.Parse(DefaultBaseURL)
	if err != nil {
		log.Error(err)
	}
	return &Client{
		BaseURL:    url,
		userToken:  userToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// SetUserToken is used for setting config
func (c *Client) SetUserToken(userToken *UserToken) error {
	var err error
	if userToken != nil {
		c.userToken = userToken
		return err
	}
	err = fmt.Errorf("No token to set")
	return err
}

// GetUserToken returns UserToken
func (c *Client) GetUserToken() *UserToken {
	return c.userToken
}

// GetTokens gets access token with usernme and password
func (c *Client) GetTokens(login Login) (*UserToken, error) {
	body := login
	req, err := c.newRequest("POST", "api/accounts/token", body)
	if err != nil {
		return nil, err
	}
	var userToken *UserToken
	_, err = c.do(req, &userToken)
	return userToken, err
}

// RefreshTokens gets a new token with refreshtoken
func (c *Client) RefreshTokens() (*UserToken, error) {
	body := refreshBody{
		AccessToken:  c.userToken.AccessToken,
		RefreshToken: c.userToken.RefreshToken,
	}
	req, err := c.newRequest("POST", "api/accounts/refresh_token", body)
	if err != nil {
		return nil, err
	}
	var userToken *UserToken
	resp, err := c.do(req, &userToken)

	if resp.StatusCode != 200 {
		return nil, err
	}
	return userToken, err
}

// GetSites get a list of Sites
func (c *Client) GetSites() ([]Site, error) {
	req, err := c.newRequest("GET", "api/sites", nil)
	if err != nil {
		return nil, err
	}
	if c.userToken.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.userToken.AccessToken)
	} else {
		err = fmt.Errorf("no access token")
		return nil, err
	}
	var sites []Site
	resp, err := c.do(req, &sites)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, err
	}
	return sites, err

}

// GetChargers for user
func (c *Client) GetChargers() ([]Charger, error) {
	req, err := c.newRequest("GET", "api/chargers", nil)
	if err != nil {
		return nil, err
	}
	if c.userToken.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.userToken.AccessToken)
	} else {
		err = fmt.Errorf("no access token")
		return nil, err
	}
	var chargers []Charger
	resp, err := c.do(req, &chargers)
	// TODO: Check and handle status code
	log.Debug("Http status code: ", resp.StatusCode)
	return chargers, err
}

// GetChargerConfig gets configuration for one charger
func (c *Client) GetChargerConfig(chargerID string) (*ChargerConfig, error) {
	req, err := c.newRequest("GET", "api/chargers/"+chargerID+"/config", nil)
	if err != nil {
		return nil, err
	}
	if c.userToken.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.userToken.AccessToken)
	} else {
		err = fmt.Errorf("no access token")
		return nil, err
	}
	var chargerConfig *ChargerConfig
	resp, err := c.do(req, &chargerConfig)
	// TODO: Check and handle status code
	log.Debug("Http status code: ", resp.StatusCode)
	return chargerConfig, err
}

// GetChargerState gets the state of the charger
func (c *Client) GetChargerState(chargerID string) (*ChargerState, error) {
	req, err := c.newRequest("GET", "api/chargers/"+chargerID+"/state", nil)
	if err != nil {
		return nil, err
	}
	if c.userToken.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.userToken.AccessToken)
	} else {
		err = fmt.Errorf("no access token")
		return nil, err
	}
	var chargerState *ChargerState
	resp, err := c.do(req, &chargerState)
	// TODO: Check and handle status code
	log.Debug("Http status code: ", resp.StatusCode)
	return chargerState, err
}

// ControlCharger starts charging on charger
func (c *Client) ControlCharger(chargerID string, control ControlType) error {
	uri := "api/chargers/" + chargerID + "/commands/" + control.String()
	log.Debug(uri)
	req, err := c.newRequest("POST", uri, nil)
	if err != nil {
		return err
	}
	if c.userToken.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.userToken.AccessToken)
	} else {
		err = fmt.Errorf("no access token")
		return err
	}

	var controlState *Control
	_, err = c.do(req, &controlState)

	if err != nil {
		return err
	}
	return err
}

func (c *Client) newRequest(method, path string, body interface{}) (*http.Request, error) {
	rel := &url.URL{Path: path}
	u := c.BaseURL.ResolveReference(rel)
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	return req, nil
}
func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {

		return nil, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(v)

	return resp, err
}
