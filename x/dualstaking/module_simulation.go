package dualstaking

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/lavanet/lava/v2/testutil/sample"
	dualstakingsimulation "github.com/lavanet/lava/v2/x/dualstaking/simulation"
	"github.com/lavanet/lava/v2/x/dualstaking/types"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = dualstakingsimulation.FindAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
	_ = rand.Rand{}
)

const (
	opWeightMsgDelegate = "op_weight_msg_delegate"
	// TODO: Determine the simulation weight value
	defaultWeightMsgDelegate int = 100

	opWeightMsgRedelegate = "op_weight_msg_redelegate"
	// TODO: Determine the simulation weight value
	defaultWeightMsgRedelegate int = 100

	opWeightMsgUnbond = "op_weight_msg_unbond"
	// TODO: Determine the simulation weight value
	defaultWeightMsgUnbond int = 100

	opWeightMsgClaimRewards = "op_weight_msg_claim_rewards"
	// TODO: Determine the simulation weight value
	defaultWeightMsgClaimRewards int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	dualstakingGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&dualstakingGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// ProposalContents doesn't return any content functions for governance proposals.
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalMsg {
	return nil
}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgDelegate int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgDelegate, &weightMsgDelegate, nil,
		func(_ *rand.Rand) {
			weightMsgDelegate = defaultWeightMsgDelegate
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgDelegate,
		dualstakingsimulation.SimulateMsgDelegate(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgRedelegate int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgRedelegate, &weightMsgRedelegate, nil,
		func(_ *rand.Rand) {
			weightMsgRedelegate = defaultWeightMsgRedelegate
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgRedelegate,
		dualstakingsimulation.SimulateMsgRedelegate(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgUnbond int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgUnbond, &weightMsgUnbond, nil,
		func(_ *rand.Rand) {
			weightMsgUnbond = defaultWeightMsgUnbond
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUnbond,
		dualstakingsimulation.SimulateMsgUnbond(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgClaimRewards int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgClaimRewards, &weightMsgClaimRewards, nil,
		func(_ *rand.Rand) {
			weightMsgClaimRewards = defaultWeightMsgClaimRewards
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgClaimRewards,
		dualstakingsimulation.SimulateMsgClaimRewards(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
