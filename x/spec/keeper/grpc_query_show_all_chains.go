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
		// get the spec's name and chain ID
		chainName := spec.GetName()
		chainId := spec.GetIndex()

		expectedInterfaces := make(map[string]bool)
		fullspec, err := k.ExpandSpec(ctx, spec)
		if err != nil {
			return nil, err
		}
		// get the spec's expected interfaces
		expectedInterfaces = k.getExpectedInterfacesForSpecInner(&fullspec, expectedInterfaces)

		// copy the expectedInterfaces's keys (which are the interface names) to a string list
		apiInterfacesNames := getInterfacesNamesFromMap(expectedInterfaces)

		apiCount := uint64(0)

		for _, api := range fullspec.Apis {
			if api.Enabled {
				apiCount++
			}
		}
		// create a chainInfoEntry which includes the chain's name, ID and enabled interfaces
		chainInfoEntry := types.ShowAllChainsInfoStruct{ChainName: chainName, ChainID: chainId, EnabledApiInterfaces: apiInterfacesNames, ApiCount: apiCount}

		// add the chainInfoEntry to the chainInfoList
		chainInfoList = append(chainInfoList, &chainInfoEntry)
	}

	return &types.QueryShowAllChainsResponse{ChainInfoList: chainInfoList}, nil
}

func getInterfacesNamesFromMap(expectedInterfaces map[string]bool) []string {
	var apiInterfacesNames []string
	for apiInterfacesName := range expectedInterfaces {
		apiInterfacesNames = append(apiInterfacesNames, apiInterfacesName)
	}

	return apiInterfacesNames
}
