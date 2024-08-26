package upgrades

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/cosmos/gogoproto/proto"
	ibctypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/lavanet/lava/v2/app/keepers"
	protocoltypes "github.com/lavanet/lava/v2/x/protocol/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

func v_35_0(
	m *module.Manager,
	c module.Configurator,
	_ BaseAppParamManager,
	lk *keepers.LavaKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		// reset allowlist to empty
		lk.SpecKeeper.AllowlistReset(ctx)

		params := lk.SpecKeeper.GetParams(ctx)
		params.AllowlistedExpeditedMsgs = []string{
			proto.MessageName(&protocoltypes.MsgSetVersion{}),
			proto.MessageName(&spectypes.SpecAddProposal{}),
			proto.MessageName(&ibctypes.ClientUpdateProposal{}),
			proto.MessageName(&ibctypes.UpgradeProposal{}),
		}
		lk.SpecKeeper.SetParams(ctx, params)

		return m.RunMigrations(ctx, c, vm)
	}
}
