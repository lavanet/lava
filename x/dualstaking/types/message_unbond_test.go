package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgUnbond_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgUnbond
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgUnbond{
				Delegator: "invalid_address",
				Amount:    sdk.NewCoin("utest", sdk.NewInt(1)),
			},
			err: legacyerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgUnbond{
				Delegator: sample.AccAddress(),
				Amount:    sdk.NewCoin("utest", sdk.NewInt(1)),
			},
		}, {
			name: "valid amount",
			msg: MsgUnbond{
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
