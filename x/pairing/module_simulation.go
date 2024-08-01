package pairing

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	types2 "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/lavanet/lava/v2/testutil/sample"
	pairingsimulation "github.com/lavanet/lava/v2/x/pairing/simulation"
	"github.com/lavanet/lava/v2/x/pairing/types"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = pairingsimulation.FindAccount
	_ = sims.StakePerAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
)

const (
	opWeightMsgStakeProvider = "op_weight_msg_stake_provider"
	// TODO: Determine the simulation weight value
	defaultWeightMsgStakeProvider int = 100

	opWeightMsgStakeClient = "op_weight_msg_stake_client"
	// TODO: Determine the simulation weight value
	defaultWeightMsgStakeClient int = 100

	opWeightMsgUnstakeProvider = "op_weight_msg_unstake_provider"
	// TODO: Determine the simulation weight value
	defaultWeightMsgUnstakeProvider int = 100

	opWeightMsgUnstakeClient = "op_weight_msg_unstake_client"
	// TODO: Determine the simulation weight value
	defaultWeightMsgUnstakeClient int = 100

	opWeightMsgRelayPayment = "op_weight_msg_relay_payment"
	// TODO: Determine the simulation weight value
	defaultWeightMsgRelayPayment int = 100

	opWeightMsgFreeze = "op_weight_msg_freeze"
	// TODO: Determine the simulation weight value
	defaultWeightMsgFreeze int = 100

	opWeightMsgUnfreeze = "op_weight_msg_unfreeze"
	// TODO: Determine the simulation weight value
	defaultWeightMsgUnfreeze int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	pairingGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&pairingGenesis)
}

// ProposalContents doesn't return any content functions for governance proposals
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalMsg {
	return nil
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
//	pairingParams := types.DefaultParams()
//	return []simtypes.ParamChange{
//		simulation.NewSimParamChange(types.ModuleName, string(types.KeyFraudStakeSlashingFactor), func(r *rand.Rand) string {
//			return string(types.Amino.MustMarshalJSON(pairingParams.FraudStakeSlashingFactor))
//		}),
//		simulation.NewSimParamChange(types.ModuleName, string(types.KeyFraudSlashingAmount), func(r *rand.Rand) string {
//			return string(types.Amino.MustMarshalJSON(pairingParams.FraudSlashingAmount))
//		}),
//		simulation.NewSimParamChange(types.ModuleName, string(types.KeyEpochBlocksOverlap), func(r *rand.Rand) string {
//			return string(types.Amino.MustMarshalJSON(pairingParams.EpochBlocksOverlap))
//		}),
//		simulation.NewSimParamChange(types.ModuleName, string(types.KeyRecommendedEpochNumToCollectPayment), func(r *rand.Rand) string {
//			return string(types.Amino.MustMarshalJSON(pairingParams.RecommendedEpochNumToCollectPayment))
//		}),
//	}
// }

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgStakeProvider int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgStakeProvider, &weightMsgStakeProvider, nil,
		func(_ *rand.Rand) {
			weightMsgStakeProvider = defaultWeightMsgStakeProvider
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgStakeProvider,
		pairingsimulation.SimulateMsgStakeProvider(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgStakeClient int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgStakeClient, &weightMsgStakeClient, nil,
		func(_ *rand.Rand) {
			weightMsgStakeClient = defaultWeightMsgStakeClient
		},
	)

	var weightMsgUnstakeProvider int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgUnstakeProvider, &weightMsgUnstakeProvider, nil,
		func(_ *rand.Rand) {
			weightMsgUnstakeProvider = defaultWeightMsgUnstakeProvider
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUnstakeProvider,
		pairingsimulation.SimulateMsgUnstakeProvider(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgUnstakeClient int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgUnstakeClient, &weightMsgUnstakeClient, nil,
		func(_ *rand.Rand) {
			weightMsgUnstakeClient = defaultWeightMsgUnstakeClient
		},
	)

	var weightMsgRelayPayment int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgRelayPayment, &weightMsgRelayPayment, nil,
		func(_ *rand.Rand) {
			weightMsgRelayPayment = defaultWeightMsgRelayPayment
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgRelayPayment,
		pairingsimulation.SimulateMsgRelayPayment(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgFreeze int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgFreeze, &weightMsgFreeze, nil,
		func(_ *rand.Rand) {
			weightMsgFreeze = defaultWeightMsgFreeze
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgFreeze,
		pairingsimulation.SimulateMsgFreeze(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgUnfreeze int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgUnfreeze, &weightMsgUnfreeze, nil,
		func(_ *rand.Rand) {
			weightMsgUnfreeze = defaultWeightMsgUnfreeze
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUnfreeze,
		pairingsimulation.SimulateMsgUnfreeze(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
