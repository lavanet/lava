package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/rewards/types"
)

// GetIprpcRewardsCurrentId get the total number of IprpcReward
func (k Keeper) GetIprpcRewardsCurrentId(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.IprpcRewardsCurrentPrefix)
	bz := store.Get(byteKey)

	// Current doesn't exist: no element
	if bz == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetIprpcRewardsCurrentId set the total number of IprpcReward
func (k Keeper) SetIprpcRewardsCurrentId(ctx sdk.Context, current uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.IprpcRewardsCurrentPrefix)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, current)
	store.Set(byteKey, bz)
}

// SetIprpcReward set a specific IprpcReward in the store
func (k Keeper) SetIprpcReward(ctx sdk.Context, iprpcReward types.IprpcReward) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IprpcRewardPrefix))
	b := k.cdc.MustMarshal(&iprpcReward)
	store.Set(GetIprpcRewardIDBytes(iprpcReward.Id), b)
}

// GetIprpcReward returns a IprpcReward from its id
func (k Keeper) GetIprpcReward(ctx sdk.Context, id uint64) (val types.IprpcReward, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IprpcRewardPrefix))
	b := store.Get(GetIprpcRewardIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveIprpcReward removes a IprpcReward from the store
func (k Keeper) RemoveIprpcReward(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IprpcRewardPrefix))
	store.Delete(GetIprpcRewardIDBytes(id))
}

// GetAllIprpcReward returns all IprpcReward
func (k Keeper) GetAllIprpcReward(ctx sdk.Context) (list []types.IprpcReward) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IprpcRewardPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.IprpcReward
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetIprpcRewardIDBytes returns the byte representation of the ID
func GetIprpcRewardIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetIprpcRewardIDFromBytes returns ID in uint64 format from a byte array
func GetIprpcRewardIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}

// PopIprpcReward gets the lowest id IprpcReward object and removes it
func (k Keeper) PopIprpcReward(ctx sdk.Context) (types.IprpcReward, bool) {
	current := k.GetIprpcRewardsCurrentId(ctx)
	k.SetIprpcRewardsCurrentId(ctx, current+1)
	defer k.RemoveIprpcReward(ctx, current)
	return k.GetIprpcReward(ctx, current)
}

// GetCurrentIprpcReward gets the lowest id IprpcReward object
func (k Keeper) GetCurrentIprpcReward(ctx sdk.Context) (types.IprpcReward, bool) {
	current := k.GetIprpcRewardsCurrentId(ctx)
	return k.GetIprpcReward(ctx, current)
}
