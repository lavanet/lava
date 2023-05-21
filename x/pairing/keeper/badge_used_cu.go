package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

// SetBadgeUsedCu set a specific badgeUsedCu in the store from its index
func (k Keeper) SetBadgeUsedCu(ctx sdk.Context, badgeUsedCu types.BadgeUsedCu) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BadgeUsedCuKeyPrefix))
	b := k.cdc.MustMarshal(&badgeUsedCu)
	store.Set(types.BadgeUsedCuKey(
		badgeUsedCu.BadgeUsedCuMapKey,
	), b)
}

// GetBadgeUsedCu returns a badgeUsedCu from its index
func (k Keeper) GetBadgeUsedCu(
	ctx sdk.Context,
	badgeUsedCuMapKey []byte,
) (val types.BadgeUsedCu, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BadgeUsedCuKeyPrefix))

	b := store.Get(types.BadgeUsedCuKey(
		badgeUsedCuMapKey,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveBadgeUsedCu removes a badgeUsedCu from the store
func (k Keeper) RemoveBadgeUsedCu(
	ctx sdk.Context,
	badgeUsedCuMapKey []byte,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BadgeUsedCuKeyPrefix))
	store.Delete(types.BadgeUsedCuKey(
		badgeUsedCuMapKey,
	))
}

// GetAllBadgeUsedCu returns all badgeUsedCu
func (k Keeper) GetAllBadgeUsedCu(ctx sdk.Context) (list []types.BadgeUsedCu) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BadgeUsedCuKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.BadgeUsedCu
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
