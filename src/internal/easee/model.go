package easee

import "github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"

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

// ChargerState represents a charger state.
type ChargerState int

const (
	Unknown ChargerState = iota - 1
	Unavailable
	Disconnected
	ReadyToCharge
	Charging
	Finished
	Error
	Requesting
)

// ToFimpState returns a human-readable name of the state.
func (s ChargerState) ToFimpState() chargepoint.State {
	switch s {
	case Unknown:
		return chargepoint.StateUnknown
	case Unavailable:
		return chargepoint.StateUnavailable
	case Disconnected:
		return chargepoint.StateDisconnected
	case ReadyToCharge:
		return chargepoint.StateReadyToCharge
	case Charging:
		return chargepoint.StateCharging
	case Finished:
		return chargepoint.StateFinished
	case Error:
		return chargepoint.StateError
	case Requesting:
		return chargepoint.StateRequesting
	default:
		return chargepoint.StateUnknown
	}
}

// SupportedChargingStates returns all charging states supported by Easee.
func SupportedChargingStates() []ChargerState {
	return []ChargerState{
		Unavailable,
		Disconnected,
		ReadyToCharge,
		Charging,
		Finished,
		Error,
		Requesting,
		Unknown,
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
