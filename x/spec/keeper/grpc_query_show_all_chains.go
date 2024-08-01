package keeper

import (
	"context"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/spec/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ShowAllChains(goCtx context.Context, req *types.QueryShowAllChainsRequest) (*types.QueryShowAllChainsResponse, error) {
	var chainInfoList []*types.ShowAllChainsInfoStruct
	ctx := sdk.UnwrapSDKContext(goCtx)

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	// get all spec
	allSpec := k.GetAllSpec(ctx)

	// iterate over specs and extract the chains' info
	for _, spec := range allSpec {
		// get the spec's name and chain ID
		chainName := spec.GetName()
		chainId := spec.GetIndex()

		fullspec, err := k.ExpandSpec(ctx, spec)
		if err != nil {
			return nil, err
		}
		// get the spec's expected interfaces
		expectedInterfaces := k.GetExpectedServicesForExpandedSpec(fullspec, true)

		// TODO: print addons too
		apiInterfacesNames := getInterfacesNamesFromMap(expectedInterfaces)

		apiCount := uint64(0)

		for _, apiCollection := range fullspec.ApiCollections {
			if apiCollection.Enabled {
				for _, api := range apiCollection.Apis {
					if api.Enabled {
						apiCount++
					}
				}
			}
		}
		// create a chainInfoEntry which includes the chain's name, ID and enabled interfaces
		chainInfoEntry := types.ShowAllChainsInfoStruct{ChainName: chainName, ChainID: chainId, EnabledApiInterfaces: apiInterfacesNames, ApiCount: apiCount}

		// add the chainInfoEntry to the chainInfoList
		chainInfoList = append(chainInfoList, &chainInfoEntry)
	}

	return &types.QueryShowAllChainsResponse{ChainInfoList: chainInfoList}, nil
}

func getInterfacesNamesFromMap(expectedInterfaces map[epochstoragetypes.EndpointService]struct{}) []string {
	seen := make(map[string]struct{})
	var apiInterfacesNames []string
	for endpointService := range expectedInterfaces {
		if _, ok := seen[endpointService.ApiInterface]; !ok {
			seen[endpointService.ApiInterface] = struct{}{}
			apiInterfacesNames = append(apiInterfacesNames, endpointService.ApiInterface)
		}
	}
	sort.Strings(apiInterfacesNames)
	return apiInterfacesNames
}
