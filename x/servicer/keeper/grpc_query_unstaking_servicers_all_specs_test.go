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
	"github.com/lavanet/lava/x/servicer/types"
)

func TestUnstakingServicersAllSpecsQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNUnstakingServicersAllSpecs(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetUnstakingServicersAllSpecsRequest
		response *types.QueryGetUnstakingServicersAllSpecsResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetUnstakingServicersAllSpecsRequest{Id: msgs[0].Id},
			response: &types.QueryGetUnstakingServicersAllSpecsResponse{UnstakingServicersAllSpecs: msgs[0]},
		},
		{
			desc:     "Second",
			request:  &types.QueryGetUnstakingServicersAllSpecsRequest{Id: msgs[1].Id},
			response: &types.QueryGetUnstakingServicersAllSpecsResponse{UnstakingServicersAllSpecs: msgs[1]},
		},
		{
			desc:    "KeyNotFound",
			request: &types.QueryGetUnstakingServicersAllSpecsRequest{Id: uint64(len(msgs))},
			err:     sdkerrors.ErrKeyNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.UnstakingServicersAllSpecs(wctx, tc.request)
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

func TestUnstakingServicersAllSpecsQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNUnstakingServicersAllSpecs(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllUnstakingServicersAllSpecsRequest {
		return &types.QueryAllUnstakingServicersAllSpecsRequest{
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
			resp, err := keeper.UnstakingServicersAllSpecsAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.UnstakingServicersAllSpecs), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.UnstakingServicersAllSpecs),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.UnstakingServicersAllSpecsAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.UnstakingServicersAllSpecs), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.UnstakingServicersAllSpecs),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.UnstakingServicersAllSpecsAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.UnstakingServicersAllSpecs),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.UnstakingServicersAllSpecsAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
