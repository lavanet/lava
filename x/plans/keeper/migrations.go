package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v2 "github.com/lavanet/lava/x/plans/migrations/v2"
	"github.com/lavanet/lava/x/plans/types"
	projecttypes "github.com/lavanet/lava/x/projects/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate2to3 implements store migration from v1 to v2:
// Trigger the version upgrade of the planFS fixation store
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	if err := m.keeper.plansFS.MigrateVersion(ctx); err != nil {
		return fmt.Errorf("%w: plans fixation-store", err)
	}

	planIndices := m.keeper.GetAllPlanIndices(ctx)

	for _, planIndex := range planIndices {
		blocks := m.keeper.plansFS.GetAllEntryVersions(ctx, planIndex, true)
		for _, block := range blocks {
			var plan_v2 v2.PlanV2
			m.keeper.plansFS.ReadEntry(ctx, planIndex, block, &plan_v2)

			// create policy struct
			planPolicy := projecttypes.Policy{
				GeolocationProfile: uint64(1),
				TotalCuLimit:       plan_v2.ComputeUnits,
				EpochCuLimit:       plan_v2.ComputeUnitsPerEpoch,
				MaxProvidersToPair: plan_v2.MaxProvidersToPair,
			}

			// convert plan from type v2.Plan to types.Plan
			plan_v3 := types.Plan{
				Index:                    plan_v2.Index,
				Block:                    plan_v2.Block,
				Price:                    plan_v2.Price,
				OveruseRate:              plan_v2.OveruseRate,
				AllowOveruse:             plan_v2.AllowOveruse,
				Description:              plan_v2.Description,
				Type:                     plan_v2.Type,
				AnnualDiscountPercentage: plan_v2.AnnualDiscountPercentage,
				PlanPolicy:               planPolicy,
			}

			m.keeper.plansFS.ModifyEntry(ctx, planIndex, block, &plan_v3)
		}
	}

	return nil
}
