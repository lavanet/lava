package user

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/lavanet/lava/testutil/sample"
	usersimulation "github.com/lavanet/lava/x/user/simulation"
	"github.com/lavanet/lava/x/user/types"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = usersimulation.FindAccount
	_ = simappparams.StakePerAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
)

const (
	opWeightMsgStakeUser = "op_weight_msg_create_chain"
	// TODO: Determine the simulation weight value
	defaultWeightMsgStakeUser int = 100

	opWeightMsgUnstakeUser = "op_weight_msg_create_chain"
	// TODO: Determine the simulation weight value
	defaultWeightMsgUnstakeUser int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	userGenesis := types.GenesisState{
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&userGenesis)
}

// ProposalContents doesn't return any content functions for governance proposals
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RandomizedParams creates randomized  param changes for the simulator
func (am AppModule) RandomizedParams(_ *rand.Rand) []simtypes.ParamChange {
	userParams := types.DefaultParams()
	return []simtypes.ParamChange{
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyMinStake), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(userParams.MinStake))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyCoinsPerCU), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(userParams.CoinsPerCU))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyUnstakeHoldBlocks), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(userParams.UnstakeHoldBlocks))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyFraudStakeSlashingFactor), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(userParams.FraudStakeSlashingFactor))
		}),
		simulation.NewSimParamChange(types.ModuleName, string(types.KeyFraudSlashingAmount), func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(userParams.FraudSlashingAmount))
		}),
	}
}

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgStakeUser int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgStakeUser, &weightMsgStakeUser, nil,
		func(_ *rand.Rand) {
			weightMsgStakeUser = defaultWeightMsgStakeUser
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgStakeUser,
		usersimulation.SimulateMsgStakeUser(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgUnstakeUser int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgUnstakeUser, &weightMsgUnstakeUser, nil,
		func(_ *rand.Rand) {
			weightMsgUnstakeUser = defaultWeightMsgUnstakeUser
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUnstakeUser,
		usersimulation.SimulateMsgUnstakeUser(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
