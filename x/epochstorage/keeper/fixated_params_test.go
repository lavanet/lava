package keeper_test

import (
	"fmt"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	common "github.com/lavanet/lava/v2/testutil/common"
	keepertest "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/testutil/nullify"
	"github.com/lavanet/lava/v2/x/epochstorage/keeper"
	"github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

type tester struct {
	common.Tester
}

func newTester(t *testing.T) *tester {
	// note: use NewTesterRaw() to avoid the default AdvanceEpoch(), because hereinafter
	// the tests expect precise control over block advancement.
	ts := &tester{Tester: *common.NewTesterRaw(t)}
	return ts
}

func (ts *tester) createNFixatedParams(n int) []types.FixatedParams {
	return createNFixatedParams(&ts.Keepers.Epochstorage, ts.Ctx, n)
}

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
	// THIS TEST ASSUMES GENESIS BLOCKS IN EPOCH > 2

	ts := newTester(t)

	blocksInEpochInitial := ts.EpochBlocks()
	epochsToSaveInitial := ts.EpochsToSave()

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
			ts.AdvanceBlocks(tt.blocksToUpdate)
			allFixatedParams := ts.Keepers.Epochstorage.GetAllFixatedParams(ts.Ctx)
			require.Equal(t, len(ts.Keepers.Epochstorage.GetFixationRegistries()), len(allFixatedParams)) // no matter how many epochs we want only one fixation since we didnt change the params
			epochStart, _, err := ts.Keepers.Epochstorage.GetEpochStartForBlock(ts.Ctx, ts.BlockHeight())
			require.NoError(t, err)
			require.Equal(t, tt.expectedEpochStart, epochStart)
		})
	}
}

func TestParamFixationWithEpochBlocksChange(t *testing.T) {
	// THIS TEST ASSUMES GENESIS BLOCKS IN EPOCH > 2

	ts := newTester(t)

	blocksInEpochInitial := ts.EpochBlocks()
	epochsMemory_initial := ts.EpochsToSave()

	newEpochBlocksValues := []uint64{17, 30, 15, 10, 11, 10}
	type EpochCompare struct {
		Block       uint64 // advance test to this block
		Epoch       uint64 // expected epoch for the test
		EpochBlocks uint64 // expected epochBlocks for the test
	}

	wanted_epoch_change_details := []EpochCompare{
		/*00*/ {1, 0, 0},
		/*01*/ {blocksInEpochInitial + 1, blocksInEpochInitial, 0},
		/*02*/ {2*blocksInEpochInitial + 1, 2 * blocksInEpochInitial, newEpochBlocksValues[0]}, // make a param change, doesn't fixate yet
		/*03*/ {2*blocksInEpochInitial + 2, 2 * blocksInEpochInitial, 0}, // fixation wasn't reached
		/*04*/ {3*blocksInEpochInitial + 2, 3 * blocksInEpochInitial, 0},
		/*05*/ {3*blocksInEpochInitial + newEpochBlocksValues[0] + 1, 3*blocksInEpochInitial + newEpochBlocksValues[0], 0},
		/*06*/ {3*blocksInEpochInitial + 2*newEpochBlocksValues[0] + 2, 3*blocksInEpochInitial + 2*newEpochBlocksValues[0], 0},
		/*07*/ {3*blocksInEpochInitial + 7*newEpochBlocksValues[0] + 2, 3*blocksInEpochInitial + 7*newEpochBlocksValues[0], 0},
		/*08*/ {3*blocksInEpochInitial + (7+epochsMemory_initial)*newEpochBlocksValues[0] + 1, 3*blocksInEpochInitial + (7+epochsMemory_initial)*newEpochBlocksValues[0], 0},
		/*09*/ {3*blocksInEpochInitial + (7+epochsMemory_initial)*newEpochBlocksValues[0] + 2, 3*blocksInEpochInitial + (7+epochsMemory_initial)*newEpochBlocksValues[0], newEpochBlocksValues[1]},
		/*10*/ {3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0], 3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0], 0},
		/*11*/ {3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0] + 1, 3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0], newEpochBlocksValues[2]},
		/*12*/ {3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0] + newEpochBlocksValues[1] + 3, 3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0] + newEpochBlocksValues[1], 0},
		/*13*/ {3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0] + newEpochBlocksValues[1] + newEpochBlocksValues[2] + 3, 3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0] + newEpochBlocksValues[1] + newEpochBlocksValues[2], 0},
		/*14*/ {3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0] + newEpochBlocksValues[1] + (epochsMemory_initial+20)*newEpochBlocksValues[2] + 5, 3*blocksInEpochInitial + (8+epochsMemory_initial)*newEpochBlocksValues[0] + newEpochBlocksValues[1] + (epochsMemory_initial+20)*newEpochBlocksValues[2], 0},
	}
	last_epoch := wanted_epoch_change_details[len(wanted_epoch_change_details)-1].Epoch
	wanted_epoch_change_details = append(wanted_epoch_change_details, []EpochCompare{
		/*15*/ {last_epoch + 10, last_epoch, 0},
		/*16*/ {last_epoch + newEpochBlocksValues[2] + 3, last_epoch + newEpochBlocksValues[2], 0},
		/*17*/ {last_epoch + 2*newEpochBlocksValues[2], last_epoch + 2*newEpochBlocksValues[2], 0},
		/*18*/ {last_epoch + 3*newEpochBlocksValues[2] - 1, last_epoch + 2*newEpochBlocksValues[2], 0},
	}...)

	last_epoch = wanted_epoch_change_details[len(wanted_epoch_change_details)-1].Epoch
	wanted_epoch_change_details = append(wanted_epoch_change_details, []EpochCompare{
		/*19*/ {last_epoch, last_epoch, newEpochBlocksValues[3]}, // add param
		/*20*/ {last_epoch + newEpochBlocksValues[2] + 1, last_epoch + newEpochBlocksValues[2], 0},
		/*21*/ {last_epoch + newEpochBlocksValues[2] + epochsMemory_initial*newEpochBlocksValues[3] - 1, last_epoch + newEpochBlocksValues[2] + (epochsMemory_initial-1)*newEpochBlocksValues[3], 0},
		/*22*/ {last_epoch + newEpochBlocksValues[2] + epochsMemory_initial*newEpochBlocksValues[3], last_epoch + newEpochBlocksValues[2] + (epochsMemory_initial)*newEpochBlocksValues[3], 0},
		/*23*/ {last_epoch + newEpochBlocksValues[2] + epochsMemory_initial*newEpochBlocksValues[3] + 1, last_epoch + newEpochBlocksValues[2] + (epochsMemory_initial)*newEpochBlocksValues[3], 0},
		/*24*/ {last_epoch + newEpochBlocksValues[2] + (epochsMemory_initial+9)*newEpochBlocksValues[3], last_epoch + newEpochBlocksValues[2] + (epochsMemory_initial+9)*newEpochBlocksValues[3], 0},
		/*25*/ {last_epoch + newEpochBlocksValues[2] + (epochsMemory_initial+10)*newEpochBlocksValues[3], last_epoch + newEpochBlocksValues[2] + (epochsMemory_initial+10)*newEpochBlocksValues[3], 0},
	}...)

	tests := []struct {
		name             string
		expectedFixation int
	}{
		{"[00]initial", 1},
		{"[01]epoch", 1},
		{"[02]paramChange", 1},
		{"[03]paramChange+block", 1},      // fixation wasn't reached
		{"[04]paramChange+epoch", 2},      // now its fixated
		{"[05]+newEpoch", 2},              //
		{"[06]+newEpoch+block", 2},        //
		{"[07]+5 * newEpoch", 2},          //
		{"[08]+memory end * newEpoch", 1}, // memory end passed
		{"[09]another param change", 1},   // fixation wasn't reached
		{"[10]paramChange+epoch", 2},      // now its fixated
		{"[11]+block", 2},                 // another param change
		{"[12]+epoch", 3},                 // another fixated
		{"[13]+new epoch", 3},             //
		{"[14]+memory end", 1},            // memory end passed
		{"[15]memory end + 10", 1},
		{"[16]memory end + epoch", 1},
		{"[17]memory end + 2epochs", 1},
		{"[18]memory end + 3epochs -block", 1},
		{"[19]param change", 1},
		{"[20]fixate param change", 2},
		{"[21]end memory -1", 2},           // before memory end
		{"[22]end memory", 2},              // at memory end
		{"[23]end memory +1", 2},           // after memory end +1
		{"[24]end memory +fixation -1", 1}, // after memory end + diff fixation -1
		{"[25]end memory +fixation", 1},    // after memory end + diff fixation
	}
	prevBlock := 0
	newEpochBlocksVal := blocksInEpochInitial
	expectedEpochBlocks := newEpochBlocksVal

	pastEpochsToCompare := []EpochCompare{}

	for ti, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocksToLoop := int(wanted_epoch_change_details[ti].Block) - prevBlock
			for i := 0; i < blocksToLoop; i++ {
				ts.AdvanceBlock()
				if ts.Keepers.Epochstorage.IsEpochStart(ts.Ctx) {
					expectedEpochBlocks = newEpochBlocksVal
				}
				prevBlock++
			}
			earliestEpochStart := ts.Keepers.Epochstorage.GetEarliestEpochStart(ts.Ctx)

			// check epoch grid is correct
			currBlock := ts.BlockHeight()
			epochStart, _, err := ts.Keepers.Epochstorage.GetEpochStartForBlock(ts.Ctx, currBlock)

			require.NoError(t, err)
			epochBlocks, err := ts.Keepers.Epochstorage.EpochBlocks(ts.Ctx, currBlock)
			require.NoError(t, err)

			require.Equal(t, expectedEpochBlocks, epochBlocks)

			fmt.Printf("Tests for current block: %d, with epochBlocks %d\n", prevBlock, epochBlocks)
			require.Equal(t, wanted_epoch_change_details[ti].Epoch, epochStart, "GetEpochStartForBlock VS expectedEpochStart")

			// check the amount of fixations
			allFixatedParams := ts.Keepers.Epochstorage.GetAllFixatedParams(ts.Ctx)
			require.Equal(t, len(ts.Keepers.Epochstorage.GetFixationRegistries())+tt.expectedFixation-1, len(allFixatedParams), fmt.Sprintf("FixatedParamsLength VS expectedFixationLength \nEarliestEpoch start: %d\n%+v", earliestEpochStart, allFixatedParams)) // no matter how many epochs we want only one fixation since we didnt change the params

			_, found := ts.Keepers.Epochstorage.LatestFixatedParams(ts.Ctx, string(types.KeyEpochBlocks))
			require.True(t, found)

			for _, epochComapre := range pastEpochsToCompare {
				// test past grid
				epochStart, _, errEpochStart := ts.Keepers.Epochstorage.GetEpochStartForBlock(ts.Ctx, epochComapre.Block)
				epochBlocks_test, errEpochBlocks := ts.Keepers.Epochstorage.EpochBlocks(ts.Ctx, epochComapre.Block)
				if epochComapre.Block >= earliestEpochStart {
					require.NoError(t, errEpochStart)
					require.Equal(t, epochComapre.Epoch, epochStart, "pastEpochsToCompare: GetEpochStartForBlock VS expectedEpochStart")

					require.NoError(t, errEpochBlocks)
					require.Equal(t, epochComapre.EpochBlocks, epochBlocks_test)
				} else if errEpochBlocks == nil || errEpochStart == nil {
					fixation, err := ts.Keepers.Epochstorage.GetFixatedParamsForBlock(ts.Ctx, string(types.KeyEpochBlocks), epochComapre.Block)

					require.NoError(t, err)
					require.True(t, fixation.FixationBlock <= epochComapre.Block)
				}
				// require.Error(t, err, fmt.Sprintf("expected error but did not receive: epochComapre.Block: %d earliestEpochStart:%d, fixations: %+v", epochComapre.Block, earliestEpochStart, allFixatedParams))
			}

			// add the current block to blocks we compare, future tests will need to check this
			pastEpochsToCompare = append(pastEpochsToCompare, EpochCompare{Block: currBlock, Epoch: epochStart, EpochBlocks: epochBlocks})

			if wanted_epoch_change_details[ti].EpochBlocks != 0 {
				require.NotEqual(t, wanted_epoch_change_details[ti].EpochBlocks, newEpochBlocksVal)
				newEpochBlocksVal = wanted_epoch_change_details[ti].EpochBlocks
				err := ts.TxProposalChangeParam(types.ModuleName, "EpochBlocks", "\""+strconv.FormatUint(newEpochBlocksVal, 10)+"\"")
				require.NoError(t, err)
			}
		})
	}
}

func TestParamFixationWithEpochToSaveChange(t *testing.T) {
	ts := newTester(t)

	blocksInEpochInitial := ts.EpochBlocks()
	epochsMemory_initial := ts.EpochsToSave()

	tests := []struct {
		name          string
		Block         uint64 // advance test to this block
		EarliestEpoch uint64 // expected earliest epoch for the test
		EpochsToSave  uint64 // set this if not zero at the start of the test
		MumOfFixation int    // expected number of fixations in the memory
	}{
		{"FillHalfMemory", epochsMemory_initial * blocksInEpochInitial / 2, 0, 0, 1},
		{"FixateNewParam", (epochsMemory_initial/2 + 1) * blocksInEpochInitial, 0, epochsMemory_initial / 2, 2},
		{"FillMemory", epochsMemory_initial * blocksInEpochInitial, 0, 0, 2},
		{"FillMemory+epoch", (epochsMemory_initial + 1) * blocksInEpochInitial, (epochsMemory_initial + 1 - epochsMemory_initial) * blocksInEpochInitial, 0, 2},
		{"MemoryLengthChange", (epochsMemory_initial + epochsMemory_initial/2 + 2) * blocksInEpochInitial, (epochsMemory_initial + epochsMemory_initial/2 + 2 - epochsMemory_initial/2) * blocksInEpochInitial, 0, 1},
		{"FutureTest", (epochsMemory_initial * 2) * blocksInEpochInitial, (epochsMemory_initial*2 - epochsMemory_initial/2) * blocksInEpochInitial, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.EpochsToSave != 0 {
				err := ts.TxProposalChangeParam(types.ModuleName, "EpochsToSave", "\""+strconv.FormatUint(tt.EpochsToSave, 10)+"\"")
				require.NoError(t, err)
			}

			ts.AdvanceToBlock(tt.Block)
			require.Equal(t, tt.Block, ts.BlockHeight())
			require.Equal(t, tt.EarliestEpoch, ts.Keepers.Epochstorage.GetEarliestEpochStart(ts.Ctx))
			allFixatedParams := ts.Keepers.Epochstorage.GetAllFixatedParams(ts.Ctx)
			require.Equal(t, len(ts.Keepers.Epochstorage.GetFixationRegistries())-1+tt.MumOfFixation, len(allFixatedParams))
		})
	}
}
