package app_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simulationtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/ignite-hq/cli/ignite/pkg/cosmoscmd"
	"github.com/lavanet/lava/app"
	pairingmodule "github.com/lavanet/lava/x/pairing"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

func init() {
	simapp.GetSimulatorFlags()
}

type SimApp interface {
	cosmoscmd.App
	GetBaseApp() *baseapp.BaseApp
	AppCodec() codec.Codec
	SimulationManager() *module.SimulationManager
	ModuleAccountAddrs() map[string]bool
	Name() string
	LegacyAmino() *codec.LegacyAmino
	BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock
	EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock
	InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain
}

var defaultConsensusParams = &abci.ConsensusParams{
	Block: &abci.BlockParams{
		MaxBytes: 200000,
		MaxGas:   2000000,
	},
	Evidence: &tmproto.EvidenceParams{
		MaxAgeNumBlocks: 302400,
		MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
		MaxBytes:        10000,
	},
	Validator: &tmproto.ValidatorParams{
		PubKeyTypes: []string{
			tmtypes.ABCIPubKeyTypeEd25519,
		},
	},
}

// BenchmarkSimulation run the chain simulation
// Running using starport command:
// `starport chain simulate -v --numBlocks 200 --blockSize 50`
// Running as go benchmark test:
// `go test -benchmem -run=^$ -bench ^BenchmarkSimulation ./app -NumBlocks=200 -BlockSize 50 -Commit=true -Verbose=true -Enabled=true`
func BenchmarkSimulation(b *testing.B) {
	simapp.FlagEnabledValue = true
	simapp.FlagCommitValue = true
	fmt.Printf("starting simulation yarom \n")
	config, db, dir, logger, _, err := simapp.SetupSimulation("goleveldb-app-sim", "Simulation")
	require.NoError(b, err, "simulation setup failed")

	b.Cleanup(func() {
		db.Close()
		err = os.RemoveAll(dir)
		require.NoError(b, err)
	})

	encoding := cosmoscmd.MakeEncodingConfig(app.ModuleBasics)

	app := app.New(
		logger,
		db,
		nil,
		true,
		map[int64]bool{},
		app.DefaultNodeHome,
		0,
		encoding,
		simapp.EmptyAppOptions{},
	)

	simApp, ok := app.(SimApp)
	require.True(b, ok, "can't use simapp")

	// Run randomized simulations
	_, simParams, simErr := simulation.SimulateFromSeed(
		b,
		os.Stdout,
		simApp.GetBaseApp(),
		simapp.AppStateFn(simApp.AppCodec(), simApp.SimulationManager()),
		simulationtypes.RandomAccounts,
		simapp.SimulationOperations(simApp, simApp.AppCodec(), config),
		simApp.ModuleAccountAddrs(),
		config,
		simApp.AppCodec(),
	)

	// export state and simParams before the simulation error is checked
	err = simapp.CheckExportSimulation(simApp, config, simParams)
	require.NoError(b, err)
	require.NoError(b, simErr)

	if config.Commit {
		simapp.PrintStats(db)
	}
}

func BenchmarkYaromSimulation(b *testing.B) {
	simapp.FlagEnabledValue = true
	simapp.FlagCommitValue = true
	fmt.Printf("starting simulation yarom \n")
	config, db, dir, logger, _, err := simapp.SetupSimulation("goleveldb-app-sim", "Simulation")
	require.NoError(b, err, "simulation setup failed")

	b.Cleanup(func() {
		db.Close()
		err = os.RemoveAll(dir)
		require.NoError(b, err)
	})

	encoding := cosmoscmd.MakeEncodingConfig(app.ModuleBasics)

	app := app.New(
		logger,
		db,
		nil,
		true,
		map[int64]bool{},
		app.DefaultNodeHome,
		0,
		encoding,
		simapp.EmptyAppOptions{},
	)

	simApp, ok := app.(SimApp)
	require.True(b, ok, "can't use simapp")

	var a pairingmodule.AppModule
	for _, module := range simApp.SimulationManager().Modules {
		if castModule, ok := module.(pairingmodule.AppModule); ok {
			a = castModule
			break
		}
	}

	// Run randomized simulations
	_, simParams, simErr := simulation.SimulateFromSeed(
		b,
		os.Stdout,
		simApp.GetBaseApp(),
		simapp.AppStateFn(simApp.AppCodec(), simApp.SimulationManager()),
		simulationtypes.RandomAccounts,
		a.YaromStakeOperations(),
		simApp.ModuleAccountAddrs(),
		config,
		simApp.AppCodec(),
	)

	// export state and simParams before the simulation error is checked
	err = simapp.CheckExportSimulation(simApp, config, simParams)
	require.NoError(b, err)
	require.NoError(b, simErr)

	if config.Commit {
		simapp.PrintStats(db)
	}
}

// func SimulationMyOperations(app simapp.App, cdc codec.JSONCodec, config simtypes.Config) []simtypes.WeightedOperation {
// 	simState := module.SimulationState{
// 		AppParams: make(simtypes.AppParams),
// 		Cdc:       cdc,
// 	}

// 	if config.ParamsFile != "" {
// 		bz, err := ioutil.ReadFile(config.ParamsFile)
// 		if err != nil {
// 			panic(err)
// 		}

// 		err = json.Unmarshal(bz, &simState.AppParams)
// 		if err != nil {
// 			panic(err)
// 		}
// 	}

// 	simState.ParamChanges = app.SimulationManager().GenerateParamChanges(config.Seed)
// 	simState.Contents = app.SimulationManager().GetProposalContents(simState)
// 	return myOperations(simState)
// }

// func myOperations(simState module.SimulationState) []simulationtypes.WeightedOperation {

// 	operations := make([]simulationtypes.WeightedOperation, 0)
// 	simApp := app.SimulationManager()

// 	var weightMsgStakeProvider int
// 	simulationtypes.AppParams.GetOrGenerate(simulationtypes.Cdc, opWeightMsgStakeProvider, &weightMsgStakeProvider, nil,
// 		func(_ *rand.Rand) {
// 			weightMsgStakeProvider = defaultWeightMsgStakeProvider
// 		},
// 	)
// 	operations = append(operations, simulation.NewWeightedOperation(
// 		weightMsgStakeProvider,
// 		pairingsimulation.SimulateMsgStakeProvider(accountKeeper, simApp.bankKeeper, simApp.keeper),
// 	))

// 	return nil
// }

// func TestYarom(t *testing.T) {
// 	fmt.Printf("yarom was here")
// 	encoding := cosmoscmd.MakeEncodingConfig(app.ModuleBasics)
// 	genesisState = app.NewDefaultGenesisState(encoding.Marshaler)

// 	genDocProvider := func() (*tmtypes.GenesisDoc, error) {
// 		return &tmtypes.GenesisDoc{
// 			GenesisTime: time.Time{},
// 			ChainID:     "pocket-test",
// 			ConsensusParams: &types.ConsensusParams{
// 				Block: types.BlockParams{
// 					MaxBytes:   15000,
// 					MaxGas:     -1,
// 					TimeIotaMs: 1,
// 				},
// 				Evidence: types.EvidenceParams{
// 					MaxAgeNumBlocks: 1000000,
// 				},
// 				Validator: types.ValidatorParams{
// 					PubKeyTypes: []string{"ed25519"},
// 				},
// 			},
// 			Validators: nil,
// 			AppHash:    nil,
// 			AppState:   genesisState,
// 		}, nil
// 	}
// }
