package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/epochstorage/types"
)

// Function that calls all the functions that are supposed to run in epoch start
func (k Keeper) EpochStart(ctx sdk.Context) {
	block := uint64(ctx.BlockHeight())
	k.SetEpochHash(ctx)
	// save params for this epoch
	k.FixateParams(ctx, block)

	// on Epoch start we need to do:
	// 1. update Epoch start
	// 2. update the StakeStorage
	// on epoch start block end: (because other modules need this info) to clear their storages
	// 3. remove old StakeStorage
	// 4. update earliest epoch start

	k.SetEpochDetailsStart(ctx, block)

	k.StoreCurrentStakeEntries(ctx, block)

	k.UpdateEarliestEpochstart(ctx)

	k.RemoveOldEpochData(ctx)
}

// StoreCurrentStakeEntries store the current stake entries in the epoch-prefixed stake entries store
func (k Keeper) StoreCurrentStakeEntries(ctx sdk.Context, epoch uint64) {
	entries := k.GetAllStakeEntriesCurrent(ctx)
	for _, entry := range entries {
		k.SetStakeEntry(ctx, epoch, entry)
	}
}

func (k *Keeper) UpdateEarliestEpochstart(ctx sdk.Context) {
	currentBlock := uint64(ctx.BlockHeight())
	earliestEpochBlock := k.GetEarliestEpochStart(ctx)

	// we take the epochs memory size at earliestEpochBlock, and not the current one
	blocksToSaveAtEarliestEpoch, err := k.BlocksToSave(ctx, earliestEpochBlock)
	if err != nil {
		// panic:ok: critical, no recovery, avoid further corruption
		utils.LavaFormatPanic("critical: failed to advance EarliestEpochstart", err,
			utils.LogAttr("earliestEpochBlock", earliestEpochBlock),
			utils.LogAttr("fixations", k.GetAllFixatedParams(ctx)),
		)
	}

	if currentBlock <= blocksToSaveAtEarliestEpoch {
		return
	}

	lastBlockInMemory := currentBlock - blocksToSaveAtEarliestEpoch

	deletedEpochs := []uint64{}
	for earliestEpochBlock < lastBlockInMemory {
		deletedEpochs = append(deletedEpochs, earliestEpochBlock)
		earliestEpochBlock, err = k.GetNextEpoch(ctx, earliestEpochBlock)
		if err != nil {
			// panic:ok: critical, no recovery, avoid further corruption
			utils.LavaFormatPanic("critical: failed to advance EarliestEpochstart", err,
				utils.LogAttr("earliestEpochBlock", earliestEpochBlock),
				utils.LogAttr("fixations", k.GetAllFixatedParams(ctx)),
			)
		}
	}

	if len(deletedEpochs) == 0 {
		return
	}

	utils.LogLavaEvent(ctx, k.Logger(ctx), types.EarliestEpochEventName,
		map[string]string{"block": strconv.FormatUint(earliestEpochBlock, 10)},
		"updated earliest epoch block")

	// now update the earliest epoch start
	k.SetEarliestEpochStart(ctx, earliestEpochBlock, deletedEpochs)
}

func (k Keeper) RemoveOldEpochData(ctx sdk.Context) {
	for _, epoch := range k.GetDeletedEpochs(ctx) {
		k.RemoveEpochHash(ctx, epoch)
		k.RemoveAllStakeEntriesForEpoch(ctx, epoch)
	}
}

func (k Keeper) SetEpochHash(ctx sdk.Context) {
	err := k.epochHashes.Set(ctx, uint64(ctx.BlockHeight()), ctx.HeaderHash())
	if err != nil {
		panic(err)
	}
}

func (k Keeper) GetEpochHash(ctx sdk.Context, epoch uint64) []byte {
	hash, err := k.epochHashes.Get(ctx, epoch)
	if err != nil {
		utils.LavaFormatError("GetEpochHash: epoch hash not found", fmt.Errorf("not found"),
			utils.LogAttr("epoch", epoch),
			utils.LogAttr("current_block", ctx.BlockHeight()),
		)
		return []byte{}
	}

	return hash
}

func (k Keeper) RemoveEpochHash(ctx sdk.Context, epoch uint64) {
	err := k.epochHashes.Remove(ctx, epoch)
	if err != nil {
		panic(err)
	}
}
