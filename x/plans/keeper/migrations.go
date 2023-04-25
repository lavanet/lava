package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v3 "github.com/lavanet/lava/x/plans/migrations/v3"
	"github.com/lavanet/lava/x/plans/types"
	projecttypes "github.com/lavanet/lava/x/projects/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate2to3 implements store migration from v2 to v3:
// Trigger the version upgrade of the planFS fixation store
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	if err := m.keeper.plansFS.MigrateVersion(ctx); err != nil {
		return fmt.Errorf("%w: plans fixation-store", err)
	}
	return nil
}

// Migrate3to4 implements protobuf migration from v3 to v4:
// Now the plan protobuf has a plan policy
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	planIndices := m.keeper.GetAllPlanIndices(ctx)

	for _, planIndex := range planIndices {
		blocks := m.keeper.plansFS.GetAllEntryVersions(ctx, planIndex, true)
		for _, block := range blocks {
			var oldPlanStruct v3.Plan
			if found := m.keeper.plansFS.FindEntry(ctx, planIndex, block, &oldPlanStruct); !found {
				return fmt.Errorf("could not find plan with index %s", planIndex)
			}

			// create policy struct
			planPolicy := projecttypes.Policy{
				GeolocationProfile: uint64(1),
				TotalCuLimit:       oldPlanStruct.ComputeUnits,
				EpochCuLimit:       oldPlanStruct.ComputeUnitsPerEpoch,
				MaxProvidersToPair: oldPlanStruct.MaxProvidersToPair,
			}

			// convert plan from type v3.Plan to types.Plan
			newPlanStruct := types.Plan{
				Index:                    oldPlanStruct.Index,
				Block:                    oldPlanStruct.Block,
				Price:                    oldPlanStruct.Price,
				OveruseRate:              oldPlanStruct.OveruseRate,
				AllowOveruse:             oldPlanStruct.AllowOveruse,
				Description:              oldPlanStruct.Description,
				Type:                     oldPlanStruct.Type,
				AnnualDiscountPercentage: oldPlanStruct.AnnualDiscountPercentage,
				PlanPolicy:               planPolicy,
			}

			err := m.keeper.plansFS.ModifyEntry(ctx, planIndex, block, &newPlanStruct)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
