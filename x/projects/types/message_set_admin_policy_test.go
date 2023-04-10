package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgSetAdminPolicy_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgSetAdminPolicy
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgSetAdminPolicy{
				Creator: "invalid_address",
				Policy: Policy{
					EpochCuLimit:       100,
					TotalCuLimit:       1000,
					MaxProvidersToPair: 3,
				},
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgSetAdminPolicy{
				Creator: sample.AccAddress(),
				Policy: Policy{
					EpochCuLimit:       100,
					TotalCuLimit:       1000,
					MaxProvidersToPair: 3,
				},
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
