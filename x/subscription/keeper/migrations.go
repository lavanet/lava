package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/utils"
	v8 "github.com/lavanet/lava/v4/x/subscription/migrations/v8"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
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
