package keeper_test

import (
	"encoding/binary"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/app"
	"github.com/lavanet/lava/v2/x/downtime/keeper"
	v1 "github.com/lavanet/lava/v2/x/downtime/v1"
	"github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

func TestQueryServer_QueryDowntime(t *testing.T) {
	app, ctx := app.TestSetup()
	dk := app.DowntimeKeeper
	qs := keeper.NewQueryServer(dk)

	// set some downtimes
	downtime := &v1.Downtime{
		Block:    0,
		Duration: 50 * time.Minute,
	}

	app.EpochstorageKeeper.SetEpochDetails(ctx, types.EpochDetails{
		StartBlock:    uint64(ctx.BlockHeight()),
		EarliestStart: uint64(ctx.BlockHeight()),
		DeletedEpochs: []uint64{},
	})
	app.EpochstorageKeeper.SetParams(ctx, types.DefaultParams())

	raw := make([]byte, 8)
	binary.LittleEndian.PutUint64(raw, 20)
	app.EpochstorageKeeper.SetFixatedParams(ctx, types.FixatedParams{
		Index:         string(types.KeyEpochBlocks) + "0",
		Parameter:     raw,
		FixationBlock: uint64(ctx.BlockHeight()),
	})
	dk.SetDowntime(ctx, downtime.Block, downtime.Duration)

	t.Run("ok", func(t *testing.T) {
		resp, err := qs.QueryDowntime(sdk.WrapSDKContext(ctx), &v1.QueryDowntimeRequest{
			EpochStartBlock: uint64(1),
		})
		require.NoError(t, err)
		require.Equal(t, &v1.QueryDowntimeResponse{
			CumulativeDowntimeDuration: downtime.Duration,
		}, resp)
	})
}

func TestQueryServer_QueryParams(t *testing.T) {
	app, ctx := app.TestSetup()
	dk := app.DowntimeKeeper
	qs := keeper.NewQueryServer(dk)

	wantParams := v1.DefaultParams()
	dk.SetParams(ctx, wantParams)

	resp, err := qs.QueryParams(sdk.WrapSDKContext(ctx), &v1.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &v1.QueryParamsResponse{Params: &wantParams}, resp)
}
