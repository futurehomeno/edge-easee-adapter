package routing

import (
	"testing"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	mockedchargepoint "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/chargepoint"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"

	mockedadapter "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/adapter"
)

const pauseTopic = "pt:j1/mt:cmd/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1"

type offeredCurrentFn func(int) error

func (f offeredCurrentFn) SetChargepointOfferedCurrent(current int) error { return f(current) }

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

func TestDivertOfferedPause_UnderSixAmpsReachesTheControllerAndRenames(t *testing.T) {
	t.Parallel()

	for _, current := range []int{0, 3, 5} {
		thing := mockedadapter.NewThing(t)
		thing.On("Address").Return("1")
		thing.On("ServiceByTopic", pauseTopic).Return(nil)

		ad := mockedadapter.NewAdapter(t)
		ad.On("ThingByTopic", pauseTopic).Return(thing)

		got := -1
		msg := newSetCurrentMessage(current)

		assert.NoError(t, divertOfferedPause(ad, func(address string) chargepoint.AdjustableOfferedCurrentController {
			assert.Equal(t, "1", address)

			return offeredCurrentFn(func(c int) error { got = c; return nil })
		}, msg))

		assert.Equal(t, current, got)
		assert.Equal(t, chargepoint.CmdCurrentSessionSetCurrent+".paused", msg.Payload.Interface)
	}
}

func TestDivertOfferedPause_LeavesOtherValuesToTheFramework(t *testing.T) {
	t.Parallel()

	for _, current := range []int{-1, 6, 16} {
		msg := newSetCurrentMessage(current)

		assert.NoError(t, divertOfferedPause(mockedadapter.NewAdapter(t), func(string) chargepoint.AdjustableOfferedCurrentController {
			t.Fatalf("lookup must not run for %d A", current)

			return nil
		}, msg))

		assert.Equal(t, chargepoint.CmdCurrentSessionSetCurrent, msg.Payload.Interface)
	}
}

func TestDivertOfferedPause_FailedPauseKeepsTheMessage(t *testing.T) {
	t.Parallel()

	thing := mockedadapter.NewThing(t)
	thing.On("Address").Return("1")

	ad := mockedadapter.NewAdapter(t)
	ad.On("ThingByTopic", pauseTopic).Return(thing)

	msg := newSetCurrentMessage(0)

	assert.NoError(t, divertOfferedPause(ad, func(string) chargepoint.AdjustableOfferedCurrentController {
		return offeredCurrentFn(func(int) error { return assert.AnError })
	}, msg))

	assert.Equal(t, chargepoint.CmdCurrentSessionSetCurrent, msg.Payload.Interface)
}

func TestDivertOfferedPause_FailedReportIsReturned(t *testing.T) {
	t.Parallel()

	service := mockedchargepoint.NewService(t)
	service.On("SendCurrentSessionReport", true).Return(false, assert.AnError)

	thing := mockedadapter.NewThing(t)
	thing.On("Address").Return("1")
	thing.On("ServiceByTopic", pauseTopic).Return(service)

	ad := mockedadapter.NewAdapter(t)
	ad.On("ThingByTopic", pauseTopic).Return(thing)

	msg := newSetCurrentMessage(0)

	err := divertOfferedPause(ad, func(string) chargepoint.AdjustableOfferedCurrentController {
		return offeredCurrentFn(func(int) error { return nil })
	}, msg)

	assert.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, chargepoint.CmdCurrentSessionSetCurrent+".paused", msg.Payload.Interface)
}
