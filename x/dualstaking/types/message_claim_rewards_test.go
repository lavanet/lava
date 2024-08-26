package types

import (
	"testing"

	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgClaimRewards_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgClaimRewards
		err  error
	}{
		{
			name: "invalid delegator address",
			msg: MsgClaimRewards{
				Creator:  "invalid_address",
				Provider: sample.AccAddress(),
			},
			err: legacyerrors.ErrInvalidAddress,
		}, {
			name: "invalid provider address",
			msg: MsgClaimRewards{
				Creator:  sample.AccAddress(),
				Provider: "invalid_address",
			},
			err: legacyerrors.ErrInvalidAddress,
		}, {
			name: "valid addresses",
			msg: MsgClaimRewards{
				Creator:  sample.AccAddress(),
				Provider: sample.AccAddress(),
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
