package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) VerifyPairing(goCtx context.Context, req *types.QueryVerifyPairingRequest) (*types.QueryVerifyPairingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	clientAddr, err := sdk.AccAddressFromBech32(req.UserAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address %s error: %s", req.UserAddr, err)
	}
	servicerAddr, err := sdk.AccAddressFromBech32(req.ServicerAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address %s error: %s", req.UserAddr, err)
	}
	isValidPairing, isOverlap, err := k.ValidatePairingForClient(ctx, req.Spec, clientAddr, servicerAddr, types.BlockNum{Num: req.BlockNum})

	return &types.QueryVerifyPairingResponse{Valid: isValidPairing, Overlap: isOverlap}, err
}
