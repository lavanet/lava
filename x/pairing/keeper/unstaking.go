package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

func (k Keeper) UnstakeEntry(ctx sdk.Context, chainID string, creator string, unstakeDescription string) error {
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

	existingEntry, entryExists, indexInStakeStorage := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, chainID, senderAddr)
	if !entryExists {
		return utils.LavaFormatWarning("can't unstake Entry, stake entry not found for address", fmt.Errorf("stake entry not found"),
			utils.Attribute{Key: "provider", Value: creator},
			utils.Attribute{Key: "spec", Value: chainID},
		)
	}
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
		"geolocation": strconv.FormatUint(existingEntry.GetGeolocation(), 10),
		"moniker":     existingEntry.GetMoniker(),
		"stake":       existingEntry.GetStake().Amount.String(),
	}
	utils.LogLavaEvent(ctx, logger, types.ProviderUnstakeEventName, details, unstakeDescription)

	unstakeHoldBlocks, err := k.unstakeHoldBlocks(ctx, existingEntry.Chain)
	if err != nil {
		return err
	}

	return k.epochStorageKeeper.AppendUnstakeEntry(ctx, existingEntry, unstakeHoldBlocks)
}

func (k Keeper) CheckUnstakingForCommit(ctx sdk.Context) {
	// this pops all the entries that had their deadline pass
	unstakingEntriesToCredit := k.epochStorageKeeper.PopUnstakeEntries(ctx, uint64(ctx.BlockHeight()))

	if unstakingEntriesToCredit != nil {
		k.creditUnstakingEntries(ctx, unstakingEntriesToCredit) // true for providers
	}
}

func (k Keeper) refundUnstakingProvider(ctx sdk.Context, addr sdk.AccAddress, neededAmount sdk.Coin) error {
	moduleBalance := k.bankKeeper.GetBalance(ctx, k.accountKeeper.GetModuleAddress(types.ModuleName), epochstoragetypes.TokenDenom)
	if moduleBalance.IsLT(neededAmount) {
		return fmt.Errorf("insufficient balance to unstake %s (current balance: %s)", neededAmount, moduleBalance)
	}
	err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, []sdk.Coin{neededAmount})
	if err != nil {
		return fmt.Errorf("failed to send coins from module to %s: %w", addr, err)
	}
	return nil
}

func (k Keeper) creditUnstakingEntries(ctx sdk.Context, entriesToUnstake []epochstoragetypes.StakeEntry) {
	logger := k.Logger(ctx)

	for _, unstakingEntry := range entriesToUnstake {
		details := map[string]string{
			"spec":     unstakingEntry.Chain,
			"provider": unstakingEntry.Address,
			"stake":    unstakingEntry.Stake.String(),
		}

		if unstakingEntry.StakeAppliedBlock <= uint64(ctx.BlockHeight()) {
			receiverAddr, err := sdk.AccAddressFromBech32(unstakingEntry.Address)
			if err != nil {
				// this should not happen; to avoid panic we simply skip this one (thus
				// freeze the situation so it can be investigated and orderly resolved).
				utils.LavaFormatError("critical: failed to get unstaking provider address", err,
					utils.Attribute{Key: "spec", Value: unstakingEntry.Chain},
					utils.Attribute{Key: "provider", Value: unstakingEntry.Address},
				)
				continue
			}
			if unstakingEntry.Stake.Amount.GT(sdk.ZeroInt()) {
				// transfer stake money to the stake entry account
				err := k.refundUnstakingProvider(ctx, receiverAddr, unstakingEntry.Stake)
				if err != nil {
					// we should always be able to redund a provider that decides to unstake;
					// but to avoid panic, just emit a critical error and proceed
					utils.LavaFormatError("critical: failed to refund staked provider", err,
						utils.Attribute{Key: "spec", Value: unstakingEntry.Chain},
						utils.Attribute{Key: "provider", Value: receiverAddr},
						utils.Attribute{Key: "stake", Value: unstakingEntry.Stake},
					)
				} else {
					utils.LogLavaEvent(ctx, logger, types.ProviderUnstakeEventName, details, "Unstaking Providers Commit")
				}
			}
		} else {
			// found an entry that isn't handled now, but later because its stakeAppliedBlock isnt current block
			utils.LavaFormatWarning("trying to unstake while its stakeAppliedBlock wasn't reached", fmt.Errorf("unstake failed"),
				utils.Attribute{Key: "spec", Value: unstakingEntry.Chain},
				utils.Attribute{Key: "provider", Value: unstakingEntry.Address},
				utils.Attribute{Key: "stake", Value: unstakingEntry.Stake.String()},
			)
		}
	}
}

func (k Keeper) unstakeHoldBlocks(ctx sdk.Context, chainID string) (uint64, error) {
	spec, found := k.specKeeper.GetSpec(ctx, chainID)
	if !found {
		return 0, fmt.Errorf("coult not find spec %s", chainID)
	}

	if spec.ProvidersTypes == spectypes.Spec_static {
		return k.epochStorageKeeper.UnstakeHoldBlocksStatic(ctx, uint64(ctx.BlockHeight())), nil
	} else {
		return k.epochStorageKeeper.UnstakeHoldBlocks(ctx, uint64(ctx.BlockHeight())), nil
	}
}
