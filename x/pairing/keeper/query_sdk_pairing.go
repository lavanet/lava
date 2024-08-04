package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
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

	strictestPolicy, _, err := k.GetProjectStrictestPolicy(ctx, project, req.ChainID, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, err
	}

	downtimeParams := k.downtimeKeeper.GetParams(ctx)

	return &types.QuerySdkPairingResponse{Pairing: pairing, Spec: spec, MaxCu: strictestPolicy.EpochCuLimit, DowntimeParams: &downtimeParams}, err
}
