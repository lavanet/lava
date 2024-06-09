package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/rewards/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// MigrateVersion1To2 sets the min IPRPC cost to be 100LAVA = 100,000,000ulava
func (m Migrator) MigrateVersion1To2(ctx sdk.Context) error {
	return m.keeper.SetIprpcData(ctx, sdk.NewCoin(m.keeper.stakingKeeper.BondDenom(ctx), sdk.NewInt(100000000)), []string{})
}

// MigrateVersion2To3 sets PendingIbcIprpcExpiration to 3 months
func (m Migrator) MigrateVersion2To3(ctx sdk.Context) error {
	m.keeper.SetParams(ctx, types.DefaultGenesis().Params)
	return nil
}
