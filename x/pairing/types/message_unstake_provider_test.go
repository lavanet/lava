package types

import (
	"testing"

	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgUnstakeProvider_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgUnstakeProvider
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgUnstakeProvider{
				Creator:   "invalid_address",
				Validator: sample.ValAddress(),
			},
			err: legacyerrors.ErrInvalidAddress,
		},
		{
			name: "invalid validator address",
			msg: MsgUnstakeProvider{
				Creator:   sample.AccAddress(),
				Validator: "invalid_address",
			},
			err: legacyerrors.ErrInvalidAddress,
		},
		{
			name: "valid address",
			msg: MsgUnstakeProvider{
				Creator:   sample.AccAddress(),
				Validator: sample.ValAddress(),
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
