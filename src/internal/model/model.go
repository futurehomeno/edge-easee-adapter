package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/service/alarm"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/types"
	log "github.com/sirupsen/logrus"
)

const (
	CableAlwaysLockedParameter = "cable_always_locked"
)

type Credentials struct {
	AccessToken  string   `json:"accessToken"`
	ExpiresIn    int      `json:"expiresIn"`
	AccessClaims []string `json:"accessClaims"`
	TokenType    string   `json:"tokenType"`
	RefreshToken string   `json:"refreshToken"`
}

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

type ChargerDetails struct {
	Product string `json:"product"`
}

type BackPlate struct {
	ID                string `json:"id"`
	MasterBackPlateID string `json:"masterBackPlateId"`
}

type ChargerConfig struct {
	DetectedPowerGridType GridType `json:"detectedPowerGridType"`
	PhaseMode             int      `json:"phaseMode"`
}

// ChargerSiteInfo carries the site's rated current.
type ChargerSiteInfo struct {
	RatedCurrent float64 `json:"ratedCurrent"`
}

const (
	ChargingModeNormal = "normal"
	ChargingModeSlow   = "slow"
)

func SupportedChargingModes() []string {
	return []string{
		ChargingModeNormal,
		ChargingModeSlow,
	}
}

type Observation struct {
	ID        ObservationID       `json:"id"`
	ChargerID string              `json:"mid"`
	DataType  ObservationDataType `json:"dataType"`
	Timestamp time.Time           `json:"timestamp"`
	Value     string              `json:"value"`
}

func (o *Observation) Str() string {
	if o.DataType == ObservationDataTypeDouble {
		if v, err := strconv.ParseFloat(o.Value, 64); err == nil {
			return fmt.Sprintf("[%s] %s=%.2f", o.ChargerID, o.ID.Str(), v)
		}
	}

	return fmt.Sprintf("[%s] %s=%s", o.ChargerID, o.ID.Str(), o.Value)
}

func (o *Observation) IntValue() (int, error) {
	if o.DataType != ObservationDataTypeInteger {
		return 0, errors.New("observation data type is not int")
	}

	return strconv.Atoi(o.Value)
}

func (o *Observation) Float64Value() (float64, error) {
	if o.DataType != ObservationDataTypeDouble {
		return 0, errors.New("observation data type is not float64")
	}

	return strconv.ParseFloat(o.Value, 64)
}

func (o *Observation) BoolValue() (bool, error) {
	if o.DataType != ObservationDataTypeBoolean {
		return false, errors.New("observation data type is not bool")
	}

	return strconv.ParseBool(o.Value)
}

// JSONValue unmarshals the Observation value into v.
func (o *Observation) JSONValue(v any) error {
	if o.DataType != ObservationDataTypeString {
		return errors.New("observation data type is not string")
	}

	return json.Unmarshal([]byte(o.Value), v)
}

type ObservationID int

const (
	DetectedPowerGridType ObservationID = 21
	LockCablePermanently  ObservationID = 30
	PhaseMode             ObservationID = 38
	LEDBrightness         ObservationID = 40
	MaxChargerCurrent     ObservationID = 47
	DynamicChargerCurrent ObservationID = 48
	SoftwareVersion       ObservationID = 80
	NoCurrentReason       ObservationID = 96
	CableLocked           ObservationID = 103
	CableRating           ObservationID = 104
	ChargerOPState        ObservationID = 109
	OutputPhase           ObservationID = 110
	ErrorString           ObservationID = 118
	ErrorCode             ObservationID = 119
	TotalPower            ObservationID = 120
	EnergySession         ObservationID = 121
	LifetimeEnergy        ObservationID = 124
	ChargingSessionStop   ObservationID = 129
	CellRSSI              ObservationID = 130
	WiFiRSSI              ObservationID = 132
	RadioRSSI             ObservationID = 136
	ConnectionType        ObservationID = 141
	InCurrentT3           ObservationID = 183
	InCurrentT4           ObservationID = 184
	InCurrentT5           ObservationID = 185
	ChargingSessionStart  ObservationID = 223
	CloudConnected        ObservationID = 250
	CloudDisconnectReason ObservationID = 251
)

var observationIDNames = map[ObservationID]string{
	DetectedPowerGridType: "detected_power_grid_type",
	LockCablePermanently:  "lock_cable_permanently",
	PhaseMode:             "phase_mode",
	MaxChargerCurrent:     "max_charger_current",
	DynamicChargerCurrent: "dynamic_charger_current",
	CableLocked:           "cable_locked",
	CableRating:           "cable_rating",
	ChargerOPState:        "charger_op_state",
	OutputPhase:           "output_phase",
	TotalPower:            "total_power",
	EnergySession:         "energy_session",
	LifetimeEnergy:        "lifetime_energy",
	ChargingSessionStop:   "charging_session_stop",
	InCurrentT3:           "in_current_t3",
	InCurrentT4:           "in_current_t4",
	InCurrentT5:           "in_current_t5",
	ChargingSessionStart:  "charging_session_start",
	CloudConnected:        "cloud_connected",
	ErrorCode:             "error_code",
	ErrorString:           "error_string",
}

func (o ObservationID) Str() string {
	if name, ok := observationIDNames[o]; ok {
		return name
	}

	return fmt.Sprintf("unknown=%d", o)
}

// supportedObservationIDs is snapshotted once: Supported() is called for every observation
// that arrives, including the full replay burst after each reconnect.
var supportedObservationIDs = SupportedObservationIDs()

func (o ObservationID) Supported() bool {
	return slices.Contains(supportedObservationIDs, o)
}

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
		ErrorCode,
		InCurrentT3,
		InCurrentT4,
		InCurrentT5,
		CloudConnected,
		CableLocked,
		CableRating,
		LockCablePermanently,
		ChargingSessionStart,
		ChargingSessionStop,
	}
}

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

var chargerStateNames = map[ChargerState]string{
	ChargerStateUnknown:                "unknown",
	ChargerStateOffline:                "offline",
	ChargerStateDisconnected:           "disconnected",
	ChargerStateAwaitingStart:          "await_start",
	ChargerStateCharging:               "charging",
	ChargerStateCompleted:              "completed",
	ChargerStateError:                  "error",
	ChargerStateReadyToCharge:          "ready_to_charge",
	ChargerStateAwaitingAuthentication: "await_auth",
	ChargerStateDeAuthenticating:       "de_auth",
}

func (s ChargerState) Str() string {
	if name, ok := chargerStateNames[s]; ok {
		return name
	}

	return fmt.Sprintf("unknown(%d)", s)
}

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

// outputPhaseModes omits Unassigned: its zero value is the empty mode the charger reports
// while idle, which callers treat as "no reading".
var outputPhaseModes = map[OutputPhaseType]types.PhaseMode{
	P1T2T3TN:     types.PhaseModeNL1,
	P1T2T3IT:     types.PhaseModeL1L2,
	P1T2T4TN:     types.PhaseModeNL2,
	P1T2T4IT:     types.PhaseModeL3L1,
	P1T2T5TN:     types.PhaseModeNL3,
	P1T3T4IT:     types.PhaseModeL2L3,
	P2T2T3T4TN:   types.PhaseModeNL1L2,
	P2T2T4T5TN:   types.PhaseModeNL2L3,
	P2T2T3T4IT:   types.PhaseModeL1L2L3,
	P3T2T3T4T5TN: types.PhaseModeNL1L2L3,
}

func (o OutputPhaseType) ToFimpState() types.PhaseMode {
	return outputPhaseModes[o]
}

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

// fimpStates omits every state that maps to StateUnknown; the lookup falls back to it.
var fimpStates = map[ChargerState]chargepoint.State{
	ChargerStateDisconnected:           chargepoint.StateDisconnected,
	ChargerStateAwaitingStart:          chargepoint.StateReadyToCharge,
	ChargerStateCharging:               chargepoint.StateCharging,
	ChargerStateCompleted:              chargepoint.StateFinished,
	ChargerStateError:                  chargepoint.StateError,
	ChargerStateReadyToCharge:          chargepoint.StateSuspendedByEV,
	ChargerStateAwaitingAuthentication: chargepoint.StateRequesting,
}

func (s ChargerState) ToFimpState() chargepoint.State {
	if state, ok := fimpStates[s]; ok {
		return state
	}

	return chargepoint.StateUnknown
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

func SupportedAlarmEvents() []string {
	return []string{alarm.EventGroundingFault, alarm.EventGridTypeFault, alarm.EventOtherChargeErr}
}

// IsGroundFault reports whether the detected grid type indicates a ground fault.
func (g GridType) IsGroundFault() bool {
	switch g { //nolint:exhaustive
	case GridTypeWarningIT3PhaseGNDFault,
		GridTypeWarningIT1PhaseGNDFault,
		GridTypeWarningIT3PhaseGNDFaultL3,
		GridTypeWarningIT1PhaseGNDFaultL3,
		GridTypeWarningTN3PhaseGNDFault,
		GridTypeWarningTN2PhaseGNDFault,
		GridTypeErrorITGroundConnectedToPin2Or3:
		return true
	default:
		return false
	}
}

// IsWiringFault reports whether the detected grid type indicates a mis-wired installation.
func (g GridType) IsWiringFault() bool {
	switch g { //nolint:exhaustive
	case GridTypeWarningTN2PhasePin235,
		GridTypeWarningTN1PhaseNeutralOnPin3,
		GridTypeWarningTN2PhasePIN234,
		GridTypeErrorNoValidPowerGridFound,
		GridTypeErrorTN400VNeutralOnWrongPin:
		return true
	default:
		return false
	}
}

// ToFimpGridType maps an Easee grid type onto a FIMP grid type and phase count.
func (g GridType) ToFimpGridType() (types.GridType, int) {
	if g >= GridTypeWarningTN2PhasePin235 {
		log.Warnf("faulty grid type detected: %s", g)
	}

	if t, ok := easeeNetworkTypeMap[g]; ok {
		return t.gridType, t.phases
	}

	log.Warnf("unknown grid type detected: %d", g)

	return "", 0
}

var gridTypeNames = map[GridType]string{
	GridTypeNotYetDetected:                  "not yet detected",
	GridTypeTN3Phase:                        "TN 3-phase",
	GridTypeTN2PhasePin23:                   "TN 2-phase (pin 2, 3)",
	GridTypeTN1Phase:                        "TN 1-phase",
	GridTypeIT3Phase:                        "IT 3-phase",
	GridTypeIT1Phase:                        "IT 1-phase",
	GridTypeWarningTN2PhasePin235:           "TN 2-phase (pin 2, 3, 5)",
	GridTypeWarningTN1PhaseNeutralOnPin3:    "TN 1-phase (neutral on pin 3)",
	GridTypeWarningIT3PhaseGNDFault:         "IT 3-phase (ground fault)",
	GridTypeWarningIT1PhaseGNDFault:         "IT 1-phase (ground fault)",
	GridTypeWarningIT3PhaseGNDFaultL3:       "IT 3-phase (ground fault L3)",
	GridTypeWarningIT1PhaseGNDFaultL3:       "IT 1-phase (ground fault L3)",
	GridTypeWarningTN2PhasePIN234:           "TN 2-phase (pin 2, 3, 4)",
	GridTypeWarningTN3PhaseGNDFault:         "TN 3-phase (ground fault)",
	GridTypeWarningTN2PhaseGNDFault:         "TN 2-phase (ground fault)",
	GridTypeErrorNoValidPowerGridFound:      "error - no valid power grid found",
	GridTypeErrorTN400VNeutralOnWrongPin:    "error - TN 400V neutral on wrong pin",
	GridTypeErrorITGroundConnectedToPin2Or3: "error - IT ground connected to pin 2 or 3",
}

func (g GridType) String() string {
	if name, ok := gridTypeNames[g]; ok {
		return name
	}

	return "unknown"
}

type networkType struct {
	gridType types.GridType
	phases   int
}

var easeeNetworkTypeMap = map[GridType]networkType{
	GridTypeUnknown:                         {"", 0},
	GridTypeNotYetDetected:                  {"", 0},
	GridTypeErrorNoValidPowerGridFound:      {"", 0},
	GridTypeErrorTN400VNeutralOnWrongPin:    {types.GridTypeTN, 0},
	GridTypeErrorITGroundConnectedToPin2Or3: {types.GridTypeIT, 0},
	GridTypeTN3Phase:                        {types.GridTypeTN, 3},
	GridTypeTN2PhasePin23:                   {types.GridTypeTN, 2},
	GridTypeTN1Phase:                        {types.GridTypeTN, 1},
	GridTypeIT3Phase:                        {types.GridTypeIT, 3},
	GridTypeIT1Phase:                        {types.GridTypeIT, 1},
	GridTypeWarningTN2PhasePin235:           {types.GridTypeTN, 2},
	GridTypeWarningTN1PhaseNeutralOnPin3:    {types.GridTypeTN, 1},
	GridTypeWarningIT3PhaseGNDFault:         {types.GridTypeIT, 3},
	GridTypeWarningIT1PhaseGNDFault:         {types.GridTypeIT, 1},
	GridTypeWarningIT3PhaseGNDFaultL3:       {types.GridTypeIT, 3},
	GridTypeWarningIT1PhaseGNDFaultL3:       {types.GridTypeIT, 1},
	GridTypeWarningTN2PhasePIN234:           {types.GridTypeTN, 2},
	GridTypeWarningTN3PhaseGNDFault:         {types.GridTypeTN, 3},
	GridTypeWarningTN2PhaseGNDFault:         {types.GridTypeTN, 2},
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
