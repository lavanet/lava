package keeper

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) VerifyPairing(goCtx context.Context, req *types.QueryVerifyPairingRequest) (*types.QueryVerifyPairingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	foundAndActive, _, _ := k.specKeeper.IsSpecFoundAndActive(ctx, req.ChainID)
	// TODO:handle spec changes
	if !foundAndActive {
		return &types.QueryVerifyPairingResponse{Valid: false}, errors.New("spec not found or not enabled")
	}

	clientAddr, err := sdk.AccAddressFromBech32(req.Client)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address %s error: %s", req.Client, err)
	}
	providerAddr, err := sdk.AccAddressFromBech32(req.Provider)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address %s error: %s", req.Provider, err)
	}

	project, err := k.GetProjectData(ctx, clientAddr, req.ChainID, req.Block)
	if err != nil {
		return nil, err
	}

	isValidPairing, cuPerEpoch, providersToPair, err := k.ValidatePairingForClient(ctx, req.ChainID, providerAddr, req.Block, project)

	return &types.QueryVerifyPairingResponse{Valid: isValidPairing, PairedProviders: uint64(len(providersToPair)), CuPerEpoch: cuPerEpoch, ProjectId: project.Index}, err
}
