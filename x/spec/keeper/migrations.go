package keeper

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	specs := m.keeper.GetAllSpec(ctx)
	for _, spec := range specs {
		spec.Name = strings.ToLower(spec.Name)
		m.keeper.SetSpec(ctx, spec)
	}
	return nil
}
