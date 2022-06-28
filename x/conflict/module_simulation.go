package conflict

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/lavanet/lava/testutil/sample"
	conflictsimulation "github.com/lavanet/lava/x/conflict/simulation"
	"github.com/lavanet/lava/x/conflict/types"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = conflictsimulation.FindAccount
	_ = simappparams.StakePerAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
)

const (
	opWeightMsgDetection = "op_weight_msg_create_chain"
	// TODO: Determine the simulation weight value
	defaultWeightMsgDetection int = 100

	opWeightMsgConflictVoteCommit = "op_weight_msg_conflict_vote_commit"
	// TODO: Determine the simulation weight value
	defaultWeightMsgConflictVoteCommit int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	conflictGenesis := types.GenesisState{
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&conflictGenesis)
}

// ProposalContents doesn't return any content functions for governance proposals
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RandomizedParams creates randomized  param changes for the simulator
func (am AppModule) RandomizedParams(_ *rand.Rand) []simtypes.ParamChange {
	conflictParams := types.DefaultParams()
	return []simtypes.ParamChange{
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyMajorityPercent), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(conflictParams.MajorityPercent))
		}),
	}
}

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgDetection int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgDetection, &weightMsgDetection, nil,
		func(_ *rand.Rand) {
			weightMsgDetection = defaultWeightMsgDetection
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgDetection,
		conflictsimulation.SimulateMsgDetection(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgConflictVoteCommit int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgConflictVoteCommit, &weightMsgConflictVoteCommit, nil,
		func(_ *rand.Rand) {
			weightMsgConflictVoteCommit = defaultWeightMsgConflictVoteCommit
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgConflictVoteCommit,
		conflictsimulation.SimulateMsgConflictVoteCommit(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
