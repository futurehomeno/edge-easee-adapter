package signalr

import (
	"errors"
	"strconv"
	"time"
)

// Observation represents a SignalR observation data.
type Observation struct {
	ChargerID string              `json:"mid"`
	DataType  ObservationDataType `json:"dataType"`
	ID        ObservationID       `json:"id"`
	Timestamp time.Time           `json:"timestamp"`
	Value     string              `json:"value"`
}

// IntValue returns an integer representation of the Observation value.
func (o Observation) IntValue() (int, error) {
	if o.DataType != Integer {
		return 0, errors.New("observation data type is not int")
	}

	return strconv.Atoi(o.Value)
}

// Float64Value returns a float64 representation of the Observation value.
func (o Observation) Float64Value() (float64, error) {
	if o.DataType != Double {
		return 0, errors.New("observation data type is not float64")
	}

	return strconv.ParseFloat(o.Value, 64)
}

// BoolValue returns a bool representation of the Observation value.
func (o Observation) BoolValue() (bool, error) {
	if o.DataType != Boolean {
		return false, errors.New("observation data type is not bool")
	}

	return strconv.ParseBool(o.Value)
}

// ObservationID represents an Observation ID in Easee API.
type ObservationID int

// Supported returns true if the ObservationID is supported by our system.
func (o ObservationID) Supported() bool {
	for _, id := range SupportedObservationIDs() {
		if o == id {
			return true
		}
	}

	return false
}

const (
	CableLocked    ObservationID = 103
	CableRating    ObservationID = 104
	ChargerOPState ObservationID = 109
	TotalPower     ObservationID = 120
	SessionEnergy  ObservationID = 121
	LifetimeEnergy ObservationID = 124
)

// SupportedObservationIDs returns all observation IDs supported by our system.
func SupportedObservationIDs() []ObservationID {
	return []ObservationID{
		ChargerOPState,
		SessionEnergy,
		CableLocked,
		TotalPower,
		LifetimeEnergy,
		CableRating,
	}
}

// ObservationDataType represents an Observation data type.
type ObservationDataType int

const (
	Binary ObservationDataType = iota + 1
	Boolean
	Double
	Integer
	Position
	String
	Statistics
)
