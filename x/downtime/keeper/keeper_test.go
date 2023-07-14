package keeper_test

import (
	"testing"
	"time"

	"github.com/lavanet/lava/app"
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
