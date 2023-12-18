package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

func (k Keeper) UnstakeEntry(ctx sdk.Context, validator, chainID, creator, unstakeDescription string) error {
	logger := k.Logger(ctx)
	// TODO: validate chainID basic validation

	// we can unstake disabled specs, but not missing ones
	_, found := k.specKeeper.GetSpec(ctx, chainID)
	if !found {
		return utils.LavaFormatWarning("trying to unstake an entry on missing spec", fmt.Errorf("spec not found"),
			utils.Attribute{Key: "spec", Value: chainID},
		)
	}
	senderAddr, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return utils.LavaFormatWarning("invalid address", err,
			utils.Attribute{Key: "provider", Value: creator},
		)
	}

	existingEntry, entryExists, _ := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, chainID, senderAddr)
	if !entryExists {
		return utils.LavaFormatWarning("can't unstake Entry, stake entry not found for address", fmt.Errorf("stake entry not found"),
			utils.Attribute{Key: "provider", Value: creator},
			utils.Attribute{Key: "spec", Value: chainID},
		)
	}

	err = k.dualstakingKeeper.UnbondFull(ctx, existingEntry.GetAddress(), validator, existingEntry.GetAddress(), existingEntry.GetChain(), existingEntry.Stake, true)
	if err != nil {
		return utils.LavaFormatWarning("can't unbond seld delegation", err,
			utils.Attribute{Key: "address", Value: existingEntry.Address},
			utils.Attribute{Key: "spec", Value: chainID},
		)
	}

	// index might have changed in the unbond
	existingEntry, _, indexInStakeStorage := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, chainID, senderAddr)
	err = k.epochStorageKeeper.RemoveStakeEntryCurrent(ctx, chainID, indexInStakeStorage)
	if err != nil {
		return utils.LavaFormatWarning("can't remove stake Entry, stake entry not found in index", err,
			utils.Attribute{Key: "index", Value: indexInStakeStorage},
			utils.Attribute{Key: "spec", Value: chainID},
		)
	}

	details := map[string]string{
		"address":     existingEntry.GetAddress(),
		"chainID":     existingEntry.GetChain(),
		"geolocation": strconv.FormatInt(int64(existingEntry.GetGeolocation()), 10),
		"moniker":     existingEntry.GetMoniker(),
		"stake":       existingEntry.GetStake().Amount.String(),
	}
	utils.LogLavaEvent(ctx, logger, types.ProviderUnstakeEventName, details, unstakeDescription)

	unstakeHoldBlocks := k.getUnstakeHoldBlocks(ctx, existingEntry.Chain)
	return k.epochStorageKeeper.AppendUnstakeEntry(ctx, existingEntry, unstakeHoldBlocks)
}

func (k Keeper) CheckUnstakingForCommit(ctx sdk.Context) {
	// this pops all the entries that had their deadline pass
	k.epochStorageKeeper.PopUnstakeEntries(ctx, uint64(ctx.BlockHeight()))
}

// NOTE: duplicated in x/dualstaking/keeper/delegate.go; any changes should be applied there too.
func (k Keeper) getUnstakeHoldBlocks(ctx sdk.Context, chainID string) uint64 {
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
		return k.epochStorageKeeper.UnstakeHoldBlocksStatic(ctx, block)
	} else {
		return k.epochStorageKeeper.UnstakeHoldBlocks(ctx, block)
	}

	// NOT REACHED
}

func (k Keeper) UnstakeEntryForce(ctx sdk.Context, chainID, provider, unstakeDescription string) error {
	providerAddr, err := sdk.AccAddressFromBech32(provider)
	if err != nil {
		return utils.LavaFormatWarning("invalid address", err,
			utils.Attribute{Key: "provider", Value: provider},
		)
	}

	existingEntry, entryExists, _ := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, chainID, providerAddr)
	if !entryExists {
		return utils.LavaFormatWarning("can't unstake Entry, stake entry not found for address", fmt.Errorf("stake entry not found"),
			utils.Attribute{Key: "provider", Value: provider},
			utils.Attribute{Key: "spec", Value: chainID},
		)
	}
	totalAmount := existingEntry.Stake.Amount
	delegations := k.stakingKeeper.GetAllDelegatorDelegations(ctx, providerAddr)

	for _, delegation := range delegations {
		validator, found := k.stakingKeeper.GetValidator(ctx, delegation.GetValidatorAddr())
		if !found {
			continue
		}
		amount := validator.TokensFromShares(delegation.Shares).TruncateInt()
		if totalAmount.LT(amount) {
			amount = totalAmount
		}
		totalAmount = totalAmount.Sub(amount)
		err = k.dualstakingKeeper.UnbondFull(ctx, existingEntry.GetAddress(), validator.OperatorAddress, existingEntry.GetAddress(), existingEntry.GetChain(), sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), amount), true)
		if err != nil {
			return utils.LavaFormatWarning("can't unbond seld delegation", err,
				utils.Attribute{Key: "address", Value: existingEntry.Address},
				utils.Attribute{Key: "spec", Value: chainID},
			)
		}

		if totalAmount.IsZero() {
			existingEntry, _, indexInStakeStorage := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, chainID, providerAddr)
			err = k.epochStorageKeeper.RemoveStakeEntryCurrent(ctx, chainID, indexInStakeStorage)
			if err != nil {
				return utils.LavaFormatWarning("can't remove stake Entry, stake entry not found in index", err,
					utils.Attribute{Key: "index", Value: indexInStakeStorage},
					utils.Attribute{Key: "spec", Value: chainID},
				)
			}

			details := map[string]string{
				"address":     existingEntry.GetAddress(),
				"chainID":     existingEntry.GetChain(),
				"geolocation": strconv.FormatInt(int64(existingEntry.GetGeolocation()), 10),
				"moniker":     existingEntry.GetMoniker(),
				"stake":       existingEntry.GetStake().Amount.String(),
			}
			utils.LogLavaEvent(ctx, k.Logger(ctx), types.ProviderUnstakeEventName, details, unstakeDescription)

			unstakeHoldBlocks := k.getUnstakeHoldBlocks(ctx, existingEntry.Chain)
			return k.epochStorageKeeper.AppendUnstakeEntry(ctx, existingEntry, unstakeHoldBlocks)
		}
	}

	return nil
}
