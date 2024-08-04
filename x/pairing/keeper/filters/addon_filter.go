package filters

import (
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

type AddonFilter struct {
	requirements map[spectypes.CollectionData]map[string]struct{} // key is collectionData value is a map of required extensions
	mix          bool
}

func (f *AddonFilter) IsMix() bool {
	return f.mix
}

func (f *AddonFilter) InitFilter(strictestPolicy planstypes.Policy) (bool, []Filter) {
	mixFilters := map[string]AddonFilter{} // map of addon/extension name -> addon filter
	if len(strictestPolicy.ChainPolicies) > 0 {
		chainPolicy := strictestPolicy.ChainPolicies[0] // only the specific spec chain policy is there
		if len(chainPolicy.Requirements) > 0 {
			requirements := map[spectypes.CollectionData]map[string]struct{}{}
			for _, requirement := range chainPolicy.Requirements {
				if requirement.Mixed {
					// even one requirement as mix sets this filter as a mix filter
					f.mix = true
				}

				// start collecting the requirements for addons/extensions from the policy
				if requirement.Differentiator() != "" {
					if _, ok := requirements[requirement.Collection]; !ok {
						requirements[requirement.Collection] = map[string]struct{}{}
					}
					for _, extension := range requirement.Extensions {
						// register requirement of couple addon-extension
						requirements[requirement.Collection][extension] = struct{}{}

						// if the requirement is mixed and it has and addon and(!) extensions
						// create sub filters with mix=true that get all the addons/extensions separately
						// for example: if the policy defines we need addon1 with ext1,ext2 extensions,
						// the regular filter will be the combination addon+ext1+ext2 and the new sub mix
						// filters will be only with addon, only with ext1 and only with ext2
						if requirement.Mixed {
							// add addon only sub mix filter
							addonNoExtensionMixFilter := AddonFilter{
								requirements: map[spectypes.CollectionData]map[string]struct{}{
									requirement.Collection: {"": {}},
								},
								mix: true,
							}
							if _, ok := mixFilters[requirement.Collection.AddOn]; !ok {
								mixFilters[requirement.Collection.AddOn] = addonNoExtensionMixFilter
							}

							// add extension only sub mix filter
							noAddonCollection := requirement.Collection
							noAddonCollection.AddOn = ""
							extensionNoAddonMixFilter := AddonFilter{
								requirements: map[spectypes.CollectionData]map[string]struct{}{
									noAddonCollection: {extension: {}},
								},
								mix: true,
							}
							if _, ok := mixFilters[extension]; !ok {
								mixFilters[extension] = extensionNoAddonMixFilter
							}
						}
					}
				}
			}
			if len(requirements) > 0 {
				// update filter full requirements
				f.requirements = requirements

				// construct mix filters list -> sort map keys, append to list
				filters := []Filter{}
				keys := []string{}
				for key := range mixFilters {
					keys = append(keys, key)
				}
				sort.Strings(keys)

				for _, key := range keys {
					filters = append(filters, convertAddonFilterToFilter(mixFilters[key]))
				}
				return true, filters
			}
		}
	}
	return false, nil
}

func convertAddonFilterToFilter(af AddonFilter) Filter {
	return Filter(&af)
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
