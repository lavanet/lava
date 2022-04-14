package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k Keeper) CheckUnstakingForCommit(ctx sdk.Context) error {
	//this pops all the entries that had their deadline pass
	unstakingEntriesToCredit := k.epochStorageKeeper.PopUnstakeEntries(ctx, types.ModuleName, uint64(ctx.BlockHeight()))
	if unstakingEntriesToCredit == nil {
		//no entries to handle
		return nil
	}
	err := k.creditUnstakingProviders(ctx, unstakingEntriesToCredit)
	return err
}

func (k Keeper) creditUnstakingProviders(ctx sdk.Context, entriesToUnstake []epochstoragetypes.StakeEntry) error {
	logger := k.Logger(ctx)
	verifySufficientAmountAndSendFromModuleToAddress := func(ctx sdk.Context, k Keeper, addr sdk.AccAddress, neededAmount sdk.Coin) (bool, error) {
		moduleBalance := k.bankKeeper.GetBalance(ctx, k.accountKeeper.GetModuleAddress(types.ModuleName), "stake")
		if moduleBalance.IsLT(neededAmount) {
			return false, fmt.Errorf("insufficient balance for unstaking %s current balance: %s", neededAmount, moduleBalance)
		}
		err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, []sdk.Coin{neededAmount})
		if err != nil {
			return false, fmt.Errorf("invalid transfer coins from module, %s to account %s", err, addr)
		}
		return true, nil
	}
	for _, unstakingEntry := range entriesToUnstake {
		details := map[string]string{"spec": unstakingEntry.Chain, "servicer": unstakingEntry.Address, "stake": unstakingEntry.Stake.String()}
		if unstakingEntry.Deadline <= uint64(ctx.BlockHeight()) {
			// found an entry that needs handling
			receiverAddr, err := sdk.AccAddressFromBech32(unstakingEntry.Address)
			//transfer stake money to the servicer account
			valid, err := verifySufficientAmountAndSendFromModuleToAddress(ctx, k, receiverAddr, unstakingEntry.Stake)
			if !valid {
				details["error"] = err.Error()
				utils.LavaError(ctx, logger, "servicer_unstaking_credit", details, "verifySufficientAmountAndSendFromModuleToAddress Failed,")
				panic(fmt.Sprintf("error unstaking : %s", err))
			}
			utils.LogLavaEvent(ctx, logger, "servicer_unstake_commit", details, "Unstaking Providers Commit")
		} else {
			// found an entry that isn't handled now, but later because its deadline isnt current block
			utils.LavaError(ctx, logger, "servicer_unstaking", details, "trying to unstake a servicer while its deadline wasn't reached")
		}
	}
	return nil
}
