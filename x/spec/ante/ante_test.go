package ante

import (
	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
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
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			k, ctx := testkeeper.SpecKeeper(t)
			params := types2.DefaultParams()
			params.BlacklistedExpeditedMsgs = []string{
				proto.MessageName(&bankv1beta1.MsgSend{}),
			} // we whitelist MsgSend proposal

			k.SetParams(ctx, params)

			anteHandler := NewExpeditedProposalFilterAnteDecorator(k)
		})
	}
}
