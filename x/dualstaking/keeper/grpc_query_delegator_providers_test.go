package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
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

	res, err := ts.QueryDualstakingDelegatorProviders(delegator)
	require.Nil(t, err)
	delegation := res.Delegations[0]
	require.Equal(t, provider, delegation.Provider)
	require.Equal(t, delegator, delegation.Delegator)
	require.Equal(t, spec.Index, delegation.ChainID)
	require.True(t, amount.Equal(delegation.Amount))

	// partially unbond and query
	unbondAmount := amount.Sub(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.OneInt()))
	_, err = ts.TxDualstakingUnbond(delegator, provider, spec.Index, unbondAmount)
	require.Nil(t, err)

	bondedAmount := amount.Sub(unbondAmount)

	res, err = ts.QueryDualstakingDelegatorProviders(delegator)
	require.Nil(t, err)
	delegation = res.Delegations[0]
	require.Equal(t, provider, delegation.Provider)
	require.Equal(t, delegator, delegation.Delegator)
	require.Equal(t, spec.Index, delegation.ChainID)
	require.True(t, bondedAmount.Equal(delegation.Amount))

	// unbond completely and query (should not work)
	ts.AdvanceEpoch() // TODO: if you put AdvanceBlock here, we get a critical error from unbonding. why?
	_, err = ts.TxDualstakingUnbond(delegator, provider, spec.Index, bondedAmount)
	require.Nil(t, err)

	_, err = ts.QueryDualstakingDelegatorProviders(delegator)
	require.NotNil(t, err)
}
