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
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	commontypes "github.com/lavanet/lava/utils/common/types"
	lavaslices "github.com/lavanet/lava/utils/lavaslices"
	"github.com/lavanet/lava/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"golang.org/x/exp/slices"
)

// increaseDelegation increases the delegation of a delegator to a provider for a
// given chain. It updates the fixation stores for both delegations and delegators,
// and updates the (epochstorage) stake-entry.
func (k Keeper) increaseDelegation(ctx sdk.Context, delegator, provider, chainID string, amount sdk.Coin, nextEpoch uint64) error {
	// get, update and append the delegation entry
	var delegationEntry types.Delegation
	index := types.DelegationKey(provider, delegator, chainID)
	found := k.delegationFS.FindEntry(ctx, index, nextEpoch, &delegationEntry)
	if !found {
		// new delegation (i.e. not increase of existing one)
		delegationEntry = types.NewDelegation(delegator, provider, chainID, ctx.BlockTime(), k.stakingKeeper.BondDenom(ctx))
	}

	delegationEntry.AddAmount(amount)

	err := k.delegationFS.AppendEntry(ctx, index, nextEpoch, &delegationEntry)
	if err != nil {
		// append should never fail here
		return utils.LavaFormatError("critical: append delegation entry", err,
			utils.Attribute{Key: "delegator", Value: delegationEntry.Delegator},
			utils.Attribute{Key: "provider", Value: delegationEntry.Provider},
			utils.Attribute{Key: "chainID", Value: delegationEntry.ChainID},
		)
	}

	// get, update and append the delegator entry
	var delegatorEntry types.Delegator
	index = types.DelegatorKey(delegator)
	_ = k.delegatorFS.FindEntry(ctx, index, nextEpoch, &delegatorEntry)

	contains := delegatorEntry.AddProvider(provider)
	if !contains {
		err = k.delegatorFS.AppendEntry(ctx, index, nextEpoch, &delegatorEntry)
		if err != nil {
			// append should never fail here
			return utils.LavaFormatError("critical: append delegator entry", err,
				utils.Attribute{Key: "delegator", Value: delegator},
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "chainID", Value: chainID},
			)
		}
	}

	if provider != commontypes.EMPTY_PROVIDER {
		// update the stake entry
		return k.modifyStakeEntryDelegation(ctx, delegator, provider, chainID, amount, true)
	}

	return nil
}

// decreaseDelegation decreases the delegation of a delegator to a provider for a
// given chain. It updates the fixation stores for both delegations and delegators,
// and updates the (epochstorage) stake-entry.
func (k Keeper) decreaseDelegation(ctx sdk.Context, delegator, provider, chainID string, amount sdk.Coin, nextEpoch uint64) error {
	// get, update and append the delegation entry
	var delegationEntry types.Delegation
	index := types.DelegationKey(provider, delegator, chainID)
	found := k.delegationFS.FindEntry(ctx, index, nextEpoch, &delegationEntry)
	if !found {
		return types.ErrDelegationNotFound
	}

	if delegationEntry.Amount.IsLT(amount) {
		return types.ErrInsufficientDelegation
	}

	delegationEntry.SubAmount(amount)

	// if delegation now becomes zero, then remove this entry altogether;
	// otherwise just append the new version (for next epoch).
	if delegationEntry.Amount.IsZero() {
		err := k.delegationFS.DelEntry(ctx, index, nextEpoch)
		if err != nil {
			// delete should never fail here
			return utils.LavaFormatError("critical: delete delegation entry", err,
				utils.Attribute{Key: "delegator", Value: delegator},
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "chainID", Value: chainID},
			)
		}
	} else {
		err := k.delegationFS.AppendEntry(ctx, index, nextEpoch, &delegationEntry)
		if err != nil {
			// append should never fail here
			return utils.LavaFormatError("failed to update delegation entry", err,
				utils.Attribute{Key: "delegator", Value: delegator},
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "chainID", Value: chainID},
			)
		}
	}

	// get, update and append the delegator entry
	var delegatorEntry types.Delegator
	index = types.DelegatorKey(delegator)
	found = k.delegatorFS.FindEntry(ctx, index, nextEpoch, &delegatorEntry)
	if !found {
		// we found the delegation above, so the delegator must exist as well
		return utils.LavaFormatError("critical: delegator entry for delegation not found",
			types.ErrDelegationNotFound,
			utils.Attribute{Key: "delegator", Value: delegator},
			utils.Attribute{Key: "provider", Value: provider},
			utils.Attribute{Key: "chainID", Value: chainID},
		)
	}

	// if delegation now becomes zero, then remove this provider from the delegator
	// entry; and if the delegator entry becomes entry then remove it altogether.
	// otherwise just append the new version (for next epoch).
	if delegationEntry.Amount.IsZero() {
		if len(k.GetAllProviderDelegatorDelegations(ctx, delegator, provider, nextEpoch)) == 0 {
			delegatorEntry.DelProvider(provider)
			if delegatorEntry.IsEmpty() {
				err := k.delegatorFS.DelEntry(ctx, index, nextEpoch)
				if err != nil {
					// delete should never fail here
					return utils.LavaFormatError("critical: delete delegator entry", err,
						utils.Attribute{Key: "delegator", Value: delegator},
						utils.Attribute{Key: "provider", Value: provider},
						utils.Attribute{Key: "chainID", Value: chainID},
					)
				}
			} else {
				err := k.delegatorFS.AppendEntry(ctx, index, nextEpoch, &delegatorEntry)
				if err != nil {
					// append should never fail here
					return utils.LavaFormatError("failed to update delegator entry", err,
						utils.Attribute{Key: "delegator", Value: delegator},
						utils.Attribute{Key: "provider", Value: provider},
						utils.Attribute{Key: "chainID", Value: chainID},
					)
				}
			}
		}
	}

	if provider != commontypes.EMPTY_PROVIDER {
		return k.modifyStakeEntryDelegation(ctx, delegator, provider, chainID, amount, false)
	}

	return nil
}

// modifyStakeEntryDelegation modifies the (epochstorage) stake-entry of the provider for a chain based on the action (increase or decrease).
func (k Keeper) modifyStakeEntryDelegation(ctx sdk.Context, delegator, provider, chainID string, amount sdk.Coin, increase bool) (err error) {
	stakeEntry, exists := k.epochstorageKeeper.GetStakeEntryCurrent(ctx, chainID, provider)
	if !exists || provider != stakeEntry.Address {
		if increase {
			return epochstoragetypes.ErrProviderNotStaked
		}
		// For decrease, if the provider doesn't exist, return without error
		return nil
	}

	if delegator == stakeEntry.Vault {
		if increase {
			stakeEntry.Stake = stakeEntry.Stake.Add(amount)
		} else {
			stakeEntry.Stake, err = stakeEntry.Stake.SafeSub(amount)
			if err != nil {
				return fmt.Errorf("invalid or insufficient funds: %w", err)
			}
		}
	} else {
		if increase {
			stakeEntry.DelegateTotal = stakeEntry.DelegateTotal.Add(amount)
		} else {
			stakeEntry.DelegateTotal, err = stakeEntry.DelegateTotal.SafeSub(amount)
			if err != nil {
				return fmt.Errorf("invalid or insufficient funds: %w", err)
			}
		}
	}

	details := map[string]string{
		"provider_vault":    stakeEntry.Vault,
		"provider_provider": stakeEntry.Address,
		"chain_id":          stakeEntry.Chain,
		"moniker":           stakeEntry.Description.Moniker,
		"description":       stakeEntry.Description.String(),
		"stake":             stakeEntry.Stake.String(),
		"effective_stake":   stakeEntry.EffectiveStake().String() + stakeEntry.Stake.Denom,
	}

	if stakeEntry.Stake.IsLT(k.GetParams(ctx).MinSelfDelegation) {
		k.epochstorageKeeper.RemoveStakeEntryCurrent(ctx, chainID, stakeEntry.Address)
		details["min_self_delegation"] = k.GetParams(ctx).MinSelfDelegation.String()
		utils.LogLavaEvent(ctx, k.Logger(ctx), types.UnstakeFromUnbond, details, "unstaking provider due to unbond that lowered its stake below min self delegation")
		return nil
	} else if stakeEntry.EffectiveStake().LT(k.specKeeper.GetMinStake(ctx, chainID).Amount) {
		details["min_spec_stake"] = k.specKeeper.GetMinStake(ctx, chainID).String()
		utils.LogLavaEvent(ctx, k.Logger(ctx), types.FreezeFromUnbond, details, "freezing provider due to stake below min spec stake")
		stakeEntry.Freeze()
	} else if delegator == stakeEntry.Vault && stakeEntry.IsFrozen() && !stakeEntry.IsJailed(ctx.BlockTime().UTC().Unix()) {
		stakeEntry.UnFreeze(k.epochstorageKeeper.GetCurrentNextEpoch(ctx) + 1)
	}

	k.epochstorageKeeper.SetStakeEntryCurrent(ctx, stakeEntry)

	return nil
}

// delegate lets a delegator delegate an amount of coins to a provider.
// (effective on next epoch)
func (k Keeper) delegate(ctx sdk.Context, delegator, provider, chainID string, amount sdk.Coin) error {
	nextEpoch := k.epochstorageKeeper.GetCurrentNextEpoch(ctx)

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
			utils.Attribute{Key: "chainID", Value: chainID},
		)
	}

	err = k.increaseDelegation(ctx, delegator, provider, chainID, amount, nextEpoch)
	if err != nil {
		return utils.LavaFormatWarning("failed to increase delegation", err,
			utils.Attribute{Key: "delegator", Value: delegator},
			utils.Attribute{Key: "provider", Value: provider},
			utils.Attribute{Key: "amount", Value: amount.String()},
			utils.Attribute{Key: "chainID", Value: chainID},
		)
	}

	return nil
}

// Redelegate lets a delegator transfer its delegation between providers, but
// without the funds being subject to unstakeHoldBlocks witholding period.
// (effective on next epoch)
func (k Keeper) Redelegate(ctx sdk.Context, delegator, from, to, fromChainID, toChainID string, amount sdk.Coin) error {
	_, foundFrom := k.specKeeper.GetSpec(ctx, fromChainID)
	_, foundTo := k.specKeeper.GetSpec(ctx, toChainID)
	if (!foundFrom && fromChainID != commontypes.EMPTY_PROVIDER_CHAINID) ||
		(!foundTo && toChainID != commontypes.EMPTY_PROVIDER_CHAINID) {
		return utils.LavaFormatWarning("cannot redelegate with invalid chain IDs", fmt.Errorf("chain ID not found"),
			utils.LogAttr("from_chain_id", fromChainID),
			utils.LogAttr("to_chain_id", toChainID),
		)
	}

	nextEpoch := k.epochstorageKeeper.GetCurrentNextEpoch(ctx)

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

	err := k.increaseDelegation(ctx, delegator, to, toChainID, amount, nextEpoch)
	if err != nil {
		return utils.LavaFormatWarning("failed to increase delegation", err,
			utils.Attribute{Key: "delegator", Value: delegator},
			utils.Attribute{Key: "provider", Value: to},
			utils.Attribute{Key: "amount", Value: amount.String()},
		)
	}

	err = k.decreaseDelegation(ctx, delegator, from, fromChainID, amount, nextEpoch)
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
func (k Keeper) unbond(ctx sdk.Context, delegator, provider, chainID string, amount sdk.Coin) error {
	_, found := k.specKeeper.GetSpec(ctx, chainID)
	if chainID != commontypes.EMPTY_PROVIDER_CHAINID && !found {
		return utils.LavaFormatWarning("cannot unbond with invalid chain ID", fmt.Errorf("chain ID not found"),
			utils.LogAttr("chain_id", chainID))
	}

	nextEpoch := k.epochstorageKeeper.GetCurrentNextEpoch(ctx)

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

	err := k.decreaseDelegation(ctx, delegator, provider, chainID, amount, nextEpoch)
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
func (k Keeper) GetDelegatorProviders(ctx sdk.Context, delegator string, epoch uint64) (providers []string, err error) {
	_, err = sdk.AccAddressFromBech32(delegator)
	if err != nil {
		return nil, utils.LavaFormatWarning("cannot get delegator's providers", err,
			utils.Attribute{Key: "delegator", Value: delegator},
		)
	}

	var delegatorEntry types.Delegator
	prefix := types.DelegatorKey(delegator)
	k.delegatorFS.FindEntry(ctx, prefix, epoch, &delegatorEntry)

	return delegatorEntry.Providers, nil
}

func (k Keeper) GetProviderDelegators(ctx sdk.Context, provider string, epoch uint64) ([]types.Delegation, error) {
	if provider != commontypes.EMPTY_PROVIDER {
		_, err := sdk.AccAddressFromBech32(provider)
		if err != nil {
			return nil, utils.LavaFormatWarning("cannot get provider's delegators", err,
				utils.Attribute{Key: "provider", Value: provider},
			)
		}
	}

	var delegations []types.Delegation
	indices := k.delegationFS.GetAllEntryIndicesWithPrefix(ctx, provider)
	for _, ind := range indices {
		var delegation types.Delegation
		found := k.delegationFS.FindEntry(ctx, ind, epoch, &delegation)
		if !found {
			provider, delegator, chainID := types.DelegationKeyDecode(ind)
			utils.LavaFormatError("delegationFS entry index has no entry", fmt.Errorf("provider delegation not found"),
				utils.Attribute{Key: "delegator", Value: delegator},
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "chainID", Value: chainID},
			)
			continue
		}
		delegations = append(delegations, delegation)
	}

	return delegations, nil
}

func (k Keeper) GetDelegation(ctx sdk.Context, delegator, provider, chainID string, epoch uint64) (types.Delegation, bool) {
	var delegationEntry types.Delegation
	index := types.DelegationKey(provider, delegator, chainID)
	found := k.delegationFS.FindEntry(ctx, index, epoch, &delegationEntry)

	return delegationEntry, found
}

func (k Keeper) GetAllProviderDelegatorDelegations(ctx sdk.Context, delegator, provider string, epoch uint64) []types.Delegation {
	prefix := types.DelegationKey(provider, delegator, "")
	indices := k.delegationFS.GetAllEntryIndicesWithPrefix(ctx, prefix)

	var delegations []types.Delegation
	for _, ind := range indices {
		var delegation types.Delegation
		_, deleted, _, found := k.delegationFS.FindEntryDetailed(ctx, ind, epoch, &delegation)
		if !found {
			if !deleted {
				provider, delegator, chainID := types.DelegationKeyDecode(ind)
				utils.LavaFormatError("delegationFS entry index has no entry", fmt.Errorf("provider delegation not found"),
					utils.Attribute{Key: "delegator", Value: delegator},
					utils.Attribute{Key: "provider", Value: provider},
					utils.Attribute{Key: "chainID", Value: chainID},
				)
			}
			continue
		}
		delegations = append(delegations, delegation)
	}

	return delegations
}

func (k Keeper) UnbondUniformProviders(ctx sdk.Context, delegator string, amount sdk.Coin) error {
	epoch := k.epochstorageKeeper.GetCurrentNextEpoch(ctx)
	providers, err := k.GetDelegatorProviders(ctx, delegator, epoch)
	if err != nil {
		return err
	}

	// first remove from the empty provider
	if lavaslices.Contains[string](providers, commontypes.EMPTY_PROVIDER) {
		delegation, found := k.GetDelegation(ctx, delegator, commontypes.EMPTY_PROVIDER, commontypes.EMPTY_PROVIDER_CHAINID, epoch)
		if found {
			if delegation.Amount.Amount.GTE(amount.Amount) {
				// we have enough here, remove all from empty delegator and bail
				return k.unbond(ctx, delegator, commontypes.EMPTY_PROVIDER, commontypes.EMPTY_PROVIDER_CHAINID, amount)
			} else {
				// we dont have enough in the empty provider, remove everything and continue with the rest
				err = k.unbond(ctx, delegator, commontypes.EMPTY_PROVIDER, commontypes.EMPTY_PROVIDER_CHAINID, delegation.Amount)
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
		delegations = append(delegations, k.GetAllProviderDelegatorDelegations(ctx, delegator, provider, epoch)...)
	}

	slices.SortFunc(delegations, func(i, j types.Delegation) bool {
		return i.Amount.IsLT(j.Amount)
	})

	type delegationKey struct {
		provider string
		chainID  string
	}
	unbondAmount := map[delegationKey]sdk.Coin{}

	// first round of deduction
	for i := range delegations {
		key := delegationKey{provider: delegations[i].Provider, chainID: delegations[i].ChainID}
		amountToDeduct := amount.Amount.QuoRaw(int64(len(delegations) - i))
		if delegations[i].Amount.Amount.LT(amountToDeduct) {
			unbondAmount[key] = delegations[i].Amount
			amount = amount.Sub(delegations[i].Amount)
			delegations[i].Amount.Amount = sdk.ZeroInt()
		} else {
			coinToDeduct := sdk.NewCoin(delegations[i].Amount.Denom, amountToDeduct)
			unbondAmount[key] = coinToDeduct
			amount = amount.Sub(coinToDeduct)
			delegations[i].Amount = delegations[i].Amount.Sub(coinToDeduct)
		}
	}

	// we have leftovers, remove it ununiformaly
	for i := range delegations {
		if amount.IsZero() {
			break
		}
		key := delegationKey{provider: delegations[i].Provider, chainID: delegations[i].ChainID}
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
		key := delegationKey{provider: delegations[i].Provider, chainID: delegations[i].ChainID}
		err := k.unbond(ctx, delegator, delegations[i].Provider, delegations[i].ChainID, unbondAmount[key])
		if err != nil {
			return err
		}
	}

	return nil
}

// returns the difference between validators delegations and provider delegation (validators-providers)
func (k Keeper) VerifyDelegatorBalance(ctx sdk.Context, delAddr sdk.AccAddress) (math.Int, int, error) {
	nextEpoch := k.epochstorageKeeper.GetCurrentNextEpoch(ctx)
	providers, err := k.GetDelegatorProviders(ctx, delAddr.String(), nextEpoch)
	if err != nil {
		return math.ZeroInt(), 0, err
	}

	sumProviderDelegations := sdk.ZeroInt()
	for _, p := range providers {
		delegations := k.GetAllProviderDelegatorDelegations(ctx, delAddr.String(), p, nextEpoch)
		for _, d := range delegations {
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
