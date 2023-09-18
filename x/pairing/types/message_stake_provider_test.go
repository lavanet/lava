package types

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgStakeProvider_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgStakeProvider
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgStakeProvider{
				Creator:            "invalid_address",
				Moniker:            "dummyMoniker",
				DelegateLimit:      types.NewCoin("ulava", types.ZeroInt()),
				DelegateCommission: 100,
			},
			err: legacyerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgStakeProvider{
				Creator:            sample.AccAddress(),
				Moniker:            "dummyMoniker",
				DelegateLimit:      types.NewCoin("ulava", types.ZeroInt()),
				DelegateCommission: 100,
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
