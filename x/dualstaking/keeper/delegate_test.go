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
			name:      "insufficient funds",
			delegator: client1Addr,
			provider:  provider1Addr,
			chainID:   "mockspec",
			amount:    testBalance + 1,
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
	bonded := zeroCoin.AddAmount(sdk.NewIntFromUint64(uint64(testStake))) // provider self delegation

	// delegate once
	amount := sdk.NewCoin("ulava", sdk.NewInt(10000))
	_, err := ts.TxDualstakingDelegate(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)
	// not yet in effect
	bonded = bonded.Add(amount)
	stakeEntry := ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))
	require.True(t, bonded.Amount.Equal(ts.Keepers.Dualstaking.TotalBondedTokens(ts.Ctx)))
	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated = delegated.Add(amount)
	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))
	require.True(t, bonded.Amount.Equal(ts.Keepers.Dualstaking.TotalBondedTokens(ts.Ctx)))

	// delegate twice same block (fail)
	_, err = ts.TxDualstakingDelegate(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)
	bonded = bonded.Add(amount)
	_, err = ts.TxDualstakingDelegate(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)
	bonded = bonded.Add(amount)
	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))
	require.True(t, bonded.Amount.Equal(ts.Keepers.Dualstaking.TotalBondedTokens(ts.Ctx)))
	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated = delegated.Add(amount).Add(amount) // two delegations
	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))
	require.True(t, bonded.Amount.Equal(ts.Keepers.Dualstaking.TotalBondedTokens(ts.Ctx)))
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
	bonded := zeroCoin.AddAmount(sdk.NewIntFromUint64(uint64(2 * testStake))) // provider self delegation

	// delegate once
	amount := sdk.NewCoin("ulava", sdk.NewInt(10000))
	_, err := ts.TxDualstakingDelegate(
		client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)
	bonded = bonded.Add(amount)
	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated1 = delegated1.Add(amount)
	stakeEntry1 := ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	stakeEntry2 := ts.getStakeEntry(provider2Acct.Addr, ts.spec.Name)
	require.True(t, delegated1.IsEqual(stakeEntry1.DelegateTotal))
	require.True(t, bonded.Amount.Equal(ts.Keepers.Dualstaking.TotalBondedTokens(ts.Ctx)))
	require.True(t, delegated2.IsEqual(stakeEntry2.DelegateTotal))
	require.True(t, bonded.Amount.Equal(ts.Keepers.Dualstaking.TotalBondedTokens(ts.Ctx)))

	// redelegate once
	amount = sdk.NewCoin("ulava", sdk.NewInt(5000))
	_, err = ts.TxDualstakingRedelegate(
		client1Addr, provider1Addr, provider2Addr, ts.spec.Name, ts.spec.Name, amount)
	require.NoError(t, err)
	stakeEntry1 = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	stakeEntry2 = ts.getStakeEntry(provider2Acct.Addr, ts.spec.Name)
	require.True(t, delegated1.IsEqual(stakeEntry1.DelegateTotal))
	require.True(t, delegated2.IsEqual(stakeEntry2.DelegateTotal))
	require.True(t, bonded.Amount.Equal(ts.Keepers.Dualstaking.TotalBondedTokens(ts.Ctx)))
	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated1 = delegated1.Sub(amount)
	delegated2 = delegated2.Add(amount)
	stakeEntry1 = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	stakeEntry2 = ts.getStakeEntry(provider2Acct.Addr, ts.spec.Name)
	require.True(t, delegated1.IsEqual(stakeEntry1.DelegateTotal))
	require.True(t, delegated2.IsEqual(stakeEntry2.DelegateTotal))
	require.True(t, bonded.Amount.Equal(ts.Keepers.Dualstaking.TotalBondedTokens(ts.Ctx)))

	_, err = ts.TxPairingUnstakeProvider(provider1Addr, ts.spec.Name)
	bonded = bonded.SubAmount(sdk.NewIntFromUint64(uint64(testStake)))
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
	require.True(t, bonded.Amount.Equal(ts.Keepers.Dualstaking.TotalBondedTokens(ts.Ctx)))
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
	bonded := zeroCoin.AddAmount(sdk.NewIntFromUint64(uint64(2 * testStake))) // provider self delegation
	notBonded := zeroCoin

	// delegate once
	amount := sdk.NewCoin("ulava", sdk.NewInt(10000))
	_, err := ts.TxDualstakingDelegate(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)
	bonded = bonded.Add(amount)
	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated = delegated.Add(amount)
	stakeEntry := ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))
	require.True(t, bonded.Amount.Equal(ts.Keepers.Dualstaking.TotalBondedTokens(ts.Ctx)))
	require.True(t, notBonded.Amount.Equal(ts.Keepers.Dualstaking.TotalNotBondedTokens(ts.Ctx)))

	// unbond once
	amount = sdk.NewCoin("ulava", sdk.NewInt(1000))
	_, err = ts.TxDualstakingUnbond(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)
	bonded = bonded.Sub(amount)
	notBonded = notBonded.Add(amount)
	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))
	require.True(t, bonded.Amount.Equal(ts.Keepers.Dualstaking.TotalBondedTokens(ts.Ctx)))
	require.True(t, notBonded.Amount.Equal(ts.Keepers.Dualstaking.TotalNotBondedTokens(ts.Ctx)))
	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated = delegated.Sub(amount)
	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))
	require.True(t, bonded.Amount.Equal(ts.Keepers.Dualstaking.TotalBondedTokens(ts.Ctx)))
	require.True(t, notBonded.Amount.Equal(ts.Keepers.Dualstaking.TotalNotBondedTokens(ts.Ctx)))

	// unbond twice in same block, and then in next block
	_, err = ts.TxDualstakingUnbond(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)
	// _, err = ts.TxDualstakingUnbond(client1Addr, provider1Addr, ts.spec.Name, amount)
	// require.Error(t, err)
	ts.AdvanceBlock()
	_, err = ts.TxDualstakingUnbond(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)
	bonded = bonded.Sub(amount).Sub(amount)
	notBonded = notBonded.Add(amount).Add(amount)
	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))
	require.True(t, bonded.Amount.Equal(ts.Keepers.Dualstaking.TotalBondedTokens(ts.Ctx)))
	require.True(t, notBonded.Amount.Equal(ts.Keepers.Dualstaking.TotalNotBondedTokens(ts.Ctx)))
	// advance epoch to digest the delegate
	ts.AdvanceEpoch()
	// now in effect
	delegated = delegated.Sub(amount).Sub(amount)
	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))
	require.True(t, bonded.Amount.Equal(ts.Keepers.Dualstaking.TotalBondedTokens(ts.Ctx)))
	require.True(t, notBonded.Amount.Equal(ts.Keepers.Dualstaking.TotalNotBondedTokens(ts.Ctx)))

	_, err = ts.TxPairingUnstakeProvider(provider1Addr, ts.spec.Name)
	bonded = bonded.SubAmount(sdk.NewIntFromUint64(uint64(testStake)))
	notBonded = notBonded.AddAmount(sdk.NewIntFromUint64(uint64(testStake))) // provider self delegation
	require.NoError(t, err)

	// unbond from unstaking provider
	_, err = ts.TxDualstakingUnbond(client1Addr, provider1Addr, ts.spec.Name, amount)
	require.NoError(t, err)
	bonded = bonded.Sub(amount)
	notBonded = notBonded.Add(amount)
	stakeEntry = ts.getStakeEntry(provider1Acct.Addr, ts.spec.Name)
	require.True(t, delegated.IsEqual(stakeEntry.DelegateTotal))
	require.True(t, bonded.Amount.Equal(ts.Keepers.Dualstaking.TotalBondedTokens(ts.Ctx)))
	require.True(t, notBonded.Amount.Equal(ts.Keepers.Dualstaking.TotalNotBondedTokens(ts.Ctx)))
	// advance half the blocks for the unbond hold-period (for sure > epoch)
	ts.AdvanceBlocks(105)
	// now in effect (stake-entry gone, so not tracking `delegated` anymore
	require.True(t, bonded.Amount.Equal(ts.Keepers.Dualstaking.TotalBondedTokens(ts.Ctx)))
	require.True(t, notBonded.Amount.Equal(ts.Keepers.Dualstaking.TotalNotBondedTokens(ts.Ctx)))

	// check that unbonding releases the money to the delegator after unhold period:
	// advance another half the blocks for the unbond hold-period
	ts.AdvanceBlocks(105)
	// not-bonded coins will have been returned to their delegator by now
	require.True(t, ts.Keepers.Dualstaking.TotalNotBondedTokens(ts.Ctx).IsZero())
}
