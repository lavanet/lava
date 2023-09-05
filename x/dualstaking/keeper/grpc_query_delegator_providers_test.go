package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

func TestDelegatorProvidersQuery(t *testing.T) {
	ts := newTester(t)

	ts.setupForDelegation(1, 1, 0, 0) // 1 delegator, 1 staked provider

	_, delegator := ts.GetAccount(common.CONSUMER, 0)
	_, provider := ts.GetAccount(common.PROVIDER, 0)

	spec := ts.Spec("mock")

	amountUint64 := uint64(100)
	amount := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewIntFromUint64(amountUint64))

	// delegate and query
	_, err := ts.TxDualstakingDelegate(delegator, provider, spec.Index, amount)
	require.Nil(t, err)
	ts.AdvanceEpoch()

	delegation := types.NewDelegation(delegator, provider, spec.Index)
	delegation.Amount = amount

	res, err := ts.QueryDualstakingDelegatorProviders(delegator, false)
	require.Nil(t, err)
	delegationRes := res.Delegations[0]
	require.True(t, delegation.Equal(&delegationRes))

	// partially unbond and query
	unbondAmount := amount.Sub(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.OneInt()))
	_, err = ts.TxDualstakingUnbond(delegator, provider, spec.Index, unbondAmount)
	require.Nil(t, err)
	ts.AdvanceEpoch()

	bondedAmount := amount.Sub(unbondAmount)
	delegation.Amount = bondedAmount

	res, err = ts.QueryDualstakingDelegatorProviders(delegator, false)
	require.Nil(t, err)
	delegation = res.Delegations[0]
	require.True(t, delegation.Equal(&delegationRes))

	// unbond completely and query (should not work)
	// ts.AdvanceEpoch() // TODO: if you put AdvanceBlock here, we get a critical error from unbonding. why?
	_, err = ts.TxDualstakingUnbond(delegator, provider, spec.Index, bondedAmount)
	require.Nil(t, err)
	ts.AdvanceEpoch()

	_, err = ts.QueryDualstakingDelegatorProviders(delegator, false)
	require.NotNil(t, err)
}

func TestShowPendingDelegators(t *testing.T) {
	ts := newTester(t)

	ts.setupForDelegation(2, 1, 0, 0) // 2 delegators, 1 staked provider

	_, delegator1 := ts.GetAccount(common.CONSUMER, 0)
	_, delegator2 := ts.GetAccount(common.CONSUMER, 1)
	_, provider := ts.GetAccount(common.PROVIDER, 0)

	spec := ts.Spec("mock")

	amountUint64 := uint64(100)
	amount := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewIntFromUint64(amountUint64))

	delegation1 := types.NewDelegation(delegator1, provider, spec.Index)
	delegation1.Amount = amount

	// delegate without advancing an epoch
	_, err := ts.TxDualstakingDelegate(delegator1, provider, spec.Index, amount)
	require.Nil(t, err)

	// query pending delegators
	res, err := ts.QueryDualstakingDelegatorProviders(delegator1, true)
	require.Nil(t, err)
	delegationRes := res.Delegations[0]
	require.True(t, delegationRes.Equal(&delegation1))

	// query current delegators
	_, err = ts.QueryDualstakingDelegatorProviders(delegator1, false)
	require.NotNil(t, err)

	// advance epoch, delegate delegator2 and query again
	ts.AdvanceEpoch()

	delegation2 := types.NewDelegation(delegator2, provider, spec.Index)
	delegation2.Amount = amount
	_, err = ts.TxDualstakingDelegate(delegator2, provider, spec.Index, amount)
	require.Nil(t, err)

	// delegator2 should show when quering with showPending=true and not show when showPending=false
	// delegator1 should always show
	res, err = ts.QueryDualstakingDelegatorProviders(delegator1, true)
	require.Nil(t, err)
	require.Equal(t, 2, len(res.Delegations))
	for _, d := range res.Delegations {
		if d.Delegator == delegator1 {
			require.True(t, d.Equal(&delegation1))
		} else if d.Delegator == delegator2 {
			require.True(t, d.Equal(&delegation2))
		}
	}

	res, err = ts.QueryDualstakingDelegatorProviders(delegator1, false)
	require.Nil(t, err)
	require.Equal(t, 1, len(res.Delegations))
	delegationRes = res.Delegations[0]
	require.True(t, delegationRes.Equal(&delegation1))
}
