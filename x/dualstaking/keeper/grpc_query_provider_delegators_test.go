package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v3/testutil/common"
	"github.com/lavanet/lava/v3/x/dualstaking/types"
	"github.com/stretchr/testify/require"
)

func TestQueryProviderDelegatorsWithUnbonding(t *testing.T) {
	ts := newTester(t)

	ts.setupForDelegation(1, 1, 0, 0) // 1 delegator, 1 staked provider

	_, delegator := ts.GetAccount(common.CONSUMER, 0)
	_, provider := ts.GetAccount(common.PROVIDER, 0)

	spec := ts.Spec("mock")

	amountUint64 := uint64(100)
	amount := sdk.NewCoin(ts.TokenDenom(), sdk.NewIntFromUint64(amountUint64))

	// delegate and query
	_, err := ts.TxDualstakingDelegate(delegator, provider, spec.Index, amount)
	require.NoError(t, err)
	ts.AdvanceEpoch()

	delegation := types.NewDelegation(delegator, provider, spec.Index, ts.Ctx.BlockTime(), ts.TokenDenom())
	delegation.Amount = amount

	res, err := ts.QueryDualstakingProviderDelegators(provider, false)
	require.NoError(t, err)
	require.Equal(t, 2, len(res.Delegations))
	var delegationRes types.Delegation
	for _, d := range res.Delegations {
		if d.Delegator == delegation.Delegator {
			delegationRes = d
		}
	}
	require.True(t, delegation.Equal(&delegationRes))

	// partially unbond and query
	unbondAmount := amount.Sub(sdk.NewCoin(ts.TokenDenom(), sdk.OneInt()))
	_, err = ts.TxDualstakingUnbond(delegator, provider, spec.Index, unbondAmount)
	require.NoError(t, err)
	ts.AdvanceEpoch()

	bondedAmount := amount.Sub(unbondAmount)
	delegation.Amount = bondedAmount

	res, err = ts.QueryDualstakingProviderDelegators(provider, false)
	require.NoError(t, err)
	require.Equal(t, 2, len(res.Delegations))
	for _, d := range res.Delegations {
		if d.Delegator == delegation.Delegator {
			delegationRes = d
		}
	}
	require.True(t, delegation.Equal(&delegationRes))

	// unbond completely and query (should not get providers)
	_, err = ts.TxDualstakingUnbond(delegator, provider, spec.Index, bondedAmount)
	require.NoError(t, err)
	ts.AdvanceEpoch()

	res, err = ts.QueryDualstakingProviderDelegators(provider, false)
	require.NoError(t, err)
	require.Equal(t, 1, len(res.Delegations))
}

func TestQueryProviderDelegatorsWithPendingDelegations(t *testing.T) {
	ts := newTester(t)

	ts.setupForDelegation(2, 1, 0, 0) // 2 delegators, 1 staked provider

	_, delegator1 := ts.GetAccount(common.CONSUMER, 0)
	_, delegator2 := ts.GetAccount(common.CONSUMER, 1)
	_, provider := ts.GetAccount(common.PROVIDER, 0)

	spec := ts.Spec("mock")

	amountUint64 := uint64(100)
	amount := sdk.NewCoin(ts.TokenDenom(), sdk.NewIntFromUint64(amountUint64))

	delegation1 := types.NewDelegation(delegator1, provider, spec.Index, ts.Ctx.BlockTime(), ts.TokenDenom())
	delegation1.Amount = amount

	// delegate without advancing an epoch
	_, err := ts.TxDualstakingDelegate(delegator1, provider, spec.Index, amount)
	require.NoError(t, err)

	// query pending delegators
	res, err := ts.QueryDualstakingProviderDelegators(provider, true)
	require.NoError(t, err)
	require.Equal(t, 2, len(res.Delegations))
	var delegationRes types.Delegation
	for _, d := range res.Delegations {
		if d.Delegator == delegation1.Delegator {
			delegationRes = d
		}
	}
	require.True(t, delegationRes.Equal(&delegation1))

	// query current delegators
	res, err = ts.QueryDualstakingProviderDelegators(provider, false)
	require.NoError(t, err)
	require.Equal(t, 1, len(res.Delegations))

	// advance epoch, delegator1 should show in both flag values
	ts.AdvanceEpoch()

	res, err = ts.QueryDualstakingProviderDelegators(provider, true)
	require.NoError(t, err)
	require.Equal(t, 2, len(res.Delegations))
	for _, d := range res.Delegations {
		if d.Delegator == delegation1.Delegator {
			delegationRes = d
		}
	}
	require.True(t, delegationRes.Equal(&delegation1))

	res, err = ts.QueryDualstakingProviderDelegators(provider, false)
	require.NoError(t, err)
	require.Equal(t, 2, len(res.Delegations))
	for _, d := range res.Delegations {
		if d.Delegator == delegation1.Delegator {
			delegationRes = d
		}
	}
	require.True(t, delegationRes.Equal(&delegation1))

	// delegate delegator2 and query again
	delegation2 := types.NewDelegation(delegator2, provider, spec.Index, ts.Ctx.BlockTime(), ts.TokenDenom())
	delegation2.Amount = amount
	_, err = ts.TxDualstakingDelegate(delegator2, provider, spec.Index, amount)
	require.NoError(t, err)

	// delegator2 should show when quering with showPending=true and not show when showPending=false
	res, err = ts.QueryDualstakingProviderDelegators(provider, true)
	require.NoError(t, err)
	require.Equal(t, 3, len(res.Delegations))
	for _, d := range res.Delegations {
		if d.Delegator == delegator1 {
			require.True(t, d.Equal(&delegation1))
		} else if d.Delegator == delegator2 {
			require.True(t, d.Equal(&delegation2))
		}
	}

	res, err = ts.QueryDualstakingProviderDelegators(provider, false)
	require.NoError(t, err)
	require.Equal(t, 2, len(res.Delegations))
	for _, d := range res.Delegations {
		if d.Delegator == delegation1.Delegator {
			delegationRes = d
		}
	}
	require.True(t, delegationRes.Equal(&delegation1))
}

func TestQueryProviderDelegatorsProviderMultipleDelegators(t *testing.T) {
	ts := newTester(t)

	ts.setupForDelegation(3, 1, 0, 0) // 3 delegators, 1 staked provider

	_, delegator1 := ts.GetAccount(common.CONSUMER, 0)
	_, delegator2 := ts.GetAccount(common.CONSUMER, 1)
	_, delegator3 := ts.GetAccount(common.CONSUMER, 2)
	providerAcc, provider := ts.GetAccount(common.PROVIDER, 0)

	ts.AddSpec("mock1", common.CreateMockSpec())

	delegators := []string{delegator1, delegator2, delegator3}

	spec := ts.Spec("mock")
	spec1 := ts.Spec("mock1")
	err := ts.StakeProvider(providerAcc.GetVaultAddr(), provider, spec1, testStake)
	require.NoError(t, err)

	amountUint64 := uint64(100)
	amount := sdk.NewCoin(ts.TokenDenom(), sdk.NewIntFromUint64(amountUint64))

	delegations := []types.Delegation{}
	for i := 0; i < len(delegators); i++ {
		var chainID string
		if i == 0 {
			chainID = spec.Index
		} else {
			chainID = spec1.Index
		}
		_, err := ts.TxDualstakingDelegate(delegators[i], provider, chainID, amount)
		require.NoError(t, err)

		delegation := types.NewDelegation(delegators[i], provider, chainID, ts.Ctx.BlockTime(), ts.TokenDenom())
		delegation.Amount = amount
		delegations = append(delegations, delegation)
	}

	ts.AdvanceEpoch()
	res, err := ts.QueryDualstakingProviderDelegators(provider, false)
	require.Equal(t, 4, len(res.Delegations))
	require.NoError(t, err)
	for _, d := range delegations {
		for _, resD := range res.Delegations {
			if d.Delegator == resD.Delegator {
				require.True(t, resD.Equal(&d))
			}
		}
	}
}

func TestQueryProviderDelegatorsDelegatorMultipleProviders(t *testing.T) {
	ts := newTester(t)

	ts.setupForDelegation(1, 3, 0, 0) // 1 delegator, 3 staked providers

	_, delegator := ts.GetAccount(common.CONSUMER, 0)
	_, provider1 := ts.GetAccount(common.PROVIDER, 0)
	_, provider2 := ts.GetAccount(common.PROVIDER, 1)
	_, provider3 := ts.GetAccount(common.PROVIDER, 2)

	providers := []string{provider1, provider2, provider3}

	spec := ts.Spec("mock")

	amountUint64 := uint64(100)
	amount := sdk.NewCoin(ts.TokenDenom(), sdk.NewIntFromUint64(amountUint64))

	delegations := []types.Delegation{}
	for i := 0; i < len(providers); i++ {
		_, err := ts.TxDualstakingDelegate(delegator, providers[i], spec.Index, amount)
		require.NoError(t, err)

		delegation := types.NewDelegation(delegator, providers[i], spec.Index, ts.Ctx.BlockTime(), ts.TokenDenom())
		delegation.Amount = amount
		delegations = append(delegations, delegation)
	}

	ts.AdvanceEpoch()

	for i := 0; i < len(providers); i++ {
		res, err := ts.QueryDualstakingProviderDelegators(providers[i], false)
		require.NoError(t, err)
		require.Equal(t, 2, len(res.Delegations))
		var delegationRes types.Delegation
		for _, d := range res.Delegations {
			if d.Delegator == delegations[i].Delegator {
				delegationRes = d
			}
		}
		require.True(t, delegationRes.Equal(&delegations[i]))
	}
}

func TestQueryProviderDelegatorsDelegatorUnstakedProvider(t *testing.T) {
	ts := newTester(t)

	ts.setupForDelegation(1, 0, 1, 1) // 1 delegator, 1 unstaked provider, 1 unstaking provider

	_, delegator := ts.GetAccount(common.CONSUMER, 0)
	_, unstakedProvider := ts.GetAccount(common.PROVIDER, 0)
	_, unstakingProvider := ts.GetAccount(common.PROVIDER, 1)

	spec := ts.Spec("mock")

	amountUint64 := uint64(100)
	amount := sdk.NewCoin(ts.TokenDenom(), sdk.NewIntFromUint64(amountUint64))

	// shouldn't be able to delegate to unstaked provider
	_, err := ts.TxDualstakingDelegate(delegator, unstakedProvider, spec.Index, amount)
	require.Error(t, err)

	// shouldn't be able to delegate to unstaking provider (even though it didn't get its funds back, it's still considered unstaked)
	_, err = ts.TxDualstakingDelegate(delegator, unstakingProvider, spec.Index, amount)
	require.Error(t, err)
}
