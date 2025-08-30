package upgrades

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/v5/app/keepers"
)

func v_4_2_0(
	m *module.Manager,
	c module.Configurator,
	_ BaseAppParamManager,
	lk *keepers.LavaKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		accounts := lk.AccountKeeper.GetAllAccounts(ctx)
		year := int64(12 * 30 * 24 * 60 * 60)
		for _, account := range accounts {
			if vaccount, ok := account.(*authtypes.PeriodicVestingAccount); ok {
				if vaccount.StartTime > ctx.BlockTime().Unix() {
					vaccount.StartTime += year
					vaccount.EndTime += year
					lk.AccountKeeper.SetAccount(ctx, vaccount)
				}
			}
		}

		return m.RunMigrations(ctx, c, vm)
	}
}
