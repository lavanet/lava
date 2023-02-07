package keeper

import (
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
	// get packageVersionsStorage
	packageVersionsStorage, found := k.GetPackageVersionsStorage(ctx, packageIndex)
	if !found {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_package_versions_storage", map[string]string{"packageIndex": packageIndex}, "could not get packageVersionsStorage with package index")
	}

	// get packageStorage
	packageStorage := packageVersionsStorage.GetPackageStorage()

	// get packageFixationEntry
	packageFixationEntry, _, found := common.GetFixatedEntry(packageStorage.GetEntryList(), epoch)
	if !found {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_fixated_entry", map[string]string{"packageIndex": packageIndex, "epoch": strconv.FormatUint(epoch, 10)}, "could not get packageFixationEntry from packageVersionsStorage's entry list")
	}

	unmarshaledPackage, err := k.getPackageFromPackageFixationEntry(ctx, packageFixationEntry)
	if err != nil {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_package_from_package_entry", map[string]string{"err": err.Error()}, "could not get unmarshal data from packageFixationEntry to get package object")
	}

	return unmarshaledPackage, nil
}

// Function to get a package's latest version
func (k Keeper) getPackageLatestVersion(ctx sdk.Context, packageIndex string) (*types.Package, error) {
	// get packageVersionsStorage
	packageVersionsStorage, found := k.GetPackageVersionsStorage(ctx, packageIndex)
	if !found {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_package_versions_storage", map[string]string{"packageIndex": packageIndex}, "could not get packageVersionsStorage with package index")
	}

	// get the package fixation entry's latest version
	latestVersionPackageFixationEntry := common.GetLatestFixatedEntry(packageVersionsStorage.GetPackageStorage().GetEntryList())

	// get the package from the package fixation entry
	latestVersionPackage, err := k.getPackageFromPackageFixationEntry(ctx, latestVersionPackageFixationEntry)
	if err != nil {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_package_from_package_entry", map[string]string{"err": err.Error()}, "could not get unmarshal data from packageFixationEntry to get package object")
	}

	return latestVersionPackage, nil
}

// Function to extract and unmarshal the package object from a package fixation entry
func (k Keeper) getPackageFromPackageFixationEntry(ctx sdk.Context, packageFixationEntry *commontypes.Entry) (*types.Package, error) {
	// get the package object's marshalled data from the package fixation entry
	marshaledPackage, err := common.GetMarshaledDataFromEntry(ctx, packageFixationEntry)
	if err != nil {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_marshaled_data_from_entry", map[string]string{"err": err.Error()}, "could not get marshaled data from packageFixationEntry")
	}

	// unmarshal the marshaled data to get the package object
	var unmarshaledPackage *types.Package
	err = k.cdc.Unmarshal(marshaledPackage, unmarshaledPackage)
	if err != nil {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "unmarshal_package", map[string]string{"err": err.Error()}, "could not unmarshal package")
	}

	return unmarshaledPackage, nil
}

// Function to add a new package to the packageVersionsStorage. It supports addition of packages with new index and packages with existing index (index that is already saved in the storage)
func (k Keeper) AddNewPackageToStorage(ctx sdk.Context, packageToAdd *types.Package) error {
	// get current epoch and next epoch
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	// make the package's epoch the next epoch (which is when it will be added)
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
		return utils.LavaError(ctx, k.Logger(ctx), "package_duration", map[string]string{"err": err.Error()}, "duration can't be less than an epoch")
	}

	// make the package's computeUnitsPerEpoch be computeUnits / duration (in epochs)
	durationInEpochs := packageToAdd.GetDuration() / epochBlocks
	packageToAdd.ComputeUnitsPerEpoch = packageToAdd.GetComputeUnits() / durationInEpochs

	// make the package's subscriptions field zero (it's a new package, so no one is subscribed yet)
	packageToAdd.Subscriptions = 0

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

	// check if there is a package storage for the new package's index
	packageVersionsStorage, found := k.GetPackageVersionsStorage(ctx, packageToAdd.GetIndex())
	if !found {
		// packageVersionsStorage not found -> create a new entryStorage which consists the new packageFixationEntryToAdd
		entryStorage, err := common.CreateNewFixatedEntryStorage(ctx, packageFixationEntryToAdd)
		if err != nil {
			return utils.LavaError(ctx, k.Logger(ctx), "create_new_fixated_entry_storage", map[string]string{"err": err.Error()}, "could not create fixated entry storage")
		}

		// create a new packageVersionsStorage
		packageVersionsStorage = types.PackageVersionsStorage{PackageIndex: packageToAdd.GetIndex(), PackageStorage: entryStorage}
	} else {
		// packageVersionsStorage found -> get a new entry list with the new entry preprended to it
		updatedEntryList, err := common.SetFixatedEntry(ctx, packageVersionsStorage.GetPackageStorage().GetEntryList(), packageFixationEntryToAdd, true)
		if err != nil {
			return utils.LavaError(ctx, k.Logger(ctx), "set_fixated_entry", map[string]string{"err": err.Error()}, "could not set package storage")
		}

		// update the packageVersionsStorage's entry list
		packageVersionsStorage.PackageStorage.EntryList = updatedEntryList
	}

	// update the KVStore with the new packageVersionsStorage
	k.SetPackageVersionsStorage(ctx, packageVersionsStorage)

	return nil
}

// Function to delete a package from storage
func (k Keeper) deletePackage(ctx sdk.Context, packageIndex string, epoch uint64) error {
	// get the package storage of the input package ID
	packageVersionsStorage, found := k.GetPackageVersionsStorage(ctx, packageIndex)
	if !found {
		return utils.LavaError(ctx, k.Logger(ctx), "get_package_versions_storage", map[string]string{"packageIndex": packageIndex}, "could not get packageVersionsStorage with package index")
	}

	// get the fixation entry list
	entryList := packageVersionsStorage.GetPackageStorage().GetEntryList()

	// get the package fixation entry
	packageFixationEntryToDelete, _, found := common.GetFixatedEntry(entryList, epoch)
	if !found {
		return utils.LavaError(ctx, k.Logger(ctx), "get_fixated_entry", map[string]string{"packageIndex": packageIndex, "epoch": strconv.FormatUint(epoch, 10)}, "could not get package fixation entry with package index and epoch")
	}

	// get an edited entryList that doesn't include the packageFixationEntryToDelete
	updatedEntryList, err := common.SetFixatedEntry(ctx, entryList, packageFixationEntryToDelete, false)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "set_fixated_entry", map[string]string{"err": err.Error()}, "could not set package storage (delete package)")
	}

	// update the packageVersionsStorage's entry list
	packageVersionsStorage.PackageStorage.EntryList = updatedEntryList

	// update the KVStore with the new packageVersionsStorage
	k.SetPackageVersionsStorage(ctx, packageVersionsStorage)

	return nil
}

// Function to get the packages that should be deleted (package was edited and there are no subs for it. Also, currentEpoch > packageEpoch + packageDuration)
func (k Keeper) getPackagesToDelete(ctx sdk.Context) (map[string][]uint64, error) {
	// create map of packages to delete: key = packageIndex, value = list of epochs (which indicate the packages to delete)
	packagesToDeleteMap := make(map[string][]uint64)

	// get all the package storages
	allPackageStorages := k.GetAllPackageVersionsStorage(ctx)

	// get current epoch
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	// iterate over the package storages
	for _, packageStorage := range allPackageStorages {
		// get the package fixation entry list
		entryList := packageStorage.GetPackageStorage().GetEntryList()

		// check if the entryList's length is more than 1. If it's not, the package wasn't edited so it shouldn't be deleted
		if len(entryList) <= 1 {
			continue
		}

		// don't check the first packageFixationEntry is the entryList, since this is the latest package version so it should not be deleted (because it's still buyable)
		entryListWithoutLatestPackageVersion := entryList[1:]

		// iterate over the package fixation entries
		for packageIndex, packageFixationEntry := range entryListWithoutLatestPackageVersion {
			// get the package to check from the package fixation entry
			packageToCheck, err := k.getPackageFromPackageFixationEntry(ctx, packageFixationEntry)
			if err != nil {
				return nil, utils.LavaError(ctx, k.Logger(ctx), "get_package_from_package_entry", map[string]string{"err": err.Error()}, "could not get unmarshal data from packageFixationEntry to get package object")
			}

			// get the package to check's updated package
			updateOfPackageToCheckFixatioEntry := entryListWithoutLatestPackageVersion[packageIndex-1]
			updateOfPackageToCheck, err := k.getPackageFromPackageFixationEntry(ctx, updateOfPackageToCheckFixatioEntry)
			if err != nil {
				return nil, utils.LavaError(ctx, k.Logger(ctx), "get_package_from_package_entry", map[string]string{"err": err.Error()}, "could not get unmarshal data from packageFixationEntry to get package object")
			}

			// check if the package is stale (current epoch is bigger than the epoch its update was created + this package's duration)
			if currentEpoch > updateOfPackageToCheck.GetEpoch()+packageToCheck.GetDuration() {
				// check if the package also has no subscriptions
				if packageToCheck.GetSubscriptions() == 0 {
					// check if the map already has the current packageIndex key
					epochList, keyExists := packagesToDeleteMap[packageToCheck.GetIndex()]
					if !keyExists {
						// packageIndex doesn't exist in map -> create new epoch list
						packagesToDeleteMap[packageToCheck.GetIndex()] = []uint64{packageToCheck.GetEpoch()}
					} else {
						// packageIndex exist in map -> update the epoch list
						epochList := append(epochList, packageToCheck.GetEpoch())
						packagesToDeleteMap[packageToCheck.GetIndex()] = epochList
					}
				}
			}
		}
	}

	return packagesToDeleteMap, nil
}

func (k Keeper) setSubscriptions(ctx sdk.Context, packageIndex string, subscriptions uint64) error {
	// get the package storage of the input package ID
	packageVersionsStorage, found := k.GetPackageVersionsStorage(ctx, packageIndex)
	if !found {
		return utils.LavaError(ctx, k.Logger(ctx), "get_package_versions_storage", map[string]string{"packageIndex": packageIndex}, "could not get packageVersionsStorage with package index")
	}

	// get the fixation entry list
	entryList := packageVersionsStorage.GetPackageStorage().GetEntryList()

	// get the latest package fixation entry (we get the latest because it's the only buyable version)
	latestPackageFixationEntry := common.GetLatestFixatedEntry(entryList)

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

	// replace the latest package fixation entry in entryList
	entryList[0] = updatedPackageFixationEntry

	// update the packageVersionsStorage's entry list
	packageVersionsStorage.PackageStorage.EntryList = entryList

	// update the KVStore with the new packageVersionsStorage
	k.SetPackageVersionsStorage(ctx, packageVersionsStorage)

	return nil
}
