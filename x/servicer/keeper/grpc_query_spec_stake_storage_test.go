package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/servicer/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestSpecStakeStorageQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNSpecStakeStorage(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetSpecStakeStorageRequest
		response *types.QueryGetSpecStakeStorageResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetSpecStakeStorageRequest{
				Index: msgs[0].Index,
			},
			response: &types.QueryGetSpecStakeStorageResponse{SpecStakeStorage: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetSpecStakeStorageRequest{
				Index: msgs[1].Index,
			},
			response: &types.QueryGetSpecStakeStorageResponse{SpecStakeStorage: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetSpecStakeStorageRequest{
				Index: strconv.Itoa(100000),
			},
			err: status.Error(codes.InvalidArgument, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.SpecStakeStorage(wctx, tc.request)
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

func TestSpecStakeStorageQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNSpecStakeStorage(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllSpecStakeStorageRequest {
		return &types.QueryAllSpecStakeStorageRequest{
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
			resp, err := keeper.SpecStakeStorageAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.SpecStakeStorage), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.SpecStakeStorage),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.SpecStakeStorageAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.SpecStakeStorage), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.SpecStakeStorage),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.SpecStakeStorageAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.SpecStakeStorage),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.SpecStakeStorageAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
