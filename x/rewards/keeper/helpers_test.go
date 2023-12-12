package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/lavanet/lava/testutil/common"
)

const (
	testStake             int64 = 100000
	testBalance           int64 = 100000000
	allocationPoolBalance int64 = 30000000000000
	feeCollectorName            = authtypes.FeeCollectorName
)

type tester struct {
	common.Tester
}

func newTester(t *testing.T) *tester {
	ts := &tester{Tester: *common.NewTesterRaw(t)}
	return ts
}

func (ts *tester) addValidators(count int) {
	start := len(ts.Accounts(common.VALIDATOR))
	for i := 0; i < count; i++ {
		_, _ = ts.AddAccount(common.VALIDATOR, start+i, testBalance)
	}
}

func (ts *tester) feeCollector() sdk.AccAddress {
	return sdk.AccAddress([]byte(feeCollectorName))
}
