package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/rewards/types"
)

// GetIprpcRewardCount get the total number of IprpcReward
func (k Keeper) GetIprpcRewardCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.IprpcRewardsCountPrefix)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetIprpcRewardCount set the total number of IprpcReward
func (k Keeper) SetIprpcRewardCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.IprpcRewardsCountPrefix)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendIprpcReward appends a IprpcReward in the store with a new id and update the count
func (k Keeper) AppendIprpcReward(
	ctx sdk.Context,
	IprpcReward types.IprpcReward,
) uint64 {
	// Create the IprpcReward
	count := k.GetIprpcRewardCount(ctx)

	// Set the ID of the appended value
	IprpcReward.Id = count

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IprpcRewardPrefix))
	appendedValue := k.cdc.MustMarshal(&IprpcReward)
	store.Set(GetIprpcRewardIDBytes(IprpcReward.Id), appendedValue)

	// Update IprpcReward count
	k.SetIprpcRewardCount(ctx, count+1)

	return count
}

// SetIprpcReward set a specific IprpcReward in the store
func (k Keeper) SetIprpcReward(ctx sdk.Context, IprpcReward types.IprpcReward) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IprpcRewardPrefix))
	b := k.cdc.MustMarshal(&IprpcReward)
	store.Set(GetIprpcRewardIDBytes(IprpcReward.Id), b)
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
