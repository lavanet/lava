package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	v2 "github.com/lavanet/lava/x/subscription/migrations/v2"
	"github.com/lavanet/lava/x/subscription/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate2to3 implements store migration from v2 to v3:
//   - Convert subscription store to fixation store and use timers
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	keeper := m.keeper

	store := prefix.NewStore(
		ctx.KVStore(keeper.storeKey),
		v2.KeyPrefix(v2.SubscriptionKeyPrefix),
	)

	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var sub_V2 v2.Subscription
		keeper.cdc.MustUnmarshal(iterator.Value(), &sub_V2)

		utils.LavaFormatDebug("migrate:",
			utils.Attribute{Key: "subscription", Value: sub_V2.Consumer})

		sub_V3 := types.Subscription{
			Creator:       sub_V2.Creator,
			Consumer:      sub_V2.Consumer,
			Block:         sub_V2.Block,
			PlanIndex:     sub_V2.PlanIndex,
			PlanBlock:     sub_V2.PlanBlock,
			DurationTotal: sub_V2.DurationTotal,
			DurationLeft:  sub_V2.DurationLeft,
			MonthCuTotal:  sub_V2.MonthCuTotal,
			MonthCuLeft:   sub_V2.MonthCuLeft,
		}

		// each subscription entry in V2 store should have an entry in V3 store
		err := keeper.subsFS.AppendEntry(ctx, sub_V3.Consumer, sub_V3.Block, &sub_V3)
		if err != nil {
			return fmt.Errorf("%w: subscriptions %s", err, sub_V3.Consumer)
		}

		// if the subscription is alive, then set the timer for the monthly expiry.
		// otherwise, delete the entry from V3 to induce stale-period state (use the
		// block from last expiry as the block for deletion).
		if sub_V3.DurationLeft > 0 {
			expiry := sub_V2.MonthExpiryTime
			if expiry >= uint64(ctx.BlockTime().UTC().Unix()) {
				return fmt.Errorf("expiry time passed for subscription %s", sub_V3.Consumer)
			}
			keeper.subsTS.AddTimerByBlockTime(ctx, expiry, []byte(sub_V3.Consumer), []byte{})
		} else {
			keeper.subsFS.DelEntry(ctx, sub_V3.Consumer, sub_V2.PrevExpiryBlock)
		}

		store.Delete(iterator.Key())
	}

	return nil
}
