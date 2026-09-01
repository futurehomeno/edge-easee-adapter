package cache_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/edge-easee-adapter/internal/cache"
)

// A value already cached returns without registering a listener at all.
func TestCache_WaitForOfferedCurrent_ReturnsImmediatelyWhenAlreadySet(t *testing.T) {
	t.Parallel()

	c := cache.NewCache("test-charger")
	c.SetOfferedCurrent(16, time.Now())

	assert.True(t, c.WaitForOfferedCurrent(16, time.Second))
}

// The wait resolves on the cached value rather than on the delivered notification, so a
// sequence of updates ending on the awaited current succeeds however the notifications
// interleaved - including when one was dropped because a listener buffer was full.
func TestCache_WaitForOfferedCurrent_ResolvesOnTheCachedValue(t *testing.T) {
	t.Parallel()

	c := cache.NewCache("test-charger")

	result := make(chan bool, 1)

	go func() {
		result <- c.WaitForOfferedCurrent(16, 30*time.Second)
	}()

	time.Sleep(50 * time.Millisecond)

	now := time.Now()

	c.SetOfferedCurrent(8, now)
	c.SetOfferedCurrent(16, now.Add(time.Millisecond))

	select {
	case got := <-result:
		assert.True(t, got, "the wait must observe the value the cache holds")
	case <-time.After(5 * time.Second):
		t.Fatal("wait did not return although the cache holds the awaited current")
	}
}

// The converse of the case above: the awaited current is observed and then immediately
// superseded, so by the time the waiter runs the cache no longer holds it and the second
// notification was dropped on the full buffer. The delivered value is the only remaining
// evidence the charger echoed the request, so the wait must resolve on it.
func TestCache_WaitForOfferedCurrent_ResolvesOnASupersededNotification(t *testing.T) {
	t.Parallel()

	c := cache.NewCache("test-charger")

	result := make(chan bool, 1)

	go func() {
		result <- c.WaitForOfferedCurrent(16, 30*time.Second)
	}()

	time.Sleep(50 * time.Millisecond)

	now := time.Now()

	c.SetOfferedCurrent(16, now)
	c.SetOfferedCurrent(8, now.Add(time.Millisecond))

	select {
	case got := <-result:
		assert.True(t, got, "the wait must resolve on the delivered confirmation")
	case <-time.After(5 * time.Second):
		t.Fatal("wait did not return although the awaited current was observed")
	}
}

// The seed the controller writes must never outrank a real observation. Easee stamps
// observations with its own clock, so one can legitimately arrive bearing a timestamp behind
// the hub's notion of now; the zero-time seed keeps the guard from rejecting it.
func TestCache_SetCableAlwaysLocked_ObservationOverridesTheSeed(t *testing.T) {
	t.Parallel()

	c := cache.NewCache("test-charger")

	c.SetCableAlwaysLocked(true, time.Time{})

	observed, _ := c.CableAlwaysLocked()
	assert.True(t, observed, "the optimistic seed must be readable straight away")

	// An observation stamped well behind the hub's clock still wins.
	assert.True(t, c.SetCableAlwaysLocked(false, time.Now().Add(-time.Hour)))

	observed, _ = c.CableAlwaysLocked()
	assert.False(t, observed, "the observation is authoritative over the seed")
}
