package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/pairing/types"
	timertypes "github.com/lavanet/lava/v2/x/timerstore/types"
)

// SetBadgeUsedCu set a specific badgeUsedCu in the store from its index
func (k Keeper) SetBadgeUsedCu(ctx sdk.Context, badgeUsedCu types.BadgeUsedCu) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BadgeUsedCuKeyPrefix))
	b := k.cdc.MustMarshal(&badgeUsedCu)
	store.Set(badgeUsedCu.BadgeUsedCuKey, b)
}

// GetBadgeUsedCu returns a badgeUsedCu from its index
func (k Keeper) GetBadgeUsedCu(
	ctx sdk.Context,
	badgeUsedCuKey []byte,
) (val types.BadgeUsedCu, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BadgeUsedCuKeyPrefix))

	b := store.Get(badgeUsedCuKey)
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveBadgeUsedCu removes a badgeUsedCu from the store
func (k Keeper) RemoveBadgeUsedCu(
	ctx sdk.Context,
	badgeUsedCuKey []byte,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BadgeUsedCuKeyPrefix))
	if store.Has(badgeUsedCuKey) {
		store.Delete(badgeUsedCuKey)
	} else {
		// badge (epoch) timer has expired for an unknown badge: either the
		// timer was set wrongly, or the badge was incorrectly removed; and
		// we cannot even return an error about it.
		utils.LavaFormatError("critical: epoch expiry for unknown badge, skipping",
			fmt.Errorf("badge not found"),
			utils.Attribute{Key: "badge", Value: string(badgeUsedCuKey)},
			utils.Attribute{Key: "block", Value: ctx.BlockHeight()},
		)
		return
	}
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

func (k Keeper) BadgeUsedCuExpiry(ctx sdk.Context, badge types.Badge) uint64 {
	blocksToSave, err := k.epochStorageKeeper.BlocksToSave(ctx, badge.Epoch)
	if err != nil {
		utils.LavaFormatError("critical: BadgeUsedCuExpiry failed to get BlocksToSave", err,
			utils.LogAttr("badge", badge.Address),
			utils.LogAttr("block", ctx.BlockHeight()),
		)
		// on error, blocksToSave will be zero, so to avoid immediate expiry (and user
		// discontent) use a reasonable default: the current EpochToSave * EpochBlocks
		blocksToSave = k.epochStorageKeeper.BlocksToSaveRaw(ctx)
	}

	return badge.Epoch + blocksToSave
}

// InitBadgeTimers imports badges timers data (from genesis)
func (k Keeper) InitBadgeTimers(ctx sdk.Context, gs timertypes.GenesisState) {
	k.badgeTimerStore.Init(ctx, gs)
}

// ExportBadgesTimers exports badges timers data (for genesis)
func (k Keeper) ExportBadgesTimers(ctx sdk.Context) timertypes.GenesisState {
	return k.badgeTimerStore.Export(ctx)
}
