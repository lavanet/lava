package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgProofOfWork_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgProofOfWork
		err  error
	}{
		{
			name: "invalid address",
			msg:  MsgProofOfWork{},
			err:  sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg:  MsgProofOfWork{},
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
