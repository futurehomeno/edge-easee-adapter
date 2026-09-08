package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

// Easee sends the cable rating as integer or double depending on the charger - the repo's own
// fixtures disagreed about it - so the strict integer accessor dropped half of them, and the
// cable current went unreported for a locked cable.
func TestObservation_NumericIntValue_AcceptsIntegerAndDouble(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dataType model.ObservationDataType
		value    string
		want     int
	}{
		{name: "integer", dataType: model.ObservationDataTypeInteger, value: "32", want: 32},
		{name: "double", dataType: model.ObservationDataTypeDouble, value: "32.0", want: 32},
		{name: "double rounds to nearest", dataType: model.ObservationDataTypeDouble, value: "31.6", want: 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			o := model.Observation{ID: model.CableRating, DataType: tt.dataType, Value: tt.value}

			got, err := o.NumericIntValue()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestObservation_NumericIntValue_RejectsNonNumeric(t *testing.T) {
	t.Parallel()

	o := model.Observation{ID: model.CableRating, DataType: model.ObservationDataTypeString, Value: "thirty-two"}

	_, err := o.NumericIntValue()
	assert.Error(t, err)
}
