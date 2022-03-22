package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/user/types"
)

func TestUnstakingUsersAllSpecsQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.UserKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNUnstakingUsersAllSpecs(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetUnstakingUsersAllSpecsRequest
		response *types.QueryGetUnstakingUsersAllSpecsResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetUnstakingUsersAllSpecsRequest{Id: msgs[0].Id},
			response: &types.QueryGetUnstakingUsersAllSpecsResponse{UnstakingUsersAllSpecs: msgs[0]},
		},
		{
			desc:     "Second",
			request:  &types.QueryGetUnstakingUsersAllSpecsRequest{Id: msgs[1].Id},
			response: &types.QueryGetUnstakingUsersAllSpecsResponse{UnstakingUsersAllSpecs: msgs[1]},
		},
		{
			desc:    "KeyNotFound",
			request: &types.QueryGetUnstakingUsersAllSpecsRequest{Id: uint64(len(msgs))},
			err:     sdkerrors.ErrKeyNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.UnstakingUsersAllSpecs(wctx, tc.request)
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

func TestUnstakingUsersAllSpecsQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.UserKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNUnstakingUsersAllSpecs(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllUnstakingUsersAllSpecsRequest {
		return &types.QueryAllUnstakingUsersAllSpecsRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}
	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.UnstakingUsersAllSpecsAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.UnstakingUsersAllSpecs), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.UnstakingUsersAllSpecs),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.UnstakingUsersAllSpecsAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.UnstakingUsersAllSpecs), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.UnstakingUsersAllSpecs),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.UnstakingUsersAllSpecsAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.UnstakingUsersAllSpecs),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.UnstakingUsersAllSpecsAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
