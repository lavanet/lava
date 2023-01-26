package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/spec/types"
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
		// get the spec's APIs
		apis := spec.GetApis()

		// create API interface names map to efficiently check if an interface already exists in it
		apiInterfacesNamesMap := make(map[string]int)

		// iterate over the APIs
		for _, api := range apis {
			// get the API interfaces
			apiInterfaces := api.GetApiInterfaces()

			// iterate over the API interfaces
			for _, apiInterface := range apiInterfaces {
				// get the interface's name
				apiInterfaceName := apiInterface.GetInterface()

				// check if the interface exists in the map
				_, found := apiInterfacesNamesMap[apiInterfaceName]

				// if the interface name wasn't found, add it to the map (put dummy value 0)
				if !found {
					apiInterfacesNamesMap[apiInterfaceName] = 0
				}
			}
		}

		// copy the apiInterfacesNamesMap's keys (which are the interface names) to a string list
		var apiInterfacesNames []string
		for apiInterfacesName := range apiInterfacesNamesMap {
			apiInterfacesNames = append(apiInterfacesNames, apiInterfacesName)
		}

		// create a chainInfoEntry which includes the chain's name, ID and enabled interfaces
		chainInfoEntry := types.ShowAllChainsInfoStruct{ChainName: spec.GetName(), ChainID: spec.GetIndex(), EnabledApiInterfaces: apiInterfacesNames}

		// add the chainInfoEntry to the chainInfoList
		chainInfoList = append(chainInfoList, &chainInfoEntry)
	}

	return &types.QueryShowAllChainsResponse{ChainInfoList: chainInfoList}, nil
}
