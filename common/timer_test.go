package common

import (
	"math"
	"strconv"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func initCtxAndTimerStores(t *testing.T, count int) (sdk.Context, []*TimerStore) {
	ctx, cdc := initCtx(t)
	tstore := make([]*TimerStore, count)

	for i := 0; i < count; i++ {
		timerKey := "mock_timer_" + strconv.Itoa(i)
		tstore[i] = NewTimerStore(mockStoreKey, cdc, timerKey)
	}

	return ctx, tstore
}

func initCtxAndTimerStore(t *testing.T) (sdk.Context, *TimerStore) {
	ctx, tstore := initCtxAndTimerStores(t, 1)
	return ctx, tstore[0]
}

type timerTemplate struct {
	op    string
	name  string
	store int
	data  string
	value uint64
	fire  int
}

// helper to automate testing operations
func testWithTimerTemplate(t *testing.T, playbook []timerTemplate, countTS int) {
	ctx, tstore := initCtxAndTimerStores(t, countTS)

	var (
		callbackBlockCount int
		callbackBlockData  string
		callbackTimeCount  int
		callbackTimeData   string
	)

	callbackHeight := func(ctx sdk.Context, data string) {
		callbackBlockCount += 1
		callbackBlockData += data
	}

	callbackTime := func(ctx sdk.Context, data string) {
		callbackTimeCount += 1
		callbackTimeData += data
	}

	for i := range tstore {
		tstore[i] = tstore[i].
			WithCallbackByBlockHeight(callbackHeight).
			WithCallbackByBlockTime(callbackTime)
	}

	for _, play := range playbook {
		what := play.op + " " + play.name +
			" value: " + strconv.Itoa(int(play.value)) +
			" data: " + play.data
		switch play.op {
		case "addheight":
			tstore[play.store].AddTimerByBlockHeight(ctx, play.value, play.data)
		case "addtime":
			tstore[play.store].AddTimerByBlockTime(ctx, play.value, play.data)
		case "nextheight":
			value := tstore[play.store].GetNextTimeoutBlockHeight(ctx)
			require.Equal(t, play.value, value, what)
		case "nexttime":
			value := tstore[play.store].GetNextTimeoutBlockTime(ctx)
			require.Equal(t, play.value, value, what)
		case "tickheight":
			callbackBlockCount = 0
			callbackBlockData = ""
			ctx = ctx.WithBlockHeight(int64(play.value))
			tstore[play.store].Tick(ctx)
			require.Equal(t, play.fire, callbackBlockCount, what)
			require.Equal(t, play.data, callbackBlockData, what)
		case "ticktime":
			callbackTimeCount = 0
			callbackTimeData = ""
			ctx = ctx.WithBlockTime(time.Unix(int64(play.value), 0))
			tstore[play.store].Tick(ctx)
			require.Equal(t, play.fire, callbackTimeCount, what)
			require.Equal(t, play.data, callbackTimeData, what)
		}
	}
}

// Test single timer by block height
func TestTimerBlockHeight(t *testing.T) {
	playbook := []timerTemplate{
		{op: "nextheight", name: "next timeout infinity", value: math.MaxInt64},
		{op: "tickheight", name: "tick without timers", value: 100, fire: 0},
		{op: "addheight", name: "add timer no-1", value: 120, data: "no-1."},
		{op: "nextheight", name: "next timeout no-1", value: 120},
		{op: "tickheight", name: "tick before timer no-1", value: 110, fire: 0},
		{op: "tickheight", name: "tick after timer no-1", value: 130, fire: 1, data: "no-1."},
		{op: "nextheight", name: "next timeout no-1", value: math.MaxInt64},
		{op: "addheight", name: "add timer no-2", value: 140, data: "no-2."},
		{op: "tickheight", name: "tick exactly on timer no-2", value: 140, fire: 1, data: "no-2."},
		{op: "nextheight", name: "next timeout infinity again", value: math.MaxInt64},
	}

	testWithTimerTemplate(t, playbook, 1)
}

// Test single timer by block time
func TestTimerBlockTime(t *testing.T) {
	playbook := []timerTemplate{
		{op: "nexttime", name: "next timeout infinity", value: math.MaxInt64},
		{op: "ticktime", name: "tick without timers", value: 100, fire: 0},
		{op: "addtime", name: "add timer no-1", value: 120, data: "no-1."},
		{op: "nexttime", name: "next timeout no-1", value: 120},
		{op: "ticktime", name: "tick before timer no-1", value: 110, fire: 0},
		{op: "ticktime", name: "tick after timer no-1", value: 130, fire: 1, data: "no-1."},
	}

	testWithTimerTemplate(t, playbook, 1)
}

// Test new timer earlier than next timeout
func TestTimerEarlierThenNext(t *testing.T) {
	playbook := []timerTemplate{
		{op: "tickheight", name: "tick without timers", value: 100, fire: 0},
		{op: "addheight", name: "add timer no 1", value: 120, data: "no-1."},
		{op: "nextheight", name: "next timeout no-1", value: 120},
		{op: "addheight", name: "add timer no 2 (as first)", value: 110, data: "no-2."},
		{op: "nextheight", name: "next timeout no-2", value: 110},
		{op: "tickheight", name: "tick before all", value: 105, fire: 0},
		{op: "tickheight", name: "tick between no-2, no-1", value: 115, fire: 1, data: "no-2."},
		{op: "nextheight", name: "next timeout no-1 again", value: 120},
		{op: "tickheight", name: "tick after no-1", value: 125, fire: 1, data: "no-1."},
	}

	testWithTimerTemplate(t, playbook, 1)
}

// Test multiple timers (by block height)
func TestMultipleTimers(t *testing.T) {
	playbook := []timerTemplate{
		{op: "tickheight", name: "tick without timers", value: 100, fire: 0},
		{op: "addheight", name: "add timer no 1", value: 120, data: "no-1."},
		{op: "addheight", name: "add timer no 2", value: 130, data: "no-2."},
		{op: "addheight", name: "add timer no 3", value: 140, data: "no-3."},
		{op: "addheight", name: "add timer no 4", value: 150, data: "no-4."},
		{op: "tickheight", name: "tick before all", value: 110, fire: 0},
		{op: "tickheight", name: "tick between no-2, no-3", value: 135, fire: 2, data: "no-1.no-2."},
		{op: "nextheight", name: "next timeout no-3", value: 140},
		{op: "tickheight", name: "tick after all", value: 155, fire: 2, data: "no-3.no-4."},
	}

	testWithTimerTemplate(t, playbook, 1)
}
