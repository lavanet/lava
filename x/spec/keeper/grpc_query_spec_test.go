package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/testutil/nullify"
	"github.com/lavanet/lava/v2/x/spec/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestSpecQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.SpecKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNSpec(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetSpecRequest
		response *types.QueryGetSpecResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetSpecRequest{
				ChainID: msgs[0].Index,
			},
			response: &types.QueryGetSpecResponse{Spec: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetSpecRequest{
				ChainID: msgs[1].Index,
			},
			response: &types.QueryGetSpecResponse{Spec: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetSpecRequest{
				ChainID: strconv.Itoa(100000),
			},
			err: status.Error(codes.InvalidArgument, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.Spec(wctx, tc.request)
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

func TestSpecQuerySingleRaw(t *testing.T) {
	keeper, ctx := keepertest.SpecKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNSpec(keeper, ctx, 2)

	msgs[0].ApiCollections = []*types.ApiCollection{{CollectionData: types.CollectionData{ApiInterface: "stub"}}}
	msgs[1].ApiCollections = []*types.ApiCollection{{CollectionData: types.CollectionData{ApiInterface: "stub"}}}
	msgs[0].ApiCollections[0].Apis = []*types.Api{{Name: "api-0", Enabled: true}}

	msgs[1].ApiCollections[0].Apis = []*types.Api{{Name: "api-1", Enabled: true}}
	msgs[1].Imports = []string{"api-0"}

	keeper.SetSpec(ctx, msgs[0])
	keeper.SetSpec(ctx, msgs[1])

	request := &types.QueryGetSpecRequest{ChainID: msgs[1].Index}
	expected := &types.QueryGetSpecResponse{Spec: msgs[1]}

	t.Run("Raw with import", func(t *testing.T) {
		response, err := keeper.SpecRaw(wctx, request)
		require.NoError(t, err)
		require.Equal(t,
			nullify.Fill(expected),
			nullify.Fill(response),
		)
	})
}

func TestSpecQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.SpecKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNSpec(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllSpecRequest {
		return &types.QueryAllSpecRequest{
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
			resp, err := keeper.SpecAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Spec), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Spec),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.SpecAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Spec), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Spec),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.SpecAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.Spec),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.SpecAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
