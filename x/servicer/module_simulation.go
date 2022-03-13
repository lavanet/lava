package servicer

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/lavanet/lava/testutil/sample"
	servicersimulation "github.com/lavanet/lava/x/servicer/simulation"
	"github.com/lavanet/lava/x/servicer/types"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = servicersimulation.FindAccount
	_ = simappparams.StakePerAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
)

const (
// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	servicerGenesis := types.GenesisState{
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&servicerGenesis)
}

// ProposalContents doesn't return any content functions for governance proposals
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RandomizedParams creates randomized  param changes for the simulator
func (am AppModule) RandomizedParams(_ *rand.Rand) []simtypes.ParamChange {
	servicerParams := types.DefaultParams()
	return []simtypes.ParamChange{
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyMinStake), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(servicerParams.MinStake))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyCoinsPerCU), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(servicerParams.CoinsPerCU))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyUnstakeHoldBlocks), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(servicerParams.UnstakeHoldBlocks))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyFraudStakeSlashingFactor), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(servicerParams.FraudStakeSlashingFactor))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyFraudSlashingAmount), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(servicerParams.FraudSlashingAmount))
		}),
	}
}

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
