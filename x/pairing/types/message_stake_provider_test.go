package types

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	commontypes "github.com/lavanet/lava/common/types"
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
				DelegateLimit:      types.NewCoin(commontypes.TokenDenom, types.ZeroInt()),
				DelegateCommission: 100,
				Validator:          sample.ValAddress(),
				Amount:             types.NewCoin(commontypes.TokenDenom, math.OneInt()),
			},
			err: legacyerrors.ErrInvalidAddress,
		},
		{
			name: "invalid validator address",
			msg: MsgStakeProvider{
				Creator:            sample.AccAddress(),
				Moniker:            "dummyMoniker",
				DelegateLimit:      types.NewCoin(commontypes.TokenDenom, types.ZeroInt()),
				DelegateCommission: 100,
				Validator:          "invalid_address",
				Amount:             types.NewCoin(commontypes.TokenDenom, math.OneInt()),
			},
			err: legacyerrors.ErrInvalidAddress,
		},
		{
			name: "invalid amount",
			msg: MsgStakeProvider{
				Creator:            sample.AccAddress(),
				Moniker:            "dummyMoniker",
				DelegateLimit:      types.NewCoin(commontypes.TokenDenom, types.ZeroInt()),
				DelegateCommission: 100,
				Validator:          sample.ValAddress(),
			},
			err: legacyerrors.ErrInvalidCoins,
		},
		{
			name: "valid address",
			msg: MsgStakeProvider{
				Creator:            sample.AccAddress(),
				Moniker:            "dummyMoniker",
				DelegateLimit:      types.NewCoin(commontypes.TokenDenom, types.ZeroInt()),
				DelegateCommission: 100,
				Validator:          sample.ValAddress(),
				Amount:             types.NewCoin(commontypes.TokenDenom, math.OneInt()),
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
