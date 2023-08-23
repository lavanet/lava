package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/x/downtime/keeper"
	v1 "github.com/lavanet/lava/x/downtime/v1"
	"github.com/stretchr/testify/require"
)

func TestQueryServer_QueryDowntime(t *testing.T) {
	app, ctx := app.TestSetup()
	dk := app.DowntimeKeeper
	qs := keeper.NewQueryServer(dk)

	// set some downtimes
	downtime := &v1.Downtime{
		Block:    1,
		Duration: 50 * time.Minute,
	}

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
