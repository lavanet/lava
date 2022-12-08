package v020 // migrations.go

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

func (m Migrator) MigrateToV020(ctx sdk.Context) error {
	return initBlockLastUpdated(ctx, m.keeper)
}

func initBlockLastUpdated(ctx sdk.Context, k keeper.Keeper) error {
	specs := k.GetAllSpec(ctx)
	for _, spec := range specs {
		spec.BlockLastUpdated = uint64(0)
		k.SetSpec(ctx, spec)
	}

	return nil
}
