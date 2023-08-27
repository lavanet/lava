package keeper

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v3 "github.com/lavanet/lava/x/spec/types/migrations/v3"
	v4 "github.com/lavanet/lava/x/spec/types/migrations/v4"
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

func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	var paramsV3 v3.Params
	m.keeper.paramstore.GetParamSet(ctx, &paramsV3)

	var paramsV4 v4.Params
	paramsV4.GeolocationCount = int32(paramsV3.GeolocationCount)
	paramsV4.MaxCU = paramsV3.MaxCU

	m.keeper.paramstore.SetParamSet(ctx, &paramsV4)

	return nil
}
