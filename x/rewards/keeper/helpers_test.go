package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/stretchr/testify/require"
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

// makeBondedRatioNonZero makes BondedRatio() to be 0.25
// assumptions:
//  1. validators was created using addValidators(1) and TxCreateValidator
//  2. TxCreateValidator was used with init funds of 30000000000000/3
func (ts *tester) makeBondedRatioNonZero() {
	bondedRatio := ts.Keepers.StakingKeeper.BondedRatio(ts.Ctx)
	if bondedRatio.Equal(sdk.NewDecWithPrec(25, 2)) {
		return
	}

	// transfer the bonded pool tokens to staking module's bonded pool tokens (which is used to calculate BondedRatio)
	// in our testing env, the bonded pool account's address is sdk.AccAddress("bonded_tokens_pool")
	// in staking module's actual bonded pool, the AccAddress is different, so we manually transfer funds there
	stakingBondedPool := ts.Keepers.StakingKeeper.GetBondedPool(ts.Ctx)
	bondedPoolBalance := ts.Keepers.BankKeeper.GetBalance(ts.Ctx, testkeeper.GetModuleAddress(stakingtypes.BondedPoolName), ts.TokenDenom())
	require.False(ts.T, bondedPoolBalance.IsZero())
	err := ts.Keepers.BankKeeper.SendCoinsFromModuleToAccount(ts.Ctx, stakingtypes.BondedPoolName, stakingBondedPool.GetAddress(), sdk.NewCoins(bondedPoolBalance))
	require.Nil(ts.T, err)
	stakingBondedPoolBalance := ts.Keepers.BankKeeper.GetBalance(ts.Ctx, stakingBondedPool.GetAddress(), ts.TokenDenom())
	require.False(ts.T, stakingBondedPoolBalance.IsZero())

	bondedRatio = ts.Keepers.StakingKeeper.BondedRatio(ts.Ctx)
	require.True(ts.T, bondedRatio.Equal(sdk.NewDecWithPrec(25, 2))) // according to "valInitBalance", bondedRatio should be 0.25
}
