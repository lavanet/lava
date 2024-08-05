package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) StaticProvidersList(goCtx context.Context, req *types.QueryStaticProvidersListRequest) (*types.QueryStaticProvidersListResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	spec, found := k.specKeeper.GetSpec(ctx, req.GetChainID())
	if !found || !spec.Enabled {
		return nil, fmt.Errorf("spec %s is not found or exist", req.GetChainID())
	}
	if spec.ProvidersTypes != spectypes.Spec_static {
		return nil, fmt.Errorf("spec %s is does not support static providers", req.GetChainID())
	}

	epoch := k.epochStorageKeeper.GetEpochStart(ctx)
	stakes, found, _ := k.epochStorageKeeper.GetEpochStakeEntries(ctx, epoch, req.GetChainID())

	if !found {
		return &types.QueryStaticProvidersListResponse{}, nil
	}

	finalProviders := []epochstoragetypes.StakeEntry{}
	for _, stake := range stakes {
		if stake.StakeAppliedBlock <= epoch {
			finalProviders = append(finalProviders, stake)
		}
	}

	return &types.QueryStaticProvidersListResponse{Providers: finalProviders}, nil
}
