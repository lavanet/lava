package keeper

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
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

	details, err := spec.ValidateSpec(k.MaxCU(ctx))
	if err != nil {
		return details, err
	}

	return details, nil
}

// returns whether a spec name is a valid spec in the consensus
// first return value is found and active, second argument is found only, third argument is the provider's type (dynamic/static)
func (k Keeper) IsSpecFoundAndActive(ctx sdk.Context, chainID string) (foundAndActive bool, found bool, providersType types.Spec_ProvidersTypes) {
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
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
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
func (k Keeper) GetExpectedInterfacesForSpec(ctx sdk.Context, chainID string, mandatory bool) (expectedInterfaces map[epochstoragetypes.EndpointService]struct{}, err error) {
	expectedInterfaces = make(map[epochstoragetypes.EndpointService]struct{})
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
		expectedInterfaces = k.getExpectedInterfacesForSpecInner(&spec, expectedInterfaces, mandatory)
		return expectedInterfaces, nil
	}
	return nil, utils.LavaFormatWarning("spec not found or not enabled in GetExpectedInterfacesForSpec", nil,
		utils.Attribute{Key: "chainID", Value: chainID})
}

func (k Keeper) getExpectedInterfacesForSpecInner(spec *types.Spec, expectedInterfaces map[epochstoragetypes.EndpointService]struct{}, mandatory bool) map[epochstoragetypes.EndpointService]struct{} {
	for _, apiCollection := range spec.ApiCollections {
		if apiCollection.Enabled && (!mandatory || apiCollection.CollectionData.AddOn == "") { // if mandatory is turned on only regard empty addons as expected interfaces for spec
			service := epochstoragetypes.EndpointService{
				ApiInterface: apiCollection.CollectionData.ApiInterface,
				Addon:        apiCollection.CollectionData.AddOn,
			}
			// if this is an optional apiInterface, we set addon as ""
			if apiCollection.CollectionData.AddOn == apiCollection.CollectionData.ApiInterface {
				service.Addon = ""
			}
			expectedInterfaces[service] = struct{}{}
		}
	}
	return expectedInterfaces
}

func (k Keeper) IsFinalizedBlock(ctx sdk.Context, chainID string, requestedBlock int64, latestBlock int64) bool {
	spec, found := k.GetSpec(ctx, chainID)
	if !found {
		return false
	}
	return types.IsFinalizedBlock(requestedBlock, latestBlock, spec.BlockDistanceForFinalizedData)
}
