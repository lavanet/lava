package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/epochstorage/keeper"
	"github.com/lavanet/lava/x/epochstorage/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNStakeStorage(k *keeper.Keeper, ctx sdk.Context, n int) []types.StakeStorage {
	items := []types.StakeStorage{}
	stakeEntries := createNStakeEntriesCurrent(k, ctx, n)
	for _, entry := range stakeEntries {
		items = append(items, types.StakeStorage{
			Index:        entry.Address,
			StakeEntries: []types.StakeEntry{entry},
		})
	}
	return items
}

func TestStakeStorageQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNStakeStorage(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetStakeStorageRequest
		response *types.QueryGetStakeStorageResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetStakeStorageRequest{
				Index: msgs[0].Index,
			},
			response: &types.QueryGetStakeStorageResponse{StakeStorage: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetStakeStorageRequest{
				Index: msgs[1].Index,
			},
			response: &types.QueryGetStakeStorageResponse{StakeStorage: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetStakeStorageRequest{
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
			response, err := keeper.StakeStorage(wctx, tc.request)
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
