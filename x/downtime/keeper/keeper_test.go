package keeper_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/x/downtime/types"
	v1 "github.com/lavanet/lava/x/downtime/v1"
	"github.com/stretchr/testify/require"
)

func TestParams(t *testing.T) {
	app, ctx := app.TestSetup()
	keeper := app.DowntimeKeeper

	keeper.SetParams(ctx, v1.DefaultParams())
	require.Equal(t, v1.DefaultParams(), keeper.GetParams(ctx))
}

func TestLastBlockTime(t *testing.T) {
	app, ctx := app.TestSetup()
	keeper := app.DowntimeKeeper

	// no last block time set
	_, ok := keeper.GetLastBlockTime(ctx)
	require.False(t, ok)

	// set last block time
	keeper.SetLastBlockTime(ctx)

	// last block time set
	lastBlockTime, ok := keeper.GetLastBlockTime(ctx)
	require.True(t, ok)
	require.Equal(t, ctx.BlockTime(), lastBlockTime)
}

func TestDowntime(t *testing.T) {
	app, ctx := app.TestSetup()
	keeper := app.DowntimeKeeper

	// set downtime
	expected := 1 * time.Minute
	keeper.SetDowntime(ctx, 1, expected)
	got, ok := keeper.GetDowntime(ctx, 1)
	require.True(t, ok)
	require.Equal(t, expected, got)

	// if it does not exist then it should return false
	_, ok = keeper.GetDowntime(ctx, 2)
	require.False(t, ok)
}

func TestHadDowntimes(t *testing.T) {
	app, ctx := app.TestSetup()
	keeper := app.DowntimeKeeper

	// no downtime
	has, _ := keeper.HadDowntimeBetween(ctx, 1, 2)
	require.False(t, has)

	// set downtime
	keeper.SetDowntime(ctx, 1, 1*time.Minute)
	// set another downtime
	keeper.SetDowntime(ctx, 2, 1*time.Minute)
	has, duration := keeper.HadDowntimeBetween(ctx, 1, 2)
	require.True(t, has)
	require.Equal(t, 2*time.Minute, duration)

	// test same block
	has, duration = keeper.HadDowntimeBetween(ctx, 1, 1)
	require.True(t, has)
	require.Equal(t, 1*time.Minute, duration)

	// out of range
	has, duration = keeper.HadDowntimeBetween(ctx, 1, 3)
	require.True(t, has)
	require.Equal(t, 2*time.Minute, duration)
}

func TestBeginBlock(t *testing.T) {
	// anonymous function to move into next block with provided duration
	nextBlock := func(ctx sdk.Context, elapsedTime time.Duration) sdk.Context {
		return ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(elapsedTime))
	}

	app, ctx := app.TestSetup()
	ctx = ctx.WithBlockTime(time.Now().UTC()).WithBlockHeight(1)
	keeper := app.DowntimeKeeper
	keeper.SetParams(ctx, v1.DefaultParams())

	// start with no block time recorded as of now
	keeper.BeginBlock(ctx)
	lbt, ok := keeper.GetLastBlockTime(ctx)
	require.True(t, ok)
	require.Equal(t, ctx.BlockTime(), lbt)

	// move into next block
	ctx = nextBlock(ctx, 1*time.Minute)

	// run begin block again to check if block time is updated
	keeper.BeginBlock(ctx)
	lbt, ok = keeper.GetLastBlockTime(ctx)
	require.True(t, ok)
	require.Equal(t, ctx.BlockTime(), lbt)

	// move into next block –– forcing a downtime
	ctx = nextBlock(ctx, keeper.GetParams(ctx).DowntimeDuration)

	// run begin block again to check if downtime is recorded
	keeper.BeginBlock(ctx)
	hadDowntimes, duration := keeper.HadDowntimeBetween(ctx, 0, uint64(ctx.BlockHeight()))
	require.True(t, hadDowntimes)
	require.Equal(t, keeper.GetParams(ctx).DowntimeDuration, duration)

	// now check garbage collection
	// we extend the downtime duration in order not to have another downtime
	// since we're making time elapse by a lot in order to trigger garbage collection!
	gcDuration := keeper.GetParams(ctx).GarbageCollectionDuration
	keeper.SetParams(ctx, v1.Params{DowntimeDuration: gcDuration*2 + keeper.GetParams(ctx).DowntimeDuration, GarbageCollectionDuration: gcDuration})
	ctx = nextBlock(ctx, gcDuration+1*time.Second)
	keeper.BeginBlock(ctx)
	_, ok = keeper.GetDowntime(ctx, uint64(ctx.BlockHeight()-1)) // currHeight-1 because we're checking the downtime of the previous block
	require.False(t, ok)
	// now we check the garbage collection store prefix, which should be empty
	sk := app.GetKey(types.ModuleName)
	store := prefix.NewStore(ctx.KVStore(sk), types.DowntimeHeightGarbageKey)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	require.False(t, iter.Valid())
}
