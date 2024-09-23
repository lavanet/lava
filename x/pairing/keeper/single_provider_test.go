package keeper_test

import (
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v3/testutil/common"
	"github.com/stretchr/testify/require"
)

func SpecName(num int) string {
	return "spec" + strconv.Itoa(num)
}

func SetupForSingleProviderTests(ts *tester, providers, specs, clientsCount int) {
	for i := 0; i < providers; i++ {
		ts.AddAccount(common.PROVIDER, i, testBalance)
	}

	for i := 0; i < specs; i++ {
		spec := ts.spec
		spec.Index = SpecName(i)
		ts.AddSpec(spec.Index, spec)
		for i := 0; i < providers; i++ {
			acc, addr := ts.GetAccount(common.PROVIDER, i)
			d := common.MockDescription()
			err := ts.StakeProviderExtra(acc.GetVaultAddr(), addr, spec, testStake, nil, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details)
			require.NoError(ts.T, err)
		}
	}

	ts.addClient(clientsCount)
	ts.AdvanceEpoch()
}

// * unstake to see the delegations distributions
// * unstake from the rest of the chains
// * stake to get all delegations back
func TestUnstakeStake(t *testing.T) {
	ts := newTester(t)
	SetupForSingleProviderTests(ts, 1, 5, 0)

	delegator, _ := ts.AddAccount("del", 1, 1000000000)
	provider0, _ := ts.GetAccount(common.PROVIDER, 0)

	// delegate and check delegatetotal
	_, err := ts.TxDualstakingDelegate(delegator.Addr.String(), provider0.Addr.String(), types.NewInt64Coin(ts.TokenDenom(), 5000))
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		res, err := ts.QueryPairingProvider(provider0.Addr.String(), SpecName(i))
		require.NoError(t, err)
		require.Equal(t, int64(1000), res.StakeEntries[0].DelegateTotal.Amount.Int64())
	}

	// unstake spec0 provider
	_, err = ts.TxPairingUnstakeProvider(provider0.GetVaultAddr(), SpecName(0))
	require.NoError(t, err)

	// check redistribution of delegation
	for i := 1; i < 5; i++ {
		res, err := ts.QueryPairingProvider(provider0.Addr.String(), SpecName(i))
		require.NoError(t, err)
		require.Equal(t, int64(1250), res.StakeEntries[0].DelegateTotal.Amount.Int64())
	}

	// unstake all
	for i := 1; i < 5; i++ {
		_, err = ts.TxPairingUnstakeProvider(provider0.GetVaultAddr(), SpecName(i))
		require.NoError(t, err)
	}

	res, err := ts.QueryDualstakingDelegatorProviders(delegator.Addr.String())
	require.NoError(t, err)
	require.Len(t, res.Delegations, 1)
	require.Equal(t, provider0.Addr.String(), res.Delegations[0].Provider)
	require.Equal(t, int64(5000), res.Delegations[0].Amount.Amount.Int64())

	// stake again and check we got the delegation back
	d := common.MockDescription()
	err = ts.StakeProviderExtra(provider0.GetVaultAddr(), provider0.Addr.String(), ts.Spec(SpecName(0)), testStake, nil, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details)
	require.NoError(ts.T, err)

	res1, err := ts.QueryPairingProvider(provider0.Addr.String(), SpecName(0))
	require.NoError(t, err)
	require.Equal(t, int64(5000), res1.StakeEntries[0].DelegateTotal.Amount.Int64())

	// stake again on spec1 and check delegations
	err = ts.StakeProviderExtra(provider0.GetVaultAddr(), provider0.Addr.String(), ts.Spec(SpecName(1)), testStake, nil, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details)
	require.NoError(ts.T, err)

	res1, err = ts.QueryPairingProvider(provider0.Addr.String(), SpecName(0))
	require.NoError(t, err)
	require.Equal(t, int64(2500), res1.StakeEntries[0].DelegateTotal.Amount.Int64())

	res1, err = ts.QueryPairingProvider(provider0.Addr.String(), SpecName(1))
	require.NoError(t, err)
	require.Equal(t, int64(2500), res1.StakeEntries[0].DelegateTotal.Amount.Int64())
}

func TestDelegations(t *testing.T) {
	ts := newTester(t)
	SetupForSingleProviderTests(ts, 1, 5, 0)

	delegator, _ := ts.AddAccount("del", 1, 1000000000)
	provider, _ := ts.GetAccount(common.PROVIDER, 0)

	// delegate amount that does not divide by 5
	_, err := ts.TxDualstakingDelegate(delegator.Addr.String(), provider.Addr.String(), types.NewInt64Coin(ts.TokenDenom(), 4999))
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		res, err := ts.QueryPairingProvider(provider.Addr.String(), SpecName(i))
		require.NoError(t, err)
		require.Equal(t, int64(999), res.StakeEntries[0].DelegateTotal.Amount.Int64())
	}

	// unbond 4 tokens, now amount divides by 5
	_, err = ts.TxDualstakingUnbond(delegator.Addr.String(), provider.Addr.String(), types.NewInt64Coin(ts.TokenDenom(), 4))
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		res, err := ts.QueryPairingProvider(provider.Addr.String(), SpecName(i))
		require.NoError(t, err)
		require.Equal(t, int64(999), res.StakeEntries[0].DelegateTotal.Amount.Int64())
	}

	// add stake to spec0
	d := common.MockDescription()
	err = ts.StakeProviderExtra(provider.GetVaultAddr(), provider.Addr.String(), ts.Spec(SpecName(0)), 2*testStake, nil, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details)
	require.NoError(ts.T, err)

	// check delegate total is twice than others in spec0
	res, err := ts.QueryPairingProvider(provider.Addr.String(), SpecName(0))
	require.NoError(t, err)
	require.Equal(t, int64(1665), res.StakeEntries[0].DelegateTotal.Amount.Int64())

	for i := 1; i < 5; i++ {
		res, err := ts.QueryPairingProvider(provider.Addr.String(), SpecName(i))
		require.NoError(t, err)
		require.Equal(t, int64(832), res.StakeEntries[0].DelegateTotal.Amount.Int64())
	}

	// unbond with vault, should be uniformali
	_, err = ts.TxDualstakingUnbond(provider.GetVaultAddr(), provider.Addr.String(), types.NewInt64Coin(ts.TokenDenom(), testStake))
	require.NoError(t, err)

	res, err = ts.QueryPairingProvider(provider.Addr.String(), SpecName(0))
	require.NoError(t, err)
	require.Equal(t, testStake+testStake*4/5, res.StakeEntries[0].Stake.Amount.Int64())
	require.Equal(t, 4995*res.StakeEntries[0].Stake.Amount.Int64()/(5*testStake), res.StakeEntries[0].DelegateTotal.Amount.Int64())

	for i := 1; i < 5; i++ {
		res, err := ts.QueryPairingProvider(provider.Addr.String(), SpecName(i))
		require.NoError(t, err)
		require.Equal(t, testStake*4/5, res.StakeEntries[0].Stake.Amount.Int64())
		require.Equal(t, 4995*res.StakeEntries[0].Stake.Amount.Int64()/(5*testStake), res.StakeEntries[0].DelegateTotal.Amount.Int64())
	}

	// unbond all delegator
	_, err = ts.TxDualstakingUnbond(delegator.Addr.String(), provider.Addr.String(), types.NewInt64Coin(ts.TokenDenom(), 4995))
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		res, err := ts.QueryPairingProvider(provider.Addr.String(), SpecName(i))
		require.NoError(t, err)
		require.Equal(t, int64(0), res.StakeEntries[0].DelegateTotal.Amount.Int64())
	}
}

func TestUnstakeWithOperator(t *testing.T) {
	ts := newTester(t)
	SetupForSingleProviderTests(ts, 1, 5, 0)

	// delegator, _ := ts.AddAccount("del", 1, 1000000000)
	provider, _ := ts.GetAccount(common.PROVIDER, 0)

	// unstake one chain
	_, err := ts.TxPairingUnstakeProvider(provider.Addr.String(), SpecName(0))
	require.NoError(t, err)

	// check the entries got the stake of spec0
	for i := 1; i < 5; i++ {
		res, err := ts.QueryPairingProvider(provider.Addr.String(), SpecName(i))
		require.NoError(t, err)
		require.Equal(t, testStake*5/4, res.StakeEntries[0].Stake.Amount.Int64())
	}

	// unstake everything
	for i := 1; i < 5; i++ {
		_, err := ts.TxPairingUnstakeProvider(provider.Addr.String(), SpecName(i))
		require.NoError(t, err)
	}

	res, err := ts.QueryPairingProvider(provider.Addr.String(), "")
	require.NoError(t, err)
	require.Len(t, res.StakeEntries, 0)

	// stake again
	d := common.MockDescription()
	err = ts.StakeProviderExtra(provider.GetVaultAddr(), provider.Addr.String(), ts.Spec(SpecName(0)), 1, nil, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details)
	require.NoError(ts.T, err)

	// check we got all the self delegations
	res, err = ts.QueryPairingProvider(provider.Addr.String(), SpecName(0))
	require.NoError(t, err)
	require.Equal(t, testStake*5+1, res.StakeEntries[0].Stake.Amount.Int64())
}
