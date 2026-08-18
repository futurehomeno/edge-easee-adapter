package api

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/notification"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubNotifier struct{}

func (stubNotifier) Event(*notification.Event) error { return nil }

// TestAuthLossHandler_FallsBackToLocalLogout pins the recovery path for a broker that will not
// take the logout command: the framework has already cleared the credentials, so without the
// fallback the SignalR client stays connected and the lifecycle keeps claiming a session that
// is gone until the next restart.
func TestAuthLossHandler_FallsBackToLocalLogout(t *testing.T) {
	t.Parallel()

	// Never started, so PublishToTopic cannot reach a broker.
	mqtt := fimpgo.NewMqttTransport("tcp://127.0.0.1:1", "test", "", "", true, 1, 1, func(error) {})

	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		called bool
	)

	wg.Add(1)

	handler := authLossHandler(stubNotifier{}, mqtt, fimptype.EaseeService, func() error {
		defer wg.Done()

		mu.Lock()
		defer mu.Unlock()

		called = true

		return errors.New("logout failed")
	})

	handler("token rejected")

	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.Fail(t, "logout fallback was not invoked")
	}

	mu.Lock()
	defer mu.Unlock()

	assert.True(t, called)
}

// A nil fallback must not panic: the handler runs under the authenticator lock.
func TestAuthLossHandler_NilFallback(t *testing.T) {
	t.Parallel()

	mqtt := fimpgo.NewMqttTransport("tcp://127.0.0.1:1", "test", "", "", true, 1, 1, func(error) {})

	assert.NotPanics(t, func() {
		authLossHandler(stubNotifier{}, mqtt, fimptype.EaseeService, nil)("token rejected")
	})
}
