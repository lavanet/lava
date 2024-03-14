package keeper

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/spec/types"
)

// SetSpec set a specific Spec in the store from its index
func (k Keeper) SetSpec(ctx sdk.Context, spec types.Spec) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecKeyPrefix))
	b := k.cdc.MustMarshal(&spec)
	store.Set(types.SpecKey(
		spec.Index,
	), b)
}

// GetSpec returns a Spec from its index
func (k Keeper) GetSpec(
	ctx sdk.Context,
	index string,
) (val types.Spec, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecKeyPrefix))

	b := store.Get(types.SpecKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveSpec removes a Spec from the store
func (k Keeper) RemoveSpec(
	ctx sdk.Context,
	index string,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecKeyPrefix))
	store.Delete(types.SpecKey(
		index,
	))
}

// GetAllSpec returns all Spec
func (k Keeper) GetAllSpec(ctx sdk.Context) (list []types.Spec) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Spec
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) GetExpandedSpec(ctx sdk.Context, index string) (types.Spec, error) {
	spec, found := k.GetSpec(ctx, index)
	if found {
		return k.ExpandSpec(ctx, spec)
	}
	return types.Spec{}, fmt.Errorf("no matching spec %s", index)
}

// ExpandSpec takes a (raw) Spec and expands the "imports" field of the spec
// -if needed, recursively- to add to the current Spec those additional APIs
// from the imported Spec(s). It returns the expanded Spec.
func (k Keeper) ExpandSpec(ctx sdk.Context, spec types.Spec) (types.Spec, error) {
	depends := map[string]bool{spec.Index: true}
	inherit := map[string]bool{}

	details, err := k.doExpandSpec(ctx, &spec, depends, &inherit, spec.Index)
	if err != nil {
		return spec, utils.LavaFormatError("spec expand failed", err,
			utils.Attribute{Key: "imports", Value: details},
		)
	}

	return spec, nil
}

// RefreshSpec checks which one Spec inherits from another (just recently
// updated) Spec, and if so updates the the BlockLastUpdated of the former.
func (k Keeper) RefreshSpec(ctx sdk.Context, spec types.Spec, ancestors []types.Spec) ([]string, error) {
	depends := map[string]bool{spec.Index: true}
	inherit := map[string]bool{}

	if details, err := k.doExpandSpec(ctx, &spec, depends, &inherit, spec.Index); err != nil {
		return nil, utils.LavaFormatWarning("spec refresh failed (import)", err,
			utils.Attribute{Key: "imports", Value: details},
		)
	}

	if details, err := spec.ValidateSpec(k.MaxCU(ctx)); err != nil {
		details["invalidates"] = spec.Index
		attrs := utils.StringMapToAttributes(details)
		return nil, utils.LavaFormatWarning("spec refresh failed (invalidate)", err, attrs...)
	}

	var inherited []string
	for _, ancestor := range ancestors {
		if _, ok := inherit[ancestor.Index]; ok {
			inherited = append(inherited, ancestor.Index)
		}
	}

	if len(inherited) > 0 {
		// to get the original spec before the expanding
		spec, _ = k.GetSpec(ctx, spec.Index)
		spec.BlockLastUpdated = uint64(ctx.BlockHeight())
		k.SetSpec(ctx, spec)
	}

	return inherited, nil
}

// doExpandSpec performs the actual work and recusion for ExpandSpec above.
func (k Keeper) doExpandSpec(
	ctx sdk.Context,
	spec *types.Spec,
	depends map[string]bool,
	inherit *map[string]bool,
	details string,
) (string, error) {
	parentsCollections := map[types.CollectionData][]*types.ApiCollection{}

	if len(spec.Imports) != 0 {
		var parents []types.Spec

		// update (cumulative) inherit
		for _, index := range spec.Imports {
			(*inherit)[index] = true
		}

		// visual markers when import deepens
		details += "->["

		// recursion to get all parent specs (DFS)
		comma := ""
		for _, index := range spec.Imports {
			imported, found := k.GetSpec(ctx, index)
			// import of unknown Spec not allowed
			if !found {
				details += fmt.Sprintf("%s%s(unknown)", comma, index)
				return details, fmt.Errorf("imported spec unknown: %s", index)
			}

			details += fmt.Sprintf("%s%s", comma, index)

			// loop in the recursion not allowed
			if _, found := depends[index]; found {
				return details, fmt.Errorf("import loops not allowed for spec: %s", index)
			}

			depends[index] = true
			details, err := k.doExpandSpec(ctx, &imported, depends, inherit, details)
			if err != nil {
				return details, err
			}
			delete(depends, index)

			parents = append(parents, imported)
			comma = ","
		}

		details += "]"

		for _, parent := range parents {
			for _, parentCollection := range parent.ApiCollections {
				// ignore disabled apiCollections
				if !parentCollection.Enabled {
					continue
				}
				if parentsCollections[parentCollection.CollectionData] == nil {
					parentsCollections[parentCollection.CollectionData] = []*types.ApiCollection{}
				}
				parentsCollections[parentCollection.CollectionData] = append(parentsCollections[parentCollection.CollectionData], parentCollection)
			}
		}
	}

	myCollections := map[types.CollectionData]*types.ApiCollection{}
	for _, collection := range spec.ApiCollections {
		myCollections[collection.CollectionData] = collection
	}

	for _, collection := range spec.ApiCollections {
		err := collection.InheritAllFields(myCollections, parentsCollections[collection.CollectionData])
		if err != nil {
			return details, err
		}
		delete(parentsCollections, collection.CollectionData)
	}

	// combine left over apis not overwritten by current spec
	err := spec.CombineCollections(parentsCollections)
	if err != nil {
		return details, err
	}

	return details, nil
}

func (k Keeper) ValidateSpec(ctx sdk.Context, spec types.Spec) (map[string]string, error) {
	spec, err := k.ExpandSpec(ctx, spec)
	if err != nil {
		details := map[string]string{"imports": strings.Join(spec.Imports, ",")}
		return details, err
	}

	if err := utils.ValidateCoins(ctx, k.stakingKeeper.BondDenom(ctx), spec.MinStakeProvider, false); err != nil {
		details := map[string]string{
			"spec":    spec.Name,
			"status":  strconv.FormatBool(spec.Enabled),
			"chainID": spec.Index,
		}

		return details, utils.LavaFormatError("MinStakeProvider is invalid", err)
	}

	details, err := spec.ValidateSpec(k.MaxCU(ctx))
	if err != nil {
		return details, err
	}

	return details, nil
}

// IsSpecFoundAndActive returns whether a spec name is a valid spec in the consensus.
// It returns whether the spec is active (and found), whether it was found, and the
// provider's type (e.g. dynamic/static).
func (k Keeper) IsSpecFoundAndActive(ctx sdk.Context, chainID string) (foundAndActive, found bool, providersType types.Spec_ProvidersTypes) {
	spec, found := k.GetSpec(ctx, chainID)
	foundAndActive = false
	if found {
		foundAndActive = spec.Enabled
		providersType = spec.ProvidersTypes
	}
	return
}

// GetSpecIDBytes returns the byte representation of the ID
func GetSpecIDBytes(id uint64) []byte {
	return sigs.EncodeUint64(id)
}

// GetSpecIDFromBytes returns ID in uint64 format from a byte array
func GetSpecIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) GetAllChainIDs(ctx sdk.Context) (chainIDs []string) {
	// TODO: make this with an iterator
	allSpecs := k.GetAllSpec(ctx)
	for _, spec := range allSpecs {
		chainIDs = append(chainIDs, spec.Index)
	}
	return
}

// returns map[apiInterface][]addons
func (k Keeper) GetExpectedServicesForSpec(ctx sdk.Context, chainID string, mandatory bool) (expectedServices map[epochstoragetypes.EndpointService]struct{}, err error) {
	var spec types.Spec
	spec, found := k.GetSpec(ctx, chainID)
	if found && spec.Enabled {
		spec, err := k.ExpandSpec(ctx, spec)
		if err != nil {
			// spec expansion should work because all specs on chain must be valid;
			// to avoid panic return an error so the caller can bail.
			return nil, utils.LavaFormatError("critical: failed to expand spec on chain", err,
				utils.Attribute{Key: "chainID", Value: chainID},
			)
		}
		expectedServices = k.GetExpectedServicesForExpandedSpec(spec, mandatory)
		return expectedServices, nil
	}
	return nil, utils.LavaFormatWarning("spec not found or not enabled in GetExpectedServicesForSpec", nil,
		utils.Attribute{Key: "chainID", Value: chainID})
}

func (k Keeper) GetExpectedServicesForExpandedSpec(expandedSpec types.Spec, mandatory bool) map[epochstoragetypes.EndpointService]struct{} {
	expectedServices := make(map[epochstoragetypes.EndpointService]struct{})
	for _, apiCollection := range expandedSpec.ApiCollections {
		if apiCollection.Enabled && (!mandatory || apiCollection.CollectionData.AddOn == "") { // if mandatory is turned on only regard empty addons as expected interfaces for spec
			service := epochstoragetypes.EndpointService{
				ApiInterface: apiCollection.CollectionData.ApiInterface,
				Addon:        apiCollection.CollectionData.AddOn,
				Extension:    "",
			}
			// if this is an optional apiInterface, we set addon as ""
			if apiCollection.CollectionData.AddOn == apiCollection.CollectionData.ApiInterface {
				service.Addon = ""
			}
			expectedServices[service] = struct{}{}
			// add extensions when not asking for mandatory
			if !mandatory {
				for _, extension := range apiCollection.Extensions {
					service.Extension = extension.Name
					expectedServices[service] = struct{}{}
				}
			}
		}
	}
	return expectedServices
}

func (k Keeper) IsFinalizedBlock(ctx sdk.Context, chainID string, requestedBlock, latestBlock int64) bool {
	spec, found := k.GetSpec(ctx, chainID)
	if !found {
		return false
	}
	return types.IsFinalizedBlock(requestedBlock, latestBlock, int64(spec.BlockDistanceForFinalizedData))
}

// returns the reward per contributor
func (k Keeper) GetContributorReward(ctx sdk.Context, chainId string) (contributors []sdk.AccAddress, percentage math.LegacyDec) {
	spec, found := k.GetSpec(ctx, chainId)
	if !found {
		return nil, math.LegacyZeroDec()
	}
	if spec.ContributorPercentage == nil || len(spec.Contributor) == 0 {
		return nil, math.LegacyZeroDec()
	}
	for _, contributorAddrSt := range spec.Contributor {
		contributorAddr, err := sdk.AccAddressFromBech32(contributorAddrSt)
		if err != nil {
			// this should never happen as it's verified
			utils.LavaFormatError("invalid contributor address", err)
			return nil, math.LegacyZeroDec()
		}
		contributors = append(contributors, contributorAddr)
	}
	return contributors, *spec.ContributorPercentage
}

func (k Keeper) GetMinStake(ctx sdk.Context, chainID string) sdk.Coin {
	spec, found := k.GetSpec(ctx, chainID)
	if !found {
		utils.LavaFormatError("critical: failed to get spec for chainID",
			fmt.Errorf("unknown chainID"),
			utils.Attribute{Key: "chainID", Value: chainID},
		)
	}

	return spec.MinStakeProvider
}
