package keeper

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v7 "github.com/lavanet/lava/v4/x/plans/migrations/v7"
	v8 "github.com/lavanet/lava/v4/x/plans/migrations/v8"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
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
