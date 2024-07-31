package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/testutil/nullify"
	"github.com/lavanet/lava/v2/x/epochstorage/types"
)

func TestEpochDetailsQuery(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	item := createTestEpochDetails(keeper, ctx)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetEpochDetailsRequest
		response *types.QueryGetEpochDetailsResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetEpochDetailsRequest{},
			response: &types.QueryGetEpochDetailsResponse{EpochDetails: item},
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.EpochDetails(wctx, tc.request)
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
