package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	dualstakingkeeper "github.com/lavanet/lava/x/dualstaking/keeper"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

// test that creation of a validator creates a delegation from the validator to an empty provider
func TestCreateValidator(t *testing.T) {
	ts := newTester(t)
	ts.addValidators(1)

	validator, _ := ts.GetAccount(common.VALIDATOR, 0)

	amount := sdk.NewIntFromUint64(100)
	_, err := ts.TxCreateValidator(validator, amount)
	require.Nil(t, err)

	res, err := ts.QueryDualstakingProviderDelegators(dualstakingkeeper.EMPTY_PROVIDER, true)
	require.Nil(t, err)
	require.Equal(t, res.Delegations[0].Delegator, validator.Addr.String())
}

// test that delegating to a validator also delegates to an empty provider
func TestDelegateToValidator(t *testing.T) {
	ts := newTester(t)
	ts.addValidators(1)

	validator, _ := ts.GetAccount(common.VALIDATOR, 0)

	amount := sdk.NewIntFromUint64(100)
	_, err := ts.TxCreateValidator(validator, amount)
	require.Nil(t, err)

	res, err := ts.QueryDualstakingProviderDelegators(dualstakingkeeper.EMPTY_PROVIDER, true)
	require.Nil(t, err)
	require.Equal(t, res.Delegations[0].Delegator, validator.Addr.String())

	ts.addClients(1)
	delegator, _ := ts.GetAccount(common.CONSUMER, 0)
	_, err = ts.TxDelegateValidator(delegator, validator, amount)
	require.Nil(t, err)

	res, err = ts.QueryDualstakingProviderDelegators(dualstakingkeeper.EMPTY_PROVIDER, true)
	require.Nil(t, err)
	require.Equal(t, 2, len(res.Delegations))
	require.True(t, res.Delegations[0].Delegator == validator.Addr.String() || res.Delegations[0].Delegator == delegator.Addr.String())
	require.True(t, res.Delegations[1].Delegator == validator.Addr.String() || res.Delegations[1].Delegator == delegator.Addr.String())
}

// test that redelegating to a validator does not affect the delegator state
func TestReDelegateToValidator(t *testing.T) {
	ts := newTester(t)
	ts.addValidators(2)

	validator1, _ := ts.GetAccount(common.VALIDATOR, 0)
	validator2, _ := ts.GetAccount(common.VALIDATOR, 1)

	amount := sdk.NewIntFromUint64(100)
	_, err := ts.TxCreateValidator(validator1, amount)
	require.Nil(t, err)

	_, err = ts.TxCreateValidator(validator2, amount)
	require.Nil(t, err)

	delegatorsRes, err := ts.QueryDualstakingProviderDelegators(dualstakingkeeper.EMPTY_PROVIDER, true)
	require.Nil(t, err)
	require.Equal(t, 2, len(delegatorsRes.Delegations))

	ts.addClients(1)
	delegator, _ := ts.GetAccount(common.CONSUMER, 0)
	_, err = ts.TxDelegateValidator(delegator, validator1, amount)
	require.Nil(t, err)

	delegatorsRes, err = ts.QueryDualstakingProviderDelegators(dualstakingkeeper.EMPTY_PROVIDER, true)
	require.Nil(t, err)
	require.Equal(t, 3, len(delegatorsRes.Delegations))

	providersRes, err := ts.QueryDualstakingDelegatorProviders(delegator.Addr.String(), true)
	require.Nil(t, err)
	require.Equal(t, 1, len(providersRes.Delegations))
	require.Equal(t, delegator.Addr.String(), providersRes.Delegations[0].Delegator)

	_, err = ts.TxReDelegateValidator(delegator, validator1, validator2, amount)
	require.Nil(t, err)

	delegatorsRes1, err := ts.QueryDualstakingProviderDelegators(dualstakingkeeper.EMPTY_PROVIDER, true)
	require.Nil(t, err)
	require.Equal(t, delegatorsRes, delegatorsRes1)

	providersRes1, err := ts.QueryDualstakingDelegatorProviders(delegator.Addr.String(), true)
	require.Nil(t, err)
	require.Equal(t, providersRes, providersRes1)
}

// test that redelegating to a validator does not affect the delegator state
func TestReDelegateToProvider(t *testing.T) {
	ts := newTester(t)
	ts.addValidators(1)
	ts.addProviders(1)
	ts.addClients(1)

	validator, _ := ts.GetAccount(common.VALIDATOR, 0)
	amount := sdk.NewIntFromUint64(10000)
	_, err := ts.TxCreateValidator(validator, amount)
	require.Nil(t, err)

	provider, _ := ts.GetAccount(common.PROVIDER, 0)
	err = ts.StakeProvider(provider.Addr.String(), ts.spec, amount.Int64())
	require.Nil(t, err)

	ts.AdvanceEpoch()

	delegator, _ := ts.GetAccount(common.CONSUMER, 0)
	_, err = ts.TxDelegateValidator(delegator, validator, amount)
	require.Nil(t, err)

	epoch := ts.EpochStart()
	entry, err := ts.Keepers.Epochstorage.GetStakeEntryForProviderEpoch(ts.Ctx, ts.spec.Index, provider.Addr, epoch)
	require.Nil(t, err)
	require.Equal(t, amount, entry.Stake.Amount)

	providersRes, err := ts.QueryDualstakingDelegatorProviders(delegator.Addr.String(), true)
	require.Nil(t, err)
	require.Equal(t, 1, len(providersRes.Delegations))
	require.Equal(t, dualstakingkeeper.EMPTY_PROVIDER, providersRes.Delegations[0].Provider)

	ts.AdvanceEpoch()

	_, err = ts.TxDualstakingRedelegate(delegator.Addr.String(),
		dualstakingkeeper.EMPTY_PROVIDER,
		provider.Addr.String(),
		dualstakingkeeper.EMPTY_PROVIDER_CHAINID,
		entry.Chain,
		sdk.NewCoin(epochstoragetypes.TokenDenom, amount))

	require.Nil(t, err)

	providersRes1, err := ts.QueryDualstakingDelegatorProviders(delegator.Addr.String(), false)
	require.Nil(t, err)
	require.Equal(t, providersRes, providersRes1)

	providersRes1, err = ts.QueryDualstakingDelegatorProviders(delegator.Addr.String(), true)
	require.Nil(t, err)
	require.Equal(t, provider.Addr.String(), providersRes1.Delegations[0].Provider)

	ts.AdvanceEpoch()

	epoch = ts.EpochStart()
	entry, err = ts.Keepers.Epochstorage.GetStakeEntryForProviderEpoch(ts.Ctx, ts.spec.Index, provider.Addr, epoch)
	require.Nil(t, err)
	require.Equal(t, amount, entry.DelegateTotal.Amount)
	require.Equal(t, amount, entry.Stake.Amount)
}

// TestUnbondUniformProviders checks that the uniform unbond of providers (that is triggered by a validator unbond)
// works as expected. The test case is as follows:
// [10 20 50 60 70] 25 -> [0 20 50 60 70] 25 + 15/4 -> [0 0 50 60 70] 25 + 15/4 + 8.75/3
// Explanation: assume 5 providers each with different delegations (10, 20, and so on). The validator
// unbonds 25*5 tokens. In this case, each provider should have its delegation decreased by 25 tokens.
// Not all providers have delegations with 25+ token, so we decrease their whole delegation and take the
// remainder from other providers uniformly
func TestUnbondUniformProviders(t *testing.T) {
	ts := newTester(t)
	ts.addValidators(1)
	ts.addProviders(5)
	ts.addClients(1)

	// create validator and providers
	validator, _ := ts.GetAccount(common.VALIDATOR, 0)
	amount := sdk.NewIntFromUint64(10000)
	_, err := ts.TxCreateValidator(validator, amount)
	require.Nil(t, err)

	for i := 0; i < 5; i++ {
		provider, _ := ts.GetAccount(common.PROVIDER, i)
		err = ts.StakeProvider(provider.Addr.String(), ts.spec, amount.Int64())
		require.Nil(t, err)
	}

	ts.AdvanceEpoch()

	// delegate to validator (automatically delegates to empty provider)
	delegatorAcc, delegator := ts.GetAccount(common.CONSUMER, 0)
	_, err = ts.TxDelegateValidator(delegatorAcc, validator, sdk.NewInt(210))
	require.Nil(t, err)

	// redelegate from empty provider to all providers with fixed amounts
	redelegateAmts := []math.Int{
		sdk.NewInt(10),
		sdk.NewInt(20),
		sdk.NewInt(50),
		sdk.NewInt(60),
		sdk.NewInt(70),
	}
	var providers []string
	for i := 0; i < 5; i++ {
		_, provider := ts.GetAccount(common.PROVIDER, i)
		providers = append(providers, provider)
		_, err = ts.TxDualstakingRedelegate(delegatorAcc.Addr.String(),
			dualstakingkeeper.EMPTY_PROVIDER,
			provider,
			dualstakingkeeper.EMPTY_PROVIDER_CHAINID,
			ts.spec.Index,
			sdk.NewCoin(epochstoragetypes.TokenDenom, redelegateAmts[i]))
		require.Nil(t, err)
	}

	// unbond 25*5 tokens from validator
	_, err = ts.TxUnbondValidator(delegatorAcc, validator, sdk.NewInt(25*5))
	require.Nil(t, err)

	res, err := ts.QueryDualstakingDelegatorProviders(delegator, false)
	require.Nil(t, err)
	for _, d := range res.Delegations {
		switch d.Provider {
		case providers[0]:
			require.True(t, d.Amount.Amount.IsZero())
		case providers[1]:
			require.True(t, d.Amount.Amount.IsZero())
		case providers[2]:
			require.Equal(t, int64(20), d.Amount.Amount.Int64())
		case providers[3]:
			require.Equal(t, int64(30), d.Amount.Amount)
		case providers[4]:
			require.Equal(t, int64(35), d.Amount.Amount) // highest delegation is decreased by uniform amount + remainder
		default:
			require.FailNow(t, "unexpected provider in delegations")
		}
	}

	diff, err := ts.Keepers.Dualstaking.VerifyDelegatorBalance(ts.Ctx, delegatorAcc.Addr)
	require.Nil(t, err)
	require.True(t, diff.IsZero())
}
