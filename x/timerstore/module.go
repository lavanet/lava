package timerstore

import (
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
)

var (
	_ module.AppModuleBasic = AppModuleBasic{}
	_ module.AppModule      = AppModule{}
	_ module.HasServices    = AppModule{}
)

const (
	ConsensusVersion = 1 + 1
)

// AppModuleBasic implements the module.AppModuleBasic interface for the downtime module.
type AppModuleBasic struct{}

func (a AppModuleBasic) Name() string {
	return ModuleName
}

func (a AppModuleBasic) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}

func (a AppModuleBasic) RegisterInterfaces(_ cdctypes.InterfaceRegistry) {}

func (a AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}

func (a AppModuleBasic) GetTxCmd() *cobra.Command { return nil }

func (a AppModuleBasic) GetQueryCmd() *cobra.Command { return nil }

// ---- AppModule

func NewAppModule(k *Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		k:              k,
	}
}

type AppModule struct {
	AppModuleBasic

	k *Keeper
}

func (a AppModule) RegisterServices(configurator module.Configurator) {
	err := configurator.RegisterMigration(ModuleName, 1, func(ctx sdk.Context) error {
		for _, timer := range a.k.timerStores {
			err := timerMigrate1to2(ctx, timer)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}

func (a AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

func (a AppModule) BeginBlock(context sdk.Context, _ abci.RequestBeginBlock) {
	a.k.BeginBlock(context)
}
