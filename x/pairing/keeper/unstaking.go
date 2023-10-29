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

func (k Keeper) UnstakeEntry(ctx sdk.Context, chainID, creator, unstakeDescription string) error {
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

	err = k.dualStakingKeeper.Unbond(ctx, existingEntry.GetAddress(), existingEntry.GetAddress(), existingEntry.GetChain(), existingEntry.Stake)
	if err != nil {
		return utils.LavaFormatWarning("can't unbond seld delegation", err,
			utils.Attribute{Key: "address", Value: existingEntry.Address},
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
		"geolocation": strconv.FormatInt(int64(existingEntry.GetGeolocation()), 10),
		"moniker":     existingEntry.GetMoniker(),
	}
	utils.LogLavaEvent(ctx, logger, types.ProviderUnstakeEventName, details, unstakeDescription)

	unstakeHoldBlocks := k.getUnstakeHoldBlocks(ctx, existingEntry.Chain)
	return k.epochStorageKeeper.AppendUnstakeEntry(ctx, existingEntry, unstakeHoldBlocks)
}

func (k Keeper) CheckUnstakingForCommit(ctx sdk.Context) {
	// this pops all the entries that had their deadline pass
	k.epochStorageKeeper.PopUnstakeEntries(ctx, uint64(ctx.BlockHeight()))
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
