package types

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v2/testutil/sample"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	"github.com/stretchr/testify/require"
)

func TestMsgStakeProvider_ValidateBasic(t *testing.T) {
	d := stakingtypes.NewDescription("dummy", "", "", "", "")
	tests := []struct {
		name string
		msg  MsgStakeProvider
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgStakeProvider{
				Creator:            "invalid_address",
				Description:        d,
				DelegateLimit:      types.NewCoin(commontypes.TokenDenom, types.ZeroInt()),
				DelegateCommission: 100,
				Validator:          sample.ValAddress(),
				Amount:             types.NewCoin(commontypes.TokenDenom, math.OneInt()),
				Address:            sample.AccAddress(),
			},
			err: legacyerrors.ErrInvalidAddress,
		},
		{
			name: "invalid provider address",
			msg: MsgStakeProvider{
				Creator:            sample.AccAddress(),
				Description:        d,
				DelegateLimit:      types.NewCoin(commontypes.TokenDenom, types.ZeroInt()),
				DelegateCommission: 100,
				Validator:          sample.ValAddress(),
				Amount:             types.NewCoin(commontypes.TokenDenom, math.OneInt()),
				Address:            "invalid_address",
			},
			err: legacyerrors.ErrInvalidAddress,
		},
		{
			name: "invalid validator address",
			msg: MsgStakeProvider{
				Creator:            sample.AccAddress(),
				Description:        d,
				DelegateLimit:      types.NewCoin(commontypes.TokenDenom, types.ZeroInt()),
				DelegateCommission: 100,
				Validator:          "invalid_address",
				Amount:             types.NewCoin(commontypes.TokenDenom, math.OneInt()),
				Address:            sample.AccAddress(),
			},
			err: legacyerrors.ErrInvalidAddress,
		},
		{
			name: "invalid amount",
			msg: MsgStakeProvider{
				Creator:            sample.AccAddress(),
				Description:        d,
				DelegateLimit:      types.NewCoin(commontypes.TokenDenom, types.ZeroInt()),
				DelegateCommission: 100,
				Validator:          sample.ValAddress(),
				Address:            sample.AccAddress(),
			},
			err: legacyerrors.ErrInvalidCoins,
		},
		{
			name: "valid address",
			msg: MsgStakeProvider{
				Creator:            sample.AccAddress(),
				Description:        d,
				DelegateLimit:      types.NewCoin(commontypes.TokenDenom, types.ZeroInt()),
				DelegateCommission: 100,
				Validator:          sample.ValAddress(),
				Amount:             types.NewCoin(commontypes.TokenDenom, math.OneInt()),
				Address:            sample.AccAddress(),
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
