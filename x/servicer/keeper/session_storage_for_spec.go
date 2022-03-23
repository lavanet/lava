package keeper

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

// SetSessionStorageForSpec set a specific sessionStorageForSpec in the store from its index
func (k Keeper) SetSessionStorageForSpec(ctx sdk.Context, sessionStorageForSpec types.SessionStorageForSpec) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SessionStorageForSpecKeyPrefix))
	b := k.cdc.MustMarshal(&sessionStorageForSpec)
	store.Set(types.SessionStorageForSpecKey(
		sessionStorageForSpec.Index,
	), b)
}

// GetSessionStorageForSpec returns a sessionStorageForSpec from its index
func (k Keeper) GetSessionStorageForSpec(
	ctx sdk.Context,
	index string,

) (val types.SessionStorageForSpec, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SessionStorageForSpecKeyPrefix))

	b := store.Get(types.SessionStorageForSpecKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveSessionStorageForSpec removes a sessionStorageForSpec from the store
func (k Keeper) RemoveSessionStorageForSpec(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SessionStorageForSpecKeyPrefix))
	store.Delete(types.SessionStorageForSpecKey(
		index,
	))
}

// GetAllSessionStorageForSpec returns all sessionStorageForSpec
func (k Keeper) GetAllSessionStorageForSpec(ctx sdk.Context) (list []types.SessionStorageForSpec) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SessionStorageForSpecKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.SessionStorageForSpec
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) RemoveStakeStorageInSession(ctx sdk.Context) (err error) {

	if uint64(ctx.BlockHeight()) < k.BlocksToSave(ctx) {
		return nil
	}
	block := types.BlockNum{Num: uint64(ctx.BlockHeight()) - k.BlocksToSave(ctx)}
	sessionStartForTarget, _, err := k.GetSessionStartForBlock(ctx, block)
	if err != nil || sessionStartForTarget == nil {
		return err
	}
	allSpecStakeStorages := k.GetAllSpecStakeStorage(ctx)
	for _, specStakeStorage := range allSpecStakeStorages {
		specName := specStakeStorage.Index
		k.RemoveSessionStorageForSpec(ctx, k.SessionStorageKey(*sessionStartForTarget, specName))
	}
	//TODO: on param change we are going to miss all the entries before the param change on the delete, this can be handled with the previousData that we have
	// another way is to just clean up if we have too many entries, old entries dont kill us just take up space
	return nil
}

func (k Keeper) StoreSpecStakeStorageInSession(ctx sdk.Context) (err error) {
	allSpecStakeStorages := k.GetAllSpecStakeStorage(ctx)
	for _, specStakeStorage := range allSpecStakeStorages {
		err = k.SetSpecStakeStorageInSessionStorageForSpec(ctx, specStakeStorage)
	}
	return
}

func (k Keeper) SetSpecStakeStorageInSessionStorageForSpec(ctx sdk.Context, specStakeStorage types.SpecStakeStorage) error {
	currentSessionStart, found := k.GetCurrentSessionStart(ctx)
	if !found {
		return fmt.Errorf("fail due to faulty GetCurrentSessionStart in keeper")
	}
	sessionStorageForSpec := types.SessionStorageForSpec{
		Index:        k.SessionStorageKey(currentSessionStart.Block, specStakeStorage.Index),
		StakeStorage: k.CopyStakeStorageContents(ctx, specStakeStorage.StakeStorage),
	}
	k.SetSessionStorageForSpec(ctx, sessionStorageForSpec)
	return nil
}

func (k Keeper) SessionStorageKey(currentSessionStart types.BlockNum, specName string) string {
	return strconv.FormatUint(currentSessionStart.Num, 16) + specName
}

func (k Keeper) GetSpecStakeStorageInSessionStorageForSpec(ctx sdk.Context, block types.BlockNum, specName string) (currentStorage *types.StakeStorage, previousStorageForOverlapping *types.StakeStorage, err error) {
	sessionStartForTarget, overlappingPreviousdSessionStart, err := k.GetSessionStartForBlock(ctx, block)
	if err != nil || sessionStartForTarget == nil {
		return nil, nil, err
	}
	sessionStorage, found := k.GetSessionStorageForSpec(ctx, k.SessionStorageKey(*sessionStartForTarget, specName))
	if !found {
		return nil, nil, fmt.Errorf("did not manage to get GetSessionStorageForSpec: %s spec name:", sessionStartForTarget, specName)
	}
	currentStorage = sessionStorage.StakeStorage
	if overlappingPreviousdSessionStart != nil {
		sessionStorage, found := k.GetSessionStorageForSpec(ctx, k.SessionStorageKey(*overlappingPreviousdSessionStart, specName))
		if found {
			previousStorageForOverlapping = sessionStorage.StakeStorage
		}
	}
	return
}
