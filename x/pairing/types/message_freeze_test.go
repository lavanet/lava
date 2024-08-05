package types

import (
	"testing"

	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgFreeze_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgFreezeProvider
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgFreezeProvider{
				Creator: "invalid_address",
			},
			err: legacyerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgFreezeProvider{
				Creator: sample.AccAddress(),
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
