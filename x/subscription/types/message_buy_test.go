package types

import (
	"testing"

	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgBuy(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgBuy
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgBuy{
				Creator:  "invalid_address",
				Consumer: sample.AccAddress(),
				Index:    "plan-name",
				Duration: 1,
			},
			err: legacyerrors.ErrInvalidAddress,
		}, {
			name: "invalid consumer addresses",
			msg: MsgBuy{
				Creator:  sample.AccAddress(),
				Consumer: "invalid_address",
				Index:    "plan-name",
				Duration: 1,
			},
			err: legacyerrors.ErrInvalidAddress,
		}, {
			name: "valid addresses",
			msg: MsgBuy{
				Creator:  sample.AccAddress(),
				Consumer: sample.AccAddress(),
				Index:    "plan-name",
				Duration: 1,
			},
		}, {
			name: "blank plan index",
			msg: MsgBuy{
				Creator:  sample.AccAddress(),
				Consumer: sample.AccAddress(),
				Duration: 1,
			},
			err: ErrBlankParameter,
		}, {
			name: "invalid duration 0",
			msg: MsgBuy{
				Creator:  sample.AccAddress(),
				Consumer: sample.AccAddress(),
				Index:    "plan-name",
				Duration: 0,
			},
			err: ErrInvalidParameter,
		}, {
			name: "invalid duration too long",
			msg: MsgBuy{
				Creator:  sample.AccAddress(),
				Consumer: sample.AccAddress(),
				Index:    "plan-name",
				Duration: 13,
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
