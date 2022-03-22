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
	opWeightMsgStakeServicer = "op_weight_msg_create_chain"
	// TODO: Determine the simulation weight value
	defaultWeightMsgStakeServicer int = 100

	opWeightMsgUnstakeServicer = "op_weight_msg_create_chain"
	// TODO: Determine the simulation weight value
	defaultWeightMsgUnstakeServicer int = 100

	opWeightMsgProofOfWork = "op_weight_msg_create_chain"
	// TODO: Determine the simulation weight value
	defaultWeightMsgProofOfWork int = 100

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
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyServicersToPairCount), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(servicerParams.ServicersToPairCount))
		}),
	}
}

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgStakeServicer int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgStakeServicer, &weightMsgStakeServicer, nil,
		func(_ *rand.Rand) {
			weightMsgStakeServicer = defaultWeightMsgStakeServicer
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgStakeServicer,
		servicersimulation.SimulateMsgStakeServicer(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgUnstakeServicer int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgUnstakeServicer, &weightMsgUnstakeServicer, nil,
		func(_ *rand.Rand) {
			weightMsgUnstakeServicer = defaultWeightMsgUnstakeServicer
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUnstakeServicer,
		servicersimulation.SimulateMsgUnstakeServicer(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgProofOfWork int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgProofOfWork, &weightMsgProofOfWork, nil,
		func(_ *rand.Rand) {
			weightMsgProofOfWork = defaultWeightMsgProofOfWork
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgProofOfWork,
		servicersimulation.SimulateMsgProofOfWork(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
