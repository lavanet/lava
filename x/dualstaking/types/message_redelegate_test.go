package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgRedelegate_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgRedelegate
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgRedelegate{
				Delegator: "invalid_address",
				Amount:    sdk.NewCoin("utest", sdk.NewInt(1)),
			},
			err: legacyerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgRedelegate{
				Delegator: sample.AccAddress(),
				Amount:    sdk.NewCoin("utest", sdk.NewInt(1)),
			},
		}, {
			name: "valid amount",
			msg: MsgRedelegate{
				Delegator: sample.AccAddress(),
				Amount:    sdk.NewCoin("utest", sdk.NewInt(1)),
			},
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
