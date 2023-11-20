package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
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
	_, err := ts.TxCreateValidator(val, math.NewIntFromUint64(uint64(testStake)))
	require.Nil(ts.T, err)

	ts.addClients(delegatorCount)

	ts.addProviders(stakedCount)
	for i := 0; i < stakedCount; i++ {
		_, addr := ts.GetAccount(common.PROVIDER, i)
		err := ts.StakeProvider(addr, ts.spec, testStake)
		require.NoError(ts.T, err)
	}

	ts.addProviders(unstakedCount)
	ts.addProviders(unstakingCount)

	for i := 0; i < unstakingCount; i++ {
		_, addr := ts.GetAccount(common.PROVIDER, stakedCount+unstakedCount+i)
		err := ts.StakeProvider(addr, ts.spec, testStake)
		require.NoError(ts.T, err)
		ts.AdvanceEpoch()
		_, err = ts.TxPairingUnstakeProvider(addr, ts.spec.Name)
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
func (ts *tester) getStakeEntry(provider sdk.AccAddress, chainID string) epochstoragetypes.StakeEntry {
	epoch := ts.EpochStart()
	keeper := ts.Keepers.Epochstorage

	stakeEntry, err := keeper.GetStakeEntryForProviderEpoch(ts.Ctx, chainID, provider, epoch)
	if err != nil {
		panic("getStakeEntry: no stake entry: " +
			err.Error() + ": " + provider.String() + " " + chainID)
	}

	return *stakeEntry
}
