package spec_test

import (
	"testing"

	types2 "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"

	keepertest "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/testutil/nullify"
	"github.com/lavanet/lava/v2/x/spec"
	"github.com/lavanet/lava/v2/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	params := types.DefaultParams()
	params.AllowlistedExpeditedMsgs = []string{proto.MessageName(&types2.MsgUpdateParams{})}

	genesisState := types.GenesisState{
		Params: params,
		SpecList: []types.Spec{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		SpecCount: 2,

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.SpecKeeper(t)
	spec.InitGenesis(ctx, *k, genesisState)
	got := spec.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.SpecList, got.SpecList)
	require.Equal(t, genesisState.SpecCount, got.SpecCount)
	// this line is used by starport scaffolding # genesis/test/assert
}
