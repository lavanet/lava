package filters

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type AddonFilter struct {
	collections map[spectypes.CollectionData]struct{}
}

func (f *AddonFilter) InitFilter(strictestPolicy planstypes.Policy) bool {
	if len(strictestPolicy.ChainPolicies) > 0 {
		chainPolicy := strictestPolicy.ChainPolicies[0] // only the specific spec chain policy is there
		if len(chainPolicy.Collections) > 0 {
			collections := map[spectypes.CollectionData]struct{}{}
			for _, collection := range chainPolicy.Collections {
				if collection.AddOn != "" {
					collections[collection] = struct{}{}
				}
			}
			if len(collections) > 0 {
				f.collections = collections
				return true
			}
		}
	}
	return false
}

func (f *AddonFilter) Filter(ctx sdk.Context, providers []epochstoragetypes.StakeEntry, currentEpoch uint64) []bool {
	filterResult := make([]bool, len(providers))
	for i, provider := range providers {
		if isAddonSupported(f.collections, provider.GetSupportedServices()) {
			filterResult[i] = true
		}
	}

	return filterResult
}

func isAddonSupported(requirements map[spectypes.CollectionData]struct{}, endpoints []epochstoragetypes.EndpointService) bool {
	utils.LavaFormatDebug("DEBUG", utils.Attribute{
		Key:   "requirements",
		Value: requirements,
	}, utils.Attribute{
		Key:   "endpoints",
		Value: endpoints,
	})
requirementsLoop:
	for requirements := range requirements {
		for _, endpoint := range endpoints {
			if endpoint.Addon == requirements.AddOn && endpoint.ApiInterface == requirements.ApiInterface {
				// this endpoint fulfills the requirement
				continue requirementsLoop
			}
			// for optional apiInterfaces addon is ""
			if endpoint.ApiInterface == requirements.ApiInterface && requirements.AddOn == endpoint.ApiInterface && endpoint.Addon == "" {
				// this endpoint supports the optional ApiInterface
				continue requirementsLoop
			}
		}
		// no endpoint fulfills the requirement
		return false
	}
	return true
}
