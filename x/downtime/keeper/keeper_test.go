package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/app"
	v1 "github.com/lavanet/lava/x/downtime/v1"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
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
	keeper.SetLastBlockTime(ctx, ctx.BlockTime())

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

func TestBeginBlock(t *testing.T) {
	app, ctx := app.TestSetup()
	ctx = ctx.WithBlockTime(time.Now().UTC()).WithBlockHeight(0)

	epochParams := epochstoragetypes.DefaultParams()
	epochParams.EpochsToSave = 1

	app.EpochstorageKeeper.SetParams(ctx, epochParams)
	app.EpochstorageKeeper.PushFixatedParams(ctx, 0, 0)
	app.EpochstorageKeeper.SetEpochDetails(ctx, *epochstoragetypes.DefaultGenesis().EpochDetails)

	keeper := app.DowntimeKeeper
	keeper.SetParams(ctx, v1.DefaultParams())

	app.EpochstorageKeeper.BeginBlock(ctx)

	// anonymous function to move into next block with provided duration
	nextBlock := func(ctx sdk.Context, elapsedTime time.Duration) sdk.Context {
		app.EpochstorageKeeper.BeginBlock(ctx)
		return ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(elapsedTime))
	}

	epochStartBlock := func(ctx sdk.Context) uint64 {
		return app.EpochstorageKeeper.GetEpochStart(ctx)
	}

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

	keeper.BeginBlock(ctx)
	duration, hadDowntimes := keeper.GetDowntime(ctx, epochStartBlock(ctx))
	require.True(t, hadDowntimes)
	require.Equal(t, keeper.GetParams(ctx).DowntimeDuration, duration)

	// move into next block, downtime must not increase
	ctx = nextBlock(ctx, 1*time.Second)
	keeper.BeginBlock(ctx)
	duration, hadDowntimes = keeper.GetDowntime(ctx, epochStartBlock(ctx))
	require.True(t, hadDowntimes)
	require.Equal(t, keeper.GetParams(ctx).DowntimeDuration, duration)

	// move into next block, accumulating more downtime
	ctx = nextBlock(ctx, keeper.GetParams(ctx).DowntimeDuration)
	keeper.BeginBlock(ctx)

	duration, hadDowntimes = keeper.GetDowntime(ctx, epochStartBlock(ctx))
	require.True(t, hadDowntimes)
	require.Equal(t, 2*keeper.GetParams(ctx).DowntimeDuration, duration)

	// check garbage collection was done, after forcing an epoch to pass
	for !app.EpochstorageKeeper.IsEpochStart(ctx) {
		ctx = nextBlock(ctx, 1*time.Second)
	}
	keeper.BeginBlock(ctx)
	_, hadDowntime := keeper.GetDowntime(ctx, 0)
	require.False(t, hadDowntime)
}

func TestImportExportGenesis(t *testing.T) {
	app, ctx := app.TestSetup()
	keeper := app.DowntimeKeeper
	ctx = ctx.WithBlockTime(time.Now().UTC()).WithBlockHeight(1)

	t.Run("import export – no last block time", func(t *testing.T) {
		ctx, _ = ctx.CacheContext()
		wantGs := &v1.GenesisState{
			Params: v1.DefaultParams(),
			Downtimes: []*v1.Downtime{
				{
					Block:    1,
					Duration: 50 * time.Minute,
				},
			},
			LastBlockTime: nil,
		}
		err := keeper.ImportGenesis(ctx, wantGs)
		require.NoError(t, err)
		// we don't want to see LastBlockTime to be set
		_, ok := keeper.GetLastBlockTime(ctx)
		require.False(t, ok)
		// we check that if we export we have the same genesis state
		gotGs, err := keeper.ExportGenesis(ctx)
		require.NoError(t, err)
		require.Equal(t, wantGs, gotGs)
	})

	t.Run("import export – last block time", func(t *testing.T) {
		ctx, _ = ctx.CacheContext()
		wantLastBlockTime := ctx.BlockTime().Add(-1 * time.Hour)
		err := keeper.ImportGenesis(ctx, &v1.GenesisState{
			Params:        v1.DefaultParams(),
			LastBlockTime: &wantLastBlockTime,
		})
		require.NoError(t, err)

		gotLastBlockTime, ok := keeper.GetLastBlockTime(ctx)
		require.True(t, ok)
		require.Equal(t, wantLastBlockTime, gotLastBlockTime)
	})
}
