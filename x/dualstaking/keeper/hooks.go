package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// Wrapper struct
type Hooks struct {
	k Keeper
}

var _ stakingtypes.StakingHooks = Hooks{}

// Create new dualstaking hooks
func (k Keeper) Hooks() *Hooks {
	return &Hooks{k}
}

// initialize validator distribution record
func (h Hooks) AfterValidatorCreated(ctx sdk.Context, valAddr sdk.ValAddress) error {
	return nil
}

// AfterValidatorRemoved performs clean up after a validator is removed
func (h Hooks) AfterValidatorRemoved(ctx sdk.Context, _ sdk.ConsAddress, valAddr sdk.ValAddress) error {
	return nil
}

// increment period
func (h Hooks) BeforeDelegationCreated(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	return nil
}

// withdraw delegation rewards (which also increments period)
func (h Hooks) BeforeDelegationSharesModified(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	return nil
}

// create new delegation period record
func (h Hooks) AfterDelegationModified(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	providers, err := h.k.GetDelegatorProviders(ctx, delAddr.String(), uint64(ctx.BlockHeight()))
	_ = err

	sumProviderDelegations := sdk.ZeroInt()
	for _, p := range providers {
		delegations := h.k.GetAllProviderDelegatorDelegations(ctx, delAddr.String(), p, uint64(ctx.BlockHeight()))
		for _, d := range delegations {
			sumProviderDelegations = sumProviderDelegations.Add(d.Amount.Amount)
		}
	}

	sumValidatorDelegations := sdk.ZeroInt()
	delegations := h.k.stakingKeeper.GetAllDelegatorDelegations(ctx, delAddr)
	for _, d := range delegations {
		validatorAddr, err := sdk.ValAddressFromBech32(d.ValidatorAddress)
		if err != nil {
			panic(err) // shouldn't happen
		}
		v, found := h.k.stakingKeeper.GetValidator(ctx, validatorAddr)
		_ = found
		sumValidatorDelegations = sumValidatorDelegations.Add(v.TokensFromShares(d.Shares).TruncateInt())
	}

	if sumProviderDelegations.Equal(sumValidatorDelegations) {
		// do nothing, this is a redelegate
		_ = ""
	} else if sumProviderDelegations.LT(sumValidatorDelegations) {
		// less provider delegations,a delegation operation was done, delegate to empty provider
		amount := sumValidatorDelegations.Sub(sumProviderDelegations)
		h.k.Delegate(ctx, delAddr.String(), EMPTY_PROVIDER, "", sdk.NewCoin("ulava", amount))
	} else if sumProviderDelegations.GT(sumValidatorDelegations) {
		// more provider delegation, unbond operation was done, unbond from providers
		_ = ""
	}

	return nil
}

// record the slash event
func (h Hooks) BeforeValidatorSlashed(ctx sdk.Context, valAddr sdk.ValAddress, fraction sdk.Dec) error {
	return nil
}

func (h Hooks) BeforeValidatorModified(_ sdk.Context, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorBonded(_ sdk.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorBeginUnbonding(_ sdk.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeDelegationRemoved(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterUnbondingInitiated(_ sdk.Context, _ uint64) error {
	return nil
}
