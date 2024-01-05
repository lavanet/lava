package upgrades

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/app/keepers"
)

func v0_32_4_UpgradeHandler(
	m *module.Manager,
	c module.Configurator,
	bapm BaseAppParamManager,
	lk *keepers.LavaKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		// reset whitelist to empty
		lk.SpecKeeper.WhitelistReset(ctx)

		params := lk.SpecKeeper.GetParams(ctx)
		params.WhitelistedExpeditedMsgs = []string{
			// proto.MessageName(&upgradetypes.MsgCancelUpgrade{}),
			// TODO: Here we setup the whitelisted messages we want to allow via expedited proposals
		}
		lk.SpecKeeper.SetParams(ctx, params)

		return m.RunMigrations(ctx, c, vm)
	}
}
