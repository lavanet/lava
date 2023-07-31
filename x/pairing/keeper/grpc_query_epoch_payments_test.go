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
	"github.com/lavanet/lava/x/pairing/types"
)

var _ = strconv.IntSize

func TestEpochPaymentsQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNEpochPayments(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetEpochPaymentsRequest
		response *types.QueryGetEpochPaymentsResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetEpochPaymentsRequest{
				Index: msgs[0].Index,
			},
			response: &types.QueryGetEpochPaymentsResponse{EpochPayments: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetEpochPaymentsRequest{
				Index: msgs[1].Index,
			},
			response: &types.QueryGetEpochPaymentsResponse{EpochPayments: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetEpochPaymentsRequest{
				Index: strconv.Itoa(100000),
			},
			err: status.Error(codes.NotFound, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.EpochPayments(wctx, tc.request)
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

func TestEpochPaymentsQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNEpochPayments(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllEpochPaymentsRequest {
		return &types.QueryAllEpochPaymentsRequest{
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
			resp, err := keeper.EpochPaymentsAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.EpochPayments), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.EpochPayments),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.EpochPaymentsAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.EpochPayments), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.EpochPayments),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.EpochPaymentsAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.EpochPayments),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.EpochPaymentsAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
