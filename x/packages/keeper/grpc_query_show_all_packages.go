package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/packages/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ShowAllPackages(goCtx context.Context, req *types.QueryShowAllPackagesRequest) (*types.QueryShowAllPackagesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	var allPackagesInfo []*types.ShowAllPackagesInfoStruct
	ctx := sdk.UnwrapSDKContext(goCtx)

	// get all packages' unique indices
	allPackageEntryUniqueIndices := common.GetAllFixationEntryUniqueIndex(ctx, k.storeKey, k.cdc, types.UniqueIndexKeyPrefix())

	// go over all the packages' unique indices
	for _, packageEntryUniqueIndex := range allPackageEntryUniqueIndices {
		packageInfoStruct := types.ShowAllPackagesInfoStruct{}

		// get the latest version package
		latestVersionPackage, err := k.GetPackageLatestVersion(ctx, packageEntryUniqueIndex.GetUniqueIndex())
		if err != nil {
			return nil, utils.LavaError(ctx, ctx.Logger(), "get_package_latest_version", map[string]string{"err": err.Error(), "packageIndex": packageEntryUniqueIndex.GetUniqueIndex()}, "could not get the latest version of the package")
		}

		// set the packageInfoStruct
		packageInfoStruct.Index = latestVersionPackage.GetIndex()
		packageInfoStruct.Name = latestVersionPackage.GetName()
		packageInfoStruct.Price = latestVersionPackage.GetPrice()

		// append the packageInfoStruct to the allPackagesInfo list
		allPackagesInfo = append(allPackagesInfo, &packageInfoStruct)
	}
	_ = ctx

	return &types.QueryShowAllPackagesResponse{PackagesInfo: allPackagesInfo}, nil
}
