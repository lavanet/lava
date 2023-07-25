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
	downtimes := []*v1.Downtime{
		{
			Block:    1,
			Duration: 2 * time.Second,
		},
		{
			Block:    2,
			Duration: 3 * time.Second,
		},
		{
			Block:    3,
			Duration: 4 * time.Second,
		},
	}
	dk.SetDowntime(ctx, downtimes[0].Block, downtimes[0].Duration)
	dk.SetDowntime(ctx, downtimes[1].Block, downtimes[1].Duration)
	dk.SetDowntime(ctx, downtimes[2].Block, downtimes[2].Duration)

	t.Run("ok", func(t *testing.T) {
		resp, err := qs.QueryDowntime(sdk.WrapSDKContext(ctx), &v1.QueryDowntimeRequest{
			StartBlock: 1,
			EndBlock:   2,
		})
		require.NoError(t, err)
		require.Equal(t, &v1.QueryDowntimeResponse{
			Downtimes:                  downtimes[:2],
			CumulativeDowntimeDuration: 5 * time.Second,
		}, resp)
	})

	t.Run("error - invalid height", func(t *testing.T) {
		resp, err := qs.QueryDowntime(sdk.WrapSDKContext(ctx), &v1.QueryDowntimeRequest{
			StartBlock: 2,
			EndBlock:   1,
		})
		require.ErrorContains(t, err, "start block must be less than or equal to end block")
		require.Nil(t, resp)
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
