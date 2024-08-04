package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/dualstaking/types"
)

// Wrapper struct
type Hooks struct {
	k Keeper
}

const PROVIDERS_NUM_GAS_REFUND = 50

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
	originalGasMeter := ctx.GasMeter()
	ctx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	providers := 0
	defer func() {
		consumedGas := ctx.GasMeter().GasConsumed()
		ctx = ctx.WithGasMeter(originalGasMeter)
		if providers >= PROVIDERS_NUM_GAS_REFUND {
			ctx.GasMeter().ConsumeGas(consumedGas, "consume hooks gas")
		}
	}()

	if h.k.GetDisableDualstakingHook(ctx) {
		return nil
	}

	var err error
	providers, err = h.k.BalanceDelegator(ctx, delAddr)
	return err
}

func (h Hooks) BeforeValidatorSlashed(ctx sdk.Context, valAddr sdk.ValAddress, fraction sdk.Dec) error {
	slashedValidators := h.k.GetSlashedValidators(ctx)
	slashedValidators = append(slashedValidators, valAddr.String())
	h.k.SetSlashedValidators(ctx, slashedValidators)
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
	if h.k.GetDisableDualstakingHook(ctx) {
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

// DisableDualstakingHook : dualstaking uses hooks to catch delegations/unbonding tx's to do the same action on the providers delegations.
// in the case of redelegation, since the user doesnt put/takes tokens back we dont want to take action in the providers delegations.
// this flag is a local flag used to mark the next hooks to do nothing since this was cause by redelegation tx (redelegation = delegation + unbond)

// SetDisableDualstakingHook set disableDualstakingHook in the store
func (k Keeper) SetDisableDualstakingHook(ctx sdk.Context, val bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.DisableDualstakingHookPrefix))
	b := []byte{0}
	if val {
		b = []byte{1}
	}
	store.Set([]byte{0}, b)
}

// GetDisableDualstakingHook returns disableDualstakingHook
func (k Keeper) GetDisableDualstakingHook(ctx sdk.Context) bool {
	if k.storeKey == nil {
		return false
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.DisableDualstakingHookPrefix))

	b := store.Get([]byte{0})
	if b == nil {
		return false
	}

	return b[0] != 0
}
