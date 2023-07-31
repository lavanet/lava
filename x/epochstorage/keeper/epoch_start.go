package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Function that calls all the functions that are supposed to run in epoch start
func (k Keeper) EpochStart(ctx sdk.Context) {
	block := uint64(ctx.BlockHeight())

	// save params for this epoch
	k.FixateParams(ctx, block)

	// on Epoch start we need to do:
	// 1. update Epoch start
	// 2. update the StakeStorage
	// on epoch start block end: (because other modules need this info) to clear their storages
	// 3. remove old StakeStorage
	// 4. update earliest epoch start

	k.SetEpochDetailsStart(ctx, block)

	k.StoreCurrentEpochStakeStorage(ctx, block)

	k.UpdateEarliestEpochstart(ctx)

	k.RemoveOldEpochData(ctx)
}
