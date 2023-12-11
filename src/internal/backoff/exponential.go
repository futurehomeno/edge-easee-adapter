package backoff

import (
	"sync/atomic"
	"time"
)

// Exponential is a struct that handles backoff duration.
type Exponential struct {
	initialBackoff       time.Duration
	repeatedBackoff      time.Duration
	finalBackoff         time.Duration
	initialFailureCount  uint32
	repeatedFailureCount uint32

	failures atomic.Uint32
}

// NewExponential creates ExponentialBackoff struct.
func NewExponential(initialBackoff, repeatedBackoff, finalBackoff time.Duration,
	initialFailureCount, repeatedFailureCount uint32,
) *Exponential {
	return &Exponential{
		initialBackoff:       initialBackoff,
		repeatedBackoff:      repeatedBackoff,
		finalBackoff:         finalBackoff,
		initialFailureCount:  initialFailureCount,
		repeatedFailureCount: repeatedFailureCount,
	}
}

// Reset resets exponential backoff failures.
func (e *Exponential) Reset() {
	e.failures.Swap(0)
}

// Next increases failure counter and calculates next backoff duration.
func (e *Exponential) Next() time.Duration {
	failures := e.failures.Add(1)

	if failures <= e.initialFailureCount {
		return e.initialBackoff
	}

	failuresAfterInitial := failures - e.initialFailureCount
	if failuresAfterInitial <= e.repeatedFailureCount {
		return e.repeatedBackoff
	}

	return e.finalBackoff
}
