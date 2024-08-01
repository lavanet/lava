package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/lavanet/lava/v2/testutil/common"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"github.com/stretchr/testify/require"
)

const (
	testStake   int64 = 100000
	testBalance int64 = 100000000
)

type tester struct {
	common.Tester
	plan planstypes.Plan
	spec spectypes.Spec
}

func newTester(t *testing.T) *tester {
	ts := &tester{Tester: *common.NewTester(t)}

	ts.plan = ts.AddPlan("mock", common.CreateMockPlan()).Plan("mock")
	ts.spec = ts.AddSpec("mock", common.CreateMockSpec()).Spec("mock")

	ts.AdvanceEpoch()

	return ts
}

func (ts *tester) setupForDelegation(delegatorCount, stakedCount, unstakedCount, unstakingCount int) *tester {
	ts.addValidators(1)
	val, _ := ts.GetAccount(common.VALIDATOR, 0)
	ts.TxCreateValidator(val, math.NewIntFromUint64(uint64(testStake)))

	ts.addClients(delegatorCount)

	err := ts.addProviders(stakedCount)
	require.NoError(ts.T, err)
	for i := 0; i < stakedCount; i++ {
		acc, provider := ts.GetAccount(common.PROVIDER, i)
		err := ts.StakeProvider(acc.GetVaultAddr(), provider, ts.spec, testStake)
		require.NoError(ts.T, err)
	}

	err = ts.addProviders(unstakedCount)
	require.NoError(ts.T, err)
	err = ts.addProviders(unstakingCount)
	require.NoError(ts.T, err)

	for i := 0; i < unstakingCount; i++ {
		acc, provider := ts.GetAccount(common.PROVIDER, stakedCount+unstakedCount+i)
		err := ts.StakeProvider(acc.GetVaultAddr(), provider, ts.spec, testStake)
		require.NoError(ts.T, err)
		ts.AdvanceEpoch()
		_, err = ts.TxPairingUnstakeProvider(acc.GetVaultAddr(), ts.spec.Name)
		require.NoError(ts.T, err)
	}

	ts.AdvanceEpoch()

	return ts
}

func (ts *tester) addClients(count int) {
	start := len(ts.Accounts(common.CONSUMER))
	for i := 0; i < count; i++ {
		_, _ = ts.AddAccount(common.CONSUMER, start+i, testBalance)
	}
}

// addProvider: with default endpoints, geolocation, moniker
func (ts *tester) addProviders(count int) error {
	start := len(ts.Accounts(common.PROVIDER))
	for i := 0; i < count; i++ {
		_, _ = ts.AddAccount(common.PROVIDER, start+i, testBalance)
	}
	return nil
}

func (ts *tester) addValidators(count int) {
	start := len(ts.Accounts(common.VALIDATOR))
	for i := 0; i < count; i++ {
		_, _ = ts.AddAccount(common.VALIDATOR, start+i, testBalance)
	}
}

// getStakeEntry find the stake entry of a given provider + chainID
func (ts *tester) getStakeEntry(provider string, chainID string) epochstoragetypes.StakeEntry {
	epoch := ts.EpochStart()
	keeper := ts.Keepers.Epochstorage

	stakeEntry, found := keeper.GetStakeEntryForProviderEpoch(ts.Ctx, chainID, provider, epoch)
	if !found {
		panic("getStakeEntry: no stake entry: " + provider + " " + chainID)
	}

	return stakeEntry
}

func (ts *tester) verifyDelegatorsBalance() {
	accounts := ts.AccountsMap()
	for key, account := range accounts {
		diff, _, err := ts.Keepers.Dualstaking.VerifyDelegatorBalance(ts.Ctx, account.Addr)
		require.Nil(ts.T, err)
		require.True(ts.T, diff.IsZero(), "delegator balance is not 0", key)
	}
}
