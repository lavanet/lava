package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/x/packages/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ShowPackageInfo(goCtx context.Context, req *types.QueryShowPackageInfoRequest) (*types.QueryShowPackageInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var packageToPrint types.Package
	err := k.packagesFs.GetEntry(ctx, req.GetPackageIndex(), uint64(ctx.BlockHeight()), &packageToPrint, commontypes.DO_NOTHING)
	if err != nil {
		return nil, status.Error(codes.NotFound, "package not found")
	}

	return &types.QueryShowPackageInfoResponse{PackageInfo: &packageToPrint}, nil
}
