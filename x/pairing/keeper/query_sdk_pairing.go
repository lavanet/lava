package keeper

import (
	"context"
	"fmt"

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

	clientAddr, err := sdk.AccAddressFromBech32(req.Client)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address %s error: %s", req.Client, err)
	}

	project, err := k.GetProjectData(ctx, clientAddr, req.ChainID, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, err
	}

	strictestPolicy, err := k.GetProjectStrictestPolicy(ctx, project, req.ChainID)
	if err != nil {
		return nil, err
	}

	return &types.QuerySdkPairingResponse{Pairing: pairing, Spec: spec, MaxCu: strictestPolicy.EpochCuLimit}, err
}
