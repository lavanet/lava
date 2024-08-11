package types

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestFundIprpc_ValidateBasic(t *testing.T) {
	tests := []struct {
		name  string
		msg   MsgFundIprpc
		valid bool
	}{
		{
			name: "valid",
			msg: MsgFundIprpc{
				Creator:  sample.AccAddress(),
				Duration: 2,
				Amounts:  sdk.NewCoins(sdk.NewCoin("denom", math.OneInt())),
				Spec:     "spec",
			},
			valid: true,
		},
		{
			name: "invalid creator address",
			msg: MsgFundIprpc{
				Creator:  "invalid_address",
				Duration: 2,
				Amounts:  sdk.NewCoins(sdk.NewCoin("denom", math.OneInt())),
				Spec:     "spec",
			},
			valid: false,
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
