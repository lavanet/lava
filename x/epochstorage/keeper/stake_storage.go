package keeper

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/epochstorage/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
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

func (k Keeper) StakeStorageKey(block uint64, chainID string) string {
	return strconv.FormatUint(block, 10) + chainID
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

func (k Keeper) GetStakeEntryByAddressCurrent(ctx sdk.Context, chainID string, address string) (types.StakeEntry, bool) {
	if !utils.IsBech32Address(address) {
		utils.LavaFormatWarning("cannot get stake entry of invalid address", fmt.Errorf("invalid address"),
			utils.LogAttr("address", address),
		)
	}
	stakeStorage, found := k.GetStakeStorageCurrent(ctx, chainID)
	if !found {
		return types.StakeEntry{}, false
	}

	return stakeStorage.GetStakeEntryByAddressFromStorage(address)
}

func (k Keeper) RemoveStakeEntryCurrent(ctx sdk.Context, chainID string, address string) error {
	if !utils.IsBech32Address(address) {
		utils.LavaFormatWarning("cannot get stake entry of invalid address", fmt.Errorf("invalid address"),
			utils.LogAttr("address", address),
		)
	}
	stakeStorage, found := k.GetStakeStorageCurrent(ctx, chainID)
	if !found {
		return legacyerrors.ErrNotFound
	}

	for i, entry := range stakeStorage.StakeEntries {
		if entry.IsAddressVaultOrProvider(address) {
			stakeStorage.StakeEntries = append(stakeStorage.StakeEntries[:i], stakeStorage.StakeEntries[i+1:]...)
			k.SetStakeStorageCurrent(ctx, chainID, stakeStorage)
			return nil
		}
	}

	return legacyerrors.ErrNotFound
}

func (k Keeper) AppendStakeEntryFromStorage(ctx sdk.Context, stakeStorageIndex string, stakeEntry types.StakeEntry, isUnstake bool) {
	stakeStorage, found := k.GetStakeStorage(ctx, stakeStorageIndex)
	var entries []types.StakeEntry
	if !found {
		entries = []types.StakeEntry{stakeEntry}
		// create a new one
		stakeStorage = types.StakeStorage{Index: stakeStorageIndex, StakeEntries: entries, EpochBlockHash: nil}
	} else {
		// the following code inserts stakeEntry into the existing entries by stake
		entries = stakeStorage.StakeEntries
		// sort func needs to return true if the inserted entry is less than the existing entry
		sortFunc := func(i int) bool {
			return stakeEntry.StakeAppliedBlock <= entries[i].StakeAppliedBlock
		}
		if !isUnstake {
			sortFunc = func(i int) bool {
				return stakeEntry.EffectiveStake().LT(entries[i].EffectiveStake())
			}
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
	k.SetStakeStorage(ctx, stakeStorage)
}

func (k Keeper) AppendStakeEntryCurrent(ctx sdk.Context, chainID string, stakeEntry types.StakeEntry) {
	k.AppendStakeEntryFromStorage(ctx, k.stakeStorageKeyCurrent(chainID), stakeEntry, false)
}

func (k Keeper) ModifyStakeEntryCurrentFromStorage(ctx sdk.Context, stakeStorage types.StakeStorage, stakeEntry types.StakeEntry) {
	// TODO: more efficient: only create a new list once, after the second index is identified
	// remove the given index, then store the new entry in the sorted list at the right place
	entries := []types.StakeEntry{}
	for i, entry := range stakeStorage.StakeEntries {
		if entry.Address == stakeEntry.Address && entry.Vault == stakeEntry.Vault {
			entries = append(entries, stakeStorage.StakeEntries[:i]...)
			entries = append(entries, stakeStorage.StakeEntries[i+1:]...)
			break
		}
	}
	// the following code inserts stakeEntry into the existing entries by stake
	// sort func needs to return true if the inserted entry is less than the existing entry
	sortFunc := func(i int) bool {
		return stakeEntry.EffectiveStake().LT(entries[i].EffectiveStake())
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
	k.SetStakeStorage(ctx, stakeStorage)
}

func (k Keeper) ModifyStakeEntryCurrent(ctx sdk.Context, chainID string, stakeEntry types.StakeEntry) {
	// this stake storage entries are sorted by stake amount
	stakeStorage, found := k.GetStakeStorageCurrent(ctx, chainID)
	if !found {
		// should not happen since caller is expected to validate chainID first;
		// do nothing and return to avoid panic.
		utils.LavaFormatError("critical: ModifyStakeEntryCurrent with unknown chain", legacyerrors.ErrNotFound,
			utils.LogAttr("chainID", chainID),
			utils.LogAttr("stakeAddr", stakeEntry.Address),
		)
		return
	}
	k.ModifyStakeEntryCurrentFromStorage(ctx, stakeStorage, stakeEntry)
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

func (k Keeper) UnstakeEntryByAddress(ctx sdk.Context, address string) (value types.StakeEntry, found bool) {
	if !utils.IsBech32Address(address) {
		utils.LavaFormatWarning("cannot get stake entry of invalid address", fmt.Errorf("invalid address"),
			utils.LogAttr("address", address),
		)
	}
	stakeStorage, found := k.GetStakeStorageUnstake(ctx)
	if !found {
		return types.StakeEntry{}, false
	}
	return stakeStorage.GetStakeEntryByAddressFromStorage(address)
}

func (k Keeper) ModifyUnstakeEntry(ctx sdk.Context, stakeEntry types.StakeEntry) {
	// this stake storage entries are sorted by stake amount
	stakeStorage, found := k.GetStakeStorageUnstake(ctx)
	if !found {
		// should not happen since stake storage must always exist; do nothing to avoid panic
		utils.LavaFormatError("critical: ModifyUnstakeEntry failed to get stakeStorage", legacyerrors.ErrNotFound,
			utils.LogAttr("stakeAddr", stakeEntry.Address),
		)
		return
	}
	k.ModifyStakeEntryCurrentFromStorage(ctx, stakeStorage, stakeEntry)
}

func (k Keeper) AppendUnstakeEntry(ctx sdk.Context, stakeEntry types.StakeEntry, unstakeHoldBlocks uint64) {
	// update unstake stakeAppliedBlock to the higher among params (unstakeholdblocks and blockstosave)
	blockHeight := uint64(ctx.BlockHeight())
	stakeEntry.StakeAppliedBlock = blockHeight + unstakeHoldBlocks
	k.AppendStakeEntryFromStorage(ctx, types.StakeStorageKeyUnstakeConst, stakeEntry, true)
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

func (k Keeper) GetStakeStorageEpoch(ctx sdk.Context, block uint64, chainID string) (stakeStorage types.StakeStorage, found bool) {
	key := k.StakeStorageKey(block, chainID)
	return k.GetStakeStorage(ctx, key)
}

func (k Keeper) GetStakeEntryForProviderEpoch(ctx sdk.Context, chainID string, address string, epoch uint64) (types.StakeEntry, bool) {
	if !utils.IsBech32Address(address) {
		utils.LavaFormatWarning("cannot get stake entry of invalid address", fmt.Errorf("invalid address"),
			utils.LogAttr("address", address),
		)
	}
	stakeStorage, found := k.GetStakeStorageEpoch(ctx, epoch, chainID)
	if !found {
		return types.StakeEntry{}, false
	}
	return stakeStorage.GetStakeEntryByAddressFromStorage(address)
}

func (k Keeper) GetStakeEntryForAllProvidersEpoch(ctx sdk.Context, chainID string, epoch uint64) (entrys *[]types.StakeEntry, err error) {
	stakeStorage, found := k.GetStakeStorageEpoch(ctx, epoch, chainID)
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

func (k Keeper) GetUnstakeHoldBlocks(ctx sdk.Context, chainID string) uint64 {
	_, found, providerType := k.specKeeper.IsSpecFoundAndActive(ctx, chainID)
	if !found {
		utils.LavaFormatError("critical: failed to get spec for chainID",
			fmt.Errorf("unknown chainID"),
			utils.Attribute{Key: "chainID", Value: chainID},
		)
	}

	// note: if spec was not found, the default choice is Spec_dynamic == 0

	block := uint64(ctx.BlockHeight())
	if providerType == spectypes.Spec_static {
		return k.UnstakeHoldBlocksStatic(ctx, block)
	} else {
		return k.UnstakeHoldBlocks(ctx, block)
	}

	// NOT REACHED
}
