package upgrades

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/v4/app/keepers"
)

func v_4_1_1(
	m *module.Manager,
	c module.Configurator,
	_ BaseAppParamManager,
	lk *keepers.LavaKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		accounts := lk.AccountKeeper.GetAllAccounts(ctx)
		for _, account := range accounts {

			if vaccount, ok := account.(*authtypes.PeriodicVestingAccount); ok {
				if vaccount.StartTime > ctx.BlockTime().Unix() {
					// Add one month (30 days) in seconds
					vaccount.StartTime += 30 * 24 * 60 * 60
					lk.AccountKeeper.SetAccount(ctx, vaccount)
				}
				continue
			}

			if vaccount, ok := account.(*authtypes.ContinuousVestingAccount); ok {
				if vaccount.StartTime > ctx.BlockTime().Unix() {
					// Add one month (30 days) in seconds
					vaccount.StartTime += 30 * 24 * 60 * 60
					vaccount.EndTime += 30 * 24 * 60 * 60
					lk.AccountKeeper.SetAccount(ctx, vaccount)
				}
				continue
			}
		}

		return m.RunMigrations(ctx, c, vm)
	}
}
