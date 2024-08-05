package protocol

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
	protocolsimulation "github.com/lavanet/lava/v2/x/protocol/simulation"
	"github.com/lavanet/lava/v2/x/protocol/types"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = protocolsimulation.FindAccount
	_ = sims.StakePerAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
)

const (
// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	protocolGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&protocolGenesis)
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
//	protocolParams := types.DefaultParams()
//	return []simtypes.ParamChange{
//		simulation.NewSimParamChange(types.ModuleName, string(types.KeyVersion), func(r *rand.Rand) string {
//			return string(types.Amino.MustMarshalJSON(protocolParams.Version))
//		}),
//	}
// }

// RegisterStoreDecoder registers a decoder
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}
