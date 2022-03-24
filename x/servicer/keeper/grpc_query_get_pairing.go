package keeper

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetPairing(goCtx context.Context, req *types.QueryGetPairingRequest) (*types.QueryGetPairingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	specName := types.SpecName{Name: req.SpecName}
	err := specName.ValidateBasic()
	if err != nil {
		return nil, fmt.Errorf("invalid spec name: %s, error: %s", req.SpecName, err)
	}

	clientAddr, err := sdk.AccAddressFromBech32(req.UserAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address %s error: %s", req.UserAddr, err)
	}

	foundAndActive, _, specID := k.specKeeper.IsSpecFoundAndActive(ctx, specName.Name)
	if !foundAndActive {
		return nil, errors.New("spec not found or not enabled")
	}
	servicers, err := k.GetPairingForClient(ctx, specID, clientAddr)
	if err != nil {
		return nil, fmt.Errorf("could not get pairing for spec ID: %d, client addr: %s, blockHeight: %d, err: %s", specID, clientAddr, ctx.BlockHeight(), err)
	}
	respStakeStorage := types.StakeStorage{Staked: servicers}
	return &types.QueryGetPairingResponse{Servicers: &respStakeStorage}, nil
}
