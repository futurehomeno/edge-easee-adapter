package signalr

import "github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"

// ObservationID represents an Observation ID in Easee API.
type ObservationID int

const (
	MaxChargerCurrent     ObservationID = 47
	DynamicChargerCurrent ObservationID = 48
	CableLocked           ObservationID = 103
	CableRating           ObservationID = 104
	ChargerOPState        ObservationID = 109
	TotalPower            ObservationID = 120
	EnergySession         ObservationID = 121
	LifetimeEnergy        ObservationID = 124
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
		MaxChargerCurrent,
		DynamicChargerCurrent,
		ChargerOPState,
		CableLocked,
		TotalPower,
		LifetimeEnergy,
		EnergySession,
		CableRating,
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
func (s ChargerState) ToFimpState() chargepoint.State {
	switch s {
	case ChargerStateUnknown:
		return chargepoint.StateUnknown
	case ChargerStateOffline:
		return chargepoint.StateUnavailable
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
		return "authenticating"
	case ChargerStateDeAuthenticating:
		return "de-authenticating"
	default:
		return chargepoint.StateUnknown
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
