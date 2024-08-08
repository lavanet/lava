package keeper

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v2 "github.com/lavanet/lava/v2/x/plans/migrations/v2"
	v3 "github.com/lavanet/lava/v2/x/plans/migrations/v3"
	v7 "github.com/lavanet/lava/v2/x/plans/migrations/v7"
	v8 "github.com/lavanet/lava/v2/x/plans/migrations/v8"
	projectsv3 "github.com/lavanet/lava/v2/x/projects/migrations/v3"
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
	planIndices := m.keeper.plansFS.AllEntryIndicesFilter(ctx, "", nil)

	for _, planIndex := range planIndices {
		blocks := m.keeper.plansFS.GetAllEntryVersions(ctx, planIndex)
		for _, block := range blocks {
			var plan_v2 v2.Plan
			m.keeper.plansFS.ReadEntry(ctx, planIndex, block, &plan_v2)

			// create policy struct
			planPolicy := projectsv3.Policy{
				GeolocationProfile: uint64(1),
				TotalCuLimit:       plan_v2.ComputeUnits,
				EpochCuLimit:       plan_v2.ComputeUnitsPerEpoch,
				MaxProvidersToPair: plan_v2.MaxProvidersToPair,
			}

			// convert plan from type v2.Plan to types.Plan
			plan_v3 := v3.Plan{
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
	// This migration used to call a deprecated fixationstore function called MigrateVersionAndPrefix

	return nil
}

// Migrate4to5 implements store migration from v4 to v5:
//   - Trigger the version upgrade of the planFS fixation store (so it will
//     call the version upgrade of its timer store).
func (m Migrator) Migrate4to5(ctx sdk.Context) error {
	// This migration used to call a deprecated fixationstore function called MigrateVersion

	return nil
}

// Migrate5to6 implements store migration from v5 to v6:
// -- trigger fixation migration, deleteat and live variables
func (m Migrator) Migrate5to6(ctx sdk.Context) error {
	// This migration used to call a deprecated fixationstore function called MigrateVersion

	return nil
}

// Migrate6to7 implements store migration from v6 to v7:
// -- trigger fixation migration (v4->v5), initialize IsLatest field
func (m Migrator) Migrate6to7(ctx sdk.Context) error {
	// This migration used to call a deprecated fixationstore function called MigrateVersion

	return nil
}

func (m Migrator) Migrate7to8(ctx sdk.Context) error {
	plansInds := m.keeper.plansFS.GetAllEntryIndices(ctx)
	for _, ind := range plansInds {
		blocks := m.keeper.plansFS.GetAllEntryVersions(ctx, ind)
		for _, block := range blocks {
			var p7 v7.Plan
			m.keeper.plansFS.ReadEntry(ctx, ind, block, &p7)

			var cp8arr []v8.ChainPolicy
			for _, cp7 := range p7.PlanPolicy.ChainPolicies {
				cp8 := v8.ChainPolicy{
					ChainId: cp7.ChainId,
					Apis:    cp7.Apis,
				}

				var req8arr []v8.ChainRequirement
				for _, req7 := range cp7.Requirements {
					req8 := v8.ChainRequirement{
						Extensions: req7.Extensions,
						Collection: req7.Collection,
					}
					req8arr = append(req8arr, req8)
				}

				cp8.Requirements = req8arr
				cp8arr = append(cp8arr, cp8)
			}

			var geoInt32 int32
			if p7.PlanPolicy.GeolocationProfile <= math.MaxInt32 {
				geoInt32 = int32(p7.PlanPolicy.GeolocationProfile)
			} else {
				geoInt32 = math.MaxInt32
			}

			p8 := v8.Plan{
				Index:                    p7.Index,
				Block:                    p7.Block,
				Price:                    p7.Price,
				AllowOveruse:             p7.AllowOveruse,
				OveruseRate:              p7.OveruseRate,
				Description:              p7.Description,
				Type:                     p7.Type,
				AnnualDiscountPercentage: p7.AnnualDiscountPercentage,
				PlanPolicy: v8.Policy{
					ChainPolicies:         cp8arr,
					GeolocationProfile:    geoInt32,
					TotalCuLimit:          p7.PlanPolicy.TotalCuLimit,
					EpochCuLimit:          p7.PlanPolicy.EpochCuLimit,
					MaxProvidersToPair:    p7.PlanPolicy.MaxProvidersToPair,
					SelectedProvidersMode: v8.SELECTED_PROVIDERS_MODE(p7.PlanPolicy.SelectedProvidersMode),
					SelectedProviders:     p7.PlanPolicy.SelectedProviders,
				},
			}

			m.keeper.plansFS.ModifyEntry(ctx, ind, block, &p8)
		}
	}

	return nil
}
