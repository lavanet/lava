package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/conflict/types"
)

// SetConflictVote set a specific conflictVote in the store from its index
func (k Keeper) SetConflictVote(ctx sdk.Context, conflictVote types.ConflictVote) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ConflictVoteKeyPrefix))
	b := k.cdc.MustMarshal(&conflictVote)
	store.Set(types.ConflictVoteKey(
		conflictVote.Index,
	), b)
}

// GetConflictVote returns a conflictVote from its index
func (k Keeper) GetConflictVote(
	ctx sdk.Context,
	index string,
) (val types.ConflictVote, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ConflictVoteKeyPrefix))

	b := store.Get(types.ConflictVoteKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveConflictVote removes a conflictVote from the store
func (k Keeper) RemoveConflictVote(
	ctx sdk.Context,
	index string,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ConflictVoteKeyPrefix))
	store.Delete(types.ConflictVoteKey(
		index,
	))
}

// GetAllConflictVote returns all conflictVote
func (k Keeper) GetAllConflictVote(ctx sdk.Context) (list []types.ConflictVote) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ConflictVoteKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.ConflictVote
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
