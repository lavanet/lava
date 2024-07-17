package types

import (
	"testing"

	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/testutil/sample"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	"github.com/stretchr/testify/require"
)

func TestMsgSetPolicy_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgSetPolicy
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgSetPolicy{
				Creator: "invalid_address",
				Policy: &planstypes.Policy{
					EpochCuLimit:       100,
					TotalCuLimit:       1000,
					MaxProvidersToPair: 3,
					GeolocationProfile: 1,
				},
			},
			err: legacyerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgSetPolicy{
				Creator: sample.AccAddress(),
				Policy: &planstypes.Policy{
					EpochCuLimit:       100,
					TotalCuLimit:       1000,
					MaxProvidersToPair: 3,
					GeolocationProfile: 1,
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
