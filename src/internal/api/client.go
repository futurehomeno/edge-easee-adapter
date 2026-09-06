package api

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

// Client is a wrapper around the Easee HTTP Client with authentication capabilities.
type Client interface {
	UpdateMaxCurrent(chargerID string, current float64) error
	// UpdateDynamicCurrent updates dynamic charger current, dynamic current is used as offered current.
	UpdateDynamicCurrent(chargerID string, current float64) error
	StopCharging(chargerID string) error
	ChargerConfig(chargerID string) (*model.ChargerConfig, error)
	// ChargerSiteInfo retrieves charger rated current, rated current is used as supported max current.
	ChargerSiteInfo(chargerID string) (*model.ChargerSiteInfo, error)
	Chargers() ([]model.Charger, error)
	ChargerDetails(chargerID string) (model.ChargerDetails, error)
	SetCableAlwaysLocked(chargerID string, locked bool) error
	SetPhaseMode(chargerID string, phaseMode int) error
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
	log.Infof("[%s] Update max current to %.1f", chargerID, current)
	token, err := a.auth.AccessToken()
	if err != nil {
		return a.tokenError(err)
	}

	return a.httpClient.UpdateMaxCurrent(token, chargerID, current)
}

func (a *apiClient) SetCableAlwaysLocked(chargerID string, locked bool) error {
	log.Infof("[%s] Set cable always locked to %t", chargerID, locked)
	token, err := a.auth.AccessToken()
	if err != nil {
		return a.tokenError(err)
	}

	return a.httpClient.SetCableAlwaysLocked(token, chargerID, locked)
}

func (a *apiClient) SetPhaseMode(chargerID string, phaseMode int) error {
	log.Infof("[%s] Set phase mode to %d", chargerID, phaseMode)
	token, err := a.auth.AccessToken()
	if err != nil {
		return a.tokenError(err)
	}

	return a.httpClient.SetPhaseMode(token, chargerID, phaseMode)
}

func (a *apiClient) UpdateDynamicCurrent(chargerID string, current float64) error {
	log.Infof("[%s] Update dynamic current to %.1f", chargerID, current)
	token, err := a.auth.AccessToken()
	if err != nil {
		return a.tokenError(err)
	}

	return a.httpClient.UpdateDynamicCurrent(token, chargerID, current)
}

func (a *apiClient) StopCharging(chargerID string) error {
	log.Infof("[%s] Stop charging", chargerID)
	token, err := a.auth.AccessToken()
	if err != nil {
		return a.tokenError(err)
	}

	return a.httpClient.StopCharging(token, chargerID)
}

func (a *apiClient) ChargerSiteInfo(chargerID string) (*model.ChargerSiteInfo, error) {
	log.Debugf("[%s] Get charger site info", chargerID)
	token, err := a.auth.AccessToken()
	if err != nil {
		return nil, a.tokenError(err)
	}

	return a.httpClient.ChargerSiteInfo(token, chargerID)
}

func (a *apiClient) ChargerConfig(chargerID string) (*model.ChargerConfig, error) {
	log.Debugf("[%s] Get charger config", chargerID)
	token, err := a.auth.AccessToken()
	if err != nil {
		return nil, a.tokenError(err)
	}

	return a.httpClient.ChargerConfig(token, chargerID)
}

func (a *apiClient) Chargers() ([]model.Charger, error) {
	log.Infof("Get chargers")
	token, err := a.auth.AccessToken()
	if err != nil {
		return nil, a.tokenError(err)
	}

	return a.httpClient.Chargers(token)
}

func (a *apiClient) ChargerDetails(chargerID string) (model.ChargerDetails, error) {
	log.Debugf("[%s] Get charger details", chargerID)
	token, err := a.auth.AccessToken()
	if err != nil {
		return model.ChargerDetails{}, a.tokenError(err)
	}

	return a.httpClient.ChargerDetails(token, chargerID)
}

func (a *apiClient) Ping() error {
	log.Debugf("Ping")
	token, err := a.auth.AccessToken()
	if err != nil {
		return a.tokenError(err)
	}

	return a.httpClient.Ping(token)
}

func (a *apiClient) tokenError(err error) error {
	return fmt.Errorf("get token err: %w", err)
}
