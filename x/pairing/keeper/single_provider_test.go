package keeper_test

import (
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v3/testutil/common"
	"github.com/lavanet/lava/v3/x/pairing/client/cli"
	pairingtypes "github.com/lavanet/lava/v3/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func SpecName(num int) string {
	return "spec" + strconv.Itoa(num)
}

func SetupForSingleProviderTests(ts *tester, providers, specs, clientsCount int) {
	for i := 0; i < providers; i++ {
		ts.AddAccount(common.PROVIDER, i, 100*testBalance)
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

	ts.AdvanceEpoch()

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

	ts.AdvanceEpoch()

	res1, err = ts.QueryPairingProvider(provider0.Addr.String(), SpecName(0))
	require.NoError(t, err)
	require.Equal(t, int64(2500), res1.StakeEntries[0].DelegateTotal.Amount.Int64())

	res1, err = ts.QueryPairingProvider(provider0.Addr.String(), SpecName(1))
	require.NoError(t, err)
	require.Equal(t, int64(2500), res1.StakeEntries[0].DelegateTotal.Amount.Int64())
}

// * unstake to see the delegations distributions
// * unstake from the rest of the chains
// * stake to get all delegations back with a new vault
func TestUnstakeStakeNewVault(t *testing.T) {
	ts := newTester(t)
	SetupForSingleProviderTests(ts, 1, 5, 0)

	delegator, _ := ts.AddAccount("del", 1, 1000000000)
	provider0, _ := ts.GetAccount(common.PROVIDER, 0)
	provider1, _ := ts.AddAccount(common.PROVIDER, 1, 1000000000)

	// delegate and check delegatetotal
	_, err := ts.TxDualstakingDelegate(delegator.Addr.String(), provider0.Addr.String(), types.NewInt64Coin(ts.TokenDenom(), 5000))
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		res, err := ts.QueryPairingProvider(provider0.Addr.String(), SpecName(i))
		require.NoError(t, err)
		require.Equal(t, int64(1000), res.StakeEntries[0].DelegateTotal.Amount.Int64())
	}

	ts.AdvanceEpoch()

	for i := 0; i < 5; i++ {
		res, err := ts.QueryPairingProvider(provider0.Addr.String(), SpecName(i))
		require.NoError(t, err)
		require.Equal(t, int64(1000), res.StakeEntries[0].DelegateTotal.Amount.Int64())
	}

	// unstake all
	for i := 0; i < 5; i++ {
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
	err = ts.StakeProviderExtra(provider1.GetVaultAddr(), provider0.Addr.String(), ts.Spec(SpecName(0)), testStake, nil, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details)
	require.NoError(ts.T, err)

	res1, err := ts.QueryPairingProvider(provider0.Addr.String(), SpecName(0))
	require.NoError(t, err)
	require.Equal(t, int64(5000), res1.StakeEntries[0].DelegateTotal.Amount.Int64())

	// stake again on spec1 and check delegations
	err = ts.StakeProviderExtra(provider1.GetVaultAddr(), provider0.Addr.String(), ts.Spec(SpecName(1)), testStake, nil, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details)
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

func TestChangeCommisionAndDescription(t *testing.T) {
	ts := newTester(t)
	SetupForSingleProviderTests(ts, 0, 5, 0)
	provider, _ := ts.AddAccount(common.PROVIDER, 0, testBalance)

	d := common.MockDescription()
	err := ts.StakeProviderFull(provider.GetVaultAddr(), provider.Addr.String(), ts.Spec(SpecName(0)), 100, nil, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details, 50)
	require.NoError(ts.T, err)

	md, err := ts.Keepers.Epochstorage.GetMetadata(ts.Ctx, provider.Addr.String())
	require.NoError(ts.T, err)
	require.Equal(t, d, md.Description)
	require.Equal(t, md.DelegateCommission, uint64(50))

	d.Moniker = "test2"
	err = ts.StakeProviderFull(provider.GetVaultAddr(), provider.Addr.String(), ts.Spec(SpecName(1)), 100, nil, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details, 49)
	require.NoError(ts.T, err)

	md, err = ts.Keepers.Epochstorage.GetMetadata(ts.Ctx, provider.Addr.String())
	require.NoError(ts.T, err)
	require.Equal(t, d, md.Description)
	require.Equal(t, md.DelegateCommission, uint64(49))

	res, err := ts.QueryPairingProviders(ts.Spec(SpecName(0)).Index, true)
	require.NoError(ts.T, err)
	require.Equal(t, d, res.StakeEntry[0].Description)
	require.Equal(t, uint64(49), res.StakeEntry[0].DelegateCommission)

	res1, err := ts.QueryPairingProvider(provider.Addr.String(), "")
	require.NoError(ts.T, err)
	require.Equal(t, d, res1.StakeEntries[0].Description)
	require.Equal(t, uint64(49), res1.StakeEntries[0].DelegateCommission)
}

func TestMoveStake(t *testing.T) {
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

	_, err = ts.TxPairingMoveStake(provider0.GetVaultAddr(), SpecName(1), SpecName(0), testStake/2)
	require.NoError(t, err)

	res, err := ts.QueryPairingProvider(provider0.Addr.String(), SpecName(0))
	require.NoError(t, err)
	require.Equal(t, int64(1500), res.StakeEntries[0].DelegateTotal.Amount.Int64())

	res, err = ts.QueryPairingProvider(provider0.Addr.String(), SpecName(1))
	require.NoError(t, err)
	require.Equal(t, int64(500), res.StakeEntries[0].DelegateTotal.Amount.Int64())

	for i := 2; i < 5; i++ {
		res, err := ts.QueryPairingProvider(provider0.Addr.String(), SpecName(i))
		require.NoError(t, err)
		require.Equal(t, int64(1000), res.StakeEntries[0].DelegateTotal.Amount.Int64())
	}
}

func TestPairingWithDelegationDistributions(t *testing.T) {
	ts := newTester(t)
	SetupForSingleProviderTests(ts, 0, 2, 1)

	stake := int64(1000)
	delegator, _ := ts.AddAccount("del", 1, stake*10000)
	client, _ := ts.GetAccount(common.CONSUMER, 0)
	spec := ts.Spec(SpecName(0))

	// stake for spec0
	for i := 0; i < 100; i++ {
		acc, addr := ts.AddAccount(common.PROVIDER, i, testBalance)
		err := ts.StakeProvider(acc.GetVaultAddr(), addr, spec, stake)
		require.NoError(ts.T, err)
	}
	provider, _ := ts.GetAccount(common.PROVIDER, 0)

	// delegate a lot of stake to the first provider
	_, err := ts.TxDualstakingDelegate(delegator.Addr.String(), provider.Addr.String(), types.NewInt64Coin(ts.TokenDenom(), stake*10000))
	require.NoError(t, err)

	numOfPairing := 0
	for i := 0; i < 100; i++ {
		ts.AdvanceEpoch()

		// make sure the provider is in the pairing list
		res, err := ts.QueryPairingGetPairing(spec.Index, client.Addr.String())
		require.NoError(t, err)
		found := false
		for _, p := range res.Providers {
			if p.Address == provider.Addr.String() {
				found = true
				break
			}
		}
		if found {
			numOfPairing++
		}
	}
	require.Greater(t, numOfPairing, 95)

	spec = ts.Spec(SpecName(1))
	// stake to spec1
	for i := 0; i < 100; i++ {
		acc, addr := ts.GetAccount(common.PROVIDER, i)
		err := ts.StakeProvider(acc.GetVaultAddr(), addr, spec, stake)
		require.NoError(ts.T, err)
	}

	// check that provider is in pairing in spec1
	numOfPairing = 0
	for i := 0; i < 100; i++ {
		ts.AdvanceEpoch()

		// make sure the provider is in the pairing list
		res, err := ts.QueryPairingGetPairing(spec.Index, client.Addr.String())
		require.NoError(t, err)
		found := false
		for _, p := range res.Providers {
			if p.Address == provider.Addr.String() {
				found = true
				break
			}
		}
		if found {
			numOfPairing++
		}
	}
	require.Greater(t, numOfPairing, 95)
}

func TestDistributionCli(t *testing.T) {
	ts := newTester(t)
	SetupForSingleProviderTests(ts, 1, 100, 0)

	provider, _ := ts.GetAccount(common.PROVIDER, 0)

	res, err := ts.QueryPairingProvider(provider.Addr.String(), "")
	require.NoError(t, err)

	// distribute each provider 0.1% and last with 90.1%
	distribution := ""
	for i := 0; i < 99; i++ {
		distribution = distribution + SpecName(i) + "," + "0.1" + ","
	}
	distribution = distribution + SpecName(99) + "," + "90.1"

	// get the msgs and run
	msgs, err := cli.CalculateDistbiruitions(provider.Addr.String(), res.StakeEntries, distribution)
	require.NoError(t, err)
	for _, msgRaw := range msgs {
		msg, ok := msgRaw.(*pairingtypes.MsgMoveProviderStake)
		require.True(t, ok)
		require.Equal(t, provider.Addr.String(), msg.Creator)
		require.Equal(t, SpecName(99), msg.DstChain)
		require.NotEqual(t, SpecName(99), msg.SrcChain)
		require.Equal(t, int64(90000), msg.Amount.Amount.Int64())

		_, err := ts.Servers.PairingServer.MoveProviderStake(ts.Ctx, msg)
		require.NoError(t, err)
	}

	// check the stake entries on chain
	for i := 0; i < 99; i++ {
		res, err = ts.QueryPairingProvider(provider.Addr.String(), SpecName(i))
		require.NoError(t, err)
		require.Equal(t, int64(10000), res.StakeEntries[0].Stake.Amount.Int64())
	}

	res, err = ts.QueryPairingProvider(provider.Addr.String(), SpecName(99))
	require.NoError(t, err)
	require.Equal(t, int64(9010000), res.StakeEntries[0].Stake.Amount.Int64())

	// distribute back to 1% each
	res, err = ts.QueryPairingProvider(provider.Addr.String(), "")
	require.NoError(t, err)

	distribution = ""
	for i := 0; i < 100; i++ {
		distribution = distribution + SpecName(i) + "," + "1" + ","
	}
	distribution = distribution[:len(distribution)-1]

	msgs, err = cli.CalculateDistbiruitions(provider.Addr.String(), res.StakeEntries, distribution)
	require.NoError(t, err)
	for _, msgRaw := range msgs {
		msg, ok := msgRaw.(*pairingtypes.MsgMoveProviderStake)
		require.True(t, ok)
		require.Equal(t, provider.Addr.String(), msg.Creator)
		require.Equal(t, SpecName(99), msg.SrcChain)
		require.NotEqual(t, SpecName(99), msg.DstChain)
		require.Equal(t, int64(90000), msg.Amount.Amount.Int64())

		_, err := ts.Servers.PairingServer.MoveProviderStake(ts.Ctx, msg)
		require.NoError(t, err)
	}

	for i := 0; i < 100; i++ {
		res, err = ts.QueryPairingProvider(provider.Addr.String(), SpecName(i))
		require.NoError(t, err)
		require.Equal(t, testStake, res.StakeEntries[0].Stake.Amount.Int64())
	}
}
