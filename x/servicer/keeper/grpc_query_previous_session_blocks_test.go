package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/servicer/types"
)

func TestPreviousSessionBlocksQuery(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	item := createTestPreviousSessionBlocks(keeper, ctx)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetPreviousSessionBlocksRequest
		response *types.QueryGetPreviousSessionBlocksResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetPreviousSessionBlocksRequest{},
			response: &types.QueryGetPreviousSessionBlocksResponse{PreviousSessionBlocks: item},
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.PreviousSessionBlocks(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t,
					nullify.Fill(tc.response),
					nullify.Fill(response),
				)
			}
		})
	}
}
