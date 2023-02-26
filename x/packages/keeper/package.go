package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/packages/types"
)

// AddPackage adds a new package to the KVStore. The function returns if the added package is a first version package
func (k Keeper) AddPackage(ctx sdk.Context, packageToAdd types.Package) error {
	// overwrite the packageToAdd's block field with the current block height
	packageToAdd.Block = uint64(ctx.BlockHeight())

	// TODO: verify the CU per epoch field

	err := k.packagesFs.AppendEntry(ctx, packageToAdd.GetIndex(), packageToAdd.GetBlock(), &packageToAdd)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "AddPackage_add_fixated_entry_failed", map[string]string{"packageToAdd": packageToAdd.String()}, "could not add new package fixated entry to storage")
	}

	return nil
}
