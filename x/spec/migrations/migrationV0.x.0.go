package migrations

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/spec/keeper"
)

type Migrator struct {
	keeper keeper.Keeper
}

func NewMigrator(keeper keeper.Keeper) Migrator {
	return Migrator{keeper: keeper}
}

func (m Migrator) MigrateToV0X0(ctx sdk.Context) error {
	return updateSpecsVersion(ctx, m.keeper)
}

func updateSpecsVersion(ctx sdk.Context, k keeper.Keeper) error {
	specs := k.GetAllSpec(ctx)
	for spec := range specs {
		for api := range specs[spec].Apis {
			for apiinterface := range specs[spec].Apis[api].ApiInterfaces {
				specs[spec].Apis[api].ApiInterfaces[apiinterface].Deterministic = specs[spec].Apis[api].Category.Reserved
			}
		}
		k.SetSpec(ctx, specs[spec])
	}

	return nil
}
