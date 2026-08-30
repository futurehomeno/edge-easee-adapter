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
