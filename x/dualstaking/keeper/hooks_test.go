package keeper_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v2/testutil/common"
	"github.com/lavanet/lava/v2/utils"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	"github.com/lavanet/lava/v2/utils/sigs"
	dualstakingtypes "github.com/lavanet/lava/v2/x/dualstaking/types"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

// test that creation of a validator creates a delegation from the validator to an empty provider
func TestCreateValidator(t *testing.T) {
	ts := newTester(t)
	ts.addValidators(1)

	validator, _ := ts.GetAccount(common.VALIDATOR, 0)

	amount := sdk.NewIntFromUint64(100)
	ts.TxCreateValidator(validator, amount)

	res, err := ts.QueryDualstakingProviderDelegators(commontypes.EMPTY_PROVIDER, true)
	require.NoError(t, err)
	require.Equal(t, res.Delegations[0].Delegator, validator.Addr.String())
}

// test that delegating to a validator also delegates to an empty provider
func TestDelegateToValidator(t *testing.T) {
	ts := newTester(t)
	ts.addValidators(1)

	validator, _ := ts.GetAccount(common.VALIDATOR, 0)

	amount := sdk.NewIntFromUint64(100)
	ts.TxCreateValidator(validator, amount)

	res, err := ts.QueryDualstakingProviderDelegators(commontypes.EMPTY_PROVIDER, true)
	require.NoError(t, err)
	require.Equal(t, res.Delegations[0].Delegator, validator.Addr.String())

	ts.addClients(1)
	delegator, _ := ts.GetAccount(common.CONSUMER, 0)
	_, err = ts.TxDelegateValidator(delegator, validator, amount)
	require.NoError(t, err)

	res, err = ts.QueryDualstakingProviderDelegators(commontypes.EMPTY_PROVIDER, true)
	require.NoError(t, err)
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
	ts.TxCreateValidator(validator1, amount)
	ts.TxCreateValidator(validator2, amount)

	delegatorsRes, err := ts.QueryDualstakingProviderDelegators(commontypes.EMPTY_PROVIDER, true)
	require.NoError(t, err)
	require.Equal(t, 2, len(delegatorsRes.Delegations))

	ts.addClients(1)
	delegator, _ := ts.GetAccount(common.CONSUMER, 0)
	_, err = ts.TxDelegateValidator(delegator, validator1, amount)
	require.NoError(t, err)

	delegatorsRes, err = ts.QueryDualstakingProviderDelegators(commontypes.EMPTY_PROVIDER, true)
	require.NoError(t, err)
	require.Equal(t, 3, len(delegatorsRes.Delegations))

	providersRes, err := ts.QueryDualstakingDelegatorProviders(delegator.Addr.String(), true)
	require.NoError(t, err)
	require.Equal(t, 1, len(providersRes.Delegations))
	require.Equal(t, delegator.Addr.String(), providersRes.Delegations[0].Delegator)

	_, err = ts.TxReDelegateValidator(delegator, validator1, validator2, amount)
	require.NoError(t, err)

	delegatorsRes1, err := ts.QueryDualstakingProviderDelegators(commontypes.EMPTY_PROVIDER, true)
	require.NoError(t, err)
	require.Equal(t, delegatorsRes, delegatorsRes1)

	providersRes1, err := ts.QueryDualstakingDelegatorProviders(delegator.Addr.String(), true)
	require.NoError(t, err)
	require.Equal(t, providersRes, providersRes1)
}

// test that redelegating to a validator does not affect the delegator state
func TestReDelegateToProvider(t *testing.T) {
	ts := newTester(t)
	ts.addValidators(1)
	err := ts.addProviders(1)
	require.NoError(t, err)
	ts.addClients(1)

	validator, _ := ts.GetAccount(common.VALIDATOR, 0)
	amount := sdk.NewIntFromUint64(10000)
	ts.TxCreateValidator(validator, amount)

	acc, provider := ts.GetAccount(common.PROVIDER, 0)
	err = ts.StakeProvider(acc.GetVaultAddr(), provider, ts.spec, amount.Int64())
	require.NoError(t, err)

	ts.AdvanceEpoch()

	delegator, _ := ts.GetAccount(common.CONSUMER, 0)
	_, err = ts.TxDelegateValidator(delegator, validator, amount)
	require.NoError(t, err)

	epoch := ts.EpochStart()
	entry, found := ts.Keepers.Epochstorage.GetStakeEntryForProviderEpoch(ts.Ctx, ts.spec.Index, provider, epoch)
	require.True(t, found)
	require.Equal(t, amount, entry.Stake.Amount)

	providersRes, err := ts.QueryDualstakingDelegatorProviders(delegator.Addr.String(), true)
	require.NoError(t, err)
	require.Equal(t, 1, len(providersRes.Delegations))
	require.Equal(t, commontypes.EMPTY_PROVIDER, providersRes.Delegations[0].Provider)

	ts.AdvanceEpoch()

	_, err = ts.TxDualstakingRedelegate(delegator.Addr.String(),
		commontypes.EMPTY_PROVIDER,
		provider,
		commontypes.EMPTY_PROVIDER_CHAINID,
		entry.Chain,
		sdk.NewCoin(ts.TokenDenom(), amount))

	require.NoError(t, err)

	providersRes1, err := ts.QueryDualstakingDelegatorProviders(delegator.Addr.String(), false)
	require.NoError(t, err)
	require.Equal(t, providersRes, providersRes1)

	providersRes1, err = ts.QueryDualstakingDelegatorProviders(delegator.Addr.String(), true)
	require.NoError(t, err)
	require.Equal(t, provider, providersRes1.Delegations[0].Provider)

	ts.AdvanceEpoch()

	epoch = ts.EpochStart()
	entry, found = ts.Keepers.Epochstorage.GetStakeEntryForProviderEpoch(ts.Ctx, ts.spec.Index, provider, epoch)
	require.True(t, found)
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
	err := ts.addProviders(5)
	require.NoError(t, err)
	ts.addClients(1)

	// create validator and providers
	validator, _ := ts.GetAccount(common.VALIDATOR, 0)
	amount := sdk.NewIntFromUint64(10000)
	ts.TxCreateValidator(validator, amount)

	for i := 0; i < 5; i++ {
		provider, _ := ts.GetAccount(common.PROVIDER, i)
		err := ts.StakeProvider(provider.GetVaultAddr(), provider.Addr.String(), ts.spec, amount.Int64())
		require.NoError(t, err)
	}

	ts.AdvanceEpoch()

	// delegate to validator (automatically delegates to empty provider)
	delegatorAcc, delegator := ts.GetAccount(common.CONSUMER, 0)
	_, err = ts.TxDelegateValidator(delegatorAcc, validator, sdk.NewInt(210))
	require.NoError(t, err)

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
			commontypes.EMPTY_PROVIDER,
			provider,
			commontypes.EMPTY_PROVIDER_CHAINID,
			ts.spec.Index,
			sdk.NewCoin(ts.TokenDenom(), redelegateAmts[i]))
		require.NoError(t, err)
	}

	// unbond 25*5 tokens from validator
	_, err = ts.TxUnbondValidator(delegatorAcc, validator, sdk.NewInt(25*5))
	require.NoError(t, err)

	res, err := ts.QueryDualstakingDelegatorProviders(delegator, true)
	require.Len(t, res.Delegations, 3)
	require.NoError(t, err)
	for _, d := range res.Delegations {
		switch d.Provider {
		case providers[2]:
			require.Equal(t, int64(19), d.Amount.Amount.Int64())
		case providers[3]:
			require.Equal(t, int64(28), d.Amount.Amount.Int64())
		case providers[4]:
			require.Equal(t, int64(38), d.Amount.Amount.Int64()) // highest delegation is decreased by uniform amount + remainder
		default:
			require.FailNow(t, "unexpected provider in delegations")
		}
	}

	diff, _, err := ts.Keepers.Dualstaking.VerifyDelegatorBalance(ts.Ctx, delegatorAcc.Addr)
	require.NoError(t, err)
	require.True(t, diff.IsZero())
}

func TestValidatorSlash(t *testing.T) {
	ts := newTester(t)
	_, _ = ts.AddAccount(common.VALIDATOR, 0, testBalance*1000000000)
	amount := sdk.NewIntFromUint64(1000000000)

	// create valAcc and providers
	valAcc, _ := ts.GetAccount(common.VALIDATOR, 0)
	ts.TxCreateValidator(valAcc, amount)
	ts.AdvanceEpoch()

	// sanity check: validator should have 1000000000 tokens
	val := ts.GetValidator(valAcc.Addr)
	require.Equal(t, amount, val.Tokens)

	// sanity check: empty provider should have delegation of 1000000000 tokens
	resQ, err := ts.QueryDualstakingProviderDelegators(commontypes.EMPTY_PROVIDER, false)
	require.NoError(t, err)
	require.Equal(t, 1, len(resQ.Delegations))
	require.Equal(t, amount, resQ.Delegations[0].Amount.Amount)

	// slash 0.6*ConsensusPowerTokens = 0.6*100000 from the validator and check balance
	expectedTokensToBurn := ts.SlashValidator(valAcc, sdk.NewDecWithPrec(6, 1), 1, ts.Ctx.BlockHeight()) // fraction = 0.6

	// sanity check: validator should have 0.6*ConsensusPowerTokens = 0.6*100000
	val = ts.GetValidator(valAcc.Addr)
	require.Equal(t, amount.Sub(expectedTokensToBurn), val.Tokens)

	// check: the only delegation should be validator delegated to empty provider
	// the delegation amount should be original_amount(=1000000000) - expectedTokensToBurn
	res, err := ts.QueryDualstakingDelegatorProviders(valAcc.Addr.String(), true)
	require.NoError(t, err)
	require.Equal(t, 1, len(res.Delegations)) // empty provider
	require.Equal(t, amount.Sub(expectedTokensToBurn), res.Delegations[0].Amount.Amount)

	// sanity check: verify that provider-validator delegations are equal
	diff, _, err := ts.Keepers.Dualstaking.VerifyDelegatorBalance(ts.Ctx, valAcc.Addr)
	require.NoError(t, err)
	require.True(t, diff.IsZero())
}

// TestValidatorAndProvidersSlash checks that after a validator gets slashed, the delegators' providers
// get slashed as expected (by an equal amount to preserve the validators-providers delegation balance)
func TestValidatorAndProvidersSlash(t *testing.T) {
	ts := newTester(t)
	ts.addValidators(1)
	err := ts.addProviders(5)
	require.NoError(t, err)
	_, _ = ts.AddAccount(common.CONSUMER, 0, testBalance*1000000000)

	power := int64(1)
	consensusPowerTokens := ts.Keepers.StakingKeeper.TokensFromConsensusPower(ts.Ctx, power)
	stake := consensusPowerTokens.MulRaw(10)

	// create valAcc and providers
	valAcc, _ := ts.GetAccount(common.VALIDATOR, 0)
	ts.TxCreateValidator(valAcc, stake)

	providersAccs := []sigs.Account{}
	for i := 0; i < 5; i++ {
		provider, _ := ts.GetAccount(common.PROVIDER, i)
		err := ts.StakeProvider(provider.GetVaultAddr(), provider.Addr.String(), ts.spec, stake.Int64())
		require.NoError(t, err)
		providersAccs = append(providersAccs, provider)
	}
	ts.AdvanceEpoch()

	// sanity check: validator should have 1000000000*6 tokens (initial stake + 5 provider stakes via dualstaking)
	expectedValidatorTokens := stake.MulRaw(6)
	val := ts.GetValidator(valAcc.Addr)
	require.Equal(t, expectedValidatorTokens, val.Tokens)

	// delegate to validator (automatically delegates to empty provider)
	delegatorAcc, delegator := ts.GetAccount(common.CONSUMER, 0)
	_, err = ts.TxDelegateValidator(delegatorAcc, valAcc, consensusPowerTokens.MulRaw(250))
	require.NoError(t, err)
	delegatorValDelegations := ts.Keepers.StakingKeeper.GetAllDelegatorDelegations(ts.Ctx, delegatorAcc.Addr)
	require.Equal(t, 1, len(delegatorValDelegations))
	val = ts.GetValidator(valAcc.Addr)
	require.Equal(t, consensusPowerTokens.MulRaw(250), val.TokensFromShares(delegatorValDelegations[0].Shares).TruncateInt())
	expectedValidatorTokens = expectedValidatorTokens.Add(consensusPowerTokens.MulRaw(250))
	ts.AdvanceEpoch() // advance epoch to apply the empty provider delegation (that happens automatically when delegating to the validator)

	// sanity check: empty provider should have val_stake(=1000000000) + 250*consensusPowerTokens tokens in two delegations
	resQ, err := ts.QueryDualstakingProviderDelegators(commontypes.EMPTY_PROVIDER, false)
	require.NoError(t, err)
	require.Equal(t, 2, len(resQ.Delegations))
	require.Equal(t, stake.Add(consensusPowerTokens.MulRaw(250)), resQ.Delegations[0].Amount.Amount.Add(resQ.Delegations[1].Amount.Amount))

	// redelegate all the empty provider's funds to all providers with fixed amounts
	redelegateAmts := []math.Int{
		consensusPowerTokens.MulRaw(15),
		consensusPowerTokens.MulRaw(15),
		consensusPowerTokens.MulRaw(55),
		consensusPowerTokens.MulRaw(60),
		consensusPowerTokens.MulRaw(105),
	}
	var providers []string
	for i := 0; i < 5; i++ {
		_, provider := ts.GetAccount(common.PROVIDER, i)
		providers = append(providers, provider)

		_, err = ts.TxDualstakingRedelegate(delegatorAcc.Addr.String(),
			commontypes.EMPTY_PROVIDER,
			provider,
			commontypes.EMPTY_PROVIDER_CHAINID,
			ts.spec.Index,
			sdk.NewCoin(ts.TokenDenom(), redelegateAmts[i]))
		require.NoError(t, err)
		ts.AdvanceEpoch()

		// verify delegation is applied (should be 2 delegations: self delegation + redelegate amount)
		resQ, err = ts.QueryDualstakingProviderDelegators(provider, false)
		require.NoError(t, err)
		require.Equal(t, 2, len(resQ.Delegations))
		require.Equal(t, redelegateAmts[i].Add(stake), resQ.Delegations[0].Amount.Amount.Add(resQ.Delegations[1].Amount.Amount))
	}

	// since we emptied empty_provider, we wait until its delegation entry in the fixation
	// store is actually deleted (and not just marked for deletion)
	ts.AdvanceBlockUntilStale()

	// sanity check: redelegate from provider0 to provider1 and check delegations balance
	_, err = ts.TxDualstakingRedelegate(delegator, providers[0], providers[1], ts.spec.Index, ts.spec.Index, sdk.NewCoin(ts.TokenDenom(), consensusPowerTokens.MulRaw(5)))
	require.NoError(t, err)
	ts.AdvanceEpoch() // apply redelegation
	diff, _, err := ts.Keepers.Dualstaking.VerifyDelegatorBalance(ts.Ctx, delegatorAcc.Addr)
	require.NoError(t, err)
	require.True(t, diff.IsZero())

	// sanity check: unbond some of provider2's funds and check delegations balance
	_, err = ts.TxDualstakingUnbond(delegator, providers[2], ts.spec.Index, sdk.NewCoin(ts.TokenDenom(), consensusPowerTokens.MulRaw(5)))
	require.NoError(t, err)
	ts.AdvanceEpoch() // apply unbond
	diff, _, err = ts.Keepers.Dualstaking.VerifyDelegatorBalance(ts.Ctx, delegatorAcc.Addr)
	require.NoError(t, err)
	require.True(t, diff.IsZero())
	expectedValidatorTokens = expectedValidatorTokens.Sub(consensusPowerTokens.MulRaw(5))

	// get the delegator's provider delegations before the slash
	res, err := ts.QueryDualstakingDelegatorProviders(delegator, true)
	require.NoError(t, err)
	delegationsBeforeSlash := res.Delegations
	slices.SortFunc(delegationsBeforeSlash, func(i, j dualstakingtypes.Delegation) bool {
		return i.Amount.IsLT(j.Amount)
	})

	// slash consensusPowerTokens*0.6 tokens from the validator and check balance
	expectedTokensSlashed := ts.SlashValidator(valAcc, sdk.NewDecWithPrec(6, 1), power, ts.Ctx.BlockHeight()) // fraction = 0.6
	expectedValidatorTokens = expectedValidatorTokens.Sub(expectedTokensSlashed)
	val = ts.GetValidator(valAcc.Addr)
	require.Equal(t, expectedValidatorTokens, val.Tokens)

	// hard coded effective fraction of slash
	fraction := sdk.MustNewDecFromStr("0.001967213114754099")

	// both the validator and providers have a single delegation that was created by their
	// self delegation. Check that the new amount after slash is (1-fraction) * old_amount
	res, err = ts.QueryDualstakingDelegatorProviders(valAcc.Addr.String(), true)
	require.NoError(t, err)
	require.Len(t, res.Delegations, 1)
	require.Equal(t, sdk.OneDec().Sub(fraction).MulInt(stake).RoundInt(), res.Delegations[0].Amount.Amount)

	for _, p := range providersAccs {
		res, err = ts.QueryDualstakingDelegatorProviders(p.GetVaultAddr(), true)
		require.NoError(t, err)
		require.Len(t, res.Delegations, 1)
		require.Equal(t, sdk.OneDec().Sub(fraction).MulInt(stake).RoundInt(), res.Delegations[0].Amount.Amount)
	}

	// the total token to deduct from the delegator's provider delegations is:
	// total_providers_delegations * fraction = (245 * consensus_power_tokens) * fraction
	res, err = ts.QueryDualstakingDelegatorProviders(delegator, true)
	require.NoError(t, err)
	require.Len(t, res.Delegations, 5) // 5 providers from redelegations
	totalDelegations := math.ZeroInt()
	for _, d := range res.Delegations {
		totalDelegations = totalDelegations.Add(d.Amount.Amount)
	}
	require.Equal(t, sdk.OneDec().Sub(fraction).MulInt(consensusPowerTokens.MulRaw(245)).RoundInt(), totalDelegations)

	// verify once again that the delegator's delegations balance is preserved
	diff, _, err = ts.Keepers.Dualstaking.VerifyDelegatorBalance(ts.Ctx, delegatorAcc.Addr)
	require.NoError(t, err)
	require.True(t, diff.IsZero())
}

// TestCancelUnbond checks that the providers-validators delegations balance is preserved when
// a delegator (to a validator) cancels its unbond request
func TestCancelUnbond(t *testing.T) {
	ts := newTester(t)
	ts.addValidators(1)
	ts.addClients(1)
	amount := sdk.NewIntFromUint64(10000)

	// create validator and providers
	validator, _ := ts.GetAccount(common.VALIDATOR, 0)
	ts.TxCreateValidator(validator, amount)

	ts.AdvanceEpoch()

	// delegate to validator (automatically delegates to empty provider)
	delegator, _ := ts.GetAccount(common.CONSUMER, 0)
	_, err := ts.TxDelegateValidator(delegator, validator, sdk.NewInt(250))
	require.NoError(t, err)

	// unbond and advance blocks
	_, err = ts.TxUnbondValidator(delegator, validator, sdk.NewInt(250))
	require.NoError(t, err)
	ts.verifyDelegatorsBalance()

	unbondBlock := ts.Ctx.BlockHeight()
	ts.AdvanceEpoch()

	// cancel the unbond TX and check for balances
	_, err = ts.TxCancelUnbondValidator(delegator, validator, unbondBlock, sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(50)))
	require.NoError(t, err)

	diff, _, err := ts.Keepers.Dualstaking.VerifyDelegatorBalance(ts.Ctx, delegator.Addr)
	require.NoError(t, err)
	require.True(t, diff.IsZero())
}

// TestHooksRandomDelegations creates lots of random delegations through the dualstaking module
// the goal is to verify that all redelegations that are triggered from dualstaking delegation TX
// succeed
func TestHooksRandomDelegations(t *testing.T) {
	ts := newTester(t)
	_, _ = ts.AddAccount(common.VALIDATOR, 0, testBalance*1000000000)
	_, _ = ts.AddAccount(common.PROVIDER, 0, testBalance*1000000000)
	_, _ = ts.AddAccount(common.CONSUMER, 0, testBalance*1000000000)
	amount := sdk.NewIntFromUint64(1000)

	// create validatorAcc and providers
	validatorAcc, _ := ts.GetAccount(common.VALIDATOR, 0)
	ts.TxCreateValidator(validatorAcc, amount)

	providerAcc, provider := ts.GetAccount(common.PROVIDER, 0)
	err := ts.StakeProvider(providerAcc.GetVaultAddr(), providerAcc.Addr.String(), ts.spec, amount.Int64())
	require.NoError(t, err)

	ts.AdvanceEpoch()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	delegations := r.Perm(int(amount.Int64())) // array of 1000 elements with random number between [0,10000)

	prevDelegatorAcc, prevDelegator := ts.GetAccount(common.CONSUMER, 0)

	for i, d := range delegations {
		_, _ = ts.AddAccount(common.CONSUMER, i+1, testBalance*1000000000)
		delegatorAcc, delegator := ts.GetAccount(common.CONSUMER, i+1)
		d += 1
		d *= 1000000037 // avoid delegating zero
		if d%2 == 0 {
			delegatorAcc = prevDelegatorAcc
			delegator = prevDelegator
		}
		_, err := ts.TxDualstakingDelegate(delegator, provider, ts.spec.Index, sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(int64(d))))
		require.NoError(t, err)

		_, found := ts.Keepers.StakingKeeper.GetDelegation(ts.Ctx, delegatorAcc.Addr, sdk.ValAddress(validatorAcc.Addr))
		require.True(t, found)

		valConsAddr := sdk.GetConsAddress(validatorAcc.ConsKey.PubKey())
		ts.Keepers.SlashingKeeper.Slash(ts.Ctx, valConsAddr, sdk.NewDecWithPrec(1, 1), 1, ts.Ctx.BlockHeight())
	}
}

// TestNotRoundedShares checks that the delegate TX works in a specific case in which it
// failed in the past due to not-rounded shares value
func TestNotRoundedShares(t *testing.T) {
	ts := newTester(t)
	_, _ = ts.AddAccount(common.VALIDATOR, 0, testBalance*10000000000)
	_, _ = ts.AddAccount(common.PROVIDER, 0, testBalance*10000000000)
	_, _ = ts.AddAccount(common.CONSUMER, 0, testBalance*10000000000)
	delAmount := sdk.NewIntFromUint64(1000000000000)

	delegatorAcc, delegator := ts.GetAccount(common.CONSUMER, 0)

	validatorAcc, _ := ts.GetAccount(common.VALIDATOR, 0)
	ts.TxCreateValidator(validatorAcc, math.NewIntFromUint64(4495000000001))

	val, found := ts.Keepers.StakingKeeper.GetValidator(ts.Ctx, sdk.ValAddress(validatorAcc.Addr))
	require.True(t, found)
	val.DelegatorShares = sdk.MustNewDecFromStr("4540404040405.050505050505050505")
	ts.Keepers.StakingKeeper.SetValidator(ts.Ctx, val)

	providerAcc, provider := ts.GetAccount(common.PROVIDER, 0)
	err := ts.StakeProvider(providerAcc.GetVaultAddr(), providerAcc.Addr.String(), ts.spec, delAmount.Int64())
	require.NoError(t, err)

	shares := sdk.MustNewDecFromStr("1010101010101.010101010101010101")
	require.NoError(t, err)
	ts.Keepers.StakingKeeper.SetDelegation(ts.Ctx, stakingtypes.NewDelegation(delegatorAcc.Addr, sdk.ValAddress(validatorAcc.Addr), shares))

	_, err = ts.TxDualstakingDelegate(delegator, provider, ts.spec.Index, sdk.NewCoin(ts.TokenDenom(), delAmount))
	require.NoError(t, err)
}

func TestUnbondValidatorButNotRemoveStakeEntry(t *testing.T) {
	ts := newTester(t)
	ts.addValidators(2)
	err := ts.addProviders(5)
	require.NoError(t, err)
	ts.addClients(2)

	// create validator and providers
	validator, _ := ts.GetAccount(common.VALIDATOR, 0)
	amount := sdk.NewIntFromUint64(9999)
	ts.TxCreateValidator(validator, amount)

	validator2, _ := ts.GetAccount(common.VALIDATOR, 1)
	amount2 := sdk.NewIntFromUint64(9998)
	ts.TxCreateValidator(validator2, amount2)

	delegatorAcc1, _ := ts.GetAccount(common.CONSUMER, 0)
	_, err = ts.TxDelegateValidator(delegatorAcc1, validator, sdk.NewInt(9999))
	require.NoError(t, err)

	delegatorAcc2, _ := ts.GetAccount(common.CONSUMER, 0)
	_, err = ts.TxDelegateValidator(delegatorAcc2, validator, sdk.NewInt(9998))
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		provider, _ := ts.GetAccount(common.PROVIDER, i)
		err := ts.StakeProvider(provider.GetVaultAddr(), provider.Addr.String(), ts.spec, sdk.NewIntFromUint64(9999).Int64())
		require.NoError(t, err)
	}

	providerAcct, provider := ts.GetAccount(common.PROVIDER, 0)

	// provider completely unbond from validator, delegation is removed
	_, err = ts.TxUnbondValidator(*providerAcct.Vault, validator, sdk.NewInt(9999))
	require.NoError(t, err)

	// other delegator should not be able to delegate to the provider
	_, err = ts.TxDualstakingRedelegate(delegatorAcc1.Addr.String(),
		commontypes.EMPTY_PROVIDER,
		provider,
		commontypes.EMPTY_PROVIDER_CHAINID,
		ts.spec.Index,
		sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(9999)))
	require.Error(t, err)

	// checking that provider is not found
	_, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, provider)
	require.False(t, found)

	_, err = ts.QueryDualstakingProviderDelegators(provider, true)
	require.NoError(t, err)
}

// TestUndelegateProvider checks for a bug that when a provider unstakes, its delegations are not
// transferred to the empty provider. If the bug persists, this unit test fails
func TestUndelegateProvider(t *testing.T) {
	ts := newTester(t)
	ts.addValidators(2)
	err := ts.addProviders(5)
	require.NoError(t, err)
	ts.addClients(2)

	// create validator and providers
	validator, _ := ts.GetAccount(common.VALIDATOR, 0)
	amount := sdk.NewIntFromUint64(9999)
	ts.TxCreateValidator(validator, amount)

	validator2, _ := ts.GetAccount(common.VALIDATOR, 1)
	amount2 := sdk.NewIntFromUint64(9998)
	ts.TxCreateValidator(validator2, amount2)

	delegatorAcc1, _ := ts.GetAccount(common.CONSUMER, 0)
	_, err = ts.TxDelegateValidator(delegatorAcc1, validator, sdk.NewInt(9999))
	require.NoError(t, err)

	delegatorAcc2, _ := ts.GetAccount(common.CONSUMER, 1)
	_, err = ts.TxDelegateValidator(delegatorAcc2, validator, sdk.NewInt(9998))
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		provider, _ := ts.GetAccount(common.PROVIDER, i)
		err := ts.StakeProvider(provider.GetVaultAddr(), provider.Addr.String(), ts.spec, amount.Int64())
		require.NoError(t, err)
	}

	providerAcct, provider := ts.GetAccount(common.PROVIDER, 0)

	utils.LavaFormatInfo("addresses:\n",
		utils.LogAttr("vault", providerAcct.GetVaultAddr()),
		utils.LogAttr("provider", provider),
		utils.LogAttr("delegator1", delegatorAcc1.Addr.String()),
	)

	// delegator1 redelegates 9999 to the provider
	_, err = ts.TxDualstakingRedelegate(delegatorAcc1.Addr.String(),
		commontypes.EMPTY_PROVIDER,
		provider,
		commontypes.EMPTY_PROVIDER_CHAINID,
		ts.spec.Index,
		sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(9999)))
	require.NoError(t, err)

	ts.AdvanceEpoch()

	res2, err := ts.QueryDualstakingProviderDelegators(provider, true)
	require.NoError(t, err)
	fmt.Println("Delegation of Provider before provider is removed", res2)

	unstakeHoldBlocks := ts.Keepers.Epochstorage.UnstakeHoldBlocks(ts.Ctx, ts.BlockHeight())
	unstakeHoldBlocksStatic := ts.Keepers.Epochstorage.UnstakeHoldBlocksStatic(ts.Ctx, ts.BlockHeight())

	_, err = ts.TxPairingUnstakeProvider(providerAcct.GetVaultAddr(), ts.spec.Index)
	require.NoError(t, err)

	ts.AdvanceBlocks(unstakeHoldBlocks)

	_, found := ts.Keepers.Epochstorage.UnstakeEntryByAddress(ts.Ctx, providerAcct.GetVaultAddr())
	require.True(t, found)

	ts.AdvanceBlocks(unstakeHoldBlocksStatic - unstakeHoldBlocks)

	// checking that provider can't be found
	_, found = ts.Keepers.Epochstorage.UnstakeEntryByAddress(ts.Ctx, providerAcct.GetVaultAddr())
	require.False(t, found)

	ts.AdvanceEpoch()

	_, found = ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, provider)
	require.False(t, found)

	// delegation of the removed provider
	// the provider is removed but the delegation is still remained
	res2, err = ts.QueryDualstakingProviderDelegators(provider, true)
	require.NoError(t, err)
	fmt.Println("Delegation of Provider after provider is removed", res2)

	// stake provider again
	err = ts.StakeProvider(providerAcct.GetVaultAddr(), providerAcct.Addr.String(), ts.spec, sdk.NewIntFromUint64(1000).Int64())
	require.NoError(t, err)

	stakeEntry, found := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, provider)
	require.True(t, found)
	fmt.Println("Stake entry of re-staked provider", stakeEntry.String())

	// delegator1 should be able to redelegate back to the empty provider
	_, err = ts.TxDualstakingRedelegate(delegatorAcc1.Addr.String(),
		provider,
		commontypes.EMPTY_PROVIDER,
		ts.spec.Index,
		commontypes.EMPTY_PROVIDER_CHAINID,
		sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(1)))
	require.NoError(t, err)

	stakeEntry, found = ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Index, provider)
	require.True(t, found)
	fmt.Println("Stake entry of re-staked provider after del1 9999 redelegation", stakeEntry.String())

	// another one delegates to provider
	// delegator2 delegates 9998 to the provider
	_, err = ts.TxDualstakingRedelegate(delegatorAcc2.Addr.String(),
		commontypes.EMPTY_PROVIDER,
		provider,
		commontypes.EMPTY_PROVIDER_CHAINID,
		ts.spec.Index,
		sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(9998)))
	require.NoError(t, err)

	// delegator 1 can only redelegate 9998 to Empty Provider
	ts.AdvanceEpoch()

	_, err = ts.TxDualstakingRedelegate(delegatorAcc1.Addr.String(),
		provider,
		commontypes.EMPTY_PROVIDER,
		ts.spec.Index,
		commontypes.EMPTY_PROVIDER_CHAINID,
		sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(9999)))
	require.Error(t, err)

	_, err = ts.TxDualstakingRedelegate(delegatorAcc1.Addr.String(),
		provider,
		commontypes.EMPTY_PROVIDER,
		ts.spec.Index,
		commontypes.EMPTY_PROVIDER_CHAINID,
		sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(9998)))
	require.NoError(t, err)
}
