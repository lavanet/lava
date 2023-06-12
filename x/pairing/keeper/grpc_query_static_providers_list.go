package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingfilters "github.com/lavanet/lava/x/pairing/keeper/filters"
	"github.com/lavanet/lava/x/pairing/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
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
	stakes, found, _ := k.epochStorageKeeper.GetEpochStakeEntries(ctx, epoch, epochstoragetypes.ProviderKey, req.GetChainID())

	if !found {
		return &types.QueryStaticProvidersListResponse{}, nil
	}

	servicersToPairCount, err := k.ServicersToPairCount(ctx, epoch)
	if err != nil {
		return nil, err
	}

	finalProviders := []epochstoragetypes.StakeEntry{}
	var geolocationFilter pairingfilters.GeolocationFilter
	policy := projectstypes.Policy{
		GeolocationProfile:    uint64(1),
		SelectedProvidersMode: projectstypes.SELECTED_PROVIDERS_MODE_DISABLED,
	}
	geoFilterActive := geolocationFilter.InitFilter(policy)
	if !geoFilterActive {
		return nil, utils.LavaFormatError("geolocation filter should be active according to the mode", fmt.Errorf("geo filter not active"),
			utils.Attribute{Key: "selected_providers_mode", Value: policy.SelectedProvidersMode},
			utils.Attribute{Key: "geolocation_profile", Value: policy.GeolocationProfile},
		)
	}

	for i := uint64(0); i < k.specKeeper.GeolocationCount(ctx); i++ {
		validProviders, err := pairingfilters.FilterProviders(ctx, []pairingfilters.Filter{&geolocationFilter}, stakes, policy, epoch)
		if err != nil {
			return nil, err
		}
		validProviders = k.returnSubsetOfProvidersByHighestStake(ctx, validProviders, servicersToPairCount)
		finalProviders = append(finalProviders, validProviders...)
		policy.GeolocationProfile <<= 1
	}

	return &types.QueryStaticProvidersListResponse{Providers: finalProviders}, nil
}
