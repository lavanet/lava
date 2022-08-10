package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/lavanet/lava/x/spec"
	"github.com/stretchr/testify/require"
)

func SimulateParamChange(ctx sdk.Context, paramKeeper paramskeeper.Keeper, subspace string, key string, value string) (err error) {
	proposal := &paramproposal.ParameterChangeProposal{Changes: []paramproposal.ParamChange{{Subspace: subspace, Key: key, Value: value}}}
	err = spec.HandleParameterChangeProposal(ctx, paramKeeper, proposal)
	return
}

func TestServicersToPair(t *testing.T) {
	_, keepers, ctx := keepertest.InitAllKeepers(t)

	//init keepers state
	keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ctx), *types.DefaultGenesis().EpochDetails)

	blocksInEpoch := keepers.Epochstorage.EpochBlocksRaw(sdk.UnwrapSDKContext(ctx))
	epochsMemory := keepers.Epochstorage.EpochsToSaveRaw(sdk.UnwrapSDKContext(ctx))
	blocksInMemory := blocksInEpoch * epochsMemory

	servicersToPairInitial, err := keepers.Pairing.GetFixatedServicersToPairForBlock(sdk.UnwrapSDKContext(ctx), uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
	servicersToParCount := servicersToPairInitial.ServicersToPairCount
	require.Nil(t, err)

	tests := []struct {
		name                    string
		Block                   uint64 //advance test to this block
		ServicersToPair         uint64 //set this if not zero at the start of the test
		ExpectedServicersToPair uint64
		NumOfFixation           uint64 //expected number of fixations in the memory
	}{
		{"FillHalfMemory", blocksInMemory / 2, 0, servicersToParCount, 1},
		{"ParamChange", blocksInMemory / 2, 2 * servicersToParCount, servicersToParCount, 1},
		{"ParamChange + eoch", blocksInMemory/2 + 2*blocksInEpoch, 0, servicersToParCount, 2},
	}

	pastTests := []struct {
		Block                   uint64
		ExpectedServicersToPair uint64
	}{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ServicersToPair != 0 {
				err := SimulateParamChange(sdk.UnwrapSDKContext(ctx), keepers.ParamsKeeper, pairingtypes.ModuleName, string(pairingtypes.KeyServicersToPairCount), "\""+strconv.FormatUint(tt.ServicersToPair, 10)+"\"")
				require.NoError(t, err)
			}

			ctx = keepertest.AdvanceToBlock(ctx, keepers, tt.Block)

			require.Equal(t, tt.Block, uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
			servicersToPair, err := keepers.Pairing.GetFixatedServicersToPairForBlock(sdk.UnwrapSDKContext(ctx), uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
			require.Nil(t, err)
			require.Equal(t, tt.ExpectedServicersToPair, servicersToPair.ServicersToPairCount)
			allFixatedParams := keepers.Epochstorage.GetAllFixatedParams(sdk.UnwrapSDKContext(ctx))
			require.Equal(t, tt.NumOfFixation, uint64(len(allFixatedParams)))

			for _, pasttest := range pastTests {
				servicersToPair, err := keepers.Pairing.GetFixatedServicersToPairForBlock(sdk.UnwrapSDKContext(ctx), pasttest.Block)
				require.Nil(t, err)
				require.Equal(t, pasttest.ExpectedServicersToPair, servicersToPair.ServicersToPairCount)
			}

			pastTests = append(pastTests, struct {
				Block                   uint64
				ExpectedServicersToPair uint64
			}{Block: tt.Block, ExpectedServicersToPair: tt.ExpectedServicersToPair})

		})
	}
}
