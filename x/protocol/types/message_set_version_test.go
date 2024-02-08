package types

import (
	"testing"

	"github.com/lavanet/lava/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestSetIprpcData_ValidateBasic(t *testing.T) {
	tests := []struct {
		name  string
		msg   MsgSetVersion
		valid bool
	}{
		{
			name: "invalid authority address",
			msg: MsgSetVersion{
				Authority: "invalid_address",
			},
			valid: false,
		},
		{
			name: "invalid subscription address",
			msg: MsgSetVersion{
				Authority: sample.AccAddress(),
			},
			valid: false,
		},
		{
			name: "valid message",
			msg: MsgSetVersion{
				Authority: sample.AccAddress(),
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
