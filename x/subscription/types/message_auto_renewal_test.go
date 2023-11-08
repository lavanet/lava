package types

import (
	"testing"

	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgAutoRenewal_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgAutoRenewal
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgAutoRenewal{
				Creator: "invalid_address",
				Enable:  false,
			},
			err: legacyerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgAutoRenewal{
				Creator: sample.AccAddress(),
				Enable:  false,
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
