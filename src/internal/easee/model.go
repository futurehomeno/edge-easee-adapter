package easee

import (
	"time"
)

const (
	ServiceName = "easee"
)

// BackPlate structure
type BackPlate struct {
	ID                string `json:"id"`
	MasterBackPlateID string `json:"masterBackPlateId"`
}

// Charger structure
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

// ChargerState structure for state of chager
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

type ChargerMode int

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

const (
	ChargerModeUnavailable   = "unavailable"
	ChargerModeDisconnected  = "disconnected"
	ChargerModeReadyToCharge = "ready_to_charge"
	ChargerModeCharging      = "charging"
	ChargerModeFinished      = "finished"
	ChargerModeError         = "error"
	ChargerModeRequesting    = "requesting"
	ChargerModeUnknown       = "unknown"
)

type LoginData struct {
	AccessToken  string   `json:"accessToken"`
	ExpiresIn    int      `json:"expiresIn"`
	AccessClaims []string `json:"accessClaims"`
	TokenType    string   `json:"tokenType"`
	RefreshToken string   `json:"refreshToken"`
}

type loginBody struct {
	Username string `json:"userName"`
	Password string `json:"password"`
}

type refreshBody struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type cableLockBody struct {
	State bool `json:"state"`
}
