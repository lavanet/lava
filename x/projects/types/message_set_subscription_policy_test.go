package types

import (
	"testing"

	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/testutil/sample"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	"github.com/stretchr/testify/require"
)

func TestMsgSetSubscriptionPolicy_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgSetSubscriptionPolicy
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgSetSubscriptionPolicy{
				Creator: "invalid_address",
				Policy: &planstypes.Policy{
					EpochCuLimit:       10,
					TotalCuLimit:       100,
					MaxProvidersToPair: 3,
					GeolocationProfile: 1,
				},
			},
			err: legacyerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgSetSubscriptionPolicy{
				Creator: sample.AccAddress(),
				Policy: &planstypes.Policy{
					EpochCuLimit:       10,
					TotalCuLimit:       100,
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
