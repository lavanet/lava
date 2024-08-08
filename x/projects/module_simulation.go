package projects

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/testutil/sims"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/lavanet/lava/v2/testutil/sample"
	projectssimulation "github.com/lavanet/lava/v2/x/projects/simulation"
	"github.com/lavanet/lava/v2/x/projects/types"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = projectssimulation.FindAccount
	_ = sims.StakePerAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
)

const (
	opWeightMsgAddKeys = "op_weight_msg_add_keys"
	// TODO: Determine the simulation weight value
	defaultWeightMsgAddKeys int = 100

	opWeightMsgDelKeys = "op_weight_msg_del_keys"
	// TODO: Determine the simulation weight value
	defaultWeightMsgDelKeys int = 100

	opWeightMsgSetPolicy = "op_weight_msg_set_admin_policy"
	// TODO: Determine the simulation weight value
	defaultWeightMsgSetPolicy int = 100

	opWeightMsgSetSubscriptionPolicy = "op_weight_msg_set_subscription_policy"
	// TODO: Determine the simulation weight value
	defaultWeightMsgSetSubscriptionPolicy int = 100

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
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalMsg {
	return nil
}

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgAddKeys int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgAddKeys, &weightMsgAddKeys, nil,
		func(_ *rand.Rand) {
			weightMsgAddKeys = defaultWeightMsgAddKeys
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgAddKeys,
		projectssimulation.SimulateMsgAddKeys(am.keeper),
	))

	var weightMsgDelKeys int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgDelKeys, &weightMsgDelKeys, nil,
		func(_ *rand.Rand) {
			weightMsgDelKeys = defaultWeightMsgDelKeys
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgDelKeys,
		projectssimulation.SimulateMsgDelKeys(am.keeper),
	))

	var weightMsgSetPolicy int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgSetPolicy, &weightMsgSetPolicy, nil,
		func(_ *rand.Rand) {
			weightMsgSetPolicy = defaultWeightMsgSetPolicy
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSetPolicy,
		projectssimulation.SimulateMsgSetPolicy(am.keeper),
	))

	var weightMsgSetSubscriptionPolicy int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgSetSubscriptionPolicy, &weightMsgSetSubscriptionPolicy, nil,
		func(_ *rand.Rand) {
			weightMsgSetSubscriptionPolicy = defaultWeightMsgSetSubscriptionPolicy
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSetSubscriptionPolicy,
		projectssimulation.SimulateMsgSetSubscriptionPolicy(am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
