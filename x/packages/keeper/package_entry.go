package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	common "github.com/lavanet/lava/common"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/packages/types"
)

// SetPackage set a specific package in its fixated form
func (k Keeper) SetPackage(ctx sdk.Context, packageIndex string, packageEntry types.Package) error {
	entry, found := common.GetEntry(ctx, k.storeKey, types.PackageEntryKeyPrefix, k.cdc, packageIndex)
	if !found {
		return utils.LavaError(ctx, k.Logger(ctx), "SetPackage_package_not_found", map[string]string{"packageIndex": packageIndex}, "could not find the package to set")
	}
	b := k.cdc.MustMarshal(&packageEntry)
	entry.MarshaledData = b
	common.SetEntry(ctx, k.storeKey, types.PackageEntryKeyPrefix, k.cdc, entry)
	return nil
}

// AddPackageEntry set a specific packageEntry in the store from its index
func (k Keeper) AddPackage(ctx sdk.Context, packageEntry types.Package) {
	b := k.cdc.MustMarshal(&packageEntry)
	common.AddFixatedEntry(ctx, k.storeKey, types.PackageEntryKeyPrefix, k.cdc, packageEntry.Index, b)
}

// GetPackageEntry returns a packageEntry from its index
func (k Keeper) GetPackageForBlock(
	ctx sdk.Context,
	packageIndex string,
	block uint64,
) (val types.Package, found bool, entryIndex string) {
	entry, found := common.GetEntryForBlock(ctx, k.storeKey, types.PackageEntryKeyPrefix, k.cdc, packageIndex, block)
	if !found {
		return types.Package{}, false, ""
	}
	k.cdc.MustUnmarshal(entry.MarshaledData, &val)
	return val, true, entry.GetIndex()
}

// RemovePackageEntry removes a packageEntry from the store
func (k Keeper) RemovePackageEntry(
	ctx sdk.Context,
	packageIndex string,
) {
	common.RemoveEntry(ctx, k.storeKey, types.PackageEntryKeyPrefix, packageIndex)
}

//GetAllEntriesForIndex returns all fixated packages for the index
func (k Keeper) GetAllEntriesForIndex(ctx sdk.Context, packageIndex string) []types.Package {
	entries := common.GetAllEntriesForIndex(ctx, k.storeKey, types.PackageEntryKeyPrefix, k.cdc, packageIndex)
	packages := []types.Package{}
	for _, entry := range entries {
		var val types.Package
		k.cdc.MustUnmarshal(entry.MarshaledData, &val)
		packages = append(packages, val)
	}
	return packages
}

// Function to get a package's latest version
func (k Keeper) GetPackageLatestVersion(ctx sdk.Context, packageIndex string) (*types.Package, error) {
	// get the requested package's latest fixated entry
	latestPackage, found, _ := k.GetPackageForBlock(ctx, packageIndex, uint64(ctx.BlockHeight()))
	if !found {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_package_by_epoch", map[string]string{"packageIndex": packageIndex}, "could not get packageFixationEntry with index and epoch")
	}

	return &latestPackage, nil
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

	// make the package's subscriptions field zero (it's a new package, so no one is subscribed yet)
	packageToAdd.Subscriptions = uint64(0)

	// marshal the package
	marshaledPackage, err := k.cdc.Marshal(packageToAdd)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "marshal_new_package", map[string]string{"err": err.Error()}, "could not marshal package")
	}

	// set the fixated entry and update the older versions' indices
	isFirstVersion, err := common.AddFixatedEntry(ctx, k.storeKey, types.PackageEntryKeyPrefix, k.cdc, packageToAdd.Index, marshaledPackage)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "set_fixated_entry_and_update_index", map[string]string{"packageIndex": packageToAdd.GetIndex()}, "could not set fixated entry with index")
	}

	// if this package is the first of its kind, add its index to the unique index list
	if isFirstVersion {
		k.AppendPackageUniqueIndex(ctx, types.PackageUniqueIndex{PackageUniqueIndex: packageToAdd.GetIndex()})
	}

	return nil
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
		packageVersionsList := common.GetAllEntriesForIndex(ctx, k.storeKey, types.PackageEntryKeyPrefix, k.cdc, uniquePackageIndex.GetPackageUniqueIndex())

		// go over the package indices and check if they need to be deleted (i starts from 1 since the first package is the latest version and is still buyable)
		for i := 1; i < len(packageVersionsList); i++ {
			// check if the package should be deleted
			packageShouldBeDeleted, err := k.packageShouldBeDeleted(ctx, currentEpoch, packageVersionsList[i], packageVersionsList[i-1])
			if err != nil {
				return utils.LavaError(ctx, k.Logger(ctx), "package_should_be_deleted", map[string]string{"packageIndexToBeDeleted": packageVersionsList[i].GetIndex()}, "could not check if package with index should be deleted")
			}

			// if the package should be deleted -> remove from storage (KVStore)
			if packageShouldBeDeleted {
				common.RemoveEntry(ctx, k.storeKey, types.PackageEntryKeyPrefix, packageVersionsList[i].GetIndex())
			}
		}
	}

	return nil
}

func (k Keeper) packageShouldBeDeleted(ctx sdk.Context, currentEpoch uint64, packageFixationEntryToCheck *commontypes.Entry, packageFixationEntryToCheckUpdate *commontypes.Entry) (bool, error) {
	//TODO: think about how this needs to happen
	return false, nil
}

func (k Keeper) AddSubscription(ctx sdk.Context, packageIndex string) error {
	latestPackage, err := k.GetPackageLatestVersion(ctx, packageIndex)
	if err != nil {
		return err
	}

	latestPackage.Subscriptions++

	k.SetPackage(ctx, packageIndex, *latestPackage)

	return nil
}
