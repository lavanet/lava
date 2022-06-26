package pairing

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/lavanet/lava/testutil/sample"
	pairingsimulation "github.com/lavanet/lava/x/pairing/simulation"
	"github.com/lavanet/lava/x/pairing/types"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = pairingsimulation.FindAccount
	_ = simappparams.StakePerAccount
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
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RandomizedParams creates randomized  param changes for the simulator
func (am AppModule) RandomizedParams(_ *rand.Rand) []simtypes.ParamChange {
	pairingParams := types.DefaultParams()
	return []simtypes.ParamChange{
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyMinStakeProvider), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(pairingParams.MinStakeProvider))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyMinStakeClient), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(pairingParams.MinStakeClient))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyMintCoinsPerCU), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(pairingParams.MintCoinsPerCU))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyBurnCoinsPerCU), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(pairingParams.BurnCoinsPerCU))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyFraudStakeSlashingFactor), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(pairingParams.FraudStakeSlashingFactor))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyFraudSlashingAmount), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(pairingParams.FraudSlashingAmount))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyServicersToPairCount), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(pairingParams.ServicersToPairCount))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyEpochBlocksOverlap), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(pairingParams.EpochBlocksOverlap))
		}),
	}
}

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
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgStakeClient,
		pairingsimulation.SimulateMsgStakeClient(am.accountKeeper, am.bankKeeper, am.keeper),
	))

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
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUnstakeClient,
		pairingsimulation.SimulateMsgUnstakeClient(am.accountKeeper, am.bankKeeper, am.keeper),
	))

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

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
