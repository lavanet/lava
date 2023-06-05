package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/testutil/sample"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	"github.com/stretchr/testify/require"
)

func TestMsgAddProject_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgAddProject
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgAddProject{
				Creator: "invalid_address",
				ProjectData: projectstypes.ProjectData{
					Name:   "validName",
					Policy: &projectstypes.Policy{MaxProvidersToPair: 3},
					ProjectKeys: []projectstypes.ProjectKey{
						projectstypes.ProjectAdminKey("invalid address"),
					},
				},
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgAddProject{
				Creator: sample.AccAddress(),
				ProjectData: projectstypes.ProjectData{
					Name:   "validName",
					Policy: &projectstypes.Policy{MaxProvidersToPair: 3},
					ProjectKeys: []projectstypes.ProjectKey{
						projectstypes.ProjectAdminKey(sample.AccAddress()),
					},
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
