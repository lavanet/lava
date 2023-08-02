package keeper

import (
	"context"

    "github.com/lavanet/lava/x/pairing/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) SdkPairing(goCtx context.Context,  req *types.QuerySdkPairingRequest) (*types.QuerySdkPairingResponse, error) {
	if req == nil {
        return nil, status.Error(codes.InvalidArgument, "invalid request")
    }

	ctx := sdk.UnwrapSDKContext(goCtx)

    // TODO: Process the query
    _ = ctx

	return &types.QuerySdkPairingResponse{}, nil
}
