package api

import (
	"fmt"

	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

// Client is a wrapper around the Easee HTTP Client with authentication capabilities.
type Client interface {
	// UpdateMaxCurrent updates max charger current.
	UpdateMaxCurrent(chargerID string, current float64) error
	// UpdateDynamicCurrent updates dynamic charger current, dynamic current is used as offered current.
	UpdateDynamicCurrent(chargerID string, current float64) error
	// StopCharging stops charging session for the selected charger.
	StopCharging(chargerID string) error
	// ChargerConfig retrieves charger config.
	ChargerConfig(chargerID string) (*model.ChargerConfig, error)
	// ChargerSiteInfo retrieves charger rated current, rated current is used as supported max current.
	ChargerSiteInfo(chargerID string) (*model.ChargerSiteInfo, error)
	// ChargerSessions retrieves at most two latest charging sessions including current if present.
	ChargerSessions(chargerID string) (model.ChargeSessions, error)
	// Chargers returns all available chargers.
	Chargers() ([]model.Charger, error)
	ChargerDetails(chargerID string) (model.ChargerDetails, error)
	SetCableAlwaysLocked(chargerID string, locked bool) error
	// Ping checks if an external service is available.
	Ping() error
}

type apiClient struct {
	httpClient HTTPClient
	auth       Authenticator
}

func NewAPIClient(http HTTPClient, auth Authenticator) Client {
	return &apiClient{
		httpClient: http,
		auth:       auth,
	}
}

func (a *apiClient) UpdateMaxCurrent(chargerID string, current float64) error {
	token, err := a.auth.AccessToken()
	if err != nil {
		return a.tokenError(err)
	}

	return a.httpClient.UpdateMaxCurrent(token, chargerID, current)
}

func (a *apiClient) SetCableAlwaysLocked(chargerID string, locked bool) error {
	token, err := a.auth.AccessToken()
	if err != nil {
		return a.tokenError(err)
	}

	return a.httpClient.SetCableAlwaysLocked(token, chargerID, locked)
}

func (a *apiClient) UpdateDynamicCurrent(chargerID string, current float64) error {
	token, err := a.auth.AccessToken()
	if err != nil {
		return a.tokenError(err)
	}

	return a.httpClient.UpdateDynamicCurrent(token, chargerID, current)
}

func (a *apiClient) StopCharging(chargerID string) error {
	token, err := a.auth.AccessToken()
	if err != nil {
		return a.tokenError(err)
	}

	return a.httpClient.StopCharging(token, chargerID)
}

func (a *apiClient) ChargerSiteInfo(chargerID string) (*model.ChargerSiteInfo, error) {
	token, err := a.auth.AccessToken()
	if err != nil {
		return nil, a.tokenError(err)
	}

	return a.httpClient.ChargerSiteInfo(token, chargerID)
}

func (a *apiClient) ChargerSessions(chargerID string) (model.ChargeSessions, error) {
	token, err := a.auth.AccessToken()
	if err != nil {
		return nil, a.tokenError(err)
	}

	return a.httpClient.ChargerSessions(token, chargerID)
}

func (a *apiClient) ChargerConfig(chargerID string) (*model.ChargerConfig, error) {
	token, err := a.auth.AccessToken()
	if err != nil {
		return nil, a.tokenError(err)
	}

	return a.httpClient.ChargerConfig(token, chargerID)
}

func (a *apiClient) Chargers() ([]model.Charger, error) {
	token, err := a.auth.AccessToken()
	if err != nil {
		return nil, a.tokenError(err)
	}

	return a.httpClient.Chargers(token)
}

func (a *apiClient) ChargerDetails(chargerID string) (model.ChargerDetails, error) {
	token, err := a.auth.AccessToken()
	if err != nil {
		return model.ChargerDetails{}, a.tokenError(err)
	}

	return a.httpClient.ChargerDetails(token, chargerID)
}

func (a *apiClient) Ping() error {
	token, err := a.auth.AccessToken()
	if err != nil {
		return a.tokenError(err)
	}

	return a.httpClient.Ping(token)
}

func (a *apiClient) tokenError(err error) error {
	return fmt.Errorf("unable to get access token: %w", err)
}
