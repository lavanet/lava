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
		var apiInterfacesNames []string

		// iterate over the APIs
		for _, api := range apis {
			// get the API interfaces
			apiInterfaces := api.GetApiInterfaces()

			// iterate over the API interfaces
			for _, apiInterface := range apiInterfaces {
				// get the interface's name
				apiInterfaceName := apiInterface.GetInterface()

				// if the name wasn't already added to the apiInterfacesNames list, add it
				if !checkIfInterfaceInList(apiInterfacesNames, apiInterfaceName) {
					apiInterfacesNames = append(apiInterfacesNames, apiInterfaceName)
				}
			}
		}

		// create a chainInfoEntry which includes the chain's name, ID and enabled interfaces
		chainInfoEntry := types.ShowAllChainsInfoStruct{ChainName: spec.GetName(), ChainID: spec.GetIndex(), EnabledApiInterfaces: apiInterfacesNames}

		// add the chainInfoEntry to the chainInfoList
		chainInfoList = append(chainInfoList, &chainInfoEntry)
	}

	return &types.QueryShowAllChainsResponse{ChainInfoList: chainInfoList}, nil
}

func checkIfInterfaceInList(apiInterfacesNames []string, apiInterfaceToCheck string) bool {
	for _, apiInterface := range apiInterfacesNames {
		if apiInterfaceToCheck == apiInterface {
			return true
		}
	}
	return false
}
