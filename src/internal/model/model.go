package model

import (
	"errors"
	"strconv"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	log "github.com/sirupsen/logrus"
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

// ObservationID represents an Observation ID in Easee API.
type ObservationID int

const (
	DetectedPowerGridType ObservationID = 21
	PhaseMode             ObservationID = 38
	MaxChargerCurrent     ObservationID = 47
	DynamicChargerCurrent ObservationID = 48
	CableRating           ObservationID = 104
	ChargerOPState        ObservationID = 109
	OutputPhase           ObservationID = 110
	TotalPower            ObservationID = 120
	EnergySession         ObservationID = 121
	LifetimeEnergy        ObservationID = 124
	InCurrentT3           ObservationID = 183
	InCurrentT4           ObservationID = 184
	InCurrentT5           ObservationID = 185
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
	}
}

// ObservationDataType represents an Observation data type.
type ObservationDataType int

const (
	ObservationDataTypeBinary     ObservationDataType = 1
	ObservationDataTypeBoolean    ObservationDataType = 2
	ObservationDataTypeDouble     ObservationDataType = 3
	ObservationDataTypeInteger    ObservationDataType = 4
	ObservationDataTypePosition   ObservationDataType = 5
	ObservationDataTypeString     ObservationDataType = 6
	ObservationDataTypeStatistics ObservationDataType = 7
)

// ChargerState represents an observation charger state.
type ChargerState int

const (
	ChargerStateUnknown                ChargerState = -1
	ChargerStateOffline                ChargerState = 0
	ChargerStateDisconnected           ChargerState = 1
	ChargerStateAwaitingStart          ChargerState = 2
	ChargerStateCharging               ChargerState = 3
	ChargerStateCompleted              ChargerState = 4
	ChargerStateError                  ChargerState = 5
	ChargerStateReadyToCharge          ChargerState = 6
	ChargerStateAwaitingAuthentication ChargerState = 7
	ChargerStateDeAuthenticating       ChargerState = 8
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
		return chargepoint.StateRequesting
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
	GridTypeErrorNoValidPowerGridFound      GridType = 50
	GridTypeErrorTN400VNeutralOnWrongPin    GridType = 51
	GridTypeErrorITGroundConnectedToPin2Or3 GridType = 52
	GridTypeWarningIT3PhaseGNDFaultL3       GridType = 34
	GridTypeWarningIT1PhaseGNDFaultL3       GridType = 35
	GridTypeWarningTN2PhasePIN234           GridType = 36
	GridTypeWarningTN3PhaseGNDFault         GridType = 37
	GridTypeWarningTN2PhaseGNDFault         GridType = 38

	GridTypeFirstInvalid = GridTypeWarningTN2PhasePin235
)

type networkType struct {
	gridType chargepoint.GridType
	phase    int
}

var easeeNetworkTypeMap = map[GridType]networkType{
	GridTypeUnknown:                         {"", 0},
	GridTypeNotYetDetected:                  {"", 0},
	GridTypeTN3Phase:                        {chargepoint.GridTypeTN, 3},
	GridTypeTN2PhasePin23:                   {chargepoint.GridTypeTN, 2},
	GridTypeTN1Phase:                        {chargepoint.GridTypeTN, 1},
	GridTypeIT3Phase:                        {chargepoint.GridTypeIT, 3},
	GridTypeIT1Phase:                        {chargepoint.GridTypeIT, 1},
	GridTypeWarningTN2PhasePin235:           {chargepoint.GridTypeTN, 2},
	GridTypeWarningTN1PhaseNeutralOnPin3:    {chargepoint.GridTypeTN, 1},
	GridTypeWarningIT3PhaseGNDFault:         {chargepoint.GridTypeIT, 3},
	GridTypeWarningIT1PhaseGNDFault:         {chargepoint.GridTypeIT, 1},
	GridTypeErrorNoValidPowerGridFound:      {"", 0},
	GridTypeErrorTN400VNeutralOnWrongPin:    {chargepoint.GridTypeTN, 0},
	GridTypeErrorITGroundConnectedToPin2Or3: {chargepoint.GridTypeIT, 0},
	GridTypeWarningIT3PhaseGNDFaultL3:       {chargepoint.GridTypeIT, 3},
	GridTypeWarningIT1PhaseGNDFaultL3:       {chargepoint.GridTypeIT, 1},
	GridTypeWarningTN2PhasePIN234:           {chargepoint.GridTypeTN, 2},
	GridTypeWarningTN3PhaseGNDFault:         {chargepoint.GridTypeTN, 3},
	GridTypeWarningTN2PhaseGNDFault:         {chargepoint.GridTypeTN, 2},
}

// ToFimpGridType returns grid type and phases.
func (g GridType) ToFimpGridType() (chargepoint.GridType, int) {
	if g >= GridTypeFirstInvalid {
		log.Warnf("Invalid grid type state %v", g)
	}

	if networkType, ok := easeeNetworkTypeMap[g]; ok {
		return networkType.gridType, networkType.phase
	}

	log.Warnf("Unknown grid type: %v", g)

	return "", 0
}
