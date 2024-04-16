package api

import (
	"time"

	"github.com/futurehomeno/edge-easee-adapter/internal/maper"
)

// Credentials stands for Easee API credentials.
type Credentials struct {
	AccessToken  string   `json:"accessToken"`
	ExpiresIn    int      `json:"expiresIn"`
	AccessClaims []string `json:"accessClaims"`
	TokenType    string   `json:"tokenType"`
	RefreshToken string   `json:"refreshToken"`
}

// Charger represents charger data.
type Charger struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Color         int       `json:"color"`
	CreatedOn     string    `json:"createdOn"`
	UpdatedOn     string    `json:"updatedOn"`
	BackPlate     BackPlate `json:"backPlate"`
	LevelOfAccess int       `json:"levelOfAccess"`
	ProductCode   int       `json:"productCode"`
}

// ChargerDetails represents charger's details.
type ChargerDetails struct {
	Product string `json:"product"`
}

// BackPlate represents charger's back plate.
type BackPlate struct {
	ID                string `json:"id"`
	MasterBackPlateID string `json:"masterBackPlateId"`
}

// ChargerConfig represents charger config.
type ChargerConfig struct {
	DetectedPowerGridType maper.GridType `json:"detectedPowerGridType"`
	PhaseMode             int            `json:"phaseMode"`
}

// ChargerSiteInfo represents charger rate current.
type ChargerSiteInfo struct {
	RatedCurrent float64 `json:"ratedCurrent"`
}

// ChargeSessions represents charge sessions.
type ChargeSessions []*ChargeSession

// Latest returns latest charge session.
func (c ChargeSessions) Latest() *ChargeSession {
	if len(c) < 1 {
		return nil
	}

	return c[0]
}

// Previous returns previous charge session.
func (c ChargeSessions) Previous() *ChargeSession {
	if len(c) < 2 {
		return nil
	}

	return c[1]
}

// ChargeSession represents charger session.
type ChargeSession struct {
	CarConnected    time.Time `json:"carConnected"`
	CarDisconnected time.Time `json:"carDisconnected"`
	KiloWattHours   float64   `json:"kiloWattHours"`
	IsComplete      bool      `json:"isComplete"`
}

// loginBody represents a login request body.
type loginBody struct {
	Username string `json:"userName"`
	Password string `json:"password"`
}

// refreshBody represents a token refresh request body.
type refreshBody struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// The following struct is currently commented out because it is not needed for the current functionality.
// However, it may be useful in implementing SetCableAlwaysLock method.
// cableLockBody represents a cable lock request body.
// type cableLockBody struct {
// 	State bool `json:"state"`
// }

// maxCurrentBody represents a charger max current request body.
type maxCurrentBody struct {
	MaxChargerCurrent float64 `json:"maxChargerCurrent"`
}

// dynamicCurrentBody represents a charger dynamic current request body.
type dynamicCurrentBody struct {
	DynamicChargerCurrent float64 `json:"dynamicChargerCurrent"`
}
