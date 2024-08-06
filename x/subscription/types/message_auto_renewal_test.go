package types

import (
	"testing"

	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgAutoRenewal_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgAutoRenewal
		err  error
	}{
		{
			name: "creator invalid address",
			msg: MsgAutoRenewal{
				Creator:  "invalid_address",
				Consumer: sample.AccAddress(),
				Enable:   false,
			},
			err: legacyerrors.ErrInvalidAddress,
		},
		{
			name: "consumer invalid address",
			msg: MsgAutoRenewal{
				Creator:  sample.AccAddress(),
				Consumer: "invalid_address",
				Enable:   false,
			},
			err: legacyerrors.ErrInvalidAddress,
		},
		{
			name: "valid address",
			msg: MsgAutoRenewal{
				Creator:  sample.AccAddress(),
				Consumer: sample.AccAddress(),
				Enable:   false,
			},
		},
		{
			name: "valid plan",
			msg: MsgAutoRenewal{
				Creator:  sample.AccAddress(),
				Consumer: sample.AccAddress(),
				Enable:   true,
				Index:    "free",
			},
		},
		{
			name: "plan not empty on disable",
			msg: MsgAutoRenewal{
				Creator:  sample.AccAddress(),
				Consumer: sample.AccAddress(),
				Enable:   false,
				Index:    "free",
			},
			err: ErrInvalidParameter,
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
