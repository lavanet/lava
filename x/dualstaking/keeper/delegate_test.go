package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	"github.com/stretchr/testify/require"
)

var zeroCoin = sdk.NewCoin("ulava", sdk.ZeroInt())

func TestDelegateFail(t *testing.T) {
	ts := newTester(t)

	// 1 delegator, 1 provider staked, 1 provider unstaked, 1 provider unstaking
	ts.setupForDelegation(1, 1, 1, 1)

	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	_, provider1Addr := ts.GetAccount(common.PROVIDER, 0)
	_, provider2Addr := ts.GetAccount(common.PROVIDER, 1)
	_, provider3Addr := ts.GetAccount(common.PROVIDER, 2)

	template := []struct {
		name      string
		delegator string
		provider  string
		chainID   string
		amount    int64
	}{
		{
			name:      "bad delegator",
			delegator: "invalid",
			provider:  provider1Addr,
			chainID:   "mockspec",
			amount:    1,
		},
		{
			name:      "bad provider",
			delegator: client1Addr,
			provider:  "invalid",
			chainID:   "mockspec",
			amount:    1,
		},
		{
			name:      "bad chainID",
			delegator: client1Addr,
			provider:  provider1Addr,
			chainID:   "invalid",
			amount:    1,
		},
		{
			name:      "bad amount",
			delegator: client1Addr,
			provider:  provider1Addr,
			chainID:   "mockspec",
			amount:    -1,
		},
		{
			name:      "provider not staked",
			delegator: client1Addr,
			provider:  provider2Addr,
			chainID:   "mockspec",
			amount:    10000,
		},
		{
			name:      "provider unstaking",
			delegator: client1Addr,
			provider:  provider3Addr,
			chainID:   "mockspec",
			amount:    10000,
		},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			amount := sdk.NewCoin("ulava", sdk.ZeroInt())
			amount.Amount = amount.Amount.Add(sdk.NewInt(tt.amount))
			_, err := ts.TxDualstakingDelegate(tt.delegator, tt.provider, tt.chainID, amount)
			require.Error(t, err, tt.name)
		})
	}
}

func TestDelegate(t *testing.T) {
	ts := newTester(t)

	// 1 delegator, 1 provider staked, 0 provider unstaked, 0 provider unstaking
	ts.setupForDelegation(1, 1, 0, 0)

	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	provider1Acct, provider1Addr := ts.GetAccount(common.PROVIDER, 0)

	delegated := zeroCoin

	// delegate once
	amount := sdk.NewCoin("ulava", sdk.NewInt(10000))
	_, err := ts.TxDualstakingDelegate(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)
	// not yet in effect

	stakeEntry := ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))
	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated = delegated.Add(amount)
	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))

	// delegate twice same block (fail)
	_, err = ts.TxDualstakingDelegate(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)

	_, err = ts.TxDualstakingDelegate(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)

	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))
	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated = delegated.Add(amount).Add(amount) // two delegations
	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))

	ts.verifyDelegatorsBalance()
}

func TestRedelegateFail(t *testing.T) {
	ts := newTester(t)

	// 1 delegator, 2 provider staked, 1 provider unstaked, 1 provider unstaking
	ts.setupForDelegation(1, 2, 1, 1)

	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	_, provider1Addr := ts.GetAccount(common.PROVIDER, 0)
	_, provider2Addr := ts.GetAccount(common.PROVIDER, 1)
	_, provider3Addr := ts.GetAccount(common.PROVIDER, 2)
	_, provider4Addr := ts.GetAccount(common.PROVIDER, 3)

	// delegate once for setup
	amount := sdk.NewCoin("ulava", sdk.NewInt(10000))
	_, err := ts.TxDualstakingDelegate(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)

	template := []struct {
		name      string
		delegator string
		provider1 string
		provider2 string
		chainID   string
		amount    int64
	}{
		{
			name:      "bad delegator",
			delegator: "invalid",
			provider1: provider1Addr,
			provider2: provider2Addr,
			chainID:   "mockspec",
			amount:    1,
		},
		{
			name:      "bad provider1",
			delegator: client1Addr,
			provider1: "invalid",
			provider2: provider2Addr,
			chainID:   "mockspec",
			amount:    1,
		},
		{
			name:      "bad provider2",
			delegator: client1Addr,
			provider1: provider1Addr,
			provider2: "invalid",
			chainID:   "mockspec",
			amount:    1,
		},
		{
			name:      "bad chainID",
			delegator: client1Addr,
			provider1: provider1Addr,
			provider2: provider2Addr,
			chainID:   "invalid",
			amount:    1,
		},
		{
			name:      "bad amount",
			delegator: client1Addr,
			provider1: provider1Addr,
			provider2: provider2Addr,
			chainID:   "mockspec",
			amount:    -1,
		},
		{
			name:      "insufficient funds",
			delegator: client1Addr,
			provider1: provider1Addr,
			provider2: provider2Addr,
			chainID:   "mockspec",
			amount:    10000 + 1,
		},
		{
			name:      "provider2 not staked",
			delegator: client1Addr,
			provider1: provider1Addr,
			provider2: provider3Addr,
			chainID:   "mockspec",
			amount:    10000,
		},
		{
			name:      "provider2 unstaking",
			delegator: client1Addr,
			provider1: provider1Addr,
			provider2: provider4Addr,
			chainID:   "mockspec",
			amount:    10000,
		},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			amount := sdk.NewCoin("ulava", sdk.ZeroInt())
			amount.Amount = amount.Amount.Add(sdk.NewInt(tt.amount))
			_, err := ts.TxDualstakingRedelegate(
				tt.delegator, tt.provider1, tt.provider2, tt.chainID, tt.chainID, amount)
			require.Error(t, err, tt.name)
		})
	}
}

func TestRedelegate(t *testing.T) {
	ts := newTester(t)

	// 1 delegator, 2 provider staked, 0 provider unstaked, 0 provider unstaking
	ts.setupForDelegation(1, 2, 0, 0)

	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	provider1Acct, provider1Addr := ts.GetAccount(common.PROVIDER, 0)
	provider2Acct, provider2Addr := ts.GetAccount(common.PROVIDER, 1)

	delegated1 := zeroCoin
	delegated2 := zeroCoin

	// delegate once
	amount := sdk.NewCoin("ulava", sdk.NewInt(10000))
	_, err := ts.TxDualstakingDelegate(
		client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)

	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated1 = delegated1.Add(amount)
	stakeEntry1 := ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	stakeEntry2 := ts.getStakeEntry(provider2Acct.Addr, ts.spec.Name)
	require.True(t, delegated1.IsEqual(stakeEntry1.DelegateTotal))
	require.True(t, delegated2.IsEqual(stakeEntry2.DelegateTotal))

	// redelegate once
	amount = sdk.NewCoin("ulava", sdk.NewInt(5000))
	_, err = ts.TxDualstakingRedelegate(
		client1Addr, provider1Addr, provider2Addr, ts.spec.Name, ts.spec.Name, amount)
	require.NoError(t, err)
	stakeEntry1 = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	stakeEntry2 = ts.getStakeEntry(provider2Acct.Addr, ts.spec.Name)
	require.True(t, delegated1.IsEqual(stakeEntry1.DelegateTotal))
	require.True(t, delegated2.IsEqual(stakeEntry2.DelegateTotal))
	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated1 = delegated1.Sub(amount)
	delegated2 = delegated2.Add(amount)
	stakeEntry1 = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	stakeEntry2 = ts.getStakeEntry(provider2Acct.Addr, ts.spec.Name)
	require.True(t, delegated1.IsEqual(stakeEntry1.DelegateTotal))
	require.True(t, delegated2.IsEqual(stakeEntry2.DelegateTotal))

	_, err = ts.TxPairingUnstakeProvider(provider1Addr, ts.spec.Name)
	require.NoError(t, err)

	// redelegate from unstaking provider
	amount = sdk.NewCoin("ulava", sdk.NewInt(5000))
	_, err = ts.TxDualstakingRedelegate(
		client1Addr, provider1Addr, provider2Addr, ts.spec.Name, ts.spec.Name, amount)
	require.NoError(t, err)
	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated2 = delegated2.Add(amount)
	stakeEntry2 = ts.getStakeEntry(provider2Acct.Addr, ts.spec.Name)
	require.True(t, delegated2.IsEqual(stakeEntry2.DelegateTotal))

	ts.verifyDelegatorsBalance()
}

func TestUnbondFail(t *testing.T) {
	ts := newTester(t)

	// 1 delegator, 1 provider staked, 1 provider unstaked, 0 provider unstaking
	ts.setupForDelegation(1, 1, 1, 0)

	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	_, provider1Addr := ts.GetAccount(common.PROVIDER, 0)
	_, provider2Addr := ts.GetAccount(common.PROVIDER, 1)

	// delegate once for setup
	amount := sdk.NewCoin("ulava", sdk.NewInt(10000))
	_, err := ts.TxDualstakingDelegate(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)

	template := []struct {
		name      string
		delegator string
		provider  string
		chainID   string
		amount    int64
	}{
		{
			name:      "bad delegator",
			delegator: "invalid",
			provider:  provider1Addr,
			chainID:   "mockspec",
			amount:    1,
		},
		{
			name:      "bad provider",
			delegator: client1Addr,
			provider:  "invalid",
			chainID:   "mockspec",
			amount:    1,
		},
		{
			name:      "bad chainID",
			delegator: client1Addr,
			provider:  provider1Addr,
			chainID:   "invalid",
			amount:    1,
		},
		{
			name:      "bad amount",
			delegator: client1Addr,
			provider:  provider1Addr,
			chainID:   "mockspec",
			amount:    -1,
		},
		{
			name:      "insufficient funds",
			delegator: client1Addr,
			provider:  provider1Addr,
			chainID:   "mockspec",
			amount:    10000 + 1,
		},
		{
			name:      "provider not staked",
			delegator: client1Addr,
			provider:  provider2Addr,
			chainID:   "mockspec",
			amount:    10000,
		},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			amount := sdk.NewCoin("ulava", sdk.ZeroInt())
			amount.Amount = amount.Amount.Add(sdk.NewInt(tt.amount))
			_, err := ts.TxDualstakingUnbond(tt.delegator, tt.provider, tt.chainID, amount)
			require.Error(t, err, tt.name)
		})
	}
}

func TestUnbond(t *testing.T) {
	ts := newTester(t)

	// 1 delegator, 2 provider staked, 0 provider unstaked, 0 provider unstaking
	ts.setupForDelegation(1, 2, 0, 0)

	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	provider1Acct, provider1Addr := ts.GetAccount(common.PROVIDER, 0)

	delegated := zeroCoin

	// delegate once
	amount := sdk.NewCoin("ulava", sdk.NewInt(10000))
	_, err := ts.TxDualstakingDelegate(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)

	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated = delegated.Add(amount)
	stakeEntry := ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))

	// unbond once
	amount = sdk.NewCoin("ulava", sdk.NewInt(1000))
	_, err = ts.TxDualstakingUnbond(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)

	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))

	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated = delegated.Sub(amount)
	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))

	// unbond twice in same block, and then in next block
	_, err = ts.TxDualstakingUnbond(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)
	// _, err = ts.TxDualstakingUnbond(client1Addr, provider1Addr, ts.spec.Name, amount)
	// require.Error(t, err)
	ts.AdvanceBlock()
	_, err = ts.TxDualstakingUnbond(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)

	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))

	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated = delegated.Sub(amount).Sub(amount)
	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))

	_, err = ts.TxPairingUnstakeProvider(provider1Addr, ts.spec.Name)
	require.NoError(t, err)

	// unbond from unstaking provider
	_, err = ts.TxDualstakingUnbond(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)
	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))

	// advance half the blocks for the unbond hold-period (for sure > epoch)
	ts.AdvanceBlocks(105)
	// now in effect (stake-entry gone, so not tracking `delegated` anymore

	// check that unbonding releases the money to the delegator after unhold period:
	// advance another half the blocks for the unbond hold-period
	ts.AdvanceBlocks(105)
	// not-blablacoins will have been returned to their delegator by now

	ts.verifyDelegatorsBalance()
}

func TestBondUnbondBond(t *testing.T) {
	ts := newTester(t)

	// 1 delegator, 2 provider staked, 0 provider unstaked, 0 provider unstaking
	ts.setupForDelegation(1, 2, 0, 0)

	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	provider1Acct, provider1Addr := ts.GetAccount(common.PROVIDER, 0)

	delegated := zeroCoin

	// delegate once
	amount := sdk.NewCoin("ulava", sdk.NewInt(10000))
	_, err := ts.TxDualstakingDelegate(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)

	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated = delegated.Add(amount)
	stakeEntry := ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))

	// unbond once
	_, err = ts.TxDualstakingUnbond(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)
	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))

	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated = delegated.Sub(amount)
	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))

	// delegate second time
	_, err = ts.TxDualstakingDelegate(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)

	ts.AdvanceEpoch()

	ts.verifyDelegatorsBalance()
}

func TestDualstakingUnbondStakeIsLowerThanMinStakeCausesFreeze(t *testing.T) {
	ts := newTester(t)

	// 0 delegator, 1 provider staked, 0 provider unstaked, 0 provider unstaking
	ts.setupForDelegation(0, 1, 0, 0)

	provider1Acct, provider1Addr := ts.GetAccount(common.PROVIDER, 0)

	staked := sdk.NewCoin("ulava", sdk.NewInt(testStake))

	// unbond once
	_, err := ts.TxDualstakingUnbond(provider1Addr, provider1Addr, ts.spec.Name, staked)
	require.NoError(t, err)

	stakeEntry := ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, staked.IsEqual(stakeEntry.Stake))

	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	staked = staked.Sub(staked)
	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, staked.IsEqual(stakeEntry.Stake))
	require.True(t, stakeEntry.IsFrozen())
}

func TestDualstakingBondStakeIsGreaterThanMinStakeCausesUnFreeze(t *testing.T) {
	ts := newTester(t)

	// 0 delegator, 0 provider staked, 1 provider unstaked, 0 provider unstaking
	ts.setupForDelegation(0, 1, 0, 0)

	provider1Acct, provider1Addr := ts.GetAccount(common.PROVIDER, 0)

	staked := sdk.NewCoin("ulava", sdk.NewInt(testStake))

	// delegate once
	amount := sdk.NewCoin("ulava", sdk.NewInt(10000))
	_, err := ts.TxDualstakingDelegate(provider1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)

	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	staked = staked.Add(amount)
	stakeEntry := ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, staked.IsEqual(stakeEntry.Stake))
	require.False(t, stakeEntry.IsFrozen())
}

func TestDualstakingRedelegateFreezeOneUnFreezeOther(t *testing.T) {
	ts := newTester(t)

	// 0 delegator, 2 provider staked, 0 provider unstaked, 0 provider unstaking
	ts.setupForDelegation(0, 2, 0, 0)

	provider1Acct, provider1Addr := ts.GetAccount(common.PROVIDER, 0)
	provider2Acct, provider2Addr := ts.GetAccount(common.PROVIDER, 1)

	stake := sdk.NewCoin("ulava", sdk.NewInt(testStake))

	// redelegate once
	_, err := ts.TxDualstakingRedelegate(provider1Addr, provider1Addr, provider2Addr, ts.spec.Name, ts.spec.Name, stake)
	require.NoError(t, err)

	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect

	stakeEntry := ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, stakeEntry.Stake.IsZero())
	require.True(t, stakeEntry.IsFrozen())

	stakeEntry = ts.getStakeEntry(provider2Acct.Addr, ts.spec.Name)
	require.True(t, stake.IsEqual(stakeEntry.Stake))
	require.True(t, stake.IsEqual(stakeEntry.DelegateTotal))
	require.False(t, stakeEntry.IsFrozen())

	// redelegate again
	_, err = ts.TxDualstakingRedelegate(provider2Addr, provider2Addr, provider1Addr, ts.spec.Name, ts.spec.Name, stake)
	require.NoError(t, err)

	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect

	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, stakeEntry.Stake.IsZero())
	require.True(t, stake.IsEqual(stakeEntry.DelegateTotal))
	require.True(t, stakeEntry.IsFrozen())

	stakeEntry = ts.getStakeEntry(provider2Acct.Addr, ts.spec.Name)
	require.True(t, stakeEntry.Stake.IsZero())
	require.True(t, stake.IsEqual(stakeEntry.DelegateTotal))
	require.True(t, stakeEntry.IsFrozen())
}

func TestStakingUnbondStakeIsLowerThanMinStakeCausesFreeze(t *testing.T) {
	ts := newTester(t)

	// 0 delegator, 1 provider staked, 0 provider unstaked, 0 provider unstaking
	ts.setupForDelegation(0, 1, 0, 0)

	provider1Acct, _ := ts.GetAccount(common.PROVIDER, 0)
	validator1Acct, _ := ts.GetAccount(common.VALIDATOR, 0)

	stakeInt := sdk.NewInt(testStake)
	stake := sdk.NewCoin("ulava", stakeInt)

	stakeEntry := ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, stake.IsEqual(stakeEntry.Stake))
	require.False(t, stakeEntry.IsFrozen())

	// unbond once
	_, err := ts.TxUnbondValidator(provider1Acct, validator1Acct, stakeInt)
	require.NoError(t, err)

	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect

	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, stakeEntry.Stake.IsZero())
	require.True(t, stakeEntry.IsFrozen())
}
