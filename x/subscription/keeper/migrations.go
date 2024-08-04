package keeper

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	v2 "github.com/lavanet/lava/v2/x/subscription/migrations/v2"
	v5 "github.com/lavanet/lava/v2/x/subscription/migrations/v5"
	v6 "github.com/lavanet/lava/v2/x/subscription/migrations/v6"
	v8 "github.com/lavanet/lava/v2/x/subscription/migrations/v8"
	"github.com/lavanet/lava/v2/x/subscription/types"
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
	utils.LavaFormatDebug("migrate: subscriptions")

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
			Creator:         sub_V2.Creator,
			Consumer:        sub_V2.Consumer,
			Block:           sub_V2.Block,
			PlanIndex:       sub_V2.PlanIndex,
			PlanBlock:       sub_V2.PlanBlock,
			DurationTotal:   sub_V2.DurationTotal,
			DurationLeft:    sub_V2.DurationLeft,
			MonthExpiryTime: sub_V2.MonthExpiryTime,
			MonthCuTotal:    sub_V2.MonthCuTotal,
			MonthCuLeft:     sub_V2.MonthCuLeft,
		}

		// each subscription entry in V2 store should have an entry in V3 store
		err := keeper.subsFS.AppendEntry(ctx, sub_V3.Consumer, sub_V3.Block, &sub_V3)
		if err != nil {
			return fmt.Errorf("%w: subscriptions %s", err, sub_V3.Consumer)
		}

		// if the subscription has expired, then delete the entry from V3 store to induce
		// stale-period state (use the block of last expiry as the block for deletion).
		// otherwise, the subscription is alive, but the current month may have expired
		// between since the upgrade proposal took effect (and until now); if indeed so,
		// then invoke advanceMonth() since the current block is the (month) expiry block.
		// otherwise, set the timer for the monthly expiry as already was set in V2.
		if sub_V3.DurationLeft > 0 {
			expiry := sub_V2.MonthExpiryTime
			if expiry <= uint64(ctx.BlockTime().UTC().Unix()) {
				utils.LavaFormatDebug("  subscription live, month expired",
					utils.Attribute{Key: "expiry", Value: time.Unix(int64(expiry), 0)},
					utils.Attribute{Key: "blockTime", Value: ctx.BlockTime().UTC()},
				)
				keeper.advanceMonth(ctx, []byte(sub_V3.Consumer))
			} else {
				utils.LavaFormatDebug("  subscription live, future expiry",
					utils.Attribute{Key: "expiry", Value: time.Unix(int64(expiry), 0)},
					utils.Attribute{Key: "blockTime", Value: ctx.BlockTime().UTC()},
				)
				keeper.subsTS.AddTimerByBlockTime(ctx, expiry, []byte(sub_V3.Consumer), []byte{})
			}
		} else {
			utils.LavaFormatDebug("  subscription deleted",
				utils.Attribute{Key: "block", Value: sub_V2.PrevExpiryBlock})
			keeper.subsFS.DelEntry(ctx, sub_V3.Consumer, sub_V2.PrevExpiryBlock)
		}

		store.Delete(iterator.Key())
	}

	return nil
}

// Migrate3to4 implements store migration from v3 to v4:
// -- trigger fixation migration (v4->v5), initialize IsLatest field
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	// This migration used to call a deprecated fixationstore function called MigrateVersionAndPrefix

	return nil
}

// Migrate4to5 implements store migration from v4 to v5:
// -- rename the DurationTotal field to DurationBought
// -- introduce two new fields: DurationTotal (with new meaning) and cluster
// -- assign the subscription's cluster
func (m Migrator) Migrate4to5(ctx sdk.Context) error {
	utils.LavaFormatDebug("migrate 4->5: subscriptions")

	keeper := m.keeper

	indices := keeper.subsFS.AllEntryIndicesFilter(ctx, "", nil)
	for _, ind := range indices {
		blocks := keeper.subsFS.GetAllEntryVersions(ctx, ind)

		for _, block := range blocks {
			var sub_V5 v5.Subscription
			keeper.subsFS.ReadEntry(ctx, ind, block, &sub_V5)
			utils.LavaFormatDebug("migrate:",
				utils.Attribute{Key: "subscription", Value: sub_V5.Consumer})

			sub_V5.Cluster = v5.GetClusterKey(sub_V5)

			keeper.subsFS.ModifyEntry(ctx, ind, block, &sub_V5)
		}
	}
	return nil
}

// Migrate5to6 implements store migration from v5 to v6:
// -- find old subscriptions and trigger advance month to make them expire
func (m Migrator) Migrate5to6(ctx sdk.Context) error {
	indices := m.keeper.GetAllSubscriptionsIndices(ctx)
	currentTime := ctx.BlockTime().UTC().Unix()
	for _, ind := range indices {
		sub, found := m.keeper.GetSubscription(ctx, ind)
		if !found {
			utils.LavaFormatError("cannot migrate sub", fmt.Errorf("sub not found"),
				utils.Attribute{Key: "sub", Value: sub},
			)
		}

		if sub.MonthExpiryTime < uint64(currentTime) {
			m.keeper.advanceMonth(ctx, []byte(ind))
		}
	}

	return nil
}

// Migrate6to7 implements store migration from v6 to v7:
// -- if subscription's auto_renewal = true, set auto_renewal_next_plan to the current's subscription plan
func (m Migrator) Migrate6to7(ctx sdk.Context) error {
	utils.LavaFormatDebug("migrate 6->7: subscriptions")

	for _, index := range m.keeper.subsFS.GetAllEntryIndices(ctx) {
		for _, block := range m.keeper.subsFS.GetAllEntryVersions(ctx, index) {
			var subscriptionV6 v6.Subscription
			var subscriptionV7 types.Subscription
			foundOld := m.keeper.subsFS.FindEntry(ctx, index, block, &subscriptionV6)
			foundNew := m.keeper.subsFS.FindEntry(ctx, index, block, &subscriptionV7)
			if !foundOld || !foundNew {
				utils.LavaFormatError("cannot migrate sub", fmt.Errorf("sub not found"),
					utils.Attribute{Key: "index", Value: index},
					utils.Attribute{Key: "block", Value: block},
				)
				continue
			}

			if subscriptionV6.AutoRenewal {
				subscriptionV7.AutoRenewalNextPlan = subscriptionV7.PlanIndex
			} else {
				subscriptionV7.AutoRenewalNextPlan = types.AUTO_RENEWAL_PLAN_NONE
			}

			m.keeper.subsFS.ModifyEntry(ctx, index, block, &subscriptionV7)
		}
	}

	return nil
}

// Migrate7to8 implements store migration from v7 to v8:
// init new credit field
func (m Migrator) Migrate7to8(ctx sdk.Context) error {
	utils.LavaFormatDebug("migrate 7->8: subscriptions")

	for _, index := range m.keeper.subsFS.GetAllEntryIndices(ctx) {
		for _, block := range m.keeper.subsFS.GetAllEntryVersions(ctx, index) {
			// read current subscription from fixation to new subscription struct
			var s8 v8.Subscription
			m.keeper.subsFS.ReadEntry(ctx, index, block, &s8)

			// calculate sub's credit
			plan, found := m.keeper.plansKeeper.FindPlan(ctx, s8.PlanIndex, s8.PlanBlock)
			if !found {
				utils.LavaFormatError("cannot migrate sub", fmt.Errorf("sub's plan not found"),
					utils.Attribute{Key: "consumer", Value: index},
					utils.Attribute{Key: "sub_block", Value: block},
					utils.Attribute{Key: "plan", Value: s8.PlanIndex},
					utils.Attribute{Key: "plan_block", Value: s8.PlanBlock},
				)
				continue
			}
			creditAmount := plan.Price.Amount.MulRaw(int64(s8.DurationLeft))
			credit := sdk.NewCoin(m.keeper.stakingKeeper.BondDenom(ctx), creditAmount)

			// calculate future sub's credit
			if s8.FutureSubscription != nil {
				futurePlan, found := m.keeper.plansKeeper.FindPlan(ctx, s8.FutureSubscription.PlanIndex, s8.FutureSubscription.PlanBlock)
				if !found {
					utils.LavaFormatError("cannot migrate sub", fmt.Errorf("sub's future plan not found"),
						utils.Attribute{Key: "consumer", Value: index},
						utils.Attribute{Key: "sub_block", Value: block},
						utils.Attribute{Key: "plan", Value: s8.PlanIndex},
						utils.Attribute{Key: "plan_block", Value: s8.PlanBlock},
					)
					continue
				}

				futureCreditAmount := futurePlan.Price.Amount.MulRaw(int64(s8.FutureSubscription.DurationBought))
				futureCredit := sdk.NewCoin(m.keeper.stakingKeeper.BondDenom(ctx), futureCreditAmount)
				s8.FutureSubscription.Credit = futureCredit
			}

			s8.Credit = credit

			// modify sub entry
			m.keeper.subsFS.ModifyEntry(ctx, index, block, &s8)
		}
	}

	return nil
}
