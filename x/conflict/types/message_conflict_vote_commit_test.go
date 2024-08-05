package types

import (
	"testing"

	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgConflictVoteCommit_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgConflictVoteCommit
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgConflictVoteCommit{
				Creator: "invalid_address",
			},
			err: legacyerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgConflictVoteCommit{
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
