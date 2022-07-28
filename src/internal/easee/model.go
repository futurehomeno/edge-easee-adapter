package easee

import (
	"time"
)

const (
	// ServiceName represents Easee service name.
	ServiceName = "easee"
)

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

// BackPlate represents charger's back plate.
type BackPlate struct {
	ID                string `json:"id"`
	MasterBackPlateID string `json:"masterBackPlateId"`
}

// ChargerConfig represents charger config.
type ChargerConfig struct {
	MaxChargerCurrent float64 `json:"maxChargerCurrent"`
}

const (
	// ChargerStateUnavailable represents an "unavailable" state of the charger.
	ChargerStateUnavailable = "unavailable"
	// ChargerStateDisconnected represents a "disconnected" state of the charger.
	ChargerStateDisconnected = "disconnected"
	// ChargerStateReadyToCharge represents a "ready to charge" state of the charger.
	ChargerStateReadyToCharge = "ready_to_charge"
	// ChargerStateCharging represents a "charging" state of the charger.
	ChargerStateCharging = "charging"
	// ChargerStateFinished represents a "finished" state of the charger.
	ChargerStateFinished = "finished"
	// ChargerStateError represents an "error" state of the charger.
	ChargerStateError = "error"
	// ChargerStateRequesting represents a "requesting" state of the charger.
	ChargerStateRequesting = "requesting"
	// ChargerStateUnknown represents an "unknown" state of the charger.
	ChargerStateUnknown = "unknown"
)

// ChargerState represents a charger state.
type ChargerState int

// String returns a string representation of ChargerState.
func (m ChargerState) String() string {
	switch m {
	case 0:
		return ChargerStateUnavailable
	case 1:
		return ChargerStateDisconnected
	case 2:
		return ChargerStateReadyToCharge
	case 3:
		return ChargerStateCharging
	case 4:
		return ChargerStateFinished
	case 5:
		return ChargerStateError
	case 6:
		return ChargerStateRequesting
	default:
		return ChargerStateUnknown
	}
}

// SupportedChargingStates returns all charging states supported by Easee.
func SupportedChargingStates() []string {
	return []string{
		ChargerStateUnavailable,
		ChargerStateDisconnected,
		ChargerStateReadyToCharge,
		ChargerStateCharging,
		ChargerStateFinished,
		ChargerStateError,
		ChargerStateRequesting,
		ChargerStateUnknown,
	}
}

const (
	// ChargingModeNormal represents a "normal" charging mode.
	ChargingModeNormal = "normal"
	// ChargingModeSlow represents a "slow" charging mode.
	ChargingModeSlow = "slow"
)

// SupportedChargingModes returns all charging modes supported by Easee.
func SupportedChargingModes() []string {
	return []string{
		ChargingModeNormal,
		ChargingModeSlow,
	}
}

// ObservationID represents an observation ID in Easee API.
type ObservationID int

const (
	// ChargerOPState represents a "charger state" observation.
	ChargerOPState ObservationID = 109
	// SessionEnergy represents a "session energy" observation.
	SessionEnergy ObservationID = 121
	// CableLocked represents a "cable locked" observation.
	CableLocked ObservationID = 103
	// TotalPower represents a "total power" observation.
	TotalPower ObservationID = 120
	// LifetimeEnergy represents a "lifetime energy" observation.
	LifetimeEnergy ObservationID = 124
)

// Observation represents a single observation in Easee API.
type Observation struct {
	Value     interface{} `json:"value"`
	Timestamp time.Time   `json:"timestamp"`
}

// Credentials stands for Easee API credentials.
type Credentials struct {
	AccessToken  string   `json:"accessToken"`
	ExpiresIn    int      `json:"expiresIn"`
	AccessClaims []string `json:"accessClaims"`
	TokenType    string   `json:"tokenType"`
	RefreshToken string   `json:"refreshToken"`
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

// cableLockBody represents a cable lock request body.
type cableLockBody struct {
	State bool `json:"state"`
}

// chargerCurrentBody represents a charger current request body.
type chargerCurrentBody struct {
	DynamicChargerCurrent float64 `json:"dynamicChargerCurrent"`
}
