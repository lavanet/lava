package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) SdkPairing(goCtx context.Context, req *types.QueryGetPairingRequest) (*types.QuerySdkPairingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	pairing, spec, err := k.getPairing(ctx, req)
	if err != nil {
		return nil, err
	}

	project, err := k.GetProjectData(ctx, sdk.AccAddress(req.Client), req.ChainID, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, err
	}

	strictestPolicy, err := k.GetProjectStrictestPolicy(ctx, project, req.ChainID)
	if err != nil {
		return nil, err
	}

	return &types.QuerySdkPairingResponse{Pairing: pairing, Spec: spec, Cu: strictestPolicy.EpochCuLimit}, err
}
