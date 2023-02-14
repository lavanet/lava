package keeper

import (
	"math"
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	common "github.com/lavanet/lava/common"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/packages/types"
)

// SetPackageEntry set a specific packageEntry in the store from its index
func (k Keeper) SetPackageEntry(ctx sdk.Context, packageEntry types.PackageEntry) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PackageEntryKeyPrefix))
	b := k.cdc.MustMarshal(&packageEntry)
	store.Set(types.PackageEntryKey(
		packageEntry.PackageIndex,
	), b)
}

// GetPackageEntry returns a packageEntry from its index
func (k Keeper) GetPackageEntry(
	ctx sdk.Context,
	packageIndex string,
) (val types.PackageEntry, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PackageEntryKeyPrefix))

	b := store.Get(types.PackageEntryKey(
		packageIndex,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemovePackageEntry removes a packageEntry from the store
func (k Keeper) RemovePackageEntry(
	ctx sdk.Context,
	packageIndex string,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PackageEntryKeyPrefix))
	store.Delete(types.PackageEntryKey(
		packageIndex,
	))
}

// GetAllPackageEntry returns all packageEntry
func (k Keeper) GetAllPackageEntry(ctx sdk.Context) (list []types.PackageEntry) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PackageEntryKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.PackageEntry
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// Function to get a package by package index + block
func (k Keeper) getPackageByBlock(ctx sdk.Context, packageIndex string, block uint64) (*types.Package, error) {
	// get the requested package's fixated entry
	packageFixationEntry, found := common.GetEntryFromStorage(ctx, packageIndex, block, k.GetFixationEntryByIndex)
	if !found {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_package_by_block", map[string]string{"packageIndex": packageIndex, "epoch": strconv.FormatUint(block, 10)}, "could not get packageFixationEntry with index and epoch")
	}

	// get the package from the fixation entry
	unmarshaledPackage, err := k.unmarshalPackage(ctx, packageFixationEntry)
	if err != nil {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_package_from_package_entry", map[string]string{"err": err.Error()}, "could not get package from package fixation entry")
	}

	return unmarshaledPackage, nil
}

// Function to get a package's latest version
func (k Keeper) GetPackageLatestVersion(ctx sdk.Context, packageIndex string) (*types.Package, error) {
	// get the requested package's latest fixated entry
	latestVersionPackageFixationEntry, found := common.GetEntryFromStorage(ctx, packageIndex, math.MaxUint64, k.GetFixationEntryByIndex)
	if !found {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_package_by_epoch", map[string]string{"packageIndex": packageIndex}, "could not get packageFixationEntry with index and epoch")
	}

	// get the package from the package fixation entry
	latestVersionPackage, err := k.unmarshalPackage(ctx, latestVersionPackageFixationEntry)
	if err != nil {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_package_from_package_entry", map[string]string{"err": err.Error()}, "could not get package from package fixation entry")
	}

	return latestVersionPackage, nil
}

// Function to extract and unmarshal the package object from a package fixation entry
func (k Keeper) unmarshalPackage(ctx sdk.Context, packageFixationEntry *commontypes.Entry) (*types.Package, error) {
	// get the marshaled data from the package fixation entry
	marshaledPackage := packageFixationEntry.GetMarshaledData()

	// unmarshal the marshaled data to get the package object
	var unmarshaledPackage types.Package
	err := k.cdc.Unmarshal(marshaledPackage, &unmarshaledPackage)
	if err != nil {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "unmarshal_package", map[string]string{"err": err.Error()}, "could not unmarshal package")
	}

	return &unmarshaledPackage, nil
}

// Function to add a new package to the packageEntry. It supports addition of packages with new index and packages with existing index (index that is already saved in the storage)
func (k Keeper) AddNewPackageToStorage(ctx sdk.Context, packageToAdd *types.Package) error {
	// get current epoch and next epoch
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	// make the package's epoch field's value be the next epoch (which is when it will be available for purchase)
	nextEpoch, err := k.epochStorageKeeper.GetNextEpoch(ctx, currentEpoch)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "get_next_epoch", map[string]string{"err": err.Error()}, "could not get next epoch")
	}
	packageToAdd.Epoch = nextEpoch

	// get current epochBlocks
	epochBlocks, err := k.epochStorageKeeper.EpochBlocks(ctx, currentEpoch)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "epoch_blocks", map[string]string{"err": err.Error()}, "could not get epoch blocks")
	}

	// make the package's computeUnitsPerEpoch be computeUnits / duration (in epochs)
	durationInEpochs := packageToAdd.GetDuration() / epochBlocks
	packageToAdd.ComputeUnitsPerEpoch = packageToAdd.GetComputeUnits() / durationInEpochs

	// make the package's subscriptions field zero (it's a new package, so no one is subscribed yet)
	packageToAdd.Subscriptions = uint64(0)

	// marshal the package
	marshaledPackage, err := k.cdc.Marshal(packageToAdd)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "marshal_new_package", map[string]string{"err": err.Error()}, "could not marshal package")
	}

	// create a new packageVersionEntry
	packageFixationEntryToAdd, err := common.CreateNewFixatedEntry(ctx, currentEpoch, marshaledPackage)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "create_new_fixated_entry", map[string]string{"err": err.Error()}, "could not create fixated entry")
	}

	// set the fixated entry and update the older versions' indices
	isFirstVersion, err := common.AddFixatedEntryToStorage(ctx, packageFixationEntryToAdd, packageToAdd.GetIndex(), k.GetFixationEntryByIndex, k.SetFixationEntry, k.RemovePackageEntry)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "set_fixated_entry_and_update_index", map[string]string{"packageIndex": packageToAdd.GetIndex()}, "could not set fixated entry with index")
	}

	// if this package is the first of its kind, add its index to the unique index list
	if isFirstVersion {
		k.AppendPackageUniqueIndex(ctx, types.PackageUniqueIndex{PackageUniqueIndex: packageToAdd.GetIndex()})
	}

	return nil
}

func (k Keeper) SetFixationEntryByIndex(ctx sdk.Context, index string, entryToSet *commontypes.Entry) error {
	// get the package from the fixation entry
	packageToSet, err := k.unmarshalPackage(ctx, entryToSet)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "get_package_from_package_fixation_entry", nil, "could not get package from package fixation entry")
	}

	// check if there is a package storage for the new package's index
	packageEntry, found := k.GetPackageEntry(ctx, index)
	if !found {
		// there is no packageEntry with the package's index -> create a new entryStorage which consists the new packageFixationEntryToSet
		newPackageEntry, err := k.createPackageEntry(ctx, packageToSet.GetIndex(), entryToSet)
		if err != nil {
			return utils.LavaError(ctx, k.Logger(ctx), "create_package_versions_storage", map[string]string{"err": err.Error()}, "could not create packageEntry object")
		}
		packageEntry = *newPackageEntry
	} else {
		// found the packageEntry with the package's index -> update its fields
		packageEntry.PackageIndex = index
		packageEntry.FixatedEntry = entryToSet
	}

	k.SetPackageEntry(ctx, packageEntry)

	return nil
}

// Function to create a new packageEntry object
func (k Keeper) createPackageEntry(ctx sdk.Context, packageIndex string, fixationEntry *commontypes.Entry) (*types.PackageEntry, error) {
	// check that the fixation entry is not nil
	if fixationEntry == nil {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "create_package_version_storage", nil, "could not create package version storage object. fixation entry is nil")
	}

	// create a new packageEntry
	packageEntry := types.PackageEntry{PackageIndex: packageIndex, FixatedEntry: fixationEntry}

	return &packageEntry, nil
}

// Function to delete a package from storage
func (k Keeper) deleteOldPackages(ctx sdk.Context) error {
	// get all the unique packages indices
	uniquePackagesIndices := k.GetAllPackageUniqueIndex(ctx)

	// get current epoch
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	// go over all the packageEntry objects
	for _, uniquePackageIndex := range uniquePackagesIndices {
		// get all the package versions indices
		packageVersionsList, packageVersionsIndexList := common.GetAllEntriesFromStorageByIndex(ctx, uniquePackageIndex.GetPackageUniqueIndex(), k.GetFixationEntryByIndex)

		// go over the package indices and check if they need to be deleted (i starts from 1 since the first package is the latest version and is still buyable)
		for i := 1; i < len(packageVersionsList); i++ {
			// check if the package should be deleted
			packageShouldBeDeleted, err := k.packageShouldBeDeleted(ctx, currentEpoch, packageVersionsList[i], packageVersionsList[i-1])
			if err != nil {
				return utils.LavaError(ctx, k.Logger(ctx), "package_should_be_deleted", map[string]string{"packageIndexToBeDeleted": packageVersionsIndexList[i]}, "could not check if package with index should be deleted")
			}

			// if the package should be deleted -> remove from storage (KVStore)
			if packageShouldBeDeleted {
				err := common.RemoveEntryFromStorage(ctx, packageVersionsIndexList[i], k.RemovePackageEntry)
				if err != nil {
					return utils.LavaError(ctx, k.Logger(ctx), "remove_entry_from_storage", map[string]string{"packageIndex": packageVersionsIndexList[i]}, "could not remove package with index")
				}
			}
		}
	}

	return nil
}

func (k Keeper) packageShouldBeDeleted(ctx sdk.Context, currentEpoch uint64, packageFixationEntryToCheck *commontypes.Entry, packageFixationEntryToCheckUpdate *commontypes.Entry) (bool, error) {
	// get the package to check if should be deleted from the fixation entry
	packageToCheck, err := k.unmarshalPackage(ctx, packageFixationEntryToCheck)
	if err != nil {
		return false, utils.LavaError(ctx, k.Logger(ctx), "get_package_from_package_fixation_entry", map[string]string{"entryIndex": packageToCheck.GetIndex()}, "could not get package from package fixation entry with index")
	}

	// get the update of the package from above
	updateOfPackageToCheck, err := k.unmarshalPackage(ctx, packageFixationEntryToCheckUpdate)
	if err != nil {
		return false, utils.LavaError(ctx, k.Logger(ctx), "get_package_from_package_fixation_entry", map[string]string{"entryIndex": updateOfPackageToCheck.GetIndex()}, "could not get package from package fixation entry with index")
	}

	// check if the package to check is stale (current epoch is bigger than the epoch its update was created + this package's duration)
	if currentEpoch > updateOfPackageToCheck.GetEpoch()+packageToCheck.GetDuration() {
		return true, nil
	}

	return false, nil
}

func (k Keeper) SetSubscriptions(ctx sdk.Context, packageIndex string, subscriptions uint64) error {
	// get current epoch and next epoch
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	// get the latest package fixation entry (we get the latest because it's the only buyable version)
	latestPackageFixationEntry, found := common.GetEntryFromStorage(ctx, packageIndex, currentEpoch, k.GetFixationEntryByIndex)
	if !found {
		return utils.LavaError(ctx, k.Logger(ctx), "get_entry_from_storage", map[string]string{"epoch": strconv.FormatUint(currentEpoch, 10), "packageIndex": packageIndex}, "could not get package's fixation entry with index")
	}

	// get the package out of the package fixation entry
	packageToEdit, err := k.unmarshalPackage(ctx, latestPackageFixationEntry)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "get_package_from_package_fixation_entry", map[string]string{"err": err.Error(), "packageIndex": packageIndex}, "could not get package's from package fixation entry")
	}

	// update the subscriptions field
	packageToEdit.Subscriptions = subscriptions

	// marshal the edited package
	marshaledPackage, err := k.cdc.Marshal(packageToEdit)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "marshal_new_package", map[string]string{"err": err.Error()}, "could not marshal package")
	}

	// create a new fixated entry
	updatedPackageFixationEntry, err := common.CreateNewFixatedEntry(ctx, packageToEdit.GetEpoch(), marshaledPackage)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "create_new_fixated_entry", map[string]string{"err": err.Error()}, "could not create new fixated entry")
	}

	// create a new packageEntry object
	packageEntry, err := k.createPackageEntry(ctx, packageIndex, updatedPackageFixationEntry)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "create_packe_versions_storage", map[string]string{"index": packageIndex}, "could not create new packageEntry object with package index")
	}

	// update the KVStore with the new packageEntry
	k.SetPackageEntry(ctx, *packageEntry)

	return nil
}

// Function to get a fixation entry by index
func (k Keeper) GetFixationEntryByIndex(ctx sdk.Context, index string) (*commontypes.Entry, bool) {
	// get the package version storage object by index
	packageEntry, found := k.GetPackageEntry(ctx, index)
	if !found {
		return nil, false
	}

	// return the fixation entry
	return packageEntry.GetFixatedEntry(), true
}

// Function that is used as the setter function of the fixationEntry library. It takes packageIndexToSet and an entryToSet to create a packageEntry and set
func (k Keeper) SetFixationEntry(ctx sdk.Context, packageIndexToSet string, entryToSet *commontypes.Entry) error {
	// check that the entry is not nil
	if entryToSet == nil {
		return utils.LavaError(ctx, k.Logger(ctx), "set_fixation_entry", nil, "could not set fixationEntry. entry is nil")
	}

	// get the relevant package versions storage
	packageEntry, found := k.GetPackageEntry(ctx, packageIndexToSet)
	if !found {
		// package versions storage not found -> create a new one (with entryToSet in it)
		newPackageEntry, err := k.createPackageEntry(ctx, packageIndexToSet, entryToSet)
		if err != nil {
			return utils.LavaError(ctx, k.Logger(ctx), "create_package_versions_storage", nil, "could not create packageEntry")
		}
		packageEntry = *newPackageEntry
	} else {
		// package versions storage found -> update its value
		packageEntry.PackageIndex = packageIndexToSet
		packageEntry.FixatedEntry = entryToSet
	}

	// set the packageEntry
	k.SetPackageEntry(ctx, packageEntry)

	return nil
}
