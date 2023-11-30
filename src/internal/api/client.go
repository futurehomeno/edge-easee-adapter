package api

import "fmt"

// APIClient is a wrapper around the Easee HTTP Client with authentication capabilities.
type APIClient interface {
	// UpdateMaxCurrent updates max charger current.
	UpdateMaxCurrent(chargerID string, current float64) error
	// UpdateDynamicCurrent updates dynamic charger current, dynamic current is used as offered current.
	UpdateDynamicCurrent(chargerID string, current float64) error
	// StartCharging starts charging session for the selected charger.
	StartCharging(chargerID string) error
	// StopCharging stops charging session for the selected charger.
	StopCharging(chargerID string) error
	// SetCableLock locks/unlocks the cable for the selected charger.
	SetCableLock(chargerID string, locked bool) error
	// ChargerConfig retrieves charger config.
	ChargerConfig(chargerID string) (*ChargerConfig, error)
	// ChargerSiteInfo retrieves charger rated current, rated current is used as supported max current.
	ChargerSiteInfo(chargerID string) (*ChargerSiteInfo, error)
	// Chargers returns all available chargers.
	Chargers() ([]Charger, error)
	// Ping checks if an external service is available.
	Ping() error
}

type apiClient struct {
	httpClient HTTPClient
	auth       Authenticator
}

func NewAPIClient(http HTTPClient, auth Authenticator) APIClient {
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

func (a *apiClient) StartCharging(chargerID string) error {
	token, err := a.auth.AccessToken()
	if err != nil {
		return a.tokenError(err)
	}

	return a.httpClient.StartCharging(token, chargerID)
}

func (a *apiClient) StopCharging(chargerID string) error {
	token, err := a.auth.AccessToken()
	if err != nil {
		return a.tokenError(err)
	}

	return a.httpClient.StopCharging(token, chargerID)
}

func (a *apiClient) SetCableLock(chargerID string, locked bool) error {
	token, err := a.auth.AccessToken()
	if err != nil {
		return a.tokenError(err)
	}

	return a.httpClient.SetCableLock(token, chargerID, locked)
}

func (a *apiClient) ChargerSiteInfo(chargerID string) (*ChargerSiteInfo, error) {
	token, err := a.auth.AccessToken()
	if err != nil {
		return nil, a.tokenError(err)
	}

	return a.httpClient.ChargerSiteInfo(token, chargerID)
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
