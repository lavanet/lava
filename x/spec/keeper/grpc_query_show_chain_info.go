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
		// get info by chain name
		if spec.GetName() == req.GetChainName() || spec.GetIndex() == req.GetChainName() {
			foundChain = true

			// get chain ID
			chainID = spec.GetIndex()

			// get API methods (includes their interfaces)
			apisCollections := spec.GetApiCollections()
			for _, apiCollection := range apisCollections {
				apiInterface := apiCollection.CollectionData.ApiInterface

				apiMethods := []string{}
				// iterate over APIs
				interfaceList = append(interfaceList, apiInterface)

				for _, api := range apiCollection.Apis {
					apiMethods = append(apiMethods, api.GetName())
				}
				apiInterfacesStructList = append(apiInterfacesStructList, &types.ApiList{
					Interface:     apiInterface,
					SupportedApis: apiMethods,
				})
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
