package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/lavanet/lava/v4/testutil/keeper"
	"github.com/lavanet/lava/v4/testutil/nullify"
	"github.com/lavanet/lava/v4/x/epochstorage/types"
)

func TestEpochDetailsQuery(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	wctx := ctx
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
