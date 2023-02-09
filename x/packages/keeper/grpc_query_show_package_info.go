package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/packages/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ShowPackageInfo(goCtx context.Context, req *types.QueryShowPackageInfoRequest) (*types.QueryShowPackageInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	packageToPrint, err := k.GetPackageLatestVersion(ctx, req.GetPackageIndex())
	if err != nil {
		return nil, status.Error(codes.NotFound, "package not found")
	}

	return &types.QueryShowPackageInfoResponse{PackageInfo: packageToPrint}, nil
}
