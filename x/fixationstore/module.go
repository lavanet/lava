package fixationstore

import (
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/lavanet/lava/v2/x/fixationstore/client/cli"
	"github.com/lavanet/lava/v2/x/fixationstore/keeper"
	"github.com/lavanet/lava/v2/x/fixationstore/types"
	"github.com/spf13/cobra"
)

var (
	_ module.AppModuleBasic = AppModuleBasic{}
	_ module.AppModule      = AppModule{}
)

const (
	ConsensusVersion = 1
)

// AppModuleBasic implements the module.AppModuleBasic interface for the downtime module.
type AppModuleBasic struct{}

func (a AppModuleBasic) Name() string {
	return types.MODULE_NAME
}

func (a AppModuleBasic) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}

func (a AppModuleBasic) RegisterInterfaces(_ cdctypes.InterfaceRegistry) {}

func (a AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}

func (a AppModuleBasic) GetTxCmd() *cobra.Command { return nil }

func (a AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd("")
}

// ---- AppModule

func NewAppModule(k *keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		k:              k,
	}
}

type AppModule struct {
	AppModuleBasic

	k *keeper.Keeper
}

func (a AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

func (a AppModule) BeginBlock(_ sdk.Context, _ abci.RequestBeginBlock) {}

// RegisterServices registers a GRPC query service to respond to the
// module-specific GRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterQueryServer(cfg.QueryServer(), am.k)
}
