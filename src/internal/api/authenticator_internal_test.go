package api

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/notification"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

type stubNotifier struct{}

func (stubNotifier) Event(*notification.Event) error { return nil }

type stubPublisher struct {
	err error
	// onPublish runs inside the publish so a test can land a login exactly while the auth-loss
	// handler is blocked on the broker.
	onPublish func()
}

func (p stubPublisher) PublishToTopic(string, *fimpgo.FimpMessage) error {
	if p.onPublish != nil {
		p.onPublish()
	}

	return p.err
}

// unchangedCredentials stubs the credentials accessor for tests where no fresh login is in
// play: every call returns the same snapshot, so the fallback's staleness check always passes.
func unchangedCredentials() config.Credentials { return config.Credentials{} }

// TestAuthLossHandler_RunsLocalLogout pins the recovery path: the framework has already cleared
// the credentials, so a logout that never reaches the app leaves the SignalR client connected
// and the lifecycle claiming a session until the next restart. A successful publish is not proof
// the logout ran - the routed handler drops the command when it loses its try-lock - so the
// fallback has to run on both outcomes.
func TestAuthLossHandler_RunsLocalLogout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		publishErr  error
		fallbackErr error
	}{
		{name: "broker refuses the publish", publishErr: errors.New("publish failed"), fallbackErr: errors.New("logout failed")},
		{name: "publish succeeds but the routed handler may drop it", publishErr: nil, fallbackErr: errors.New("logout failed")},
		{name: "fallback succeeds", publishErr: nil, fallbackErr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := make(chan struct{})

			handler := authLossHandler(stubNotifier{}, stubPublisher{err: tt.publishErr}, fimptype.EaseeService, unchangedCredentials, func() error {
				close(called)

				return tt.fallbackErr
			})

			handler("token rejected")

			select {
			case <-called:
			case <-time.After(5 * time.Second):
				require.Fail(t, "logout fallback was not invoked")
			}
		})
	}
}

// A concurrent re-login between the auth-loss event and the fallback goroutine running must win:
// the fallback is cleaning up a session that no longer exists, and running it anyway would log
// the fresh session straight back out.
func TestAuthLossHandler_SkipsFallbackWhenCredentialsChanged(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	credentials := func() config.Credentials {
		// First read: the snapshot taken synchronously before the fallback goroutine spawns.
		// Second read: taken from inside the goroutine, simulating a login that landed in between.
		if calls.Add(1) == 1 {
			return config.Credentials{}
		}

		return config.Credentials{AccessToken: "fresh-login-value"} //nolint:gosec
	}

	fallbackCalled := make(chan struct{})

	handler := authLossHandler(stubNotifier{}, stubPublisher{}, fimptype.EaseeService, credentials, func() error {
		close(fallbackCalled)

		return nil
	})

	handler("token rejected")

	select {
	case <-fallbackCalled:
		require.Fail(t, "logout fallback ran after a fresh login replaced the cleared credentials")
	case <-time.After(200 * time.Millisecond):
	}
}

// The snapshot has to be taken before the notification and the publish, not next to the
// goroutine that reads it: a publish blocks while the broker is down, so a login landing in
// that window would end up captured as the session the fallback is cleaning up - and wiped.
func TestAuthLossHandler_SnapshotsCredentialsBeforePublishing(t *testing.T) {
	t.Parallel()

	var current atomic.Pointer[config.Credentials]

	current.Store(&config.Credentials{})

	publisher := stubPublisher{onPublish: func() {
		current.Store(&config.Credentials{AccessToken: "fresh-login-value"}) //nolint:gosec
	}}

	fallbackCalled := make(chan struct{})

	handler := authLossHandler(stubNotifier{}, publisher, fimptype.EaseeService,
		func() config.Credentials { return *current.Load() },
		func() error {
			close(fallbackCalled)

			return nil
		})

	handler("token rejected")

	select {
	case <-fallbackCalled:
		require.Fail(t, "logout fallback ran after a login landed while the logout was being published")
	case <-time.After(200 * time.Millisecond):
	}
}

// A credentials change in the gap between the synchronous snapshot and the goroutine actually
// running must stop the publish too, or the stale cmd.auth.logout message can still reach the
// routed handler and clear the session that replaced the one that triggered this callback.
func TestAuthLossHandler_SkipsPublishWhenCredentialsChangedBeforeGoroutineRuns(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	credentials := func() config.Credentials {
		// First read: the snapshot taken synchronously before the goroutine spawns. Second
		// read: taken from inside the goroutine, simulating a login that landed in between.
		if calls.Add(1) == 1 {
			return config.Credentials{}
		}

		return config.Credentials{AccessToken: "fresh-login-value"} //nolint:gosec
	}

	published := make(chan struct{})

	publisher := stubPublisher{onPublish: func() { close(published) }}

	handler := authLossHandler(stubNotifier{}, publisher, fimptype.EaseeService, credentials, func() error {
		return nil
	})

	handler("token rejected")

	select {
	case <-published:
		require.Fail(t, "logout was published after a fresh login replaced the cleared credentials")
	case <-time.After(200 * time.Millisecond):
	}
}

// A nil fallback must not panic: the handler runs under the authenticator lock.
func TestAuthLossHandler_NilFallback(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		authLossHandler(stubNotifier{}, stubPublisher{}, fimptype.EaseeService, unchangedCredentials, nil)("token rejected")
	})
}
