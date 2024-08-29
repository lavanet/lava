package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/dualstaking/types"
)

// balance delegators dualstaking after potential validators slashing
func (k Keeper) HandleSlashedValidators(ctx sdk.Context) {
	slashedValidators := k.GetSlashedValidators(ctx)
	for _, validator := range slashedValidators {
		k.BalanceValidatorsDelegators(ctx, validator)
	}
	k.SetSlashedValidators(ctx, []string{})
}

func (k Keeper) BalanceValidatorsDelegators(ctx sdk.Context, validator string) {
	valAcc, err := sdk.ValAddressFromBech32(validator)
	if err != nil {
		utils.LavaFormatError("failed to get validator address", err, utils.LogAttr("validator", validator))
		return
	}

	delegators := k.stakingKeeper.GetValidatorDelegations(ctx, valAcc)
	for _, delegator := range delegators {
		delAddr := delegator.GetDelegatorAddr()
		k.BalanceDelegator(ctx, delAddr)
	}
}

// SetDisableDualstakingHook set disableDualstakingHook in the store
func (k Keeper) SetSlashedValidators(ctx sdk.Context, val []string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SlashedValidatorsPrefix))
	slashedValidators := types.SlashedValidators{Validators: val}
	b := k.cdc.MustMarshal(&slashedValidators)
	store.Set([]byte{0}, b)
}

// GetDisableDualstakingHook returns disableDualstakingHook
func (k Keeper) GetSlashedValidators(ctx sdk.Context) []string {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SlashedValidatorsPrefix))

	b := store.Get([]byte{0})
	if b == nil {
		return []string{}
	}

	var val types.SlashedValidators
	k.cdc.MustUnmarshal(b, &val)
	return val.Validators
}
