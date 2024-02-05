package api

import (
	"fmt"
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
	ChargerConfig(chargerID string) (*ChargerConfig, error)
	// ChargerSiteInfo retrieves charger rated current, rated current is used as supported max current.
	ChargerSiteInfo(chargerID string) (*ChargerSiteInfo, error)
	// ChargerSessions retrieves at most two latest charging sessions including current if present.
	ChargerSessions(chargerID string) (ChargeSessions, error)
	// Chargers returns all available chargers.
	Chargers() ([]Charger, error)
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

func (a *apiClient) ChargerSiteInfo(chargerID string) (*ChargerSiteInfo, error) {
	token, err := a.auth.AccessToken()
	if err != nil {
		return nil, a.tokenError(err)
	}

	return a.httpClient.ChargerSiteInfo(token, chargerID)
}

func (a *apiClient) ChargerSessions(chargerID string) (ChargeSessions, error) {
	token, err := a.auth.AccessToken()
	if err != nil {
		return nil, a.tokenError(err)
	}

	return a.httpClient.ChargerSessions(token, chargerID)
}

func (a *apiClient) ChargerConfig(chargerID string) (*ChargerConfig, error) {
	token, err := a.auth.AccessToken()
	if err != nil {
		return nil, a.tokenError(err)
	}

	return a.httpClient.ChargerConfig(token, chargerID)
}

func (a *apiClient) Chargers() ([]Charger, error) {
	token, err := a.auth.AccessToken()
	if err != nil {
		return nil, a.tokenError(err)
	}

	return a.httpClient.Chargers(token)
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
