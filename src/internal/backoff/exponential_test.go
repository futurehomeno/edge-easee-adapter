package backoff_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/edge-easee-adapter/internal/backoff"
	"github.com/stretchr/testify/assert"
)

func TestExponentialBackoff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		backoff         *backoff.Exponential
		expectedResults []time.Duration
	}{
		{
			name:            "regular backoff",
			backoff:         backoff.NewExponential(time.Second, 2*time.Second, 3*time.Second, 1, 2),
			expectedResults: []time.Duration{time.Second, 2 * time.Second, 2 * time.Second, 3 * time.Second, 3 * time.Second},
		},
		{
			name:            "no initial backoff",
			backoff:         backoff.NewExponential(time.Second, 2*time.Second, 3*time.Second, 0, 2),
			expectedResults: []time.Duration{2 * time.Second, 2 * time.Second, 3 * time.Second},
		},
		{
			name:            "no repeated backoff",
			backoff:         backoff.NewExponential(time.Second, 2*time.Second, 3*time.Second, 2, 0),
			expectedResults: []time.Duration{time.Second, time.Second, 3 * time.Second},
		},
		{
			name:            "no initial and repeated backoff",
			backoff:         backoff.NewExponential(0, 0, time.Second, 0, 0),
			expectedResults: []time.Duration{time.Second, time.Second, time.Second, time.Second, time.Second},
		},
		{
			name:            "empty backoff",
			backoff:         backoff.NewExponential(0, 0, 0, 0, 0),
			expectedResults: []time.Duration{0, 0, 0, 0, 0},
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			for i, expected := range test.expectedResults {
				actual := test.backoff.Next()
				assert.Equal(t, expected, actual, "invalid %d backoff", i+1)
			}

			test.backoff.Reset()

			for i, expected := range test.expectedResults {
				actual := test.backoff.Next()
				assert.Equal(t, expected, actual, "invalid %d backoff after reset", i+1)
			}
		})
	}
}
