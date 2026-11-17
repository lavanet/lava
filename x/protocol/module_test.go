package protocol_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	keepertest "github.com/lavanet/lava/v5/testutil/keeper"
	"github.com/lavanet/lava/v5/x/protocol"
	"github.com/lavanet/lava/v5/x/protocol/types"
	"github.com/stretchr/testify/require"
)

// mainnetConsensusVersion is the x/protocol consensus version on mainnet before the v6.3.0 upgrade
const mainnetConsensusVersion = 28

// TestProtocolModule_UpgradeMigration runs the x/protocol migrations through a real module
// manager, as a chain upgrade does, starting from the mainnet consensus version. It checks that
// every step up to the current consensus version is registered and that the on-chain version
// params end at the compiled defaults (TARGET_VERSION and MIN_VERSION).
func TestProtocolModule_UpgradeMigration(t *testing.T) {
	tests := []struct {
		name    string
		version types.Version
	}{
		{
			name:    "mainnet params before v6.3.0",
			version: types.Version{ProviderTarget: "6.0.0", ProviderMin: "5.5.1", ConsumerTarget: "6.0.0", ConsumerMin: "5.5.1"},
		},
		{
			name:    "params already at the defaults",
			version: types.DefaultVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, ctx := keepertest.ProtocolKeeper(t)
			params := k.GetParams(ctx)
			params.Version = tt.version
			k.SetParams(ctx, params)

			registry := codectypes.NewInterfaceRegistry()
			types.RegisterInterfaces(registry)
			cdc := codec.NewProtoCodec(registry)
			msgRouter := baseapp.NewMsgServiceRouter()
			msgRouter.SetInterfaceRegistry(registry)
			queryRouter := baseapp.NewGRPCQueryRouter()
			queryRouter.SetInterfaceRegistry(registry)
			configurator := module.NewConfigurator(cdc, msgRouter, queryRouter)

			appModule := protocol.NewAppModule(cdc, *k)
			manager := module.NewManager(appModule)
			manager.RegisterServices(configurator)

			versions, err := manager.RunMigrations(ctx, configurator, module.VersionMap{types.ModuleName: mainnetConsensusVersion})
			require.NoError(t, err)
			require.Greater(t, appModule.ConsensusVersion(), uint64(mainnetConsensusVersion))
			require.Equal(t, appModule.ConsensusVersion(), versions[types.ModuleName])
			require.Equal(t, types.DefaultVersion, k.GetParams(ctx).Version)
		})
	}
}
