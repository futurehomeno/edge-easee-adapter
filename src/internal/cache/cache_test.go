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
	waiting := make(chan struct{})

	go func() {
		close(waiting)

		result <- c.WaitForOfferedCurrent(16, 30*time.Second)
	}()

	<-waiting

	// The listener registers a moment after the goroutine starts, and only a notification
	// delivered after that reproduces the supersede. Retry the pair until one is observed
	// rather than sleeping for a fixed window a loaded runner can outlast.
	now := time.Now()

	for i := 0; ; i++ {
		c.SetOfferedCurrent(16, now.Add(time.Duration(2*i)*time.Millisecond))
		c.SetOfferedCurrent(8, now.Add(time.Duration(2*i+1)*time.Millisecond))

		select {
		case got := <-result:
			assert.True(t, got, "the wait must resolve on the delivered confirmation")

			return
		case <-time.After(10 * time.Millisecond):
		}

		if i > 500 {
			t.Fatal("wait did not return although the awaited current was observed")
		}
	}
}

func TestCache_WaitForOfferedCurrent_TimesOutWhenNeverObserved(t *testing.T) {
	t.Parallel()

	c := cache.NewCache("test-charger")

	assert.False(t, c.WaitForOfferedCurrent(16, 50*time.Millisecond))
}

// The seed the controller writes must never outrank a real observation. Easee stamps
// observations with its own clock, so one can legitimately arrive bearing a timestamp behind
// the hub's notion of now; the seed bypasses the ordering guard so it never suppresses one.
func TestCache_SeedCableAlwaysLocked_ObservationOverridesTheSeed(t *testing.T) {
	t.Parallel()

	c := cache.NewCache("test-charger")

	c.SeedCableAlwaysLocked(true)

	observed, _ := c.CableAlwaysLocked()
	assert.True(t, observed, "the optimistic seed must be readable straight away")

	// An observation stamped well behind the hub's clock still wins.
	assert.True(t, c.SetCableAlwaysLocked(false, time.Now().Add(-time.Hour)))

	observed, _ = c.CableAlwaysLocked()
	assert.False(t, observed, "the observation is authoritative over the seed")
}

// The seed must also land when the cache already holds a real observation: a zero-time
// seed would be rejected by the ordering guard, leaving cmd.param.set echoing the old value.
func TestCache_SeedCableAlwaysLocked_AppliesOverAPopulatedCache(t *testing.T) {
	t.Parallel()

	c := cache.NewCache("test-charger")

	assert.True(t, c.SetCableAlwaysLocked(false, time.Now()))

	c.SeedCableAlwaysLocked(true)

	observed, _ := c.CableAlwaysLocked()
	assert.True(t, observed, "the seed must overwrite an already populated cache")
}
