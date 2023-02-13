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

// SetPackageVersionsStorage set a specific packageVersionsStorage in the store from its index
func (k Keeper) SetPackageVersionsStorage(ctx sdk.Context, packageVersionsStorage types.PackageVersionsStorage) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PackageVersionsStorageKeyPrefix))
	b := k.cdc.MustMarshal(&packageVersionsStorage)
	store.Set(types.PackageVersionsStorageKey(
		packageVersionsStorage.PackageIndex,
	), b)
}

func (k Keeper) SetFixationEntry(ctx sdk.Context, entryIndexToSet string, entryToSet *commontypes.Entry) error {
	// check that the entry is not nil
	if entryToSet == nil {
		return utils.LavaError(ctx, k.Logger(ctx), "set_fixation_entry", nil, "could not set fixationEntry. entry is nil")
	}

	// get the relevant package versions storage
	packageVersionsStorage, found := k.GetPackageVersionsStorage(ctx, entryIndexToSet)
	if !found {
		// package versions storage not found -> create a new one (with entryToSet in it)
		newPackageVersionsStorage, err := k.createPackageVersionsStorage(ctx, entryIndexToSet, entryToSet)
		if err != nil {
			return utils.LavaError(ctx, k.Logger(ctx), "create_package_versions_storage", nil, "could not create packageVersionsStorage")
		}
		packageVersionsStorage = *newPackageVersionsStorage
	} else {
		// package versions storage found -> update its value
		packageVersionsStorage.PackageIndex = entryIndexToSet
		packageVersionsStorage.FixatedEntry = entryToSet
	}

	// set the packageVersionsStorage
	k.SetPackageVersionsStorage(ctx, packageVersionsStorage)

	return nil
}

// GetPackageVersionsStorage returns a packageVersionsStorage from its index
func (k Keeper) GetPackageVersionsStorage(
	ctx sdk.Context,
	packageIndex string,
) (val types.PackageVersionsStorage, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PackageVersionsStorageKeyPrefix))

	b := store.Get(types.PackageVersionsStorageKey(
		packageIndex,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemovePackageVersionsStorage removes a packageVersionsStorage from the store
func (k Keeper) RemovePackageVersionsStorage(
	ctx sdk.Context,
	packageIndex string,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PackageVersionsStorageKeyPrefix))
	store.Delete(types.PackageVersionsStorageKey(
		packageIndex,
	))
}

// GetAllPackageVersionsStorage returns all packageVersionsStorage
func (k Keeper) GetAllPackageVersionsStorage(ctx sdk.Context) (list []types.PackageVersionsStorage) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PackageVersionsStorageKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.PackageVersionsStorage
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// Function to get a package by package index + epoch
func (k Keeper) getPackageByEpoch(ctx sdk.Context, packageIndex string, epoch uint64) (*types.Package, error) {
	// get the requested package's fixated entry
	packageFixationEntry, found := common.GetEntryFromStorage(ctx, packageIndex, epoch, k.GetFixationEntryByIndex)
	if !found {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_package_by_epoch", map[string]string{"packageIndex": packageIndex, "epoch": strconv.FormatUint(epoch, 10)}, "could not get packageFixationEntry with index and epoch")
	}

	// get the package from the fixation entry
	unmarshaledPackage, err := k.getPackageFromPackageFixationEntry(ctx, packageFixationEntry)
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
	latestVersionPackage, err := k.getPackageFromPackageFixationEntry(ctx, latestVersionPackageFixationEntry)
	if err != nil {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_package_from_package_entry", map[string]string{"err": err.Error()}, "could not get package from package fixation entry")
	}

	return latestVersionPackage, nil
}

// Function to extract and unmarshal the package object from a package fixation entry
func (k Keeper) getPackageFromPackageFixationEntry(ctx sdk.Context, packageFixationEntry *commontypes.Entry) (*types.Package, error) {
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

// Function to add a new package to the packageVersionsStorage. It supports addition of packages with new index and packages with existing index (index that is already saved in the storage)
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

	// check that the duration is at least an epoch
	if packageToAdd.GetDuration() < epochBlocks {
		return utils.LavaError(ctx, k.Logger(ctx), "package_duration", map[string]string{"epochBlocks": strconv.FormatUint(epochBlocks, 10), "packageDuration": strconv.FormatUint(packageToAdd.GetDuration(), 10)}, "duration can't be less than an epoch")
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
	isFirstVersion, err := common.AddFixatedEntryToStorage(ctx, packageFixationEntryToAdd, packageToAdd.GetIndex(), k.GetFixationEntryByIndex, k.SetFixationEntry, k.RemovePackageVersionsStorage)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "set_fixated_entry_and_update_index", map[string]string{"packageIndex": packageToAdd.GetIndex()}, "could not set fixated entry with index")
	}

	// if this package is the first of its kind, add its index to the unique index list
	if isFirstVersion {
		k.AppendPackageUniqueIndex(ctx, types.PackageUniqueIndex{Index: packageToAdd.GetIndex()})
	}

	return nil
}

func (k Keeper) SetFixationEntryByIndex(ctx sdk.Context, index string, entryToSet *commontypes.Entry) error {
	// get the package from the fixation entry
	packageToSet, err := k.getPackageFromPackageFixationEntry(ctx, entryToSet)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "get_package_from_package_fixation_entry", nil, "could not get package from package fixation entry")
	}

	// check if there is a package storage for the new package's index
	packageVersionsStorage, found := k.GetPackageVersionsStorage(ctx, index)
	if !found {
		// there is no packageVersionsStorage with the package's index -> create a new entryStorage which consists the new packageFixationEntryToSet
		newPackageVersionsStorage, err := k.createPackageVersionsStorage(ctx, packageToSet.GetIndex(), entryToSet)
		if err != nil {
			return utils.LavaError(ctx, k.Logger(ctx), "create_package_versions_storage", map[string]string{"err": err.Error()}, "could not create packageVersionsStorage object")
		}
		packageVersionsStorage = *newPackageVersionsStorage
	} else {
		// found the packageVersionsStorage with the package's index -> update its fields
		packageVersionsStorage.PackageIndex = index
		packageVersionsStorage.FixatedEntry = entryToSet
	}

	k.SetPackageVersionsStorage(ctx, packageVersionsStorage)

	return nil
}

// Function to create a new packageVersionsStorage object
func (k Keeper) createPackageVersionsStorage(ctx sdk.Context, packageIndex string, fixationEntry *commontypes.Entry) (*types.PackageVersionsStorage, error) {
	// check that the fixation entry is not nil
	if fixationEntry == nil {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "create_package_version_storage", nil, "could not create package version storage object. fixation entry is nil")
	}

	// create a new packageVersionsStorage
	packageVersionsStorage := types.PackageVersionsStorage{PackageIndex: packageIndex, FixatedEntry: fixationEntry}

	return &packageVersionsStorage, nil
}

// Function to delete a package from storage
func (k Keeper) deletePackages(ctx sdk.Context) error {
	// get all the unique packages indices
	uniquePackagesIndices := k.GetAllPackageUniqueIndex(ctx)

	// get current epoch
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	// go over all the packageVersionsStorage objects
	for _, uniquePackageIndex := range uniquePackagesIndices {
		// get all the package versions indices
		packageVersionsList, packageVersionsIndexList := common.GetAllEntriesFromStorageByIndex(ctx, uniquePackageIndex.GetIndex(), k.GetFixationEntryByIndex)

		// go over the package indices and check if they need to be deleted (i starts from 1 since the first package is the latest version and is still buyable)
		for i := 1; i < len(packageVersionsList); i++ {
			// check if the package should be deleted
			packageShouldBeDeleted, err := k.packageShouldBeDeleted(ctx, currentEpoch, packageVersionsList[i], packageVersionsList[i-1])
			if err != nil {
				return utils.LavaError(ctx, k.Logger(ctx), "package_should_be_deleted", map[string]string{"packageIndexToBeDeleted": packageVersionsIndexList[i]}, "could not check if package with index should be deleted")
			}

			// if the package should be deleted -> remove from storage (KVStore)
			if packageShouldBeDeleted {
				err := common.RemoveEntryFromStorage(ctx, packageVersionsIndexList[i], k.RemovePackageVersionsStorage)
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
	packageToCheck, err := k.getPackageFromPackageFixationEntry(ctx, packageFixationEntryToCheck)
	if err != nil {
		return false, utils.LavaError(ctx, k.Logger(ctx), "get_package_from_package_fixation_entry", map[string]string{"entryIndex": packageToCheck.GetIndex()}, "could not get package from package fixation entry with index")
	}

	// get the update of the package from above
	updateOfPackageToCheck, err := k.getPackageFromPackageFixationEntry(ctx, packageFixationEntryToCheckUpdate)
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
	packageToEdit, err := k.getPackageFromPackageFixationEntry(ctx, latestPackageFixationEntry)
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

	// create a new packageVersionsStorage object
	packageVersionsStorage, err := k.createPackageVersionsStorage(ctx, packageIndex, updatedPackageFixationEntry)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "create_packe_versions_storage", map[string]string{"index": packageIndex}, "could not create new packageVersionsStorage object with package index")
	}

	// update the KVStore with the new packageVersionsStorage
	k.SetPackageVersionsStorage(ctx, *packageVersionsStorage)

	return nil
}

// Function to get a fixation entry by index
func (k Keeper) GetFixationEntryByIndex(ctx sdk.Context, index string) (*commontypes.Entry, bool) {
	// get the package version storage object by index
	packageVersionStorage, found := k.GetPackageVersionsStorage(ctx, index)
	if !found {
		return nil, false
	}

	// return the fixation entry
	return packageVersionStorage.GetFixatedEntry(), true
}
