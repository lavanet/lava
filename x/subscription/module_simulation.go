package subscription

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/testutil/sims"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/lavanet/lava/v2/testutil/sample"
	subscriptionsimulation "github.com/lavanet/lava/v2/x/subscription/simulation"
	"github.com/lavanet/lava/v2/x/subscription/types"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = subscriptionsimulation.FindAccount
	_ = sims.StakePerAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
)

const (
	opWeightMsgBuy = "op_weight_msg_buy"
	// TODO: Determine the simulation weight value
	defaultWeightMsgBuy int = 100

	opWeightMsgAddProject = "op_weight_msg_add_project"
	// TODO: Determine the simulation weight value
	defaultWeightMsgAddProject int = 100

	opWeightMsgDelProject = "op_weight_msg_del_project"
	// TODO: Determine the simulation weight value
	defaultWeightMsgDelProject int = 100

	opWeightMsgAutoRenewal = "op_weight_msg_auto_renewal"
	// TODO: Determine the simulation weight value
	defaultWeightMsgAutoRenewal int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	subscriptionGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&subscriptionGenesis)
}

// ProposalContents doesn't return any content functions for governance proposals
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalMsg {
	return nil
}

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgBuy int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgBuy, &weightMsgBuy, nil,
		func(_ *rand.Rand) {
			weightMsgBuy = defaultWeightMsgBuy
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgBuy,
		subscriptionsimulation.SimulateMsgBuy(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgAddProject int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgAddProject, &weightMsgAddProject, nil,
		func(_ *rand.Rand) {
			weightMsgAddProject = defaultWeightMsgAddProject
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgAddProject,
		subscriptionsimulation.SimulateMsgAddProject(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgDelProject int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgDelProject, &weightMsgDelProject, nil,
		func(_ *rand.Rand) {
			weightMsgDelProject = defaultWeightMsgDelProject
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgDelProject,
		subscriptionsimulation.SimulateMsgDelProject(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgAutoRenewal int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgAutoRenewal, &weightMsgAutoRenewal, nil,
		func(_ *rand.Rand) {
			weightMsgAutoRenewal = defaultWeightMsgAutoRenewal
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgAutoRenewal,
		subscriptionsimulation.SimulateMsgAutoRenewal(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
