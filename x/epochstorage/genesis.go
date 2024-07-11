package epochstorage

import (
	"strconv"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/epochstorage/keeper"
	"github.com/lavanet/lava/x/epochstorage/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the stakeStorage
	for _, elem := range genState.StakeStorageList {
		for _, entry := range elem.StakeEntries {
			if elem.Index != "" {
				epoch, err := strconv.ParseUint(elem.Index, 10, 64)
				if err != nil {
					panic(errors.Wrapf(err, "failed InitGenesis of epochstorage"))
				}
				k.SetStakeEntry(ctx, epoch, entry)
			} else {
				k.SetStakeEntryCurrent(ctx, entry)
			}
		}
	}

	// Set if defined
	if genState.EpochDetails != nil {
		k.SetEpochDetails(ctx, *genState.EpochDetails)
	}
	// Set all the fixatedParams
	for _, elem := range genState.FixatedParamsList {
		k.SetFixatedParams(ctx, elem)
	}
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	// get all stake entries
	genesis.StakeStorageList = k.GetAllStakeEntriesForGenesis(ctx)

	// Get all epochDetails
	epochDetails, found := k.GetEpochDetails(ctx)
	if found {
		genesis.EpochDetails = &epochDetails
	}
	genesis.FixatedParamsList = k.GetAllFixatedParams(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
