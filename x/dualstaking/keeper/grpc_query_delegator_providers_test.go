package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/testutil/common"
	"github.com/lavanet/lava/v4/x/dualstaking/types"
	"github.com/stretchr/testify/require"
)

func TestQueryWithUnbonding(t *testing.T) {
	ts := newTester(t)

	ts.setupForDelegation(1, 1, 0, 0) // 1 delegator, 1 staked provider

	_, delegator := ts.GetAccount(common.CONSUMER, 0)
	_, provider := ts.GetAccount(common.PROVIDER, 0)

	amountUint64 := uint64(100)
	amount := sdk.NewCoin(ts.TokenDenom(), sdk.NewIntFromUint64(amountUint64))

	// delegate and query
	_, err := ts.TxDualstakingDelegate(delegator, provider, amount)
	require.NoError(t, err)
	ts.AdvanceEpoch()

	delegation := types.NewDelegation(delegator, provider, ts.Ctx.BlockTime(), ts.TokenDenom())
	delegation.Amount = amount

	res, err := ts.QueryDualstakingDelegatorProviders(delegator)
	require.NoError(t, err)
	delegationRes := res.Delegations[0]
	require.True(t, delegation.Equal(&delegationRes))

	// partially unbond and query
	unbondAmount := amount.Sub(sdk.NewCoin(ts.TokenDenom(), sdk.OneInt()))
	_, err = ts.TxDualstakingUnbond(delegator, provider, unbondAmount)
	require.NoError(t, err)
	ts.AdvanceEpoch()

	bondedAmount := amount.Sub(unbondAmount)
	delegation.Amount = bondedAmount

	res, err = ts.QueryDualstakingDelegatorProviders(delegator)
	require.NoError(t, err)
	delegationRes = res.Delegations[0]
	require.True(t, delegation.Equal(&delegationRes))

	// unbond completely and query (should not get providers)
	_, err = ts.TxDualstakingUnbond(delegator, provider, bondedAmount)
	require.NoError(t, err)
	ts.AdvanceEpoch()

	res, err = ts.QueryDualstakingDelegatorProviders(delegator)
	require.NoError(t, err)
	require.Equal(t, 0, len(res.Delegations))
}

func TestQueryDelegations(t *testing.T) {
	ts := newTester(t)

	ts.setupForDelegation(2, 1, 0, 0) // 2 delegators, 1 staked provider

	_, delegator1 := ts.GetAccount(common.CONSUMER, 0)
	_, delegator2 := ts.GetAccount(common.CONSUMER, 1)
	_, provider := ts.GetAccount(common.PROVIDER, 0)

	amountUint64 := uint64(100)
	amount := sdk.NewCoin(ts.TokenDenom(), sdk.NewIntFromUint64(amountUint64))

	delegation1 := types.NewDelegation(delegator1, provider, ts.Ctx.BlockTime(), ts.TokenDenom())
	delegation1.Amount = amount

	// delegate without advancing an epoch
	_, err := ts.TxDualstakingDelegate(delegator1, provider, amount)
	require.NoError(t, err)

	// query pending delegators
	res, err := ts.QueryDualstakingDelegatorProviders(delegator1)
	require.NoError(t, err)
	require.Equal(t, 1, len(res.Delegations))
	delegationRes := res.Delegations[0]
	require.True(t, delegationRes.Equal(&delegation1))

	// advance epoch, delegator1 should show in both flag values
	ts.AdvanceEpoch()

	res, err = ts.QueryDualstakingDelegatorProviders(delegator1)
	require.NoError(t, err)
	require.Equal(t, 1, len(res.Delegations))
	delegationRes = res.Delegations[0]
	require.True(t, delegationRes.Equal(&delegation1))

	// delegate delegator2 and query again
	delegation2 := types.NewDelegation(delegator2, provider, ts.Ctx.BlockTime(), ts.TokenDenom())

	delegation2.Amount = amount
	_, err = ts.TxDualstakingDelegate(delegator2, provider, amount)
	require.NoError(t, err)

	// delegator2 should show when quering with showPending=true and not show when showPending=false
	res, err = ts.QueryDualstakingDelegatorProviders(delegator2)
	require.NoError(t, err)
	require.Equal(t, 1, len(res.Delegations))
	delegationRes = res.Delegations[0]
	require.True(t, delegationRes.Equal(&delegation2))
}

func TestQueryProviderMultipleDelegators(t *testing.T) {
	ts := newTester(t)

	ts.setupForDelegation(3, 1, 0, 0) // 3 delegators, 1 staked provider

	_, delegator1 := ts.GetAccount(common.CONSUMER, 0)
	_, delegator2 := ts.GetAccount(common.CONSUMER, 1)
	_, delegator3 := ts.GetAccount(common.CONSUMER, 2)
	providerAcc, provider := ts.GetAccount(common.PROVIDER, 0)

	ts.AddSpec("mock1", common.CreateMockSpec())

	delegators := []string{delegator1, delegator2, delegator3}

	spec1 := ts.Spec("mock1")
	err := ts.StakeProvider(providerAcc.GetVaultAddr(), provider, spec1, testStake)
	require.NoError(t, err)

	amountUint64 := uint64(100)
	amount := sdk.NewCoin(ts.TokenDenom(), sdk.NewIntFromUint64(amountUint64))

	delegations := []types.Delegation{}
	for i := 0; i < len(delegators); i++ {
		_, err := ts.TxDualstakingDelegate(delegators[i], provider, amount)
		require.NoError(t, err)

		delegation := types.NewDelegation(delegators[i], provider, ts.Ctx.BlockTime(), ts.TokenDenom())
		delegation.Amount = amount
		delegations = append(delegations, delegation)
	}

	ts.AdvanceEpoch()

	for i := 0; i < len(delegations); i++ {
		res, err := ts.QueryDualstakingDelegatorProviders(delegations[i].Delegator)
		require.NoError(t, err)
		require.Equal(t, 1, len(res.Delegations))
		resDelegation := res.Delegations[0]
		require.True(t, resDelegation.Equal(&delegations[i]))
	}
}

func TestQueryDelegatorMultipleProviders(t *testing.T) {
	ts := newTester(t)

	ts.setupForDelegation(1, 3, 0, 0) // 1 delegator, 3 staked providers

	_, delegator := ts.GetAccount(common.CONSUMER, 0)
	_, provider1 := ts.GetAccount(common.PROVIDER, 0)
	_, provider2 := ts.GetAccount(common.PROVIDER, 1)
	_, provider3 := ts.GetAccount(common.PROVIDER, 2)

	providers := []string{provider1, provider2, provider3}

	amountUint64 := uint64(100)
	amount := sdk.NewCoin(ts.TokenDenom(), sdk.NewIntFromUint64(amountUint64))

	delegations := map[string]types.Delegation{}
	for i := 0; i < len(providers); i++ {
		_, err := ts.TxDualstakingDelegate(delegator, providers[i], amount)
		require.NoError(t, err)

		delegation := types.NewDelegation(delegator, providers[i], ts.Ctx.BlockTime(), ts.TokenDenom())
		delegation.Amount = amount
		delegations[providers[i]] = delegation
	}

	ts.AdvanceEpoch()

	res, err := ts.QueryDualstakingDelegatorProviders(delegator)
	require.NoError(t, err)
	require.Equal(t, 3, len(res.Delegations))
	for i := 0; i < len(delegations); i++ {
		resDelegation := res.Delegations[i]
		delegation := delegations[resDelegation.Provider]
		require.True(t, resDelegation.Equal(&delegation))
	}
}

func TestQueryDelegatorUnstakedProvider(t *testing.T) {
	ts := newTester(t)

	ts.setupForDelegation(1, 0, 1, 1) // 1 delegator, 1 unstaked provider, 1 unstaking provider

	_, delegator := ts.GetAccount(common.CONSUMER, 0)
	_, unstakedProvider := ts.GetAccount(common.PROVIDER, 0)
	_, unstakingProvider := ts.GetAccount(common.PROVIDER, 1)

	amountUint64 := uint64(100)
	amount := sdk.NewCoin(ts.TokenDenom(), sdk.NewIntFromUint64(amountUint64))

	// shouldn't be able to delegate to unstaked provider
	_, err := ts.TxDualstakingDelegate(delegator, unstakedProvider, amount)
	require.Error(t, err)

	// shouldn't be able to delegate to unstaking provider (even though it didn't get its funds back, it's still considered unstaked)
	_, err = ts.TxDualstakingDelegate(delegator, unstakingProvider, amount)
	require.Error(t, err)
}
