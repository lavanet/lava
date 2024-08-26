package ante_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v2/testutil/common"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	"github.com/lavanet/lava/v2/x/dualstaking/ante"
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
		{newAuthzMsg([]sdk.Msg{newStakingRedelegateMsg()}), true},
		{newAuthzMsg([]sdk.Msg{newAuthzMsg([]sdk.Msg{newStakingRedelegateMsg()})}), true},
	}

	rf := ante.NewRedelegationFlager(ts.Keepers.Dualstaking)

	for _, testCase := range testCases {
		_, err := rf.AnteHandle(ts.Ctx, TxMock{msgs: []sdk.Msg{testCase.msg}}, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) { return })
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

func newAuthzMsg(msgs []sdk.Msg) *authz.MsgExec {
	msg := authz.NewMsgExec(sdk.AccAddress("grantee"), msgs)
	return &msg
}

type TxMock struct {
	msgs []sdk.Msg
}

func (tmock TxMock) GetMsgs() []sdk.Msg {
	return tmock.msgs
}

func (tmock TxMock) ValidateBasic() error {
	return nil
}
