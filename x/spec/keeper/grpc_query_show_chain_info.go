package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/spec/types"
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
	var spec types.Spec

	for _, s := range k.GetAllSpec(ctx) {
		if s.Name == req.ChainName || s.Index == req.ChainName {
			spec = s
			break
		}
	}

	if spec.Index == "" {
		return nil, fmt.Errorf("spec not found")
	}

	fullspec, err := k.ExpandSpec(ctx, spec)
	if err != nil {
		return nil, err
	}

	// get the spec's expected interfaces (mandatory)
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
			Addon:         apiCollection.CollectionData.AddOn,
		})
	}

	return &types.QueryShowChainInfoResponse{ChainID: spec.Index, Interfaces: mandatoryInterfaceList, SupportedApisInterfaceList: apiInterfacesStructList, OptionalInterfaces: optionalInterfaceList}, nil
}
