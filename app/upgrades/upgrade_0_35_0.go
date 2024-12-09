package upgrades

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/gogoproto/proto"
	ibctypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/lavanet/lava/v4/app/keepers"
	protocoltypes "github.com/lavanet/lava/v4/x/protocol/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
)

func v_35_0(
	m *module.Manager,
	c module.Configurator,
	_ BaseAppParamManager,
	lk *keepers.LavaKeepers,
) upgradetypes.UpgradeHandler {
	return func(context context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(context)
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
