package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgSubscribe_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgSubscribe
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgSubscribe{
				Creator:  "invalid_address",
				Consumer: sample.AccAddress(),
				Index:    "plan-name",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "invalid consumer addresses",
			msg: MsgSubscribe{
				Creator:  sample.AccAddress(),
				Consumer: "invalid_address",
				Index:    "plan-name",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid addresses",
			msg: MsgSubscribe{
				Creator:  sample.AccAddress(),
				Consumer: sample.AccAddress(),
				Index:    "plan-name",
			},
		}, {
			name: "blank plan index",
			msg: MsgSubscribe{
				Creator:  sample.AccAddress(),
				Consumer: sample.AccAddress(),
			},
			err: ErrBlankParameter,
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
