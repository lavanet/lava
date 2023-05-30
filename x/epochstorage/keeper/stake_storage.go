package keeper

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
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

func (k Keeper) RemoveOldEpochData(ctx sdk.Context) {
	for _, block := range k.GetDeletedEpochs(ctx) {
		allChainIDs := k.specKeeper.GetAllChainIDs(ctx)
		for _, chainID := range allChainIDs {
			k.RemoveStakeStorageByBlockAndChain(ctx, block, chainID)
		}
	}
}

func (k *Keeper) UpdateEarliestEpochstart(ctx sdk.Context) {
	currentBlock := uint64(ctx.BlockHeight())
	earliestEpochBlock := k.GetEarliestEpochStart(ctx)
	blocksToSaveAtEarliestEpoch, err := k.BlocksToSave(ctx, earliestEpochBlock) // we take the epochs memory size at earliestEpochBlock, and not the current one
	deletedEpochs := []uint64{}
	if err != nil {
		// this is critical, no recovery from this
		panic(fmt.Sprintf("Critical Error: could not progress EarliestEpochstart %s\nearliestEpochBlock: %d, fixations: %+v", err, earliestEpochBlock, k.GetAllFixatedParams(ctx)))
	}
	if currentBlock <= blocksToSaveAtEarliestEpoch {
		return
	}
	lastBlockInMemory := currentBlock - blocksToSaveAtEarliestEpoch
	changed := false
	for earliestEpochBlock < lastBlockInMemory {
		deletedEpochs = append(deletedEpochs, earliestEpochBlock)
		earliestEpochBlock, err = k.GetNextEpoch(ctx, earliestEpochBlock)
		if err != nil {
			// this is critical, no recovery from this
			panic(fmt.Sprintf("Critical Error: could not progress EarliestEpochstart %s", err))
		}
		changed = true
	}

	if !changed {
		return
	}

	logger := k.Logger(ctx)
	// now update the earliest epoch start
	utils.LogLavaEvent(ctx, logger, types.EarliestEpochEventName, map[string]string{"block": strconv.FormatUint(earliestEpochBlock, 10)}, "updated earliest epoch block")
	k.SetEarliestEpochStart(ctx, earliestEpochBlock, deletedEpochs)
}

func (k Keeper) StakeStorageKey(block uint64, chainID string) string {
	return strconv.FormatUint(block, 10) + chainID
}

func (k Keeper) removeAllEntriesPriorToBlockNumber(ctx sdk.Context, block uint64, allChainID []string) {
	allStorage := k.GetAllStakeStorage(ctx)
	for _, chainId := range allChainID {
		for _, entry := range allStorage {
			if strings.Contains(entry.Index, chainId) {
				if (len(chainId)) > len(entry.Index) {
					panic(fmt.Sprintf("storageType + chainId length out of range %d vs %d\n more info: entry.Index: %s, chainId: %s", len(chainId), len(entry.Index), entry.Index, chainId))
				}
				storageBlock := entry.Index[:(len(entry.Index) - len(chainId))]
				blockHeight, err := strconv.ParseUint(storageBlock, 10, 64)
				if err != nil {
					if storageBlock == "" {
						// if storageBlock is empty its stake entry current. so we dont remove it.
						continue
					}
					panic("failed to convert storage block to int: " + storageBlock)
				}
				if blockHeight < block {
					k.RemoveStakeStorage(ctx, entry.Index)
				}
			}
		}
	}
}

func (k Keeper) RemoveAllEntriesPriorToBlockNumber(ctx sdk.Context, block uint64, allChainID []string) {
	k.removeAllEntriesPriorToBlockNumber(ctx, block, allChainID)
}

func (k Keeper) RemoveStakeStorageByBlockAndChain(ctx sdk.Context, block uint64, chainID string) {
	key := k.StakeStorageKey(block, chainID)
	k.RemoveStakeStorage(ctx, key)
}

// -------------------------------------------------- current staking list --------------------------------------------

func (k Keeper) stakeStorageKeyCurrent(chainID string) string {
	return chainID
}

// used to get the latest
func (k Keeper) GetStakeStorageCurrent(ctx sdk.Context, chainID string) (types.StakeStorage, bool) {
	return k.GetStakeStorage(ctx, k.stakeStorageKeyCurrent(chainID))
}

func (k Keeper) SetStakeStorageCurrent(ctx sdk.Context, chainID string, stakeStorage types.StakeStorage) {
	stakeStorage.Index = k.stakeStorageKeyCurrent(chainID)
	k.SetStakeStorage(ctx, stakeStorage)
}

func (k Keeper) stakeEntryIndexByAddress(ctx sdk.Context, stakeStorage types.StakeStorage, address sdk.AccAddress) (index uint64, found bool) {
	// the following finds the address of stakeEntry and returns it
	entries := stakeStorage.StakeEntries
	for idx, entry := range entries {
		entryAddr, err := sdk.AccAddressFromBech32(entry.Address)
		if err != nil {
			panic("invalid account address inside StakeStorage: " + entry.Address)
		}
		if entryAddr.Equals(address) {
			// found the right thing
			index = uint64(idx)
			found = true
			// remove from the stakeStorage, i checked it supports idx == length-1
			return
		}
	}
	return 0, false
}

func (k Keeper) GetStakeEntryByAddressFromStorage(ctx sdk.Context, stakeStorage types.StakeStorage, address sdk.AccAddress) (value types.StakeEntry, found bool, index uint64) {
	idx, found := k.stakeEntryIndexByAddress(ctx, stakeStorage, address)
	if !found {
		return types.StakeEntry{}, false, 0
	}
	// found the right thing
	value = stakeStorage.StakeEntries[idx]
	found = true
	index = idx
	return
}

func (k Keeper) GetStakeEntryByAddressCurrent(ctx sdk.Context, chainID string, address sdk.AccAddress) (value types.StakeEntry, found bool, index uint64) {
	stakeStorage, found := k.GetStakeStorageCurrent(ctx, chainID)
	if !found {
		return types.StakeEntry{}, false, 0
	}
	// the following finds the address of stakeEntry and returns it
	idx, found := k.stakeEntryIndexByAddress(ctx, stakeStorage, address)
	if !found {
		return types.StakeEntry{}, false, 0
	}
	// found the right thing
	value = stakeStorage.StakeEntries[idx]
	found = true
	index = idx
	return
}

func (k Keeper) RemoveStakeEntryCurrent(ctx sdk.Context, chainID string, idx uint64) error {
	stakeStorage, found := k.GetStakeStorageCurrent(ctx, chainID)
	if !found {
		return errors.ErrNotFound
	}
	if idx >= uint64(len(stakeStorage.StakeEntries)) {
		return errors.ErrNotFound
	}
	stakeStorage.StakeEntries = append(stakeStorage.StakeEntries[:idx], stakeStorage.StakeEntries[idx+1:]...)
	k.SetStakeStorageCurrent(ctx, chainID, stakeStorage)
	return nil
}

func (k Keeper) AppendStakeEntryCurrent(ctx sdk.Context, chainID string, stakeEntry types.StakeEntry) {
	// this stake storage entries are sorted by stake amount
	stakeStorage, found := k.GetStakeStorageCurrent(ctx, chainID)
	var entries []types.StakeEntry
	if !found {
		entries = []types.StakeEntry{stakeEntry}
		// create a new one
		stakeStorage = types.StakeStorage{Index: k.stakeStorageKeyCurrent(chainID), StakeEntries: entries, EpochBlockHash: nil}
	} else {
		// the following code inserts stakeEntry into the existing entries by stake
		entries = stakeStorage.StakeEntries
		// sort func needs to return true if the inserted entry is less than the existing entry
		sortFunc := func(i int) bool {
			return stakeEntry.Stake.Amount.LT(entries[i].Stake.Amount)
		}
		// returns the smallest index in which the sort func is true
		index := sort.Search(len(entries), sortFunc)
		if index < len(entries) {
			entries = append(entries[:index+1], entries[index:]...)
			entries[index] = stakeEntry
		} else {
			// put in the end
			entries = append(entries, stakeEntry)
		}
	}
	stakeStorage.StakeEntries = entries
	k.SetStakeStorageCurrent(ctx, chainID, stakeStorage)
}

func (k Keeper) ModifyStakeEntryCurrent(ctx sdk.Context, chainID string, stakeEntry types.StakeEntry, removeIndex uint64) {
	// this stake storage entries are sorted by stake amount
	stakeStorage, found := k.GetStakeStorageCurrent(ctx, chainID)
	if !found {
		panic("called modify when there is no stakeStorage")
	}
	// TODO: more efficient: only create a new list once, after the second index is identified
	// remove the given index, then store the new entry in the sorted list at the right place
	entries := []types.StakeEntry{}
	entries = append(entries, stakeStorage.StakeEntries[:removeIndex]...)
	entries = append(entries, stakeStorage.StakeEntries[removeIndex+1:]...)
	// the following code inserts stakeEntry into the existing entries by stake
	// sort func needs to return true if the inserted entry is less than the existing entry
	sortFunc := func(i int) bool {
		return stakeEntry.Stake.Amount.LT(entries[i].Stake.Amount)
	}
	// returns the smallest index in which the sort func is true
	index := sort.Search(len(entries), sortFunc)
	if index < len(entries) {
		entries = append(entries[:index+1], entries[index:]...)
		entries[index] = stakeEntry
	} else {
		entries = append(entries, stakeEntry)
	}
	stakeStorage.StakeEntries = entries
	k.SetStakeStorageCurrent(ctx, chainID, stakeStorage)
}

// -------------------------------------------------- unstaking list --------------------------------------------

// used to get the unstaking entries
func (k Keeper) GetStakeStorageUnstake(ctx sdk.Context) (types.StakeStorage, bool) {
	return k.GetStakeStorage(ctx, types.StakeStorageKeyUnstakeConst)
}

func (k Keeper) SetStakeStorageUnstake(ctx sdk.Context, stakeStorage types.StakeStorage) {
	stakeStorage.Index = types.StakeStorageKeyUnstakeConst
	k.SetStakeStorage(ctx, stakeStorage)
}

func (k Keeper) UnstakeEntryByAddress(ctx sdk.Context, address sdk.AccAddress) (value types.StakeEntry, found bool, index uint64) {
	stakeStorage, found := k.GetStakeStorageUnstake(ctx)
	if !found {
		return types.StakeEntry{}, false, 0
	}
	// the following finds the address of stakeEntry and returns it
	idx, found := k.stakeEntryIndexByAddress(ctx, stakeStorage, address)
	if !found {
		return types.StakeEntry{}, false, 0
	}
	// found the right thing
	value = stakeStorage.StakeEntries[idx]
	found = true
	index = idx
	return
}

func (k Keeper) ModifyUnstakeEntry(ctx sdk.Context, stakeEntry types.StakeEntry, removeIndex uint64) {
	// this stake storage entries are sorted by stake amount
	stakeStorage, found := k.GetStakeStorageUnstake(ctx)
	if !found {
		panic("called modify when there is no stakeStorage")
	}
	// TODO: more efficient: only create a new list once, after the second index is identified
	// remove the given index, then store the new entry in the sorted list at the right place
	entries := []types.StakeEntry{}
	entries = append(entries, stakeStorage.StakeEntries[:removeIndex]...)
	entries = append(entries, stakeStorage.StakeEntries[removeIndex+1:]...)
	// the following code inserts stakeEntry into the existing entries by stake
	// sort func needs to return true if the inserted entry is less than the existing entry
	sortFunc := func(i int) bool {
		return stakeEntry.StakeAppliedBlock <= entries[i].StakeAppliedBlock
	}
	// returns the smallest index in which the sort func is true
	index := sort.Search(len(entries), sortFunc)
	if index < len(entries) {
		entries = append(entries[:index+1], entries[index:]...)
		entries[index] = stakeEntry
	} else {
		entries = append(entries, stakeEntry)
	}
	stakeStorage.StakeEntries = entries
	k.SetStakeStorageUnstake(ctx, stakeStorage)
}

func (k Keeper) AppendUnstakeEntry(ctx sdk.Context, stakeEntry types.StakeEntry, unstakeHoldBlocks uint64) error {
	// update unstake stakeAppliedBlock to the higher among params (unstakeholdblocks and blockstosave)

	blockHeight := uint64(ctx.BlockHeight())

	stakeEntry.StakeAppliedBlock = blockHeight + unstakeHoldBlocks

	// this stake storage entries are sorted by stakeAppliedBlock
	stakeStorage, found := k.GetStakeStorageUnstake(ctx)
	var entries []types.StakeEntry
	if !found {
		entries = []types.StakeEntry{stakeEntry}
		// create a new one
		stakeStorage = types.StakeStorage{Index: types.StakeStorageKeyUnstakeConst, StakeEntries: entries, EpochBlockHash: nil}
	} else {
		// the following code inserts stakeEntry into the existing entries by stakeAppliedBlock
		entries = stakeStorage.StakeEntries
		// sort func needs to return true if the inserted entry is less than the existing entry
		sortFunc := func(i int) bool {
			return stakeEntry.StakeAppliedBlock <= entries[i].StakeAppliedBlock
		}
		// returns the smallest index in which the sort func is true
		index := sort.Search(len(entries), sortFunc)
		if index < len(entries) {
			entries = append(entries[:index+1], entries[index:]...)
			entries[index] = stakeEntry
		} else {
			// put in the end
			entries = append(entries, stakeEntry)
		}
	}
	stakeStorage.StakeEntries = entries
	k.SetStakeStorageUnstake(ctx, stakeStorage)

	return nil
}

// Returns the unstaking Entry if its stakeAppliedBlock is lower than the provided block
func (k Keeper) PopUnstakeEntries(ctx sdk.Context, block uint64) (value []types.StakeEntry) {
	stakeStorage, found := k.GetStakeStorageUnstake(ctx)
	if !found {
		// utils.LavaError(ctx, k.Logger(ctx), "emptyStakeStorage", map[string]string{"storageType": storageType}, "stakeStorageUnstake Empty!")
		return nil
	}
	found_idx := -1
	// the unstaking is a sorted list so just chekcing until an entry stakeAppliedBlock is too big
	for idx, entry := range stakeStorage.StakeEntries {
		if entry.StakeAppliedBlock <= block {
			// found an enrty that its stakeAppliedBlock is less equal to the wanted block number
			value = append(value, entry)
			// remove from the unstaking stakeStorage everything before this index
			found_idx = idx
		} else {
			// no need to keep iterating the sorted list
			break
		}
	}
	if found_idx >= 0 {
		stakeStorage.StakeEntries = stakeStorage.StakeEntries[found_idx+1:]
		k.SetStakeStorageUnstake(ctx, stakeStorage)
		return
	}
	return nil
}

// ------------------------------------------------

// takes the current stake storage and puts it in epoch storage
func (k Keeper) StoreCurrentEpochStakeStorage(ctx sdk.Context, block uint64) {
	allChainIDs := k.specKeeper.GetAllChainIDs(ctx)
	for _, chainID := range allChainIDs {
		tmpStorage, found := k.GetStakeStorageCurrent(ctx, chainID)
		if !found {
			// no storage for this spec yet
			continue
		}
		newStorage := tmpStorage.Copy()
		newStorage.Index = k.StakeStorageKey(block, chainID)
		newStorage.EpochBlockHash = ctx.HeaderHash() // set the current block hash for pairing to work without accessing history
		k.SetStakeStorage(ctx, newStorage)
	}
}

func (k Keeper) getStakeStorageEpoch(ctx sdk.Context, block uint64, chainID string) (stakeStorage types.StakeStorage, found bool) {
	key := k.StakeStorageKey(block, chainID)
	return k.GetStakeStorage(ctx, key)
}

func (k Keeper) GetStakeEntryForProviderEpoch(ctx sdk.Context, chainID string, selectedProvider sdk.AccAddress, epoch uint64) (entry *types.StakeEntry, err error) {
	stakeStorage, found := k.getStakeStorageEpoch(ctx, epoch, chainID)
	if !found {
		return nil, fmt.Errorf("could not find stakeStorage - epoch %d, chainID %s provider %s", epoch, chainID, selectedProvider.String())
	}
	providerStakeEntry, found, _ := k.GetStakeEntryByAddressFromStorage(ctx, stakeStorage, selectedProvider)
	if !found {
		return nil, fmt.Errorf("could not find stakeEntry - epoch %d for provider %s, chainID %s", epoch, selectedProvider.String(), chainID)
	}
	entry = &providerStakeEntry
	return
}

func (k Keeper) GetStakeEntryForAllProvidersEpoch(ctx sdk.Context, chainID string, epoch uint64) (entrys *[]types.StakeEntry, err error) {
	stakeStorage, found := k.getStakeStorageEpoch(ctx, epoch, chainID)
	if !found {
		return nil, fmt.Errorf("could not find stakeStorage - epoch %d, chainID %s", epoch, chainID)
	}

	return &stakeStorage.StakeEntries, nil
}

func (k Keeper) GetEpochStakeEntries(ctx sdk.Context, block uint64, chainID string) (entries []types.StakeEntry, found bool, epochHash []byte) {
	key := k.StakeStorageKey(block, chainID)
	stakeStorage, found := k.GetStakeStorage(ctx, key)
	if !found {
		return nil, false, nil
	}
	return stakeStorage.StakeEntries, true, stakeStorage.EpochBlockHash
}
