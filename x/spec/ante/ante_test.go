package ante

import (
	"github.com/cosmos/cosmos-sdk/types"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	types2 "github.com/lavanet/lava/x/spec/types"
	"testing"
)

func TestNewExpeditedProposalFilterAnteDecorator(t *testing.T) {
	tests := []struct {
		name       string
		txMsgs     []types.Msg
		shouldFail bool
	}{
		{
			name:       "should fail if any of the messages are in the blacklist",
			txMsgs:     []types.Msg{},
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, ctx := testkeeper.SpecKeeper(t)
			params := types2.DefaultParams()

			k.SetParams(ctx, params)
		})
	}
}
