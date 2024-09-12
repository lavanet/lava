package keeper

// Delegation allows securing funds for a specific provider to effectively increase
// its stake so it will be paired with consumers more often. The delegators do not
// transfer the funds to the provider but only bestow the funds with it. In return
// to locking the funds there, delegators get some of the provider’s profit (after
// commission deduction).
//
// The delegated funds are stored in the module's BondedPoolName account. On request
// to terminate the delegation, they are then moved to the modules NotBondedPoolName
// account, and remain locked there for staking.UnbondingTime() witholding period
// before finally released back to the delegator. The timers for bonded funds are
// tracked are indexed by the delegator, provider, and chainID.
//
// The delegation state is stores with fixation using two maps: one for delegations
// indexed by the combination <provider,chainD,delegator>, used to track delegations
// and find/access delegations by provider (and chainID); and another for delegators
// tracking the list of providers for a delegator, indexed by the delegator.

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v3/utils"
	commontypes "github.com/lavanet/lava/v3/utils/common/types"
	lavaslices "github.com/lavanet/lava/v3/utils/lavaslices"
	"github.com/lavanet/lava/v3/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/v3/x/epochstorage/types"
	"golang.org/x/exp/slices"
)

// increaseDelegation increases the delegation of a delegator to a provider for a
// given chain. It updates the fixation stores for both delegations and delegators,
// and updates the (epochstorage) stake-entry.
func (k Keeper) increaseDelegation(ctx sdk.Context, delegator, provider string, amount sdk.Coin) error {
	// get, update the delegation entry
	delegation, err := k.delegations.Get(ctx, types.DelegationKey(provider, delegator))
	if err != nil {
		// new delegation (i.e. not increase of existing one)
		delegation = types.NewDelegation(delegator, provider, ctx.BlockTime(), k.stakingKeeper.BondDenom(ctx))
	}

	delegation.AddAmount(amount)

	err = k.delegations.Set(ctx, types.DelegationKey(provider, delegator), delegation)
	if err != nil {
		return err
	}

	if provider != commontypes.EMPTY_PROVIDER {
		// update the stake entry
		return k.AfterDelegationModified(ctx, delegator, provider, amount, true)
	}

	return nil
}

// decreaseDelegation decreases the delegation of a delegator to a provider for a
// given chain. It updates the fixation stores for both delegations and delegators,
// and updates the (epochstorage) stake-entry.
func (k Keeper) decreaseDelegation(ctx sdk.Context, delegator, provider string, amount sdk.Coin) error {
	// get, update and append the delegation entry
	delegation, found := k.GetDelegation(ctx, provider, delegator)
	if !found {
		return types.ErrDelegationNotFound
	}

	if delegation.Amount.IsLT(amount) {
		return types.ErrInsufficientDelegation
	}

	delegation.SubAmount(amount)

	if delegation.Amount.IsZero() {
		err := k.RemoveDelegation(ctx, delegation)
		if err != nil {
			return err
		}
	} else {
		err := k.SetDelegation(ctx, delegation)
		if err != nil {
			return err
		}
	}

	if provider != commontypes.EMPTY_PROVIDER {
		return k.AfterDelegationModified(ctx, delegator, provider, amount, false)
	}

	return nil
}

func (k Keeper) AfterDelegationModified(ctx sdk.Context, delegator, provider string, amount sdk.Coin, increase bool) (err error) {
	// get all entries
	metadata, err := k.epochstorageKeeper.GetMetadata(ctx, provider)
	if err != nil {
		if increase {
			return err
		} else {
			// we want to allow decreasing if the provider does not exist
			return nil
		}
	}

	// check if self delegation
	selfdelegation := delegator == provider

	entries := []epochstoragetypes.StakeEntry{}
	for _, chain := range metadata.Chains {
		entry, found := k.epochstorageKeeper.GetStakeEntryCurrent(ctx, chain, provider)
		if !found {
			panic("AfterDelegationModified: entry not found ")
		}
		entries = append(entries, entry)
		if !selfdelegation && entry.Vault == delegator {
			selfdelegation = true
		}
	}

	if increase {
		if selfdelegation {
			metadata.SelfDelegation = metadata.SelfDelegation.Add(amount)
		} else {
			metadata.TotalDelegations = metadata.TotalDelegations.Add(amount)
		}
	} else {
		if selfdelegation {
			metadata.SelfDelegation, err = metadata.SelfDelegation.SafeSub(amount)
		} else {
			metadata.TotalDelegations, err = metadata.TotalDelegations.SafeSub(amount)
		}
	}
	if err != nil {
		return err
	}
	k.epochstorageKeeper.SetMetadata(ctx, metadata)

	details := map[string]string{
		"provider": provider,
	}

	for _, entry := range entries {
		details[entry.Chain] = entry.Chain
		if entry.Stake.IsLT(k.GetParams(ctx).MinSelfDelegation) {
			k.epochstorageKeeper.RemoveStakeEntryCurrent(ctx, entry.Chain, entry.Address)
			details["min_self_delegation"] = k.GetParams(ctx).MinSelfDelegation.String()
			utils.LogLavaEvent(ctx, k.Logger(ctx), types.UnstakeFromUnbond, details, "unstaking provider due to unbond that lowered its stake below min self delegation")
			continue
		}
		entry.DelegateTotal = sdk.NewCoin(amount.Denom, metadata.TotalDelegations.Amount.Mul(entry.Stake.Amount).Quo(metadata.SelfDelegation.Amount))
		if entry.TotalStake().LT(k.specKeeper.GetMinStake(ctx, entry.Chain).Amount) {
			details["min_spec_stake"] = k.specKeeper.GetMinStake(ctx, entry.Chain).String()
			utils.LogLavaEvent(ctx, k.Logger(ctx), types.FreezeFromUnbond, details, "freezing provider due to stake below min spec stake")
			entry.Freeze()

		} else if delegator == entry.Vault && entry.IsFrozen() && !entry.IsJailed(ctx.BlockTime().UTC().Unix()) {
			entry.UnFreeze(k.epochstorageKeeper.GetCurrentNextEpoch(ctx) + 1)
		}
		k.epochstorageKeeper.SetStakeEntryCurrent(ctx, entry)
	}

	return nil
}

// delegate lets a delegator delegate an amount of coins to a provider.
// (effective on next epoch)
func (k Keeper) delegate(ctx sdk.Context, delegator, provider string, amount sdk.Coin) error {
	_, err := sdk.AccAddressFromBech32(delegator)
	if err != nil {
		return utils.LavaFormatWarning("invalid delegator address", err,
			utils.Attribute{Key: "delegator", Value: delegator},
		)
	}

	if provider != commontypes.EMPTY_PROVIDER {
		if _, err = sdk.AccAddressFromBech32(provider); err != nil {
			return utils.LavaFormatWarning("invalid provider address", err,
				utils.Attribute{Key: "provider", Value: provider},
			)
		}
	}

	if err := utils.ValidateCoins(ctx, k.stakingKeeper.BondDenom(ctx), amount, false); err != nil {
		return utils.LavaFormatWarning("failed to delegate: coin validation failed", err,
			utils.Attribute{Key: "delegator", Value: delegator},
			utils.Attribute{Key: "provider", Value: provider},
		)
	}

	err = k.increaseDelegation(ctx, delegator, provider, amount)
	if err != nil {
		return utils.LavaFormatWarning("failed to increase delegation", err,
			utils.Attribute{Key: "delegator", Value: delegator},
			utils.Attribute{Key: "provider", Value: provider},
			utils.Attribute{Key: "amount", Value: amount.String()},
		)
	}

	return nil
}

// Redelegate lets a delegator transfer its delegation between providers, but
// without the funds being subject to unstakeHoldBlocks witholding period.
// (effective on next epoch)
func (k Keeper) Redelegate(ctx sdk.Context, delegator, from, to string, amount sdk.Coin) error {
	if _, err := sdk.AccAddressFromBech32(delegator); err != nil {
		return utils.LavaFormatWarning("invalid delegator address", err,
			utils.Attribute{Key: "delegator", Value: delegator},
		)
	}

	if from != commontypes.EMPTY_PROVIDER {
		if _, err := sdk.AccAddressFromBech32(from); err != nil {
			return utils.LavaFormatWarning("invalid from-provider address", err,
				utils.Attribute{Key: "from_provider", Value: from},
			)
		}
	}

	if to != commontypes.EMPTY_PROVIDER {
		if _, err := sdk.AccAddressFromBech32(to); err != nil {
			return utils.LavaFormatWarning("invalid to-provider address", err,
				utils.Attribute{Key: "to_provider", Value: to},
			)
		}
	}

	if err := utils.ValidateCoins(ctx, k.stakingKeeper.BondDenom(ctx), amount, false); err != nil {
		return utils.LavaFormatWarning("failed to redelegate: coin validation failed", err,
			utils.Attribute{Key: "delegator", Value: delegator},
			utils.Attribute{Key: "provider", Value: to},
		)
	}

	err := k.increaseDelegation(ctx, delegator, to, amount)
	if err != nil {
		return utils.LavaFormatWarning("failed to increase delegation", err,
			utils.Attribute{Key: "delegator", Value: delegator},
			utils.Attribute{Key: "provider", Value: to},
			utils.Attribute{Key: "amount", Value: amount.String()},
		)
	}

	err = k.decreaseDelegation(ctx, delegator, from, amount)
	if err != nil {
		return utils.LavaFormatWarning("failed to decrease delegation", err,
			utils.Attribute{Key: "delegator", Value: delegator},
			utils.Attribute{Key: "provider", Value: from},
			utils.Attribute{Key: "amount", Value: amount.String()},
		)
	}

	// no need to transfer funds, because they remain in the dualstaking module
	// (specifically in types.BondedPoolName).

	return nil
}

// unbond lets a delegator get its delegated coins back from a provider. The
// delegation ends immediately, but coins are held for unstakeHoldBlocks period
// before released and transferred back to the delegator. The rewards from the
// provider will be updated accordingly (or terminate) from the next epoch.
// (effective on next epoch)
func (k Keeper) unbond(ctx sdk.Context, delegator, provider string, amount sdk.Coin) error {
	if _, err := sdk.AccAddressFromBech32(delegator); err != nil {
		return utils.LavaFormatWarning("invalid delegator address", err,
			utils.Attribute{Key: "delegator", Value: delegator},
		)
	}

	if provider != commontypes.EMPTY_PROVIDER {
		if _, err := sdk.AccAddressFromBech32(provider); err != nil {
			return utils.LavaFormatWarning("invalid provider address", err,
				utils.Attribute{Key: "provider", Value: provider},
			)
		}
	}

	if err := utils.ValidateCoins(ctx, k.stakingKeeper.BondDenom(ctx), amount, false); err != nil {
		return utils.LavaFormatWarning("failed to unbond: coin validation failed", err,
			utils.Attribute{Key: "delegator", Value: delegator},
			utils.Attribute{Key: "provider", Value: provider},
		)
	}

	err := k.decreaseDelegation(ctx, delegator, provider, amount)
	if err != nil {
		return utils.LavaFormatWarning("failed to decrease delegation", err,
			utils.Attribute{Key: "delegator", Value: delegator},
			utils.Attribute{Key: "provider", Value: provider},
			utils.Attribute{Key: "amount", Value: amount.String()},
		)
	}

	return nil
}

// GetDelegatorProviders gets all the providers the delegator is delegated to
func (k Keeper) GetDelegatorProviders(ctx sdk.Context, delegator string) (providers []string, err error) {
	_, err = sdk.AccAddressFromBech32(delegator)
	if err != nil {
		return nil, utils.LavaFormatWarning("cannot get delegator's providers", err,
			utils.Attribute{Key: "delegator", Value: delegator},
		)
	}

	iter, err := k.delegations.Indexes.Number.MatchExact(ctx, delegator)
	if err != nil {
		return nil, err
	}

	for ; iter.Valid(); iter.Next() {
		key, err := iter.PrimaryKey()
		if err != nil {
			return nil, err
		}
		providers = append(providers, key.K1())
	}

	return providers, nil
}

func (k Keeper) GetProviderDelegators(ctx sdk.Context, provider string) ([]types.Delegation, error) {
	if provider != commontypes.EMPTY_PROVIDER {
		_, err := sdk.AccAddressFromBech32(provider)
		if err != nil {
			return nil, utils.LavaFormatWarning("cannot get provider's delegators", err,
				utils.Attribute{Key: "provider", Value: provider},
			)
		}
	}

	iter, err := k.delegations.Iterate(ctx, collections.NewPrefixedPairRange[string, string](provider))
	if err != nil {
		return nil, err
	}

	return iter.Values()
}

func (k Keeper) GetDelegation(ctx sdk.Context, provider, delegator string) (types.Delegation, bool) {
	delegation, err := k.delegations.Get(ctx, types.DelegationKey(provider, delegator))
	return delegation, err == nil
}

func (k Keeper) GetAllDelegations(ctx sdk.Context) ([]types.Delegation, error) {
	iter, err := k.delegations.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}

	return iter.Values()
}

func (k Keeper) SetDelegation(ctx sdk.Context, delegation types.Delegation) error {
	return k.delegations.Set(ctx, types.DelegationKey(delegation.Provider, delegation.Delegator), delegation)
}

func (k Keeper) RemoveDelegation(ctx sdk.Context, delegation types.Delegation) error {
	return k.delegations.Remove(ctx, types.DelegationKey(delegation.Provider, delegation.Delegator))
}

func (k Keeper) UnbondUniformProviders(ctx sdk.Context, delegator string, amount sdk.Coin) error {
	providers, err := k.GetDelegatorProviders(ctx, delegator)
	if err != nil {
		return err
	}

	// first remove from the empty provider
	if lavaslices.Contains[string](providers, commontypes.EMPTY_PROVIDER) {
		delegation, found := k.GetDelegation(ctx, commontypes.EMPTY_PROVIDER, delegator)
		if found {
			if delegation.Amount.Amount.GTE(amount.Amount) {
				// we have enough here, remove all from empty delegator and bail
				return k.unbond(ctx, delegator, commontypes.EMPTY_PROVIDER, amount)
			} else {
				// we dont have enough in the empty provider, remove everything and continue with the rest
				err = k.unbond(ctx, delegator, commontypes.EMPTY_PROVIDER, delegation.Amount)
				if err != nil {
					return err
				}
				amount = amount.Sub(delegation.Amount)
			}
		}
	}

	providers, _ = lavaslices.Remove[string](providers, commontypes.EMPTY_PROVIDER)

	var delegations []types.Delegation
	for _, provider := range providers {
		delegation, found := k.GetDelegation(ctx, provider, delegator)
		if found {
			delegations = append(delegations, delegation)
		}
	}

	slices.SortFunc(delegations, func(i, j types.Delegation) bool {
		return i.Amount.IsLT(j.Amount)
	})

	unbondAmount := map[string]sdk.Coin{}
	// first round of deduction
	for i := range delegations {
		amountToDeduct := amount.Amount.QuoRaw(int64(len(delegations) - i))
		if delegations[i].Amount.Amount.LT(amountToDeduct) {
			unbondAmount[delegations[i].Provider] = delegations[i].Amount
			amount = amount.Sub(delegations[i].Amount)
			delegations[i].Amount.Amount = sdk.ZeroInt()
		} else {
			coinToDeduct := sdk.NewCoin(delegations[i].Amount.Denom, amountToDeduct)
			unbondAmount[delegations[i].Provider] = coinToDeduct
			amount = amount.Sub(coinToDeduct)
			delegations[i].Amount = delegations[i].Amount.Sub(coinToDeduct)
		}
	}

	// we have leftovers, remove it ununiformaly
	for i := range delegations {
		if amount.IsZero() {
			break
		}
		key := delegations[i].Provider
		if delegations[i].Amount.Amount.LT(amount.Amount) {
			unbondAmount[key].Add(delegations[i].Amount)
			amount = amount.Sub(delegations[i].Amount)
		} else {
			unbondAmount[key].Add(amount)
			amount = amount.Sub(amount)
		}
	}

	// now unbond all
	for i := range delegations {
		key := delegations[i].Provider
		err := k.unbond(ctx, delegator, delegations[i].Provider, unbondAmount[key])
		if err != nil {
			return err
		}
	}

	return nil
}

// returns the difference between validators delegations and provider delegation (validators-providers)
func (k Keeper) VerifyDelegatorBalance(ctx sdk.Context, delAddr sdk.AccAddress) (math.Int, int, error) {
	providers, err := k.GetDelegatorProviders(ctx, delAddr.String())
	if err != nil {
		return math.ZeroInt(), 0, err
	}

	sumProviderDelegations := sdk.ZeroInt()
	for _, p := range providers {
		d, found := k.GetDelegation(ctx, p, delAddr.String())
		if found {
			sumProviderDelegations = sumProviderDelegations.Add(d.Amount.Amount)
		}
	}

	sumValidatorDelegations := sdk.ZeroInt()
	delegations := k.stakingKeeper.GetAllDelegatorDelegations(ctx, delAddr)
	for _, d := range delegations {
		v, found := k.stakingKeeper.GetValidator(ctx, d.GetValidatorAddr())
		if found {
			sumValidatorDelegations = sumValidatorDelegations.Add(v.TokensFromSharesRoundUp(d.Shares).Ceil().TruncateInt())
		}
	}

	return sumValidatorDelegations.Sub(sumProviderDelegations), len(providers), nil
}
