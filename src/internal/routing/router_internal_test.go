package routing

import (
	"sync"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"

	mockedadapter "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/adapter"
)

const pauseTopic = "pt:j1/mt:cmd/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1"

type bindings struct {
	calls []string
}

func (b *bindings) Bind(*fimpgo.Message)   { b.calls = append(b.calls, "bind") }
func (b *bindings) Unbind(*fimpgo.Message) { b.calls = append(b.calls, "unbind") }

func TestBindCommand_BindsThenUnbinds(t *testing.T) {
	t.Parallel()

	thing := mockedadapter.NewThing(t)
	thing.On("Address").Return("1")

	ad := mockedadapter.NewAdapter(t)
	ad.On("ThingByTopic", pauseTopic).Return(thing)

	ctrl := &bindings{}
	lookup := func(string) Controller { return ctrl }
	msg := newSetCurrentMessage(0)

	bindCommand(ad, lookup, true)(msg)
	bindCommand(ad, lookup, false)(msg)

	assert.Equal(t, []string{"bind", "unbind"}, ctrl.calls)
}

type heldBinding struct {
	sync.Mutex
}

func (h *heldBinding) Bind(*fimpgo.Message)   { h.Lock() }
func (h *heldBinding) Unbind(*fimpgo.Message) { h.Unlock() }

func TestBindCommand_ReportRequestDuringACommandPassesAtOnce(t *testing.T) {
	t.Parallel()

	thing := mockedadapter.NewThing(t)
	thing.On("Address").Return("1").Maybe()

	ad := mockedadapter.NewAdapter(t)
	ad.On("ThingByTopic", pauseTopic).Return(thing).Maybe()

	ctrl := &heldBinding{}
	lookup := func(string) Controller { return ctrl }
	ctrl.Lock()
	defer ctrl.Unlock()

	report := newSetCurrentMessage(0)
	report.Payload.Interface = chargepoint.CmdStateGetReport
	done := make(chan struct{})

	go func() {
		bindCommand(ad, lookup, true)(report)
		bindCommand(ad, lookup, false)(report)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("report request waited for the running command")
	}
}

func newSetCurrentMessage(current int) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: pauseTopic,
		Payload: &fimpgo.FimpMessage{
			Interface: chargepoint.CmdCurrentSessionSetCurrent,
			Service:   chargepoint.Chargepoint,
			ValueType: fimptype.VTypeInt,
			Value:     current,
		},
	}
}
