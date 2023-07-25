package downtime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/lavanet/lava/x/downtime/client/cli"
	"github.com/lavanet/lava/x/downtime/keeper"
	"github.com/lavanet/lava/x/downtime/types"
	v1 "github.com/lavanet/lava/x/downtime/v1"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/tendermint/abci/types"
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
	return types.ModuleName
}

func (a AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

func (a AppModuleBasic) RegisterInterfaces(ir cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(ir)
}

func (a AppModuleBasic) DefaultGenesis(codec codec.JSONCodec) json.RawMessage {
	return codec.MustMarshalJSON(v1.DefaultGenesisState())
}

func (a AppModuleBasic) ValidateGenesis(codec codec.JSONCodec, config client.TxEncodingConfig, message json.RawMessage) error {
	gs := new(v1.GenesisState)
	codec.MustUnmarshalJSON(message, gs)
	return gs.Validate()
}

func (a AppModuleBasic) RegisterRESTRoutes(_ client.Context, _ *mux.Router) {}

func (a AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	_ = v1.RegisterQueryHandlerClient(context.Background(), mux, v1.NewQueryClient(clientCtx))
}

func (a AppModuleBasic) GetTxCmd() *cobra.Command { return cli.NewTxCmd() }

func (a AppModuleBasic) GetQueryCmd() *cobra.Command { return cli.NewQueryCmd() }

// ---- AppModule

func NewAppModule(k keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		k:              k,
	}
}

type AppModule struct {
	AppModuleBasic

	k keeper.Keeper
}

func (a AppModule) InitGenesis(context sdk.Context, jsonCodec codec.JSONCodec, message json.RawMessage) []abci.ValidatorUpdate {
	gs := new(v1.GenesisState)
	jsonCodec.MustUnmarshalJSON(message, gs)
	err := a.k.ImportGenesis(context, gs)
	if err != nil {
		panic(err)
	}
	return nil
}

func (a AppModule) ExportGenesis(context sdk.Context, jsonCodec codec.JSONCodec) json.RawMessage {
	gs, err := a.k.ExportGenesis(context)
	if err != nil {
		panic(err)
	}
	return jsonCodec.MustMarshalJSON(gs)
}

func (a AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

func (a AppModule) Route() sdk.Route {
	return sdk.NewRoute(types.ModuleName, func(ctx sdk.Context, _ sdk.Msg) (*sdk.Result, error) {
		return nil, fmt.Errorf("legacy router should not be used")
	})
}

func (a AppModule) QuerierRoute() string {
	return types.ModuleName
}

func (a AppModule) LegacyQuerierHandler(amino *codec.LegacyAmino) sdk.Querier {
	return func(_ sdk.Context, _ []string, _ abci.RequestQuery) ([]byte, error) {
		return nil, fmt.Errorf("legacy querier should not be used")
	}
}

func (a AppModule) RegisterServices(configurator module.Configurator) {
	v1.RegisterQueryServer(configurator.QueryServer(), keeper.NewQueryServer(a.k))
}

func (a AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

func (a AppModule) BeginBlock(context sdk.Context, _ abci.RequestBeginBlock) {
	a.k.BeginBlock(context)
}
