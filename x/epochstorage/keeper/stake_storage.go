package keeper

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/epochstorage/types"
)

// SetStakeStorage set a specific stakeStorage in the store from its index
func (k Keeper) SetStakeStorage(ctx sdk.Context, stakeStorage types.StakeStorage) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StakeStorageKeyPrefix))
	b := k.cdc.MustMarshal(&stakeStorage)
	store.Set(types.StakeStorageKey(
		stakeStorage.Index,
	), b)
}

// GetStakeStorage returns a stakeStorage from its index
func (k Keeper) GetStakeStorage(
	ctx sdk.Context,
	index string,

) (val types.StakeStorage, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StakeStorageKeyPrefix))

	b := store.Get(types.StakeStorageKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveStakeStorage removes a stakeStorage from the store
func (k Keeper) RemoveStakeStorage(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StakeStorageKeyPrefix))
	store.Delete(types.StakeStorageKey(
		index,
	))
}

// GetAllStakeStorage returns all stakeStorage
func (k Keeper) GetAllStakeStorage(ctx sdk.Context) (list []types.StakeStorage) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StakeStorageKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.StakeStorage
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) RemoveOldEpochData(ctx sdk.Context, storageType string) (err error) {

	if uint64(ctx.BlockHeight()) < k.BlocksToSave(ctx) {
		return nil
	}
	block := uint64(ctx.BlockHeight()) - k.BlocksToSave(ctx)
	earliestEpochBlock := k.GetEarliestEpochStart(ctx)
	if earliestEpochBlock > block {
		return nil
	}
	//we passed the distance to earliest session block, so remove the entries and update the earliestSessionBlock
	allChainIDs := k.specKeeper.GetAllChainIDs(ctx)
	for _, chainID := range allChainIDs {
		k.RemoveStakeStorageByBlockAndChain(ctx, storageType, earliestEpochBlock, chainID)
	}
	//TODO: after a long period go over all entries and find leftovers, to make sure edge cases are handled
	return nil
}

func (k Keeper) UpdateEarliestEpochstart(ctx sdk.Context) {
	block := uint64(ctx.BlockHeight()) - k.BlocksToSave(ctx)
	earliestEpochBlock := k.GetEarliestEpochStart(ctx)
	if earliestEpochBlock > block {
		return
	}
	//now update the earliest session start
	epochBlocks := k.GetEpochBlocks(ctx, earliestEpochBlock)
	earliestEpochBlock += epochBlocks
	k.SetEarliestEpochStart(ctx, earliestEpochBlock)
}

func (k Keeper) stakeStorageKey(storageType string, block uint64, chainID string) string {
	return storageType + strconv.FormatUint(block, 10) + chainID
}

func (k Keeper) stakeStorageKeyTemp(storageType string, chainID string) string {
	return storageType + chainID
}

func (k Keeper) RemoveStakeStorageByBlockAndChain(ctx sdk.Context, storageType string, block uint64, chainID string) {
	key := k.stakeStorageKey(storageType, block, chainID)
	k.RemoveStakeStorage(ctx, key)
}

//used to get the latest
func (k Keeper) GetStakeStorageTemp(ctx sdk.Context, storageType string, chainID string) (types.StakeStorage, bool) {
	return k.GetStakeStorage(ctx, k.stakeStorageKeyTemp(storageType, chainID))
}

func (k Keeper) StoreEpochStakeStorage(ctx sdk.Context, block uint64, storageType string) {
	// takes the temporary stake storage and puts it in epoch storage
	allChainIDs := k.specKeeper.GetAllChainIDs(ctx)
	for _, chainID := range allChainIDs {
		tmpStorage, found := k.GetStakeStorageTemp(ctx, storageType, chainID)
		if !found {
			//no storage for this spec yet
			continue
		}
		newStorage := tmpStorage.Copy()
		newStorage.Index = k.stakeStorageKey(storageType, block, chainID)
	}
}
