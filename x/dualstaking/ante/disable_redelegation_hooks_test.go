package ante_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v4/testutil/common"
	commontypes "github.com/lavanet/lava/v4/utils/common/types"
	"github.com/lavanet/lava/v4/x/dualstaking/ante"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"
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
		sdk.AccAddress("del1").String(),
		sdk.ValAddress("val1").String(),
		sdk.ValAddress("val2").String(),
		sdk.NewCoin(commontypes.TokenDenom, math.OneInt()),
	)
}

func newStakingDelegateMsg() *stakingtypes.MsgDelegate {
	return stakingtypes.NewMsgDelegate(
		sdk.AccAddress("del1").String(),
		sdk.ValAddress("val1").String(),
		sdk.NewCoin(commontypes.TokenDenom, math.OneInt()),
	)
}

func newStakingUndelegateMsg() *stakingtypes.MsgUndelegate {
	return stakingtypes.NewMsgUndelegate(
		sdk.AccAddress("del1").String(),
		sdk.ValAddress("val1").String(),
		sdk.NewCoin(commontypes.TokenDenom, math.OneInt()),
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

func (tmock TxMock) GetMsgsV2() ([]protov2.Message, error) {
	msgs := make([]protov2.Message, len(tmock.msgs))
	for i, msg := range tmock.msgs {
		protoMsg, ok := msg.(protov2.Message)
		if !ok {
			return nil, fmt.Errorf("message %T does not implement protov2.Message", msg)
		}
		msgs[i] = protoMsg
	}
	return msgs, nil
}
