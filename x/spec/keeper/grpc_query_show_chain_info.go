package keeper

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
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
	var optionalInterfaceList []string
	var chainID string

	allSpec := k.GetAllSpec(ctx)
	for _, spec := range allSpec {
		// get info by chain name
		if spec.GetName() == req.GetChainName() || spec.GetIndex() == req.GetChainName() {
			// get chain ID
			chainID = spec.GetIndex()

			fullspec, err := k.ExpandSpec(ctx, spec)
			if err != nil {
				return nil, err
			}
			// get the spec's expected interfaces
			expectedInterfaces := k.GetExpectedServicesForExpandedSpec(fullspec, true)

			mandatoryInterfaceList := getInterfacesNamesFromMap(expectedInterfaces)

			// get API methods (includes their interfaces)
			apisCollections := fullspec.GetApiCollections()
			for _, apiCollection := range apisCollections {
				if !apiCollection.Enabled {
					continue
				}
				apiInterface := apiCollection.CollectionData.ApiInterface

				apiMethods := []string{}
				// iterate over APIs
				if _, ok := expectedInterfaces[epochstoragetypes.EndpointService{
					ApiInterface: apiInterface,
					Addon:        "",
					Extension:    "",
				}]; !ok {
					optionalInterfaceList = append(optionalInterfaceList, apiInterface)
				}
				for _, api := range apiCollection.Apis {
					if !api.Enabled {
						continue
					}
					apiMethods = append(apiMethods, api.GetName())
				}
				apiInterfacesStructList = append(apiInterfacesStructList, &types.ApiList{
					Interface:     apiInterface,
					SupportedApis: apiMethods,
				})
			}

			// found the chain, there is no need to further iterate
			return &types.QueryShowChainInfoResponse{ChainID: chainID, Interfaces: mandatoryInterfaceList, SupportedApisInterfaceList: apiInterfacesStructList, OptionalInterfaces: optionalInterfaceList}, nil
		}
	}
	// Didn't find the chain
	return nil, sdkerrors.Wrapf(types.ErrChainNameNotFound, "%s", req.GetChainName())
}
