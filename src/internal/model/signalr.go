package model

import "github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"

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
	P1T2T3T4IT   OutputPhaseType = 22
	P3T2T3T4T5TN OutputPhaseType = 30
)

func (o OutputPhaseType) ToFimpState() chargepoint.PhaseMode {
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
