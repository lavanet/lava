package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v4/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgRedelegate_ValidateBasic(t *testing.T) {
	oneCoin := sdk.NewCoin("utest", sdk.NewInt(1))

	tests := []struct {
		name string
		msg  MsgRedelegate
		err  error
	}{
		{
			name: "invalid delegator address",
			msg: MsgRedelegate{
				Creator:      "invalid_address",
				FromProvider: sample.AccAddress(),
				ToProvider:   sample.AccAddress(),
				Amount:       oneCoin,
				FromChainID:  "",
				ToChainID:    "",
			},
			err: legacyerrors.ErrInvalidAddress,
		}, {
			name: "invalid provider address",
			msg: MsgRedelegate{
				Creator:      sample.AccAddress(),
				FromProvider: "invalid_address",
				ToProvider:   sample.AccAddress(),
				Amount:       oneCoin,
				FromChainID:  "",
				ToChainID:    "",
			},
			err: legacyerrors.ErrInvalidAddress,
		}, {
			name: "valid address and amount",
			msg: MsgRedelegate{
				Creator:      sample.AccAddress(),
				FromProvider: sample.AccAddress(),
				ToProvider:   sample.AccAddress(),
				Amount:       oneCoin,
				FromChainID:  "",
				ToChainID:    "",
			},
		}, {
			name: "invalid amount",
			msg: MsgRedelegate{
				Creator:      sample.AccAddress(),
				FromProvider: sample.AccAddress(),
				ToProvider:   sample.AccAddress(),
				Amount:       sdk.Coin{Denom: "utest", Amount: sdk.NewInt(-1)},
				FromChainID:  "",
				ToChainID:    "",
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
