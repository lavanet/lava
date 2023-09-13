package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
	"github.com/lavanet/lava/x/spec/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

func (m Migrator) UpgradeProtocolVersionParams(ctx sdk.Context) {
	params := m.keeper.GetParams(ctx)
	params.Version = protocoltypes.DefaultGenesis().Params.Version
	m.keeper.SetParams(ctx, params)

	detailsMap := map[string]string{
		"param": string(protocoltypes.KeyVersion),
		"value": params.Version.String(),
	}

	utils.LogLavaEvent(ctx, m.keeper.Logger(ctx), types.ParamChangeEventName, detailsMap, "Gov Proposal Accepted Param Changed")
}

// Migrate2to3 implements store migration from v2 to v3:
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	m.UpgradeProtocolVersionParams(ctx)
	return nil
}
