package signalr

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"
)

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

// JSONValue returns a string representation of the Observation value.
func (o *Observation) JSONValue(v any) error {
	if o.DataType != ObservationDataTypeString {
		return errors.New("observation data type is not string")
	}

	return json.Unmarshal([]byte(o.Value), v)
}
