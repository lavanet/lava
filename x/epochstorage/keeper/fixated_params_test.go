package keeper_test

import (
	"fmt"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/epochstorage/keeper"
	"github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/spec"
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
				ctx = keepertest.AdvanceBlock(ctx, keepers)
			}
			allFixatedParams := keepers.Epochstorage.GetAllFixatedParams(sdk.UnwrapSDKContext(ctx))
			require.Equal(t, len(allFixatedParams), 1) // no matter how many epochs we want only one fixation since we didnt change the params
			epochStart, _, err := keepers.Epochstorage.GetEpochStartForBlock(sdk.UnwrapSDKContext(ctx), uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
			require.NoError(t, err)
			require.Equal(t, tt.expectedEpochStart, epochStart)

		})
	}

}

func TestParamFixationWithEpochBlocksChange(t *testing.T) {
	//THIS TEST ASSUMES GENESIS BLOCKS IN EPOCH > 2

	// AdvanceBlock(ctx context.Context, ks *Keepers)

	// servers, keepers, ctx := keepertest.InitAllKeepers(t)
	_, keepers, ctx := keepertest.InitAllKeepers(t)

	//init keepers state

	keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ctx), *types.DefaultGenesis().EpochDetails)

	// ctx = keepertest.AdvanceEpoch(ctx, keepers)
	blocksInEpochInitial := keepers.Epochstorage.EpochBlocksRaw(sdk.UnwrapSDKContext(ctx))
	epochsMemory_initial := keepers.Epochstorage.EpochsToSaveRaw(sdk.UnwrapSDKContext(ctx))
	newEpochBlocksValues := []uint64{17, 30, 15, 10, 11, 10}
	type EpochCompare struct {
		Block       uint64
		Epoch       uint64
		EpochBlocks uint64
	}
	// epochsToSaveInitial := keepers.Epochstorage.EpochsToSaveRaw(sdk.UnwrapSDKContext(ctx))
	wanted_epoch_change_details := []EpochCompare{
		{1, 0, 0},
		{blocksInEpochInitial + 1, blocksInEpochInitial, 0},
		{2*blocksInEpochInitial + 1, 2 * blocksInEpochInitial, newEpochBlocksValues[0]}, // make a param change, doesn't fixate yet
		{2*blocksInEpochInitial + 2, 2 * blocksInEpochInitial, 0},                       //fixation wasn't reached
		{3*blocksInEpochInitial + 2, 3 * blocksInEpochInitial, 0},
		{3*blocksInEpochInitial + newEpochBlocksValues[0] + 1, 3*blocksInEpochInitial + newEpochBlocksValues[0], 0},
		{3*blocksInEpochInitial + 2*newEpochBlocksValues[0] + 2, 3*blocksInEpochInitial + 2*newEpochBlocksValues[0], 0},
		{3*blocksInEpochInitial + 7*newEpochBlocksValues[0] + 2, 3*blocksInEpochInitial + 7*newEpochBlocksValues[0], 0},
		{3*blocksInEpochInitial + (7+epochsMemory_initial)*newEpochBlocksValues[0] + 1, 3*blocksInEpochInitial + (7+epochsMemory_initial)*newEpochBlocksValues[0], 0},
		{3*blocksInEpochInitial + (7+epochsMemory_initial)*newEpochBlocksValues[0] + 2, 3*blocksInEpochInitial + (7+epochsMemory_initial)*newEpochBlocksValues[0], newEpochBlocksValues[1]},
		{3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0], 3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0], 0},
		{3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0] + 1, 3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0], newEpochBlocksValues[2]},
		{3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0] + newEpochBlocksValues[1] + 3, 3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0] + newEpochBlocksValues[1], 0},
		{3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0] + newEpochBlocksValues[1] + newEpochBlocksValues[2] + 3, 3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0] + newEpochBlocksValues[1] + newEpochBlocksValues[2], 0},
		{3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0] + newEpochBlocksValues[1] + (epochsMemory_initial+20)*newEpochBlocksValues[2] + 5, 3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0] + newEpochBlocksValues[1] + (epochsMemory_initial+20)*newEpochBlocksValues[2], 0},
	}

	tests := []struct {
		name               string
		EpochChangeDetails EpochCompare
		expectedFixation   uint64
	}{
		{"[00]initial", wanted_epoch_change_details[0], 1},
		{"[01]epoch", wanted_epoch_change_details[1], 1},
		{"[02]paramChange", wanted_epoch_change_details[2], 1},
		{"[03]paramChange+block", wanted_epoch_change_details[3], 1},      //fixation wasn't reached
		{"[04]paramChange+epoch", wanted_epoch_change_details[4], 2},      //now its fixated
		{"[05]+newEpoch", wanted_epoch_change_details[5], 2},              //
		{"[06]+newEpoch+block", wanted_epoch_change_details[6], 2},        //
		{"[07]+5 * newEpoch", wanted_epoch_change_details[7], 2},          //
		{"[08]+memory end * newEpoch", wanted_epoch_change_details[8], 1}, // memory end passed
		{"[09]another param change", wanted_epoch_change_details[9], 1},   // fixation wasn't reached
		{"[10]paramChange+epoch", wanted_epoch_change_details[10], 2},     // now its fixated
		{"[11]+block", wanted_epoch_change_details[11], 2},                // another param change
		{"[12]+epoch", wanted_epoch_change_details[12], 3},                // another fixated
		{"[13]+new epoch", wanted_epoch_change_details[13], 3},            //
		{"[14]+memory end", wanted_epoch_change_details[14], 1},           // memory end passed
	}
	prevBlock := 0
	newEpochBlocksVal := blocksInEpochInitial
	expectedEpochBlocks := newEpochBlocksVal

	pastEpochsToCompare := []EpochCompare{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			blocksToLoop := int(tt.EpochChangeDetails.Block) - prevBlock
			for i := 0; i < blocksToLoop; i++ {
				ctx = keepertest.AdvanceBlock(ctx, keepers)
				if keepers.Epochstorage.IsEpochStart(sdk.UnwrapSDKContext(ctx)) {
					expectedEpochBlocks = newEpochBlocksVal
				}
				prevBlock++
			}
			earliestEpochStart := keepers.Epochstorage.GetEarliestEpochStart(sdk.UnwrapSDKContext(ctx))

			//check epoch grid is correct
			currBlock := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
			epochStart, _, err := keepers.Epochstorage.GetEpochStartForBlock(sdk.UnwrapSDKContext(ctx), currBlock)

			require.NoError(t, err)
			epochBlocks, err := keepers.Epochstorage.EpochBlocks(sdk.UnwrapSDKContext(ctx), currBlock)
			require.NoError(t, err)

			require.Equal(t, expectedEpochBlocks, epochBlocks)

			fmt.Printf("Tests for current block: %d, with epochBlocks %d\n", prevBlock, epochBlocks)
			require.Equal(t, tt.EpochChangeDetails.Epoch, epochStart, "GetEpochStartForBlock VS expectedEpochStart")

			//check the amount of fixations
			allFixatedParams := keepers.Epochstorage.GetAllFixatedParams(sdk.UnwrapSDKContext(ctx))
			require.Equal(t, tt.expectedFixation, uint64(len(allFixatedParams)), fmt.Sprintf("FixatedParamsLength VS expectedFixationLength \nEarliestEpoch start: %d\n%+v", earliestEpochStart, allFixatedParams)) // no matter how many epochs we want only one fixation since we didnt change the params

			latestParamChange, found := keepers.Epochstorage.LatestFixatedParams(sdk.UnwrapSDKContext(ctx), string(types.KeyEpochBlocks))
			require.True(t, found)

			for _, epochComapre := range pastEpochsToCompare {
				//test past grid
				epochStart, _, err := keepers.Epochstorage.GetEpochStartForBlock(sdk.UnwrapSDKContext(ctx), epochComapre.Block)

				if epochComapre.Block >= earliestEpochStart || (latestParamChange.FixationBlock < epochComapre.Block) {
					require.NoError(t, err)
					require.Equal(t, epochComapre.Epoch, epochStart, "pastEpochsToCompare: GetEpochStartForBlock VS expectedEpochStart")
				} else {
					if err == nil {
						fixation, err := keepers.Epochstorage.GetFixatedParamsForBlock(sdk.UnwrapSDKContext(ctx), string(types.KeyEpochBlocks), epochComapre.Block)
						require.NoError(t, err)
						require.True(t, fixation.FixationBlock <= epochComapre.Block)
					}
					// require.Error(t, err, fmt.Sprintf("expected error but did not receive: epochComapre.Block: %d earliestEpochStart:%d, fixations: %+v", epochComapre.Block, earliestEpochStart, allFixatedParams))
				}

				//test for past Epoch Blocks
				epochBlocks_test, err := keepers.Epochstorage.EpochBlocks(sdk.UnwrapSDKContext(ctx), epochComapre.Block)
				if epochComapre.Block >= earliestEpochStart || (latestParamChange.FixationBlock < epochComapre.Block) {
					require.NoError(t, err)
					require.Equal(t, epochComapre.EpochBlocks, epochBlocks_test)
				} else {
					if err == nil {
						fixation, err := keepers.Epochstorage.GetFixatedParamsForBlock(sdk.UnwrapSDKContext(ctx), string(types.KeyEpochBlocks), epochComapre.Block)
						require.NoError(t, err)
						require.True(t, fixation.FixationBlock <= epochComapre.Block)
					}
				}

			}

			//add the current block to blocks we compare, future tests will need to check this
			pastEpochsToCompare = append(pastEpochsToCompare, EpochCompare{Block: currBlock, Epoch: epochStart, EpochBlocks: epochBlocks})

			if tt.EpochChangeDetails.EpochBlocks != 0 {
				require.NotEqual(t, tt.EpochChangeDetails.EpochBlocks, newEpochBlocksVal)
				newEpochBlocksVal = tt.EpochChangeDetails.EpochBlocks
				err := SimulateParamChange(sdk.UnwrapSDKContext(ctx), keepers.ParamsKeeper, types.ModuleName, "EpochBlocks", "\""+strconv.FormatUint(newEpochBlocksVal, 10)+"\"")
				require.NoError(t, err)
			}

		})
	}

}

func SimulateParamChange(ctx sdk.Context, paramKeeper paramskeeper.Keeper, subspace string, key string, value string) (err error) {
	proposal := &paramproposal.ParameterChangeProposal{Changes: []paramproposal.ParamChange{{Subspace: subspace, Key: key, Value: value}}}
	err = spec.HandleParameterChangeProposal(ctx, paramKeeper, proposal)
	return
}
