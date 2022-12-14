package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	//TODO: make this with an iterator
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
		for _, api := range spec.Apis {
			for _, apiInterface := range api.ApiInterfaces {
				expectedInterfaces[apiInterface.Interface] = true
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
	return types.IsFinalizedBlock(requestedBlock, latestBlock, spec.FinalizationCriteria)
}
