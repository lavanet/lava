package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/dualstaking/types"
	"golang.org/x/exp/slices"
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
// add description
func (h Hooks) AfterDelegationModified(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	if DisableDualstakingHook {
		return nil
	}

	diff, err := h.k.VerifyDelegatorBalance(ctx, delAddr)
	if err != nil {
		return err
	}

	// if diff is zero, do nothing, this is a redelegate
	if diff.IsZero() {
		return nil
	} else if diff.IsPositive() {
		// less provider delegations,a delegation operation was done, delegate to empty provider
		err = h.k.delegate(ctx, delAddr.String(), types.EMPTY_PROVIDER, types.EMPTY_PROVIDER_CHAINID,
			sdk.NewCoin(h.k.stakingKeeper.BondDenom(ctx), diff))
		if err != nil {
			return err
		}
	} else if diff.IsNegative() {
		// more provider delegation, unbond operation was done, unbond from providers
		err = h.k.UnbondUniformProviders(ctx, delAddr.String(), sdk.NewCoin(h.k.stakingKeeper.BondDenom(ctx), diff.Neg()))
		if err != nil {
			return err
		}
	}

	diff, err = h.k.VerifyDelegatorBalance(ctx, delAddr)
	if err != nil {
		return err
	}
	// now it needs to be zero
	if !diff.IsZero() {
		return utils.LavaFormatError("validator and provider balances are not balanced", nil,
			utils.Attribute{Key: "delegator", Value: delAddr.String()},
			utils.Attribute{Key: "diff", Value: diff.String()},
		)
	}
	return nil
}

// BeforeValidatorSlashed hook unbonds funds from providers so the providers-validators delegations balance will preserve
func (h Hooks) BeforeValidatorSlashed(ctx sdk.Context, valAddr sdk.ValAddress, fraction sdk.Dec) error {
	val, found := h.k.stakingKeeper.GetValidator(ctx, valAddr)
	if !found {
		return utils.LavaFormatError("slash hook failed", fmt.Errorf("validator not found"),
			utils.Attribute{Key: "validator_address", Value: valAddr.String()},
		)
	}

	// unbond from providers according to slash
	// sort the delegations from lowest to highest so if there's a remainder,
	// remove it from the highest delegation in the last iteration
	remainingTokensToSlash := fraction.MulInt(val.Tokens).TruncateInt()
	delegations := h.k.stakingKeeper.GetValidatorDelegations(ctx, valAddr)
	slices.SortFunc(delegations, func(i, j stakingtypes.Delegation) bool {
		return val.TokensFromShares(i.Shares).LT(val.TokensFromShares(j.Shares))
	})
	for i, d := range delegations {
		tokens := val.TokensFromShares(d.Shares)
		tokensToSlash := fraction.Mul(tokens).TruncateInt()
		if i == len(delegations)-1 {
			tokensToSlash = remainingTokensToSlash
		}
		if tokensToSlash.IsPositive() {
			err := h.k.UnbondUniformProviders(ctx, d.DelegatorAddress, sdk.NewCoin(commontypes.TokenDenom, tokensToSlash))
			if err != nil {
				utils.LavaFormatError("slash hook failed", err,
					utils.Attribute{Key: "validator_address", Value: valAddr.String()},
					utils.Attribute{Key: "delegator_address", Value: d.DelegatorAddress},
					utils.Attribute{Key: "slash_amount", Value: tokensToSlash.String()},
				)
			}

			remainingTokensToSlash = remainingTokensToSlash.Sub(tokensToSlash)
		}
	}

	details := make(map[string]string)
	details["validator_address"] = valAddr.String()
	details["slash_fraction"] = fraction.String()

	utils.LogLavaEvent(ctx, h.k.Logger(ctx), types.ValidatorSlashEventName, details, "Validator slash hook event")
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

func (h Hooks) BeforeDelegationRemoved(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	if DisableDualstakingHook {
		return nil
	}

	delegation, found := h.k.stakingKeeper.GetDelegation(ctx, delAddr, valAddr)
	if !found {
		return fmt.Errorf("could not find delegation for dualstaking hook")
	}
	validator, err := h.k.stakingKeeper.GetDelegatorValidator(ctx, delAddr, valAddr)
	if err != nil {
		return utils.LavaFormatWarning("delegation removed hook failed", err,
			utils.LogAttr("validator_address", valAddr.String()),
			utils.LogAttr("delegator_address", delAddr.String()),
		)
	}
	amount := validator.TokensFromSharesRoundUp(delegation.Shares).Ceil().TruncateInt()
	err = h.k.UnbondUniformProviders(ctx, delAddr.String(), sdk.NewCoin(h.k.stakingKeeper.BondDenom(ctx), amount))
	if err != nil {
		return utils.LavaFormatError("delegation removed hook failed", err,
			utils.Attribute{Key: "validator_address", Value: valAddr.String()},
			utils.Attribute{Key: "delegator_address", Value: delAddr.String()},
			utils.Attribute{Key: "amount", Value: amount.String()},
		)
	}

	return nil
}

func (h Hooks) AfterUnbondingInitiated(_ sdk.Context, _ uint64) error {
	return nil
}
