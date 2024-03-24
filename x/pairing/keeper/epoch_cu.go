package keeper

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

// UniqueEpochSession is used to detect double spend attacks
// It's kept in a epoch-prefixed store with a unique index: provider, project ID, chain ID and session ID
//
// ProviderEpochCu is used to track the CU of a specific provider in a specific epoch
// It's kept in a epoch-prefixed store with a unique index: provider address
//
// ProviderConsumerEpochCu is used to track the CU between a specific provider and
// consumer in a specific epoch
// It's kept in a epoch-prefixed store with a unique index: provider and project ID

/* ########## UniqueEpochSession ############ */

// SetUniqueEpochSession sets a UniqueEpochSession in the store
func (k Keeper) SetUniqueEpochSession(ctx sdk.Context, provider string, project string, chainID string, sessionID uint64) {
	epoch := k.epochStorageKeeper.GetEpochStart(ctx)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.UniqueEpochSessionKeyPrefix(epoch))
	store.Set(types.UniqueEpochSessionKey(provider, project, chainID, sessionID), []byte{0})
}

// GetUniqueEpochSession checks if a UniqueEpochSession exists
func (k Keeper) GetUniqueEpochSession(ctx sdk.Context, provider string, project string, chainID string, sessionID uint64) bool {
	epoch := k.epochStorageKeeper.GetEpochStart(ctx)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.UniqueEpochSessionKeyPrefix(epoch))
	b := store.Get(types.UniqueEpochSessionKey(provider, project, chainID, sessionID))
	return b != nil
}

// RemoveUniqueEpochSessions removes all the UniqueEpochSession objects from the store for a specific epoch
func (k Keeper) RemoveUniqueEpochSessions(ctx sdk.Context, epoch uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.UniqueEpochSessionKeyPrefix(epoch))
	iterator := sdk.KVStorePrefixIterator(store, []byte{}) // Get an iterator with no prefix to iterate over all keys
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
}

// GetAllUniqueEpochSession gets all the UniqueEpochSession objects from the store for a specific epoch
func (k Keeper) GetAllUniqueEpochSession(ctx sdk.Context, epoch uint64) []string {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.UniqueEpochSessionKeyPrefix(epoch))

	iterator := sdk.KVStorePrefixIterator(store, []byte{}) // Get an iterator with no prefix to iterate over all keys
	defer iterator.Close()

	var keys []string
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		// Remove the prefix to get the actual UniqueEpochSession key
		uniqueEpochSessionKey := strings.TrimPrefix(string(key), string(types.UniqueEpochSessionKeyPrefix(epoch)))
		keys = append(keys, uniqueEpochSessionKey)
	}

	return keys
}

/* ########## ProviderEpochCu ############ */

// SetProviderEpochCu sets a ProviderEpochCu in the store
func (k Keeper) SetProviderEpochCu(ctx sdk.Context, provider string, providerEpochCu types.ProviderEpochCu) {
	epoch := k.epochStorageKeeper.GetEpochStart(ctx)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ProviderEpochCuKeyPrefix(epoch))
	b := k.cdc.MustMarshal(&providerEpochCu)
	store.Set(types.ProviderEpochCuKey(provider), b)
}

// GetProviderEpochCu returns a ProviderEpochCu for a specific epoch and provider
func (k Keeper) GetProviderEpochCu(ctx sdk.Context, epoch uint64, provider string) (val types.ProviderEpochCu, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ProviderEpochCuKeyPrefix(epoch))
	b := store.Get(types.ProviderEpochCuKey(provider))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveProviderEpochCu removes a ProviderEpochCu from the store
func (k Keeper) RemoveProviderEpochCu(ctx sdk.Context, epoch uint64, provider string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ProviderEpochCuKeyPrefix(epoch))
	store.Delete(types.ProviderEpochCuKey(provider))
}

/* ########## ProviderConsumerEpochCu ############ */

// SetProviderConsumerEpochCu sets a ProviderConsumerEpochCu in the store
func (k Keeper) SetProviderConsumerEpochCu(ctx sdk.Context, provider string, project string, ProviderConsumerEpochCu types.ProviderConsumerEpochCu) {
	epoch := k.epochStorageKeeper.GetEpochStart(ctx)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ProviderConsumerEpochCuKeyPrefix(epoch))
	b := k.cdc.MustMarshal(&ProviderConsumerEpochCu)
	store.Set(types.ProviderConsumerEpochCuKey(provider, project), b)
}

// GetProviderConsumerEpochCu returns a ProviderConsumerEpochCu for a specific epoch, provider and project
func (k Keeper) GetProviderConsumerEpochCu(ctx sdk.Context, epoch uint64, provider string, project string) (val types.ProviderConsumerEpochCu, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ProviderConsumerEpochCuKeyPrefix(epoch))
	b := store.Get(types.ProviderConsumerEpochCuKey(provider, project))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveProviderConsumerEpochCu removes a ProviderConsumerEpochCu from the store
func (k Keeper) RemoveProviderConsumerEpochCu(ctx sdk.Context, epoch uint64, provider string, project string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ProviderConsumerEpochCuKeyPrefix(epoch))
	store.Delete(types.ProviderConsumerEpochCuKey(provider, project))
}
