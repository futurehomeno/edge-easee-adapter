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

// ChargerState represents detailed state data about the charger.
type ChargerState struct {
	SmartCharging                                bool        `json:"smartCharging"`
	CableLocked                                  bool        `json:"cableLocked"`
	ChargerOpMode                                ChargerMode `json:"chargerOpMode"`
	TotalPower                                   float64     `json:"totalPower"`
	SessionEnergy                                float64     `json:"sessionEnergy"`
	EnergyPerHour                                float64     `json:"energyPerHour"`
	WiFiRSSI                                     int         `json:"wiFiRSSI"`
	CellRSSI                                     int         `json:"cellRSSI"`
	LocalRSSI                                    int         `json:"localRSSI"`
	OutputPhase                                  int         `json:"outputPhase"`
	DynamicCircuitCurrentP1                      float64     `json:"dynamicCircuitCurrentP1"`
	DynamicCircuitCurrentP2                      float64     `json:"dynamicCircuitCurrentP2"`
	DynamicCircuitCurrentP3                      float64     `json:"dynamicCircuitCurrentP3"`
	LatestPulse                                  time.Time   `json:"latestPulse"`
	ChargerFirmware                              int         `json:"chargerFirmware"`
	LatestFirmware                               int         `json:"latestFirmware"`
	Voltage                                      float64     `json:"voltage"`
	ChargerRAT                                   int         `json:"chargerRAT"`
	LockCablePermanently                         bool        `json:"lockCablePermanently"`
	InCurrentT2                                  float64     `json:"inCurrentT2"`
	InCurrentT3                                  float64     `json:"inCurrentT3"`
	InCurrentT4                                  float64     `json:"inCurrentT4"`
	InCurrentT5                                  float64     `json:"inCurrentT5"`
	OutputCurrent                                float64     `json:"outputCurrent"`
	IsOnline                                     bool        `json:"isOnline"`
	InVoltageT1T2                                float64     `json:"inVoltageT1T2"`
	InVoltageT1T3                                float64     `json:"inVoltageT1T3"`
	InVoltageT1T4                                float64     `json:"inVoltageT1T4"`
	InVoltageT1T5                                float64     `json:"inVoltageT1T5"`
	InVoltageT2T3                                float64     `json:"inVoltageT2T3"`
	InVoltageT2T4                                float64     `json:"inVoltageT2T4"`
	InVoltageT2T5                                float64     `json:"inVoltageT2T5"`
	InVoltageT3T4                                float64     `json:"inVoltageT3T4"`
	InVoltageT3T5                                float64     `json:"inVoltageT3T5"`
	InVoltageT4T5                                float64     `json:"inVoltageT4T5"`
	LedMode                                      int         `json:"ledMode"`
	CableRating                                  float64     `json:"cableRating"`
	DynamicChargerCurrent                        float64     `json:"dynamicChargerCurrent"`
	CircuitTotalAllocatedPhaseConductorCurrentL1 float64     `json:"circuitTotalAllocatedPhaseConductorCurrentL1"`
	CircuitTotalAllocatedPhaseConductorCurrentL2 float64     `json:"circuitTotalAllocatedPhaseConductorCurrentL2"`
	CircuitTotalAllocatedPhaseConductorCurrentL3 float64     `json:"circuitTotalAllocatedPhaseConductorCurrentL3"`
	CircuitTotalPhaseConductorCurrentL1          float64     `json:"circuitTotalPhaseConductorCurrentL1"`
	CircuitTotalPhaseConductorCurrentL2          float64     `json:"circuitTotalPhaseConductorCurrentL2"`
	CircuitTotalPhaseConductorCurrentL3          float64     `json:"circuitTotalPhaseConductorCurrentL3"`
	ReasonForNoCurrent                           int         `json:"reasonForNoCurrent"`
	WiFiAPEnabled                                bool        `json:"wiFiAPEnabled"`
	LifetimeEnergy                               float64     `json:"lifetimeEnergy"`
}

const (
	// ChargerModeUnavailable represents an "unavailable" state of the charger.
	ChargerModeUnavailable = "unavailable"
	// ChargerModeDisconnected represents a "disconnected" state of the charger.
	ChargerModeDisconnected = "disconnected"
	// ChargerModeReadyToCharge represents a "ready to charge" state of the charger.
	ChargerModeReadyToCharge = "ready_to_charge"
	// ChargerModeCharging represents a "charging" state of the charger.
	ChargerModeCharging = "charging"
	// ChargerModeFinished represents a "finished" state of the charger.
	ChargerModeFinished = "finished"
	// ChargerModeError represents an "error" state of the charger.
	ChargerModeError = "error"
	// ChargerModeRequesting represents a "requesting" state of the charger.
	ChargerModeRequesting = "requesting"
	// ChargerModeUnknown represents an "unknown" state of the charger.
	ChargerModeUnknown = "unknown"
)

// ChargerMode represents a charger mode.
type ChargerMode int

// String returns a string representation ChargerMode.
func (m ChargerMode) String() string {
	switch m {
	case 0:
		return ChargerModeUnavailable
	case 1:
		return ChargerModeDisconnected
	case 2:
		return ChargerModeReadyToCharge
	case 3:
		return ChargerModeCharging
	case 4:
		return ChargerModeFinished
	case 5:
		return ChargerModeError
	case 6:
		return ChargerModeRequesting
	default:
		return ChargerModeUnknown
	}
}

// SupportedChargingStates returns all charging states supported by Easee.
func SupportedChargingStates() []string {
	return []string{
		ChargerModeUnavailable,
		ChargerModeDisconnected,
		ChargerModeReadyToCharge,
		ChargerModeCharging,
		ChargerModeFinished,
		ChargerModeError,
		ChargerModeRequesting,
		ChargerModeUnknown,
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

const (
	// resultCodeSent represents a command result code.
	// The command was sent, but not yet processed by Easee cloud.
	resultCodeSent = iota
	// resultCodeExpired represents a command result code.
	// The command was sent, but was not processed on time.
	resultCodeExpired
	// resultCodeExecuted represents a command result code.
	// The command was sent and successfully executed by Easee cloud.
	resultCodeExecuted
	// resultCodeRejected represents a command result code.
	// The command was sent, but was rejected by Easee cloud.
	resultCodeRejected
)

// commandResponse represents a response returned by all command API calls.
type commandResponse struct {
	Device    string `json:"device"`
	CommandID int    `json:"commandId"`
	Ticks     int    `json:"ticks"`
}

// checkerResponse represents a response of command checker API endpoint.
type checkerResponse struct {
	ResultCode int `json:"resultCode"`
}
