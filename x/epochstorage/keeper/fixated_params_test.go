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
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNFixatedParams(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.FixatedParams {
	items := make([]types.FixatedParams, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetFixatedParams(ctx, items[i])
	}
	return items
}

func TestFixatedParamsGet(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNFixatedParams(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetFixatedParams(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestFixatedParamsRemove(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNFixatedParams(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveFixatedParams(ctx,
			item.Index,
		)
		_, found := keeper.GetFixatedParams(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestFixatedParamsGetAll(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNFixatedParams(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllFixatedParams(ctx)),
	)
}

func TestParamFixation(t *testing.T) {
	//THIS TEST ASSUMES GENESIS BLOCKS IN EPOCH > 2

	// AdvanceBlock(ctx context.Context, ks *Keepers)

	// servers, keepers, ctx := keepertest.InitAllKeepers(t)
	_, keepers, ctx := keepertest.InitAllKeepers(t)

	//init keepers state

	keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ctx), *types.DefaultGenesis().EpochDetails)

	// ctx = keepertest.AdvanceEpoch(ctx, keepers)
	blocksInEpochInitial := keepers.Epochstorage.EpochBlocksRaw(sdk.UnwrapSDKContext(ctx))
	epochsToSaveInitial := keepers.Epochstorage.EpochsToSaveRaw(sdk.UnwrapSDKContext(ctx))
	tests := []struct {
		name               string
		blocksToUpdate     uint64
		expectedEpochStart uint64
	}{
		{"genesis", 0, 0},
		{"initial", 1, 0},
		{"epoch", blocksInEpochInitial, blocksInEpochInitial},
		{"epoch plus block", 1, blocksInEpochInitial},
		{"entire memory", blocksInEpochInitial * epochsToSaveInitial, blocksInEpochInitial*epochsToSaveInitial + blocksInEpochInitial},
		{"entire memory plus block", 1, blocksInEpochInitial*epochsToSaveInitial + blocksInEpochInitial},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for i := 0; i < int(tt.blocksToUpdate); i++ {
				keepertest.AdvanceBlock(ctx, keepers)
			}
			allFixatedParams := keepers.Epochstorage.GetAllFixatedParams(sdk.UnwrapSDKContext(ctx))
			require.Equal(t, len(allFixatedParams), 1) // no matter how many epochs we want only one fixation since we didnt change the params
			epochStart, _ := keepers.Epochstorage.GetEpochStartForBlock(sdk.UnwrapSDKContext(ctx), uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
			require.Equal(t, epochStart, tt.expectedEpochStart)

		})
	}

}

func TestParamFixationWithEpochChange(t *testing.T) {
	//THIS TEST ASSUMES GENESIS BLOCKS IN EPOCH > 2

	// AdvanceBlock(ctx context.Context, ks *Keepers)

	// servers, keepers, ctx := keepertest.InitAllKeepers(t)
	_, keepers, ctx := keepertest.InitAllKeepers(t)

	//init keepers state

	keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ctx), *types.DefaultGenesis().EpochDetails)

	// ctx = keepertest.AdvanceEpoch(ctx, keepers)
	blocksInEpochInitial := keepers.Epochstorage.EpochBlocksRaw(sdk.UnwrapSDKContext(ctx))
	// epochsToSaveInitial := keepers.Epochstorage.EpochsToSaveRaw(sdk.UnwrapSDKContext(ctx))
	blocksInEpochChange := []uint64{blocksInEpochInitial * 2, blocksInEpochInitial * 3}
	tests := []struct {
		name               string
		blocksToUpdate     uint64
		expectedEpochStart uint64
		epochBlocksValue   uint64
		expectedFixation   uint64
	}{
		{"initial", 1, 0, blocksInEpochInitial, 1},
		{"epoch", blocksInEpochInitial, blocksInEpochInitial, blocksInEpochInitial, 1},
		{"paramChange", blocksInEpochInitial, blocksInEpochInitial, blocksInEpochInitial * 2, 1},
		{"paramChange+block", 1, blocksInEpochInitial, blocksInEpochChange[0], 1},
		{"paramChange+epoch", blocksInEpochInitial, blocksInEpochInitial, blocksInEpochInitial * 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for i := 0; i < int(tt.blocksToUpdate); i++ {
				keepertest.AdvanceBlock(ctx, keepers)
			}
			allFixatedParams := keepers.Epochstorage.GetAllFixatedParams(sdk.UnwrapSDKContext(ctx))
			require.Equal(t, len(allFixatedParams), tt.expectedFixation) // no matter how many epochs we want only one fixation since we didnt change the params
			epochStart, _ := keepers.Epochstorage.GetEpochStartForBlock(sdk.UnwrapSDKContext(ctx), uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
			require.Equal(t, epochStart, tt.expectedEpochStart)
			//TODO: check pas epochs before tt.expectedEpochStart

		})
	}

}
