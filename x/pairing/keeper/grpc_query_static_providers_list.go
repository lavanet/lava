package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
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
	stakes, found := k.epochStorageKeeper.GetEpochStakeEntries(ctx, epoch, epochstoragetypes.ProviderKey, req.GetChainID())

	if !found {
		return &types.QueryStaticProvidersListResponse{}, nil
	}

	servicersToPairCount, err := k.ServicersToPairCount(ctx, epoch)
	if err != nil {
		return nil, err
	}

	finalProviders := []epochstoragetypes.StakeEntry{}
	geolocation := uint64(1)
	for i := uint64(0); i < k.specKeeper.GeolocationCount(ctx); i++ {
		validProviders := k.getGeolocationProviders(ctx, stakes, geolocation)
		validProviders = k.returnSubsetOfProvidersByHighestStake(ctx, validProviders, servicersToPairCount)
		finalProviders = append(finalProviders, validProviders...)
		geolocation <<= 1
	}

	return &types.QueryStaticProvidersListResponse{Providers: finalProviders}, nil
}
