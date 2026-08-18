package api

import (
	"errors"
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

type stubPublisher struct{ err error }

func (p stubPublisher) PublishToTopic(string, *fimpgo.FimpMessage) error { return p.err }

// TestAuthLossHandler_RunsLocalLogout pins the recovery path: the framework has already cleared
// the credentials, so a logout that never reaches the app leaves the SignalR client connected
// and the lifecycle claiming a session until the next restart. A successful publish is not proof
// the logout ran - the routed handler drops the command when it loses its try-lock - so the
// fallback has to run on both outcomes.
func TestAuthLossHandler_RunsLocalLogout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		publishErr error
	}{
		{name: "broker refuses the publish", publishErr: errors.New("publish failed")},
		{name: "publish succeeds but the routed handler may drop it", publishErr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := make(chan struct{})

			handler := authLossHandler(stubNotifier{}, stubPublisher{err: tt.publishErr}, fimptype.EaseeService, func() error {
				close(called)

				return errors.New("logout failed")
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

// A nil fallback must not panic: the handler runs under the authenticator lock.
func TestAuthLossHandler_NilFallback(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		authLossHandler(stubNotifier{}, stubPublisher{}, fimptype.EaseeService, nil)("token rejected")
	})
}
