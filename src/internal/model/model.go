package model

import (
	"errors"
	"strconv"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	log "github.com/sirupsen/logrus"
)

const (
	CableAlwaysLockedParameter = "cable_always_locked"
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
	DetectedPowerGridType GridType `json:"detectedPowerGridType"`
	PhaseMode             int      `json:"phaseMode"`
}

// ChargerSiteInfo represents charger rate current.
type ChargerSiteInfo struct {
	RatedCurrent float64 `json:"ratedCurrent"`
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

// Observation represents a SignalR observation data.
type Observation struct {
	ID        ObservationID       `json:"id"`
	ChargerID string              `json:"mid"`
	DataType  ObservationDataType `json:"dataType"`
	Timestamp time.Time           `json:"timestamp"`
	Value     string              `json:"value"`
}

// IntValue returns an integer representation of the Observation value.
func (o *Observation) IntValue() (int, error) {
	if o.DataType != ObservationDataTypeInteger {
		return 0, errors.New("observation data type is not int")
	}

	return strconv.Atoi(o.Value)
}

// Float64Value returns a float64 representation of the Observation value.
func (o *Observation) Float64Value() (float64, error) {
	if o.DataType != ObservationDataTypeDouble {
		return 0, errors.New("observation data type is not float64")
	}

	return strconv.ParseFloat(o.Value, 64)
}

// BoolValue returns a bool representation of the Observation value.
func (o *Observation) BoolValue() (bool, error) {
	if o.DataType != ObservationDataTypeBoolean {
		return false, errors.New("observation data type is not bool")
	}

	return strconv.ParseBool(o.Value)
}

// StringValue returns a string representation of the Observation value.
func (o *Observation) StringValue() (string, error) {
	if o.DataType != ObservationDataTypeString {
		return "", errors.New("observation data type is not string")
	}

	return o.Value, nil
}

// ObservationID represents an Observation ID in Easee API.
type ObservationID int

const (
	DetectedPowerGridType ObservationID = 21
	LockCablePermanently  ObservationID = 30
	PhaseMode             ObservationID = 38
	MaxChargerCurrent     ObservationID = 47
	DynamicChargerCurrent ObservationID = 48
	CableLocked           ObservationID = 103
	CableRating           ObservationID = 104
	ChargerOPState        ObservationID = 109
	OutputPhase           ObservationID = 110
	TotalPower            ObservationID = 120
	EnergySession         ObservationID = 121
	LifetimeEnergy        ObservationID = 124
	ChargingSessionStop   ObservationID = 129
	InCurrentT3           ObservationID = 183
	InCurrentT4           ObservationID = 184
	InCurrentT5           ObservationID = 185
	ChargingSessionStart  ObservationID = 223
	CloudConnected        ObservationID = 250
)

// Supported returns true if the ObservationID is supported by our system.
func (o ObservationID) Supported() bool {
	for _, id := range SupportedObservationIDs() {
		if o == id {
			return true
		}
	}

	return false
}

// SupportedObservationIDs returns all observation IDs supported by our system.
func SupportedObservationIDs() []ObservationID {
	return []ObservationID{
		DetectedPowerGridType,
		PhaseMode,
		MaxChargerCurrent,
		DynamicChargerCurrent,
		ChargerOPState,
		OutputPhase,
		TotalPower,
		LifetimeEnergy,
		EnergySession,
		CableRating,
		InCurrentT3,
		InCurrentT4,
		InCurrentT5,
		CloudConnected,
		CableLocked,
		CableRating,
		LockCablePermanently,
	}
}

// ObservationDataType represents an Observation data type.
type ObservationDataType int

const (
	ObservationDataTypeBinary ObservationDataType = iota + 1
	ObservationDataTypeBoolean
	ObservationDataTypeDouble
	ObservationDataTypeInteger
	ObservationDataTypePosition
	ObservationDataTypeString
	ObservationDataTypeStatistics
)

// ChargerState represents an observation charger state.
type ChargerState int

const (
	ChargerStateUnknown ChargerState = iota - 1
	ChargerStateOffline
	ChargerStateDisconnected
	ChargerStateAwaitingStart
	ChargerStateCharging
	ChargerStateCompleted
	ChargerStateError
	ChargerStateReadyToCharge
	ChargerStateAwaitingAuthentication
	ChargerStateDeAuthenticating
)

type OutputPhaseType int

const (
	Unassigned   OutputPhaseType = 0
	P1T2T3TN     OutputPhaseType = 10
	P1T2T3IT     OutputPhaseType = 11
	P1T2T4TN     OutputPhaseType = 12
	P1T2T4IT     OutputPhaseType = 13
	P1T2T5TN     OutputPhaseType = 14
	P1T3T4IT     OutputPhaseType = 15
	P2T2T3T4TN   OutputPhaseType = 20
	P2T2T4T5TN   OutputPhaseType = 21
	P2T2T3T4IT   OutputPhaseType = 22
	P3T2T3T4T5TN OutputPhaseType = 30
)

func (o OutputPhaseType) ToFimpState() chargepoint.PhaseMode { //nolint:cyclop
	switch o { //nolint:exhaustive
	case P1T2T3TN:
		return chargepoint.PhaseModeNL1
	case P1T2T3IT:
		return chargepoint.PhaseModeL1L2
	case P1T2T4TN:
		return chargepoint.PhaseModeNL2
	case P1T2T4IT:
		return chargepoint.PhaseModeL3L1
	case P1T2T5TN:
		return chargepoint.PhaseModeNL3
	case P1T3T4IT:
		return chargepoint.PhaseModeL2L3
	case P2T2T3T4TN:
		return chargepoint.PhaseModeNL1L2
	case P2T2T4T5TN:
		return chargepoint.PhaseModeNL2L3
	case P2T2T3T4IT:
		return chargepoint.PhaseModeL1L2L3
	case P3T2T3T4T5TN:
		return chargepoint.PhaseModeNL1L2L3
	default:
		return ""
	}
}

// SupportedChargingStates returns all charging states supported by Easee.
func SupportedChargingStates() []ChargerState {
	return []ChargerState{
		ChargerStateOffline,
		ChargerStateDisconnected,
		ChargerStateAwaitingStart,
		ChargerStateCharging,
		ChargerStateCompleted,
		ChargerStateError,
		ChargerStateReadyToCharge,
		ChargerStateAwaitingAuthentication,
		ChargerStateDeAuthenticating,
	}
}

// ToFimpState returns a human-readable name of the state.
func (s ChargerState) ToFimpState() chargepoint.State { //nolint:cyclop
	switch s {
	case ChargerStateUnknown:
		return chargepoint.StateUnknown
	case ChargerStateOffline:
		return chargepoint.StateUnknown
	case ChargerStateDisconnected:
		return chargepoint.StateDisconnected
	case ChargerStateAwaitingStart:
		return chargepoint.StateReadyToCharge
	case ChargerStateCharging:
		return chargepoint.StateCharging
	case ChargerStateCompleted:
		return chargepoint.StateFinished
	case ChargerStateError:
		return chargepoint.StateError
	case ChargerStateReadyToCharge:
		return chargepoint.StateSuspendedByEV
	case ChargerStateAwaitingAuthentication:
		return chargepoint.StateRequesting
	case ChargerStateDeAuthenticating:
		return chargepoint.StateUnknown
	default:
		return chargepoint.StateUnknown
	}
}

func (s ChargerState) IsSessionFinished() bool {
	switch s { //nolint:exhaustive
	case ChargerStateUnknown,
		ChargerStateOffline,
		ChargerStateDisconnected,
		ChargerStateCompleted,
		ChargerStateError,
		ChargerStateAwaitingAuthentication,
		ChargerStateDeAuthenticating:
		return true
	default:
		return false
	}
}

// ClientState represents the state of the SignalR client.
type ClientState int

func (s ClientState) String() string {
	if s == ClientStateDisconnected {
		return "disconnected"
	}

	return "connected"
}

const (
	ClientStateDisconnected ClientState = iota
	ClientStateConnected
)

// GridType represents a grid type.
type GridType int

const (
	GridTypeUnknown                         GridType = -1
	GridTypeNotYetDetected                  GridType = 0
	GridTypeTN3Phase                        GridType = 1
	GridTypeTN2PhasePin23                   GridType = 2
	GridTypeTN1Phase                        GridType = 3
	GridTypeIT3Phase                        GridType = 4
	GridTypeIT1Phase                        GridType = 5
	GridTypeWarningTN2PhasePin235           GridType = 30
	GridTypeWarningTN1PhaseNeutralOnPin3    GridType = 31
	GridTypeWarningIT3PhaseGNDFault         GridType = 32
	GridTypeWarningIT1PhaseGNDFault         GridType = 33
	GridTypeWarningIT3PhaseGNDFaultL3       GridType = 34
	GridTypeWarningIT1PhaseGNDFaultL3       GridType = 35
	GridTypeWarningTN2PhasePIN234           GridType = 36
	GridTypeWarningTN3PhaseGNDFault         GridType = 37
	GridTypeWarningTN2PhaseGNDFault         GridType = 38
	GridTypeErrorNoValidPowerGridFound      GridType = 50
	GridTypeErrorTN400VNeutralOnWrongPin    GridType = 51
	GridTypeErrorITGroundConnectedToPin2Or3 GridType = 52
)

// ToFimpGridType returns grid type and phases.
func (g GridType) ToFimpGridType() (chargepoint.GridType, int) {
	if g >= GridTypeWarningTN2PhasePin235 {
		log.Warnf("faulty grid type detected: %s", g)
	}

	if t, ok := easeeNetworkTypeMap[g]; ok {
		return t.gridType, t.phases
	}

	log.Warnf("unknown grid type detected: %d", g)

	return "", 0
}

// String returns a human-readable name of the grid type.
func (g GridType) String() string { //nolint:cyclop
	switch g { //nolint:exhaustive
	case GridTypeNotYetDetected:
		return "not yet detected"
	case GridTypeTN3Phase:
		return "TN 3-phase"
	case GridTypeTN2PhasePin23:
		return "TN 2-phase (pin 2, 3)"
	case GridTypeTN1Phase:
		return "TN 1-phase"
	case GridTypeIT3Phase:
		return "IT 3-phase"
	case GridTypeIT1Phase:
		return "IT 1-phase"
	case GridTypeWarningTN2PhasePin235:
		return "TN 2-phase (pin 2, 3, 5)"
	case GridTypeWarningTN1PhaseNeutralOnPin3:
		return "TN 1-phase (neutral on pin 3)"
	case GridTypeWarningIT3PhaseGNDFault:
		return "IT 3-phase (ground fault)"
	case GridTypeWarningIT1PhaseGNDFault:
		return "IT 1-phase (ground fault)"
	case GridTypeWarningIT3PhaseGNDFaultL3:
		return "IT 3-phase (ground fault L3)"
	case GridTypeWarningIT1PhaseGNDFaultL3:
		return "IT 1-phase (ground fault L3)"
	case GridTypeWarningTN2PhasePIN234:
		return "TN 2-phase (pin 2, 3, 4)"
	case GridTypeWarningTN3PhaseGNDFault:
		return "TN 3-phase (ground fault)"
	case GridTypeWarningTN2PhaseGNDFault:
		return "TN 2-phase (ground fault)"
	case GridTypeErrorNoValidPowerGridFound:
		return "error - no valid power grid found"
	case GridTypeErrorTN400VNeutralOnWrongPin:
		return "error - TN 400V neutral on wrong pin"
	case GridTypeErrorITGroundConnectedToPin2Or3:
		return "error - IT ground connected to pin 2 or 3"
	default:
		return "unknown"
	}
}

type networkType struct {
	gridType chargepoint.GridType
	phases   int
}

var easeeNetworkTypeMap = map[GridType]networkType{
	GridTypeUnknown:                         {"", 0},
	GridTypeNotYetDetected:                  {"", 0},
	GridTypeErrorNoValidPowerGridFound:      {"", 0},
	GridTypeErrorTN400VNeutralOnWrongPin:    {chargepoint.GridTypeTN, 0},
	GridTypeErrorITGroundConnectedToPin2Or3: {chargepoint.GridTypeIT, 0},
	GridTypeTN3Phase:                        {chargepoint.GridTypeTN, 3},
	GridTypeTN2PhasePin23:                   {chargepoint.GridTypeTN, 2},
	GridTypeTN1Phase:                        {chargepoint.GridTypeTN, 1},
	GridTypeIT3Phase:                        {chargepoint.GridTypeIT, 3},
	GridTypeIT1Phase:                        {chargepoint.GridTypeIT, 1},
	GridTypeWarningTN2PhasePin235:           {chargepoint.GridTypeTN, 2},
	GridTypeWarningTN1PhaseNeutralOnPin3:    {chargepoint.GridTypeTN, 1},
	GridTypeWarningIT3PhaseGNDFault:         {chargepoint.GridTypeIT, 3},
	GridTypeWarningIT1PhaseGNDFault:         {chargepoint.GridTypeIT, 1},
	GridTypeWarningIT3PhaseGNDFaultL3:       {chargepoint.GridTypeIT, 3},
	GridTypeWarningIT1PhaseGNDFaultL3:       {chargepoint.GridTypeIT, 1},
	GridTypeWarningTN2PhasePIN234:           {chargepoint.GridTypeTN, 2},
	GridTypeWarningTN3PhaseGNDFault:         {chargepoint.GridTypeTN, 3},
	GridTypeWarningTN2PhaseGNDFault:         {chargepoint.GridTypeTN, 2},
}

type TimestampedValue[T any] struct {
	Value     T
	Timestamp time.Time
}

type StartChargingSession struct {
	ID         int64     `json:"Id"`
	MeterValue float64   `json:"MeterValue"`
	Start      time.Time `json:"Start"`
}

type StopChargingSession struct {
	ID              int64     `json:"Id"`
	Energy          float64   `json:"EnergyKwh"`
	MeterValueStart float64   `json:"MeterValueStart"`
	MeterValueStop  float64   `json:"MeterValueStop"`
	Start           time.Time `json:"Start"`
	Stop            time.Time `json:"Stop"`
}
