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
	"encoding/json"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/slices"
	"github.com/lavanet/lava/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

// getNextEpoch returns the block of next epoch.
// (all delegate transaction take effect in the subsequent epoch)
func (k Keeper) getNextEpoch(ctx sdk.Context) (uint64, error) {
	block := uint64(ctx.BlockHeight())
	nextEpoch, err := k.epochstorageKeeper.GetNextEpoch(ctx, block)
	if err != nil {
		return 0, utils.LavaFormatError("critical: failed to get next epoch", err,
			utils.Attribute{Key: "block", Value: block},
		)
	}
	return nextEpoch, nil
}

// validateCoins validates that the input amount is valid and non-negative
func validateCoins(amount sdk.Coin) error {
	if !amount.IsValid() {
		return utils.LavaFormatWarning("invalid coins to delegate",
			sdkerrors.ErrInvalidCoins,
			utils.Attribute{Key: "amount", Value: amount},
		)
	}
	return nil
}

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
		delegationEntry = types.NewDelegation(delegator, provider, chainID)
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

	delegatorEntry.AddProvider(provider)

	err = k.delegatorFS.AppendEntry(ctx, index, nextEpoch, &delegatorEntry)
	if err != nil {
		// append should never fail here
		return utils.LavaFormatError("critical: append delegator entry", err,
			utils.Attribute{Key: "delegator", Value: delegator},
			utils.Attribute{Key: "provider", Value: provider},
			utils.Attribute{Key: "chainID", Value: chainID},
		)
	}

	// update the stake entry
	err = k.increaseStakeEntryDelegation(ctx, provider, chainID, amount)
	if err != nil {
		return err
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
		}
	} else {
		delegatorEntry.AddProvider(provider)
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

	if err := k.decreaseStakeEntryDelegation(ctx, provider, chainID, amount); err != nil {
		return err
	}

	return nil
}

// increaseStakeEntryDelegation increases the (epochstorage) stake-entry of the provider for a chain.
func (k Keeper) increaseStakeEntryDelegation(ctx sdk.Context, provider, chainID string, amount sdk.Coin) error {
	providerAddr, err := sdk.AccAddressFromBech32(provider)
	if err != nil {
		// panic:ok: this call was alreadys successful by the caller
		utils.LavaFormatPanic("increaseStakeEntry: invalid provider address", err,
			utils.Attribute{Key: "provider", Value: provider},
		)
	}

	stakeEntry, exists, index := k.epochstorageKeeper.GetStakeEntryByAddressCurrent(ctx, chainID, providerAddr)
	if !exists {
		return types.ErrProviderNotStaked
	}

	// sanity check
	if stakeEntry.Address != provider {
		return utils.LavaFormatError("critical: delegate to provider with address mismatch", sdkerrors.ErrInvalidAddress,
			utils.Attribute{Key: "provider", Value: provider},
			utils.Attribute{Key: "address", Value: stakeEntry.Address},
		)
	}

	stakeEntry.DelegateTotal = stakeEntry.DelegateTotal.Add(amount)

	k.epochstorageKeeper.ModifyStakeEntryCurrent(ctx, chainID, stakeEntry, index)

	return nil
}

// decreaseStakeEntryDelegation decreases the (epochstorage) stake-entry of the provider for a chain.
func (k Keeper) decreaseStakeEntryDelegation(ctx sdk.Context, provider, chainID string, amount sdk.Coin) error {
	providerAddr, err := sdk.AccAddressFromBech32(provider)
	if err != nil {
		// panic:ok: this call was alreadys successful by the caller
		utils.LavaFormatPanic("decreaseStakeEntryDelegation: invalid provider address", err,
			utils.Attribute{Key: "provider", Value: provider},
		)
	}

	stakeEntry, exists, index := k.epochstorageKeeper.GetStakeEntryByAddressCurrent(ctx, chainID, providerAddr)
	if !exists {
		return nil
	}

	// sanity check
	if stakeEntry.Address != provider {
		return utils.LavaFormatError("critical: un-delegate from provider with address mismatch", sdkerrors.ErrInvalidAddress,
			utils.Attribute{Key: "provider", Value: provider},
			utils.Attribute{Key: "address", Value: stakeEntry.Address},
		)
	}

	stakeEntry.DelegateTotal, err = stakeEntry.DelegateTotal.SafeSub(amount)
	if err != nil {
		return fmt.Errorf("invalid or insufficient funds: %w", err)
	}

	k.epochstorageKeeper.ModifyStakeEntryCurrent(ctx, chainID, stakeEntry, index)

	return nil
}

// Delegate lets a delegator delegate an amount of coins to a provider.
// (effective on next epoch)
func (k Keeper) Delegate(ctx sdk.Context, delegator, provider, chainID string, amount sdk.Coin) error {
	nextEpoch, err := k.getNextEpoch(ctx)
	if err != nil {
		return err
	}

	delegatorAddr, err := sdk.AccAddressFromBech32(delegator)
	if err != nil {
		return utils.LavaFormatWarning("invalid delegator address", err,
			utils.Attribute{Key: "delegator", Value: delegator},
		)
	}

	if _, err = sdk.AccAddressFromBech32(provider); err != nil {
		return utils.LavaFormatWarning("invalid provider address", err,
			utils.Attribute{Key: "provider", Value: provider},
		)
	}

	if err := validateCoins(amount); err != nil {
		return err
	} else if amount.IsZero() {
		return nil
	}

	balance := k.bankKeeper.GetBalance(ctx, delegatorAddr, epochstoragetypes.TokenDenom)
	if balance.IsLT(amount) {
		return utils.LavaFormatWarning("insufficient funds to delegate",
			sdkerrors.ErrInsufficientFunds,
			utils.Attribute{Key: "balance", Value: balance},
			utils.Attribute{Key: "amount", Value: amount},
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

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, delegatorAddr, types.BondedPoolName, sdk.NewCoins(amount))
	if err != nil {
		return utils.LavaFormatError("failed to transfer coins to module", err,
			utils.Attribute{Key: "balance", Value: balance},
			utils.Attribute{Key: "amount", Value: amount},
		)
	}

	return nil
}

// Redelegate lets a delegator transfer its delegation between providers, but
// without the funds being subject to unstakeHoldBlocks witholding period.
// (effective on next epoch)
func (k Keeper) Redelegate(ctx sdk.Context, delegator, from, to, fromChainID, toChainID string, amount sdk.Coin) error {
	nextEpoch, err := k.getNextEpoch(ctx)
	if err != nil {
		return err
	}

	if _, err = sdk.AccAddressFromBech32(delegator); err != nil {
		return utils.LavaFormatWarning("invalid delegator address", err,
			utils.Attribute{Key: "delegator", Value: delegator},
		)
	}

	if _, err = sdk.AccAddressFromBech32(from); err != nil {
		return utils.LavaFormatWarning("invalid from-provider address", err,
			utils.Attribute{Key: "from_provider", Value: from},
		)
	}

	if _, err = sdk.AccAddressFromBech32(to); err != nil {
		return utils.LavaFormatWarning("invalid to-provider address", err,
			utils.Attribute{Key: "to_provider", Value: to},
		)
	}

	if err := validateCoins(amount); err != nil {
		return err
	} else if amount.IsZero() {
		return nil
	}

	err = k.decreaseDelegation(ctx, delegator, from, fromChainID, amount, nextEpoch)
	if err != nil {
		return utils.LavaFormatWarning("failed to decrease delegation", err,
			utils.Attribute{Key: "delegator", Value: delegator},
			utils.Attribute{Key: "provider", Value: from},
			utils.Attribute{Key: "amount", Value: amount.String()},
		)
	}

	err = k.increaseDelegation(ctx, delegator, to, toChainID, amount, nextEpoch)
	if err != nil {
		return utils.LavaFormatWarning("failed to increase delegation", err,
			utils.Attribute{Key: "delegator", Value: delegator},
			utils.Attribute{Key: "provider", Value: to},
			utils.Attribute{Key: "amount", Value: amount.String()},
		)
	}

	// no need to transfer funds, because they remain in the dualstaking module
	// (specifically in types.BondedPoolName).

	return nil
}

// Unbond lets a delegator get its delegated coins back from a provider. The
// delegation ends immediately, but coins are held for unstakeHoldBlocks period
// before released and transferred back to the delegator. The rewards from the
// provider will be updated accordingly (or terminate) from the next epoch.
// (effective on next epoch)
func (k Keeper) Unbond(ctx sdk.Context, delegator, provider, chainID string, amount sdk.Coin) error {
	nextEpoch, err := k.getNextEpoch(ctx)
	if err != nil {
		return err
	}

	if _, err = sdk.AccAddressFromBech32(delegator); err != nil {
		return utils.LavaFormatWarning("invalid delegator address", err,
			utils.Attribute{Key: "delegator", Value: delegator},
		)
	}

	if _, err = sdk.AccAddressFromBech32(provider); err != nil {
		return utils.LavaFormatWarning("invalid provider address", err,
			utils.Attribute{Key: "provider", Value: provider},
		)
	}

	if err := validateCoins(amount); err != nil {
		return err
	} else if amount.IsZero() {
		return nil
	}

	err = k.decreaseDelegation(ctx, delegator, provider, chainID, amount, nextEpoch)
	if err != nil {
		return utils.LavaFormatWarning("failed to decrease delegation", err,
			utils.Attribute{Key: "delegator", Value: delegator},
			utils.Attribute{Key: "provider", Value: provider},
			utils.Attribute{Key: "amount", Value: amount.String()},
		)
	}

	// in unbonding the funds to not return immediately to the delegator; instead
	// they transfer to the NotBondedPoolName module account, where they are held
	// for the hold period before they are finally released.

	k.bondedTokensToNotBonded(ctx, amount.Amount)

	err = k.setUnbondingTimer(ctx, delegator, provider, chainID, amount)
	if err != nil {
		return utils.LavaFormatError("failed to unbond delegated coins", err,
			utils.Attribute{Key: "amount", Value: amount},
		)
	}

	return nil
}

var space = " "

// encodeForTimer generates timer key unique to a specific delegation; thus it
// must include the delegator, provider, and chainID.
func encodeForTimer(delegator, provider, chainID string) []byte {
	dlen, plen, clen := len(delegator), len(provider), len(chainID)

	encodedKey := make([]byte, dlen+plen+clen+2)

	index := 0
	index += copy(encodedKey[index:], delegator)
	encodedKey[index] = []byte(space)[0]
	index += copy(encodedKey[index+1:], provider) + 1
	encodedKey[index] = []byte(space)[0]
	copy(encodedKey[index+1:], chainID)

	return encodedKey
}

// decodeForTimer extracts the delegation details from a timer key.
func decodeForTimer(encodedKey []byte) (string, string, string) {
	split := strings.Split(string(encodedKey), space)
	if len(split) != 3 {
		return "", "", ""
	}
	delegator, provider, chainID := split[0], split[1], split[2]
	return delegator, provider, chainID
}

// setUnbondingTimer creates a timer for an unbonding operation, that will expire
// in the spec's unbonding-hold-blocks blocks from now. only one unbonding request
// can be placed per block (for a given delegator/provider/chainID combination).
func (k Keeper) setUnbondingTimer(ctx sdk.Context, delegator, provider, chainID string, amount sdk.Coin) error {
	key := encodeForTimer(delegator, provider, chainID)

	unholdBlocks := k.getUnbondHoldBlocks(ctx, chainID)
	timeout := uint64(ctx.BlockHeight()) + unholdBlocks

	// the timer key encodes the unique delegator/provider/chainID combination.
	// the timer data holds the amount to be released in the future (marshalled).
	if k.unbondingTS.HasTimerByBlockHeight(ctx, timeout, key) {
		return types.ErrUnbondingInProgress
	}

	data, _ := json.Marshal(amount)
	k.unbondingTS.AddTimerByBlockHeight(ctx, timeout, key, data)

	return nil
}

// finalizeUnbonding is called when the unbond hold period terminated; it extracts
// the delegation details from the timer key and data and releases the bonder funds
// back to the delegator.
func (k Keeper) finalizeUnbonding(ctx sdk.Context, key []byte, data []byte) {
	delegator, provider, chainID := decodeForTimer(key)

	attrs := slices.Slice(
		utils.Attribute{Key: "delegator", Value: delegator},
		utils.Attribute{Key: "provider", Value: provider},
		utils.Attribute{Key: "chainID", Value: chainID},
		utils.Attribute{Key: "data", Value: data},
	)

	var amount sdk.Coin
	err := json.Unmarshal(data, &amount)
	if err != nil {
		utils.LavaFormatError("critical: finalizeBonding failed to decode", err, attrs...)
	}

	// sanity: delegator address must be valid
	delegatorAddr, err := sdk.AccAddressFromBech32(delegator)
	if err != nil {
		utils.LavaFormatError("critical: finalizeBonding invalid delegator", err, attrs...)
		return
	}

	// sanity: provider address must be valid
	if _, err := sdk.AccAddressFromBech32(provider); err != nil {
		utils.LavaFormatError("critical: finalizeBonding invalid provider", err, attrs...)
		return
	}

	// sanity: saved amount is valid
	if err := validateCoins(amount); err != nil {
		utils.LavaFormatError("critical: finalizeBonding invalid amount", err, attrs...)
		return
	} else if amount.IsZero() {
		utils.LavaFormatError("critical: finalizeBonding zero amount", sdkerrors.ErrInsufficientFunds, attrs...)
		return
	}

	// sanity: verify that BondedPool has enough funds
	if k.TotalNotBondedTokens(ctx).LT(amount.Amount) {
		utils.LavaFormatError("critical: finalizeBonding insufficient bonds", err, attrs...)
		return
	}

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.NotBondedPoolName, delegatorAddr, sdk.NewCoins(amount))
	if err != nil {
		utils.LavaFormatError("critical: finalizeBonding failed to transfer", err, attrs...)
	}

	details := map[string]string{
		"delegator": delegator,
		"provider":  provider,
		"chainID":   chainID,
		"amount":    amount.String(),
	}

	utils.LogLavaEvent(ctx, k.Logger(ctx), types.RefundedEventName, details, "Refunded")
}

// NOTE: duplicated in x/dualstaking/keeper/delegate.go; any changes should be applied there too.
func (k Keeper) getUnbondHoldBlocks(ctx sdk.Context, chainID string) uint64 {
	_, found, providerType := k.specKeeper.IsSpecFoundAndActive(ctx, chainID)
	if !found {
		utils.LavaFormatError("critical: failed to get spec for chainID",
			fmt.Errorf("unknown chainID"),
			utils.Attribute{Key: "chainID", Value: chainID},
		)
	}

	// note: if spec was not found, the default choice is Spec_dynamic == 0

	block := uint64(ctx.BlockHeight())
	if providerType == spectypes.Spec_static {
		return k.epochstorageKeeper.UnstakeHoldBlocksStatic(ctx, block)
	} else {
		return k.epochstorageKeeper.UnstakeHoldBlocks(ctx, block)
	}

	// NOT REACHED
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
	_, err := sdk.AccAddressFromBech32(provider)
	if err != nil {
		return nil, utils.LavaFormatWarning("cannot get provider's delegators", err,
			utils.Attribute{Key: "provider", Value: provider},
		)
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
