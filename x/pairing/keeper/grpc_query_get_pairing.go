package keeper

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/pairing/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Gets a client's provider list in a specific chain. Also returns the start block of the current epoch, time (in seconds) until there's a new pairing, the block that the chain in the request's spec was changed
func (k Keeper) GetPairing(goCtx context.Context, req *types.QueryGetPairingRequest) (*types.QueryGetPairingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	pairing, _, err := k.getPairing(ctx, req)
	return pairing, err
}

func (k Keeper) getPairing(ctx sdk.Context, req *types.QueryGetPairingRequest) (*types.QueryGetPairingResponse, *spectypes.Spec, error) {
	if req == nil {
		return nil, nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	// TODO: validate chainID
	// check client address
	clientAddr, err := sdk.AccAddressFromBech32(req.Client)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid creator address %s error: %s", req.Client, err)
	}

	// Make sure the chain ID exists and the chain's functional
	foundAndActive, _, _ := k.specKeeper.IsSpecFoundAndActive(ctx, req.ChainID)
	// TODO: handle spec changes
	if !foundAndActive {
		return nil, nil, errors.New("spec not found or not enabled")
	}

	// Get pairing list for latest block
	providers, err := k.GetPairingForClient(ctx, req.ChainID, clientAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get pairing for chainID: %s, client addr: %s, blockHeight: %d, err: %s", req.ChainID, clientAddr, ctx.BlockHeight(), err)
	}

	// Calculate the time left until the new epoch (when epoch changes, new pairing is generated)
	timeLeftToNextPairing, nextPairingBlock, err := k.calculateNextEpochTimeAndBlock(ctx)
	if err != nil {
		// we don't want to fail the query if the calculateNextEpochTime function fails. This shouldn't happen, it's a fail-safe
		utils.LavaFormatError("calculate next epoch time failed. Returning default time=0", err)
		timeLeftToNextPairing = 0
	}

	// Get current epoch start block
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	// Get the block in which there was the latest change for the current spec
	spec, err := k.specKeeper.GetExpandedSpec(ctx, req.GetChainID())
	if err != nil {
		return nil, nil, err
	}

	return &types.QueryGetPairingResponse{Providers: providers, CurrentEpoch: currentEpoch, TimeLeftToNextPairing: timeLeftToNextPairing, SpecLastUpdatedBlock: spec.BlockLastUpdated, BlockOfNextPairing: nextPairingBlock}, &spec, nil
}
