package backoff

import "time"

// ExponentialBackoff is a struct that handles backoff duration.
type ExponentialBackoff struct {
	initialBackoff       time.Duration
	repeatedBackoff      time.Duration
	finalBackoff         time.Duration
	initialFailureCount  int
	repeatedFailureCount int
	failures             int
}

// NewExponentialBackoff creates ExponentialBackoff struct.
func NewExponentialBackoff(initialBackoff time.Duration, repeatedBackoff time.Duration, finalBackoff time.Duration, initialFailureCount int, repeatedFailureCount int) *ExponentialBackoff {
	return &ExponentialBackoff{
		initialBackoff:       initialBackoff,
		repeatedBackoff:      repeatedBackoff,
		finalBackoff:         finalBackoff,
		initialFailureCount:  initialFailureCount,
		repeatedFailureCount: repeatedFailureCount,
	}
}

// Reset resets exponential backoff failures.
func (e *ExponentialBackoff) Reset() {
	e.failures = 0
}

// Next increases failure counter and calculates next backoff duration.
func (e *ExponentialBackoff) Next() time.Duration {
	e.failures++

	if e.failures <= e.initialFailureCount {
		return e.initialBackoff
	}

	failuresAfterInitial := e.failures - e.initialFailureCount
	if failuresAfterInitial <= e.repeatedFailureCount {
		return e.repeatedBackoff
	}

	return e.finalBackoff
}
