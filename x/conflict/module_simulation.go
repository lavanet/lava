package conflict

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/testutil/sims"
	types2 "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/lavanet/lava/v2/testutil/sample"
	conflictsimulation "github.com/lavanet/lava/v2/x/conflict/simulation"
	"github.com/lavanet/lava/v2/x/conflict/types"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = conflictsimulation.FindAccount
	_ = sims.StakePerAccount
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

	opWeightMsgConflictVoteReveal = "op_weight_msg_conflict_vote_reveal"
	// TODO: Determine the simulation weight value
	defaultWeightMsgConflictVoteReveal int = 100

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

// TODO: Add weighted proposals
func (AppModule) ProposalMsgs(_ module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg("op_weight_msg_update_params", 100, func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
			return &types2.MsgUpdateParams{}
		}),
	}
}

//// RandomizedParams creates randomized  param changes for the simulator
// func (am AppModule) RandomizedParams(_ *rand.Rand) []simtypes.ParamChange {
//	conflictParams := types.DefaultParams()
//	return []simtypes.ParamChange{
//		simulation.NewSimParamChange(types.ModuleName, string(types.KeyMajorityPercent), func(r *rand.Rand) string {
//			return string(types.Amino.MustMarshalJSON(conflictParams.MajorityPercent))
//		}),
//	}
// }

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

	var weightMsgConflictVoteReveal int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgConflictVoteReveal, &weightMsgConflictVoteReveal, nil,
		func(_ *rand.Rand) {
			weightMsgConflictVoteReveal = defaultWeightMsgConflictVoteReveal
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgConflictVoteReveal,
		conflictsimulation.SimulateMsgConflictVoteReveal(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
