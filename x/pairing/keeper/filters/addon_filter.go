package filters

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type AddonFilter struct {
	requirements map[spectypes.CollectionData]map[string]struct{} // key is collectionData value is a map of required extensions
}

func (f *AddonFilter) InitFilter(strictestPolicy planstypes.Policy) bool {
	if len(strictestPolicy.ChainPolicies) > 0 {
		chainPolicy := strictestPolicy.ChainPolicies[0] // only the specific spec chain policy is there
		if len(chainPolicy.Requirements) > 0 {
			requirements := map[spectypes.CollectionData]map[string]struct{}{}
			for _, requirement := range chainPolicy.Requirements {
				if requirement.Comparable() != "" {
					if _, ok := requirements[requirement.Collection]; !ok {
						requirements[requirement.Collection] = map[string]struct{}{}
					}
					for _, extension := range requirement.Extensions {
						requirements[requirement.Collection][extension] = struct{}{}
					}

				}
			}
			if len(requirements) > 0 {
				f.requirements = requirements
				return true
			}
		}
	}
	return false
}

func (f *AddonFilter) Filter(ctx sdk.Context, providers []epochstoragetypes.StakeEntry, currentEpoch uint64) []bool {
	filterResult := make([]bool, len(providers))
	for i, provider := range providers {
		if isRequirementSupported(f.requirements, provider.GetSupportedServices()) {
			filterResult[i] = true
		}
	}

	return filterResult
}

func isRequirementSupported(requirements map[spectypes.CollectionData]map[string]struct{}, services []epochstoragetypes.EndpointService) bool {
	utils.LavaFormatDebug("DEBUG", utils.Attribute{
		Key:   "requirements",
		Value: requirements,
	}, utils.Attribute{
		Key:   "services",
		Value: services,
	})
requirementsLoop:
	for requirement, requiredExtensions := range requirements {
		supportedExtensions := map[string]struct{}{}
		requiredExtensionsLen := len(requiredExtensions)
		// assumes requiredExtensions are sorted and
		for _, service := range services {
			fullfilAddonRequirement := service.Addon == requirement.AddOn && service.ApiInterface == requirement.ApiInterface
			fullfilOptionalApiInterfaceRequirement := service.ApiInterface == requirement.ApiInterface && requirement.AddOn == service.ApiInterface && service.Addon == ""
			if fullfilAddonRequirement || fullfilOptionalApiInterfaceRequirement {
				if _, ok := requiredExtensions[service.Extension]; ok {
					supportedExtensions[service.Extension] = struct{}{}
				}
				if len(supportedExtensions) >= requiredExtensionsLen {
					// this endpoint fulfills the requirement
					continue requirementsLoop
				}
			}
		}
		// no endpoint fulfills the requirement
		return false
	}
	return true
}
