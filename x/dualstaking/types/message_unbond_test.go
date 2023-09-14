package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgUnbond_ValidateBasic(t *testing.T) {
	oneCoin := sdk.NewCoin("utest", sdk.NewInt(1))

	tests := []struct {
		name string
		msg  MsgUnbond
		err  error
	}{
		{
			name: "invalid delegator address",
			msg: MsgUnbond{
				Creator:  "invalid_address",
				Provider: sample.AccAddress(),
				Amount:   oneCoin,
			},
			err: legacyerrors.ErrInvalidAddress,
		}, {
			name: "invalid provider address",
			msg: MsgUnbond{
				Creator:  sample.AccAddress(),
				Provider: "invalid_address",
				Amount:   oneCoin,
			},
			err: legacyerrors.ErrInvalidAddress,
		}, {
			name: "valid addresses and amount",
			msg: MsgUnbond{
				Creator:  sample.AccAddress(),
				Provider: sample.AccAddress(),
				Amount:   oneCoin,
			},
		}, {
			name: "valid amount",
			msg: MsgUnbond{
				Creator:  sample.AccAddress(),
				Provider: sample.AccAddress(),
				Amount:   sdk.Coin{Denom: "utest", Amount: sdk.NewInt(-1)},
			},
			err: legacyerrors.ErrInvalidCoins,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
