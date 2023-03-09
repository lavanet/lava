package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ShowChainInfo(goCtx context.Context, req *types.QueryShowChainInfoRequest) (*types.QueryShowChainInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var apiInterfacesStructList []*types.ApiList
	var interfaceList []string
	var chainID string
	foundChain := false

	allSpec := k.GetAllSpec(ctx)
	for _, spec := range allSpec {
		if !spec.Enabled {
			continue
		}

		// get info by chain name
		if spec.GetName() == req.GetChainName() {
			foundChain = true

			// get chain ID
			chainID = spec.GetIndex()

			// get API methods (includes their interfaces)
			apis := spec.GetApis()
			for _, api := range apis {
				apiInterfaces := api.GetApiInterfaces()

				// iterate over APIs
				for _, apiInterface := range apiInterfaces {
					interfaceApiStructIndex := checkInterfaceExistence(apiInterfacesStructList, apiInterface.GetInterface())

					// found an API with a new interface
					if interfaceApiStructIndex == -1 {
						interfaceList = append(interfaceList, apiInterface.GetInterface())
						apiMethods := []string{api.GetName()}
						tempApiInterfaceStruct := types.ApiList{Interface: apiInterface.GetInterface(), SupportedApis: apiMethods}
						apiInterfacesStructList = append(apiInterfacesStructList, &tempApiInterfaceStruct)
					} else {
						// found an API with an existent interface
						apiInterfacesStructList[interfaceApiStructIndex].SupportedApis = append(apiInterfacesStructList[interfaceApiStructIndex].SupportedApis, api.GetName())
					}
				}
			}

			// found the chain, there is no need to further iterate
			break
		}
	}

	// Didn't find the chain
	if !foundChain {
		return nil, sdkerrors.Wrapf(types.ErrChainNameNotFound, "%s", req.GetChainName())
	}

	return &types.QueryShowChainInfoResponse{ChainID: chainID, Interfaces: interfaceList, SupportedApisInterfaceList: apiInterfacesStructList}, nil
}

// checks if an interface is already written in one of the ApiList structs and returns the index of its incident
func checkInterfaceExistence(apis []*types.ApiList, interfaceToCheck string) (index int) {
	for i, api := range apis {
		if interfaceToCheck == api.Interface {
			return i
		}
	}
	return -1
}
