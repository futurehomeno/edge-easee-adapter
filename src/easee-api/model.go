package easee

import (
	"time"
)

// Product contains charger with state and config
type Product struct {
	Charger       *Charger       `json:"charger"`
	ChargerConfig *ChargerConfig `json:"chargerConfig"`
	ChargerState  *ChargerState  `json:"chargerState"`
}

// Site easee site structure
type Site struct {
	ID            int     `json:"id"`
	SiteKey       string  `json:"siteKey"`
	Name          string  `json:"name"`
	LevelOfAccess int     `json:"levelOfAccess"`
	Address       Address `json:"address"`
}

// Address for site structure
type Address struct {
	Street         string      `json:"street"`
	BuildingNumber string      `json:"buildingNumber"`
	Zip            string      `json:"zip"`
	Area           string      `json:"area"`
	Country        interface{} `json:"country"`
	Latitude       float64     `json:"latitude"`
	Longitude      float64     `json:"longitude"`
	Altitude       float64     `json:"altitude"`
}

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
	SmartCharging                                bool      `json:"smartCharging"`
	CableLocked                                  bool      `json:"cableLocked"`
	ChargerOpMode                                int       `json:"chargerOpMode"`
	TotalPower                                   float64   `json:"totalPower"`
	SessionEnergy                                float64   `json:"sessionEnergy"`
	EnergyPerHour                                float64   `json:"energyPerHour"`
	WiFiRSSI                                     int       `json:"wiFiRSSI"`
	CellRSSI                                     int       `json:"cellRSSI"`
	LocalRSSI                                    int       `json:"localRSSI"`
	OutputPhase                                  int       `json:"outputPhase"`
	DynamicCircuitCurrentP1                      float64   `json:"dynamicCircuitCurrentP1"`
	DynamicCircuitCurrentP2                      float64   `json:"dynamicCircuitCurrentP2"`
	DynamicCircuitCurrentP3                      float64   `json:"dynamicCircuitCurrentP3"`
	LatestPulse                                  time.Time `json:"latestPulse"`
	ChargerFirmware                              int       `json:"chargerFirmware"`
	LatestFirmware                               int       `json:"latestFirmware"`
	Voltage                                      float64   `json:"voltage"`
	ChargerRAT                                   int       `json:"chargerRAT"`
	LockCablePermanently                         bool      `json:"lockCablePermanently"`
	InCurrentT2                                  float64   `json:"inCurrentT2"`
	InCurrentT3                                  float64   `json:"inCurrentT3"`
	InCurrentT4                                  float64   `json:"inCurrentT4"`
	InCurrentT5                                  float64   `json:"inCurrentT5"`
	OutputCurrent                                float64   `json:"outputCurrent"`
	IsOnline                                     bool      `json:"isOnline"`
	InVoltageT1T2                                float64   `json:"inVoltageT1T2"`
	InVoltageT1T3                                float64   `json:"inVoltageT1T3"`
	InVoltageT1T4                                float64   `json:"inVoltageT1T4"`
	InVoltageT1T5                                float64   `json:"inVoltageT1T5"`
	InVoltageT2T3                                float64   `json:"inVoltageT2T3"`
	InVoltageT2T4                                float64   `json:"inVoltageT2T4"`
	InVoltageT2T5                                float64   `json:"inVoltageT2T5"`
	InVoltageT3T4                                float64   `json:"inVoltageT3T4"`
	InVoltageT3T5                                float64   `json:"inVoltageT3T5"`
	InVoltageT4T5                                float64   `json:"inVoltageT4T5"`
	LedMode                                      int       `json:"ledMode"`
	CableRating                                  float64   `json:"cableRating"`
	DynamicChargerCurrent                        float64   `json:"dynamicChargerCurrent"`
	CircuitTotalAllocatedPhaseConductorCurrentL1 float64   `json:"circuitTotalAllocatedPhaseConductorCurrentL1"`
	CircuitTotalAllocatedPhaseConductorCurrentL2 float64   `json:"circuitTotalAllocatedPhaseConductorCurrentL2"`
	CircuitTotalAllocatedPhaseConductorCurrentL3 float64   `json:"circuitTotalAllocatedPhaseConductorCurrentL3"`
	CircuitTotalPhaseConductorCurrentL1          float64   `json:"circuitTotalPhaseConductorCurrentL1"`
	CircuitTotalPhaseConductorCurrentL2          float64   `json:"circuitTotalPhaseConductorCurrentL2"`
	CircuitTotalPhaseConductorCurrentL3          float64   `json:"circuitTotalPhaseConductorCurrentL3"`
	ReasonForNoCurrent                           int       `json:"reasonForNoCurrent"`
	WiFiAPEnabled                                bool      `json:"wiFiAPEnabled"`
}

// ChargerConfig structure for charger config
type ChargerConfig struct {
	IsEnabled                    bool    `json:"isEnabled"`
	LockCablePermanently         bool    `json:"lockCablePermanently"`
	AuthorizationRequired        bool    `json:"authorizationRequired"`
	RemoteStartRequired          bool    `json:"remoteStartRequired"`
	SmartButtonEnabled           bool    `json:"smartButtonEnabled"`
	WiFiSSID                     string  `json:"wiFiSSID"`
	DetectedPowerGridType        int     `json:"detectedPowerGridType"`
	OfflineChargingMode          int     `json:"offlineChargingMode"`
	CircuitMaxCurrentP1          float64 `json:"circuitMaxCurrentP1"`
	CircuitMaxCurrentP2          float64 `json:"circuitMaxCurrentP2"`
	CircuitMaxCurrentP3          float64 `json:"circuitMaxCurrentP3"`
	EnableIdleCurrent            bool    `json:"enableIdleCurrent"`
	LimitToSinglePhaseCharging   bool    `json:"limitToSinglePhaseCharging"`
	PhaseMode                    int     `json:"phaseMode"`
	LocalNodeType                int     `json:"localNodeType"`
	LocalAuthorizationRequired   bool    `json:"localAuthorizationRequired"`
	LocalRadioChannel            int     `json:"localRadioChannel"`
	LocalShortAddress            int     `json:"localShortAddress"`
	LocalParentAddrOrNumOfNodes  int     `json:"localParentAddrOrNumOfNodes"`
	LocalPreAuthorizeEnabled     bool    `json:"localPreAuthorizeEnabled"`
	LocalAuthorizeOfflineEnabled bool    `json:"localAuthorizeOfflineEnabled"`
	AllowOfflineTxForUnknownID   bool    `json:"allowOfflineTxForUnknownId"`
	MaxChargerCurrent            float64 `json:"maxChargerCurrent"`
	LedStripBrightness           int     `json:"ledStripBrightness"`
}
