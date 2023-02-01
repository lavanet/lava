package keeper

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
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

	details, err := k.doExpandSpec(ctx, &spec, depends, spec.Index)
	if err != nil {
		details := map[string]string{"imports": details}
		return spec, utils.LavaError(ctx, k.Logger(ctx), "spec expand failed", details, err.Error())
	}
	return spec, nil
}

// doExpandSpec performs the actual work and recusion for ExpandSpec above.
func (k Keeper) doExpandSpec(ctx sdk.Context, spec *types.Spec, depends map[string]bool, details string) (string, error) {
	if len(spec.Imports) == 0 {
		return details, nil
	}

	var parents []types.Spec

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
		details, err := k.doExpandSpec(ctx, &imported, depends, details)
		if err != nil {
			return details, err
		}
		delete(depends, index)

		parents = append(parents, imported)
		comma = ","
	}

	details += "]"

	var mergedApis []types.ServiceApi
	mergedApisMap := make(map[string]types.ServiceApi)

	// collect all parents' Specs' APIs
	for _, imported := range parents {
		for _, api := range imported.Apis {
			if api.Enabled {
				// duplicate API(s) not allowed
				if _, found := mergedApisMap[api.Name]; found {
					return details, fmt.Errorf("duplicate imported api: %s (in spec: %s)", api.Name, imported.Index)
				}
				mergedApisMap[api.Name] = api
				mergedApis = append(mergedApis, api)
			}
		}
	}

	currentApis := make(map[string]bool)
	for _, api := range spec.Apis {
		currentApis[api.Name] = true
	}

	// merge collected APIs into current spec's APIs (unless overridden)
	for _, api := range mergedApis {
		if _, found := currentApis[api.Name]; !found {
			spec.Apis = append(spec.Apis, api)
		}
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
// first return value is found and active, second argument is found only
func (k Keeper) IsSpecFoundAndActive(ctx sdk.Context, chainID string) (foundAndActive bool, found bool) {
	spec, found := k.GetSpec(ctx, chainID)
	foundAndActive = false
	if found {
		foundAndActive = spec.Enabled
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

func (k Keeper) GetExpectedInterfacesForSpec(ctx sdk.Context, chainID string) (expectedInterfaces map[string]bool) {
	expectedInterfaces = make(map[string]bool)
	spec, found := k.GetSpec(ctx, chainID)
	if found && spec.Enabled {
		spec, err := k.ExpandSpec(ctx, spec)
		if err != nil { // should not happen! (all specs on chain must be valid)
			panic(err)
		}
		for _, api := range spec.Apis {
			if api.Enabled {
				for _, apiInterface := range api.ApiInterfaces {
					expectedInterfaces[apiInterface.Interface] = true
				}
			}
		}
	}
	return
}

func (k Keeper) IsFinalizedBlock(ctx sdk.Context, chainID string, requestedBlock int64, latestBlock int64) bool {
	spec, found := k.GetSpec(ctx, chainID)
	if !found {
		return false
	}
	return types.IsFinalizedBlock(requestedBlock, latestBlock, spec.BlockDistanceForFinalizedData)
}
