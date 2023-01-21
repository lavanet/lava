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

	geolocationMap := make(map[uint64][]*types.ProviderInfo)
	if found {
		geolocationcount := k.specKeeper.GeolocationCount(ctx)
		for _, stake := range stakes {
			for geolocation := uint64(1); geolocation <= geolocationcount; geolocation++ {
				if stake.Geolocation&1 == 1 {
					if _, ok := geolocationMap[geolocation]; !ok {
						geolocationMap[geolocation] = []*types.ProviderInfo{}
					}
					geolocationMap[geolocation] = append(geolocationMap[geolocation], &types.ProviderInfo{Address: stake.Address, Endpoints: stake.Endpoints, ExpirationEpoch: 0})
				}
			}
		}
	}

	response := types.QueryStaticProvidersListResponse{}
	response.Config.ChainId = req.GetChainID()
	response.Config.Description = spec.Name
	response.Config.Geolocations = []*types.GeoLocation{}
	for geolocation, providers := range geolocationMap {
		response.Config.Geolocations = append(response.Config.Geolocations, &types.GeoLocation{GeoLocation: geolocation, Providers: providers})
	}

	return &response, nil
}
