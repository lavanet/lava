package ante

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/gogoproto/proto"
	"github.com/lavanet/lava/app"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	types2 "github.com/lavanet/lava/x/spec/types"
	types3 "github.com/lavanet/lava/x/subscription/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewExpeditedProposalFilterAnteDecorator(t *testing.T) {
	tests := []struct {
		name       string
		theMsg     func() types.Msg
		shouldFail bool
	}{
		{
			name: "should fail if any of the messages are in the blacklist, expedited",
			theMsg: func() types.Msg {
				proposal, err := v1.NewMsgSubmitProposal(
					[]types.Msg{
						&banktypes.MsgSend{},
					},
					types.NewCoins(types.NewCoin("lava", types.NewInt(100))),
					"cosmos1qypqxpq9qcrsszgjx3ysxf7j8xq9q9qyq9q9q9",
					"metadata",
					"title",
					"summary",
					true,
				)
				require.NoError(t, err)

				return proposal
			},
			shouldFail: true,
		},
		{
			name: "should not fail if none of the messages are in the blacklist, expedited",
			theMsg: func() types.Msg {
				proposal, err := v1.NewMsgSubmitProposal(
					[]types.Msg{
						&types3.MsgAutoRenewal{},
					},
					types.NewCoins(types.NewCoin("lava", types.NewInt(100))),
					"cosmos1qypqxpq9qcrsszgjx3ysxf7j8xq9q9qyq9q9q9",
					"metadata",
					"title",
					"summary",
					true,
				)
				require.NoError(t, err)

				return proposal
			},
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			k, ctx := testkeeper.SpecKeeper(t)
			params := types2.DefaultParams()
			params.BlacklistedExpeditedMsgs = []string{
				proto.MessageName(&banktypes.MsgSend{}),
			} // we whitelist MsgSend proposal

			k.SetParams(ctx, params)

			encodingConfig := app.MakeEncodingConfig()
			clientCtx := client.Context{}.
				WithTxConfig(encodingConfig.TxConfig)

			txBuilder := clientCtx.TxConfig.NewTxBuilder()
			err := txBuilder.SetMsgs(tt.theMsg())
			require.NoError(t, err)

			tx := txBuilder.GetTx()
			anteHandler := NewExpeditedProposalFilterAnteDecorator(k)

			_, err = anteHandler.AnteHandle(ctx, tx, false, func(ctx types.Context, tx types.Tx, simulate bool) (newCtx types.Context, err error) {
				return ctx, nil
			})
			if tt.shouldFail {
				require.Error(t, err)
				require.ErrorContainsf(t, err, "proposal contains blacklisted message", "expected error to contain %s", "proposal contains blacklisted messages")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
