package ante_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/x/dualstaking/ante"
	"github.com/stretchr/testify/require"
)

// TestDisableRedelegationHooks verifies that the DisableRedelegationHooks works as expected, i.e. disables
// the dualstaking hooks only for redelegations message
func TestDisableRedelegationHooks(t *testing.T) {
	ts := *common.NewTester(t)
	testCases := []struct {
		msg               sdk.Msg
		expectedFlagValue bool
	}{
		{newStakingRedelegateMsg(), true},
		{newStakingDelegateMsg(), false},
		{newStakingUndelegateMsg(), false},
	}

	rf := ante.NewRedelegationFlager(ts.Keepers.Dualstaking)

	for _, testCase := range testCases {
		err := rf.DisableRedelegationHooks(ts.Ctx, []sdk.Msg{testCase.msg})
		require.NoError(t, err)
		disableHooksFlag := ts.Keepers.Dualstaking.GetDisableDualstakingHook(ts.Ctx)
		require.Equal(t, testCase.expectedFlagValue, disableHooksFlag)
	}
}

func newStakingRedelegateMsg() *stakingtypes.MsgBeginRedelegate {
	return stakingtypes.NewMsgBeginRedelegate(
		sdk.AccAddress("del1"),
		sdk.ValAddress("val1"),
		sdk.ValAddress("val2"),
		sdk.NewCoin(commontypes.TokenDenom, sdk.OneInt()),
	)
}

func newStakingDelegateMsg() *stakingtypes.MsgDelegate {
	return stakingtypes.NewMsgDelegate(
		sdk.AccAddress("del1"),
		sdk.ValAddress("val1"),
		sdk.NewCoin(commontypes.TokenDenom, sdk.OneInt()),
	)
}

func newStakingUndelegateMsg() *stakingtypes.MsgUndelegate {
	return stakingtypes.NewMsgUndelegate(
		sdk.AccAddress("del1"),
		sdk.ValAddress("val1"),
		sdk.NewCoin(commontypes.TokenDenom, sdk.OneInt()),
	)
}
