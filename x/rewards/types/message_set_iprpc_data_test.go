package types

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/testutil/sample"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	"github.com/stretchr/testify/require"
)

func TestSetIprpcData_ValidateBasic(t *testing.T) {
	tests := []struct {
		name  string
		msg   MsgSetIprpcData
		valid bool
	}{
		{
			name: "invalid authority address",
			msg: MsgSetIprpcData{
				Authority:          "invalid_address",
				MinIprpcCost:       sdk.NewCoin(commontypes.TokenDenom, math.ZeroInt()),
				IprpcSubscriptions: []string{sample.AccAddress()},
			},
			valid: false,
		},
		{
			name: "invalid subscription address",
			msg: MsgSetIprpcData{
				Authority:          sample.AccAddress(),
				MinIprpcCost:       sdk.NewCoin(commontypes.TokenDenom, math.ZeroInt()),
				IprpcSubscriptions: []string{sample.AccAddress(), "invalid_address"},
			},
			valid: false,
		},
		{
			name: "valid message",
			msg: MsgSetIprpcData{
				Authority:          sample.AccAddress(),
				MinIprpcCost:       sdk.NewCoin(commontypes.TokenDenom, math.ZeroInt()),
				IprpcSubscriptions: []string{sample.AccAddress()},
			},
			valid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.valid {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
		})
	}
}
