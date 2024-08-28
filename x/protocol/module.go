package protocol

import (
	"context"
	"encoding/json"
	"fmt"

	// this line is used by starport scaffolding # 1

	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/lavanet/lava/v2/x/protocol/client/cli"
	"github.com/lavanet/lava/v2/x/protocol/keeper"
	"github.com/lavanet/lava/v2/x/protocol/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// ----------------------------------------------------------------------------
// AppModuleBasic
// ----------------------------------------------------------------------------

// AppModuleBasic implements the AppModuleBasic interface for the capability module.
type AppModuleBasic struct {
	cdc codec.BinaryCodec
}

func NewAppModuleBasic(cdc codec.BinaryCodec) AppModuleBasic {
	return AppModuleBasic{cdc: cdc}
}

// Name returns the capability module's name.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

func (AppModuleBasic) RegisterCodec(cdc *codec.LegacyAmino) {
	types.RegisterCodec(cdc)
}

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterCodec(cdc)
}

// RegisterInterfaces registers the module's interface types
func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns the capability module's default genesis state.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesis())
}

// ValidateGenesis performs genesis state validation for the capability module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return genState.Validate()
}

// RegisterRESTRoutes registers the capability module's REST service handlers.
func (AppModuleBasic) RegisterRESTRoutes(clientCtx client.Context, rtr *mux.Router) {
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx))
	// this line is used by starport scaffolding # 2
}

// GetTxCmd returns the capability module's root tx command.
func (a AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns the capability module's root query command.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd(types.StoreKey)
}

// ----------------------------------------------------------------------------
// AppModule
// ----------------------------------------------------------------------------

// AppModule implements the AppModule interface for the capability module.
type AppModule struct {
	AppModuleBasic

	keeper keeper.Keeper
}

func NewAppModule(
	cdc codec.Codec,
	keeper keeper.Keeper,
) AppModule {
	return AppModule{
		AppModuleBasic: NewAppModuleBasic(cdc),
		keeper:         keeper,
	}
}

// Name returns the capability module's name.
func (am AppModule) Name() string {
	return am.AppModuleBasic.Name()
}

// RegisterServices registers a GRPC query service to respond to the
// module-specific GRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))

	migrator := keeper.NewMigrator(am.keeper)

	// register v2 -> v3 migration
	if err := cfg.RegisterMigration(types.ModuleName, 2, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v3: %w", types.ModuleName, err))
	}

	// register v3 -> v4 migration
	if err := cfg.RegisterMigration(types.ModuleName, 3, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v3: %w", types.ModuleName, err))
	}

	// register v4 -> v5 migration
	if err := cfg.RegisterMigration(types.ModuleName, 4, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v5: %w", types.ModuleName, err))
	}

	// register v5 -> v6 migration
	if err := cfg.RegisterMigration(types.ModuleName, 5, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v6: %w", types.ModuleName, err))
	}

	// register v6 -> v7 migration
	if err := cfg.RegisterMigration(types.ModuleName, 6, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v7: %w", types.ModuleName, err))
	}

	// register v7 -> v8 migration
	if err := cfg.RegisterMigration(types.ModuleName, 7, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v8: %w", types.ModuleName, err))
	}

	if err := cfg.RegisterMigration(types.ModuleName, 8, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v9: %w", types.ModuleName, err))
	}

	if err := cfg.RegisterMigration(types.ModuleName, 9, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v10: %w", types.ModuleName, err))
	}

	if err := cfg.RegisterMigration(types.ModuleName, 10, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v11: %w", types.ModuleName, err))
	}

	if err := cfg.RegisterMigration(types.ModuleName, 11, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v12: %w", types.ModuleName, err))
	}

	if err := cfg.RegisterMigration(types.ModuleName, 12, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v13: %w", types.ModuleName, err))
	}

	if err := cfg.RegisterMigration(types.ModuleName, 13, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v14: %w", types.ModuleName, err))
	}

	if err := cfg.RegisterMigration(types.ModuleName, 14, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v15: %w", types.ModuleName, err))
	}

	if err := cfg.RegisterMigration(types.ModuleName, 15, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v16: %w", types.ModuleName, err))
	}
	if err := cfg.RegisterMigration(types.ModuleName, 16, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v17: %w", types.ModuleName, err))
	}
	if err := cfg.RegisterMigration(types.ModuleName, 17, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v18: %w", types.ModuleName, err))
	}
	if err := cfg.RegisterMigration(types.ModuleName, 18, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v19: %w", types.ModuleName, err))
	}
	if err := cfg.RegisterMigration(types.ModuleName, 19, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v20: %w", types.ModuleName, err))
	}
	if err := cfg.RegisterMigration(types.ModuleName, 20, migrator.MigrateVersion); err != nil {
		// panic:ok: at start up, migration cannot proceed anyhow
		panic(fmt.Errorf("%s: failed to register migration to v21: %w", types.ModuleName, err))
	}
}

// ConsensusVersion implements ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 21 }

// RegisterInvariants registers the capability module's invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// InitGenesis performs the capability module's genesis initialization It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) []abci.ValidatorUpdate {
	var genState types.GenesisState
	// Initialize global index to index in genesis state
	cdc.MustUnmarshalJSON(gs, &genState)

	InitGenesis(ctx, am.keeper, genState)

	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the capability module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(genState)
}

// BeginBlock executes all ABCI BeginBlock logic respective to the capability module.
func (am AppModule) BeginBlock(_ sdk.Context, _ abci.RequestBeginBlock) {}

// EndBlock executes all ABCI EndBlock logic respective to the capability module. It
// returns no validator updates.
func (am AppModule) EndBlock(_ sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}
