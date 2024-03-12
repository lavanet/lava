package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testutils "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestUnstakeAndSlashProposal(t *testing.T) {
	ts := newTester(t)
	delegators := 5
	ts.setupForPayments(1, delegators, 0)

	ts.AdvanceEpoch()

	_, provider := ts.GetAccount(common.PROVIDER, 0)
	delegatorsSlashing := []types.DelegatorSlashing{}
	beforeSlashDelegation := map[string]sdk.Int{}
	for i := 0; i < delegators; i++ {
		_, delegator := ts.GetAccount(common.CONSUMER, i)
		beforeSlashDelegation[delegator] = sdk.NewInt(1000 * int64(i+1))
		_, err := ts.TxDualstakingDelegate(delegator, provider, ts.spec.Index, sdk.NewCoin(ts.BondDenom(), beforeSlashDelegation[delegator]))
		require.NoError(t, err)
		delegatorsSlashing = append(delegatorsSlashing, types.DelegatorSlashing{Delegator: delegator, SlashingAmount: sdk.NewCoin(ts.BondDenom(), sdk.NewInt(1000/3*int64(i+1)))})
	}

	ts.AdvanceEpoch()

	err := testutils.SimulateUnstakeProposal(ts.Ctx,
		ts.Keepers.Pairing,
		[]types.ProviderUnstakeInfo{{Provider: provider, ChainId: ts.spec.Index}},
		delegatorsSlashing,
	)
	require.NoError(t, err)

	result, err := ts.QueryDualstakingProviderDelegators(provider, true)
	require.NoError(t, err)
	for _, d := range result.Delegations {
		for _, s := range delegatorsSlashing {
			if d.Delegator == s.Delegator {
				require.Equal(t, beforeSlashDelegation[d.Delegator].Sub(s.SlashingAmount.Amount), d.Amount.Amount)
			}
		}
	}
}
