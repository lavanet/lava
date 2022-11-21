package keeper

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

func (k Keeper) RemoveOldEpochData(ctx sdk.Context, storageType string) {

	for _, block := range k.deletedEpochs {
		allChainIDs := k.specKeeper.GetAllChainIDs(ctx)
		for _, chainID := range allChainIDs {
			k.RemoveStakeStorageByBlockAndChain(ctx, storageType, block, chainID)
		}
	}
}

func (k *Keeper) UpdateEarliestEpochstart(ctx sdk.Context) {
	currentBlock := uint64(ctx.BlockHeight())
	earliestEpochBlock := k.GetEarliestEpochStart(ctx)
	blocksToSaveAtEarliestEpoch, err := k.BlocksToSave(ctx, earliestEpochBlock) //we take the epochs memory size at earliestEpochBlock, and not the current one
	k.deletedEpochs = []uint64{}
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
		k.deletedEpochs = append(k.deletedEpochs, earliestEpochBlock)
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
	//now update the earliest epoch start
	utils.LogLavaEvent(ctx, logger, "earliest_epoch", map[string]string{"block": strconv.FormatUint(earliestEpochBlock, 10)}, "updated earliest epoch block")
	k.SetEarliestEpochStart(ctx, earliestEpochBlock)
	return
}

func (k Keeper) StakeStorageKey(storageType string, block uint64, chainID string) string {
	return storageType + strconv.FormatUint(block, 10) + chainID
}

func (k Keeper) removeAllEntriesPriorToBlockNumber(ctx sdk.Context, storageType string, block uint64, allChainID []string) {
	allStorage := k.GetAllStakeStorage(ctx)
	for _, chainId := range allChainID {
		for _, entry := range allStorage {
			if strings.Contains(entry.Index, storageType) && strings.Contains(entry.Index, chainId) {
				if (len(storageType) + len(chainId)) > len(entry.Index) {
					panic(fmt.Sprintf("storageType + chainId length out of range %d vs %d\n more info: entry.Index: %s, storageType: %s, chainId: %s", (len(storageType) + len(chainId)), len(entry.Index), entry.Index, storageType, chainId))
				}
				storageBlock := entry.Index[len(storageType):]
				storageBlock = storageBlock[:(len(storageBlock) - len(chainId))]
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
	k.removeAllEntriesPriorToBlockNumber(ctx, types.ProviderKey, block, allChainID)
	k.removeAllEntriesPriorToBlockNumber(ctx, types.ClientKey, block, allChainID)
}

func (k Keeper) RemoveStakeStorageByBlockAndChain(ctx sdk.Context, storageType string, block uint64, chainID string) {
	key := k.StakeStorageKey(storageType, block, chainID)
	k.RemoveStakeStorage(ctx, key)
}

// -------------------------------------------------- current staking list --------------------------------------------

func (k Keeper) stakeStorageKeyCurrent(storageType string, chainID string) string {
	return storageType + chainID
}

//used to get the latest
func (k Keeper) GetStakeStorageCurrent(ctx sdk.Context, storageType string, chainID string) (types.StakeStorage, bool) {
	return k.GetStakeStorage(ctx, k.stakeStorageKeyCurrent(storageType, chainID))
}

func (k Keeper) SetStakeStorageCurrent(ctx sdk.Context, storageType string, chainID string, stakeStorage types.StakeStorage) {
	stakeStorage.Index = k.stakeStorageKeyCurrent(storageType, chainID)
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

func (k Keeper) GetStakeEntryByAddressCurrent(ctx sdk.Context, storageType string, chainID string, address sdk.AccAddress) (value types.StakeEntry, found bool, index uint64) {
	stakeStorage, found := k.GetStakeStorageCurrent(ctx, storageType, chainID)
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

func (k Keeper) RemoveStakeEntryCurrent(ctx sdk.Context, storageType string, chainID string, idx uint64) {
	stakeStorage, found := k.GetStakeStorageCurrent(ctx, storageType, chainID)
	if !found {
		return
	}
	stakeStorage.StakeEntries = append(stakeStorage.StakeEntries[:idx], stakeStorage.StakeEntries[idx+1:]...)
	k.SetStakeStorageCurrent(ctx, storageType, chainID, stakeStorage)
}

func (k Keeper) AppendStakeEntryCurrent(ctx sdk.Context, storageType string, chainID string, stakeEntry types.StakeEntry) {
	//this stake storage entries are sorted by stake amount
	stakeStorage, found := k.GetStakeStorageCurrent(ctx, storageType, chainID)
	var entries = []types.StakeEntry{}
	if !found {
		entries = []types.StakeEntry{stakeEntry}
		//create a new one
		stakeStorage = types.StakeStorage{Index: k.stakeStorageKeyCurrent(storageType, chainID), StakeEntries: entries}
	} else {
		// the following code inserts stakeEntry into the existing entries by stake
		entries = stakeStorage.StakeEntries
		//sort func needs to return true if the inserted entry is less than the existing entry
		sortFunc := func(i int) bool {
			return stakeEntry.Stake.Amount.LT(entries[i].Stake.Amount)
		}
		//returns the smallest index in which the sort func is true
		index := sort.Search(len(entries), sortFunc)
		if index < len(entries) {
			entries = append(entries[:index+1], entries[index:]...)
			entries[index] = stakeEntry
		} else {
			//put in the end
			entries = append(entries, stakeEntry)
		}
	}
	stakeStorage.StakeEntries = entries
	k.SetStakeStorageCurrent(ctx, storageType, chainID, stakeStorage)
}

func (k Keeper) ModifyStakeEntryCurrent(ctx sdk.Context, storageType string, chainID string, stakeEntry types.StakeEntry, removeIndex uint64) {
	//this stake storage entries are sorted by stake amount
	stakeStorage, found := k.GetStakeStorageCurrent(ctx, storageType, chainID)
	if !found {
		panic("called modify when there is no stakeStorage")
	}
	//TODO: more efficient: only create a new list once, after the second index is identified
	// remove the given index, then store the new entry in the sorted list at the right place
	entries := append(stakeStorage.StakeEntries[:removeIndex], stakeStorage.StakeEntries[removeIndex+1:]...)
	// the following code inserts stakeEntry into the existing entries by stake
	//sort func needs to return true if the inserted entry is less than the existing entry
	sortFunc := func(i int) bool {
		return stakeEntry.Stake.Amount.LT(entries[i].Stake.Amount)
	}
	//returns the smallest index in which the sort func is true
	index := sort.Search(len(entries), sortFunc)
	if index < len(entries) {
		entries = append(entries[:index+1], entries[index:]...)
		entries[index] = stakeEntry
	} else {
		entries = append(entries, stakeEntry)
	}
	stakeStorage.StakeEntries = entries
	k.SetStakeStorageCurrent(ctx, storageType, chainID, stakeStorage)
}

// -------------------------------------------------- unstaking list --------------------------------------------

func (k Keeper) stakeStorageKeyUnstake(storageType string) string {
	return storageType + "Unstake"
}

//used to get the unstaking entries
func (k Keeper) GetStakeStorageUnstake(ctx sdk.Context, storageType string) (types.StakeStorage, bool) {
	return k.GetStakeStorage(ctx, k.stakeStorageKeyUnstake(storageType))
}

func (k Keeper) SetStakeStorageUnstake(ctx sdk.Context, storageType string, stakeStorage types.StakeStorage) {
	stakeStorage.Index = k.stakeStorageKeyUnstake(storageType)
	k.SetStakeStorage(ctx, stakeStorage)
}

func (k Keeper) UnstakeEntryByAddress(ctx sdk.Context, storageType string, address sdk.AccAddress) (value types.StakeEntry, found bool, index uint64) {
	stakeStorage, found := k.GetStakeStorageUnstake(ctx, storageType)
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

func (k Keeper) ModifyUnstakeEntry(ctx sdk.Context, storageType string, stakeEntry types.StakeEntry, removeIndex uint64) {
	//this stake storage entries are sorted by stake amount
	stakeStorage, found := k.GetStakeStorageUnstake(ctx, storageType)
	if !found {
		panic("called modify when there is no stakeStorage")
	}
	//TODO: more efficient: only create a new list once, after the second index is identified
	// remove the given index, then store the new entry in the sorted list at the right place
	entries := append(stakeStorage.StakeEntries[:removeIndex], stakeStorage.StakeEntries[removeIndex+1:]...)
	// the following code inserts stakeEntry into the existing entries by stake
	//sort func needs to return true if the inserted entry is less than the existing entry
	sortFunc := func(i int) bool {
		return stakeEntry.Deadline <= entries[i].Deadline
	}
	//returns the smallest index in which the sort func is true
	index := sort.Search(len(entries), sortFunc)
	if index < len(entries) {
		entries = append(entries[:index+1], entries[index:]...)
		entries[index] = stakeEntry
	} else {
		entries = append(entries, stakeEntry)
	}
	stakeStorage.StakeEntries = entries
	k.SetStakeStorageUnstake(ctx, storageType, stakeStorage)
}

func (k Keeper) AppendUnstakeEntry(ctx sdk.Context, storageType string, stakeEntry types.StakeEntry) error {
	//update unstake deadline to the higher among params (unstakeholdblocks and blockstosave)
	//TODO validate in paramchange that unstakeholdblocks >= blockstosave and remove redundancy check
	blockHeight := uint64(ctx.BlockHeight())
	blocksToSave, err := k.BlocksToSave(ctx, blockHeight)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "append_stake_blockstosave_get_fail", map[string]string{}, "failed to get BlocksToSave")
	}
	stakeEntry.Deadline = blockHeight + blocksToSave
	holdBlocks := blockHeight + k.UnstakeHoldBlocks(ctx, blockHeight)
	if stakeEntry.Deadline < holdBlocks {
		stakeEntry.Deadline = holdBlocks
	}
	//this stake storage entries are sorted by deadline
	stakeStorage, found := k.GetStakeStorageUnstake(ctx, storageType)
	entries := []types.StakeEntry{}
	if !found {
		entries = []types.StakeEntry{stakeEntry}
		//create a new one
		stakeStorage = types.StakeStorage{Index: k.stakeStorageKeyUnstake(storageType), StakeEntries: entries}
	} else {
		// the following code inserts stakeEntry into the existing entries by deadline
		entries = stakeStorage.StakeEntries
		//sort func needs to return true if the inserted entry is less than the existing entry
		sortFunc := func(i int) bool {
			return stakeEntry.Deadline <= entries[i].Deadline
		}
		//returns the smallest index in which the sort func is true
		index := sort.Search(len(entries), sortFunc)
		if index < len(entries) {
			entries = append(entries[:index+1], entries[index:]...)
			entries[index] = stakeEntry
		} else {
			//put in the end
			entries = append(entries, stakeEntry)
		}
	}
	stakeStorage.StakeEntries = entries
	k.SetStakeStorageUnstake(ctx, storageType, stakeStorage)

	return nil
}

//Returns the unstaking Entry if its deadline is lower than the provided block
func (k Keeper) PopUnstakeEntries(ctx sdk.Context, storageType string, block uint64) (value []types.StakeEntry) {
	stakeStorage, found := k.GetStakeStorageUnstake(ctx, storageType)
	if !found {
		// utils.LavaError(ctx, k.Logger(ctx), "emptyStakeStorage", map[string]string{"storageType": storageType}, "stakeStorageUnstake Empty!")
		return nil
	}
	found_idx := -1
	// the unstaking is a sorted list so just chekcing until an entry deadline is too big
	for idx, entry := range stakeStorage.StakeEntries {
		if entry.Deadline <= block {
			// found an enrty that its deadline is less equal to the wanted block number
			value = append(value, entry)
			// remove from the unstaking stakeStorage everything before this index
			found_idx = idx
		} else {
			//no need to keep iterating the sorted list
			break
		}
	}
	if found_idx >= 0 {
		stakeStorage.StakeEntries = stakeStorage.StakeEntries[found_idx+1:]
		k.SetStakeStorageUnstake(ctx, storageType, stakeStorage)
		return
	}
	return nil
}

// ------------------------------------------------

// takes the current stake storage and puts it in epoch storage
func (k Keeper) StoreCurrentEpochStakeStorage(ctx sdk.Context, block uint64, storageType string) {
	allChainIDs := k.specKeeper.GetAllChainIDs(ctx)
	for _, chainID := range allChainIDs {
		tmpStorage, found := k.GetStakeStorageCurrent(ctx, storageType, chainID)
		if !found {
			//no storage for this spec yet
			continue
		}
		newStorage := tmpStorage.Copy()
		newStorage.Index = k.StakeStorageKey(storageType, block, chainID)
		k.SetStakeStorage(ctx, newStorage)
	}
}

func (k Keeper) getStakeStorageEpoch(ctx sdk.Context, block uint64, storageType string, chainID string) (stakeStorage types.StakeStorage, found bool) {
	key := k.StakeStorageKey(storageType, block, chainID)
	return k.GetStakeStorage(ctx, key)
}

// gets chainID, clientAddress, and epoch
// returns epochstoragetypes.StakeEntry which is needed to calculate allowedCU, for the selected epoch
func (k Keeper) GetStakeEntryForClientEpoch(ctx sdk.Context, chainID string, selectedClient sdk.AccAddress, epoch uint64) (entry *types.StakeEntry, err error) {
	stakeStorage, found := k.getStakeStorageEpoch(ctx, epoch, types.ClientKey, chainID)
	if !found {
		return nil, fmt.Errorf("could not find stakeStorage - epoch %d, chainID %s client %s", epoch, chainID, selectedClient.String())
	}
	clientStakeEntry, found, _ := k.GetStakeEntryByAddressFromStorage(ctx, stakeStorage, selectedClient)
	if !found {
		return nil, fmt.Errorf("could not find stakeEntry - epoch %d for client %s, chainID %s", epoch, selectedClient.String(), chainID)
	}
	entry = &clientStakeEntry
	return
}

func (k Keeper) GetStakeEntryForProviderEpoch(ctx sdk.Context, chainID string, selectedProvider sdk.AccAddress, epoch uint64) (entry *types.StakeEntry, err error) {
	stakeStorage, found := k.getStakeStorageEpoch(ctx, epoch, types.ProviderKey, chainID)
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
	stakeStorage, found := k.getStakeStorageEpoch(ctx, epoch, types.ProviderKey, chainID)
	if !found {
		return nil, fmt.Errorf("could not find stakeStorage - epoch %d, chainID %s", epoch, chainID)
	}

	return &stakeStorage.StakeEntries, nil
}

func (k Keeper) GetEpochStakeEntries(ctx sdk.Context, block uint64, storageType string, chainID string) (entries []types.StakeEntry, found bool) {
	key := k.StakeStorageKey(storageType, block, chainID)
	stakeStorage, found := k.GetStakeStorage(ctx, key)
	if !found {
		return nil, false
	}
	return stakeStorage.StakeEntries, true
}

func (k Keeper) GetEpochStakeStorage(ctx sdk.Context, block uint64, storageType string, chainID string) (stakeStorage types.StakeStorage, found bool) {
	key := k.StakeStorageKey(storageType, block, chainID)
	stakeStorage, found = k.GetStakeStorage(ctx, key)
	return stakeStorage, found
}

//append to epoch stake entries ONLY if it doesn't exist
func (k Keeper) AppendEpochStakeEntries(ctx sdk.Context, block uint64, storageType string, chainID string, stakeEntry types.StakeEntry) error {
	storage, found := k.GetEpochStakeStorage(ctx, block, storageType, chainID)
	if !found {
		entries := []types.StakeEntry{}
		//create a new one
		storage = types.StakeStorage{Index: k.StakeStorageKey(storageType, block, chainID), StakeEntries: entries}
	}
	entryAddr, err := sdk.AccAddressFromBech32(stakeEntry.Address)
	if err != nil {
		return err
	}

	for _, clientStakeEntry := range storage.StakeEntries {
		clientAddr, err := sdk.AccAddressFromBech32(clientStakeEntry.Address)
		if err != nil {
			panic(fmt.Sprintf("invalid user address saved in keeper %s, err: %s", clientStakeEntry.Address, err))
		}
		if clientAddr.Equals(entryAddr) {
			return nil //stake already exists in this epoch
		}
	}

	//put it in the right place
	entries := storage.StakeEntries
	sortFunc := func(i int) bool {
		return stakeEntry.Stake.Amount.LT(entries[i].Stake.Amount)
	}
	//returns the smallest index in which the sort func is true
	index := sort.Search(len(entries), sortFunc)
	if index < len(entries) {
		entries = append(entries[:index+1], entries[index:]...)
		entries[index] = stakeEntry
	} else {
		//put in the end
		entries = append(entries, stakeEntry)
	}

	storage.StakeEntries = entries
	k.SetStakeStorage(ctx, storage)
	return nil
}
