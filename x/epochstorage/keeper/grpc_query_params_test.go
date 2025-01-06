package keeper_test

import (
	"testing"

	testkeeper "github.com/lavanet/lava/v4/testutil/keeper"
	"github.com/lavanet/lava/v4/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := testkeeper.EpochstorageKeeper(t)
	wctx := ctx
	params := types.DefaultParams()
	keeper.SetParams(ctx, params)

	response, err := keeper.Params(wctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
