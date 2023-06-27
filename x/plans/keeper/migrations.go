package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v2 "github.com/lavanet/lava/x/plans/migrations/v2"
	"github.com/lavanet/lava/x/plans/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate2to3 implements store migration from v1 to v2:
//   - Trigger the version upgrade of the planFS fixation store
//   - Update plan policy
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	if err := m.keeper.plansFS.MigrateVersion(ctx); err != nil {
		return fmt.Errorf("%w: plans fixation-store", err)
	}

	planIndices := m.keeper.plansFS.AllEntryIndicesFilter(ctx, "", nil)

	for _, planIndex := range planIndices {
		blocks := m.keeper.plansFS.GetAllEntryVersions(ctx, planIndex)
		for _, block := range blocks {
			var plan_v2 v2.PlanV2
			m.keeper.plansFS.ReadEntry(ctx, planIndex, block, &plan_v2)

			// create policy struct
			planPolicy := types.Policy{
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

// Migrate3to4 implements store migration from v3 to v4:
//   - Trigger the version upgrade of the planFS fixation store
//   - Replace the store prefix from module-name ("plan") to "plan-fs"
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	const V4_PlanFixationStorePrefix = "plan"

	// MigrateVersionAndPrefix() will take care of calling MigrateVersion()
	// with the old prefix first, before changing to the new (current) prefix.

	if err := m.keeper.plansFS.MigrateVersionAndPrefix(ctx, V4_PlanFixationStorePrefix); err != nil {
		return fmt.Errorf("%w: plans fixation-store", err)
	}

	return nil
}

// Migrate4to5 implements store migration from v4 to v5:
//   - Trigger the version upgrade of the planFS fixation store (so it will
//     call the version upgrade of its timer store).
func (m Migrator) Migrate4to5(ctx sdk.Context) error {
	if err := m.keeper.plansFS.MigrateVersion(ctx); err != nil {
		return fmt.Errorf("%w: plans fixation-store", err)
	}
	return nil
}

// Migrate5to6 implements store migration from v5 to v6:
// -- trigger fixation migration, deleteat and live variables
func (m Migrator) Migrate5to6(ctx sdk.Context) error {
	return m.keeper.plansFS.MigrateVersionFrom(ctx, 3)
}
