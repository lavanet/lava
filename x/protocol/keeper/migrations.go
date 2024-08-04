package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	protocoltypes "github.com/lavanet/lava/v2/x/protocol/types"
	"github.com/lavanet/lava/v2/x/spec/types"
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

// MigrateVersion implements store migration: update protocol version
func (m Migrator) MigrateVersion(ctx sdk.Context) error {
	m.UpgradeProtocolVersionParams(ctx)
	return nil
}
