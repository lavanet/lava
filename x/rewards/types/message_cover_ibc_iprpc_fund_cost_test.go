package types

import (
	"testing"

	"github.com/lavanet/lava/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestCoverIbcIprpcFundCost_ValidateBasic(t *testing.T) {
	tests := []struct {
		name  string
		msg   MsgCoverIbcIprpcFundCost
		valid bool
	}{
		{
			name: "valid",
			msg: MsgCoverIbcIprpcFundCost{
				Creator: sample.AccAddress(),
				Index:   1,
			},
			valid: true,
		},
		{
			name: "invalid creator address",
			msg: MsgCoverIbcIprpcFundCost{
				Creator: "invalid_address",
				Index:   1,
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
