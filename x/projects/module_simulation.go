package projects

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/lavanet/lava/testutil/sample"
	projectssimulation "github.com/lavanet/lava/x/projects/simulation"
	"github.com/lavanet/lava/x/projects/types"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = projectssimulation.FindAccount
	_ = simappparams.StakePerAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
)

const (
	opWeightMsgAddProjectKeys = "op_weight_msg_add_project_keys"
	// TODO: Determine the simulation weight value
	defaultWeightMsgAddProjectKeys int = 100

	opWeightMsgSetProjectPolicy = "op_weight_msg_set_project_policy"
	// TODO: Determine the simulation weight value
	defaultWeightMsgSetProjectPolicy int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	projectsGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&projectsGenesis)
}

// ProposalContents doesn't return any content functions for governance proposals
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RandomizedParams creates randomized  param changes for the simulator
func (am AppModule) RandomizedParams(_ *rand.Rand) []simtypes.ParamChange {
	return []simtypes.ParamChange{}
}

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgAddProjectKeys int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgAddProjectKeys, &weightMsgAddProjectKeys, nil,
		func(_ *rand.Rand) {
			weightMsgAddProjectKeys = defaultWeightMsgAddProjectKeys
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgAddProjectKeys,
		projectssimulation.SimulateMsgAddProjectKeys(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgSetProjectPolicy int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgSetProjectPolicy, &weightMsgSetProjectPolicy, nil,
		func(_ *rand.Rand) {
			weightMsgSetProjectPolicy = defaultWeightMsgSetProjectPolicy
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSetProjectPolicy,
		projectssimulation.SimulateMsgSetProjectPolicy(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
