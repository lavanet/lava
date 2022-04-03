package keeper

import (
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/user/types"
	"github.com/rs/zerolog/log"
)

func (k Keeper) CheckUnstakingForCommit(ctx sdk.Context) error {
	deadline, found := k.GetBlockDeadlineForCallback(ctx)
	if !found {
		panic("didn't find single variable BlockDeadlineForCallback")
	}
	if deadline.Deadline.Num == 0 { //special case, theres no deadline so return
		return nil
	}
	currentBlock := ctx.BlockHeight()
	if deadline.Deadline.Num > uint64(currentBlock) { // didn't reach the first deadline
		return nil
	}
	err := k.creditUnstakingUsersAndRemoveFromCallback(ctx, uint64(currentBlock))
	return err
}

func (k Keeper) creditUnstakingUsersAndRemoveFromCallback(ctx sdk.Context, deadline uint64) error {
	unstakingUsers := k.GetAllUnstakingUsersAllSpecs(ctx)
	minDeadaline := uint64(math.MaxUint64)
	indexesForDelete := make([]uint64, 0)
	//handlng an entry needs a few things done:
	//A2. remove the entry from UnstakingUsersAllSpecs
	//A3. transfer money to the User account
	//A4. repeat for all entries with deadline
	//A5. set new deadline for next callback

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

	for idx, unstakingEntry := range unstakingUsers {
		//A4. repeat for all entries with deadline
		if unstakingEntry.Unstaking.Deadline.Num <= deadline {
			// found an entry that needs handling
			indexesForDelete = append(indexesForDelete, uint64(idx))
			//TODO: when this list is sorted just check the first elements until we reach future deadlines, instead of looping on it
			var receiverAddr sdk.AccAddress
			var err error
			receiverAddr, err = sdk.AccAddressFromBech32(unstakingEntry.Unstaking.Index)

			//A3. transfer stake money to the User account
			valid, err := verifySufficientAmountAndSendFromModuleToAddress(ctx, k, receiverAddr, unstakingEntry.Unstaking.Stake)
			if !valid {
				panic(fmt.Sprintf("error unstaking : %s", err))
			}
			eventAttributes := []sdk.Attribute{sdk.NewAttribute("user", receiverAddr.String()), sdk.NewAttribute("stake", unstakingEntry.Unstaking.Stake.String())}
			ctx.EventManager().EmitEvent(sdk.NewEvent("lava_user_unstake_commit", eventAttributes...))
		} else {
			// found an entry that isn't handled now, but later because its deadline isnt current block
			entryDeadline := unstakingEntry.Unstaking.Deadline.Num
			if entryDeadline < minDeadaline {
				minDeadaline = entryDeadline
			}
		}
	}
	//A2. remove the entry from UnstakingUsersAllSpecs, remove all the processed entries together (from the end not to affect the indexes, because we might remove more than 1)
	for idx := len(indexesForDelete) - 1; idx >= 0; idx-- {
		log.Warn().Msg(fmt.Sprintf("removing index: %d", indexesForDelete[idx]))
		k.RemoveUnstakingUsersAllSpecs(ctx, indexesForDelete[idx])
	}

	//A5. set new deadline for next callback
	if len(unstakingUsers)-len(indexesForDelete) == 0 {
		//no more deadlines, resolved all unstaking
		k.SetBlockDeadlineForCallback(ctx, types.BlockDeadlineForCallback{Deadline: types.BlockNum{Num: 0}})
	} else {
		// still some deadlines to go over, so set the closest one
		// and check sanity that deadlines are in the future
		if minDeadaline < uint64(ctx.BlockHeight()) || minDeadaline == uint64(math.MaxUint64) {
			panic(fmt.Sprintf("trying to set invalid next deadline! %d block height: %d unstaking count: %d, deleted indexes: %d \n unstaking Users: %s, length: %d\n", minDeadaline, uint64(ctx.BlockHeight()), k.GetUnstakingUsersAllSpecsCount(ctx), len(indexesForDelete), k.GetAllUnstakingUsersAllSpecs(ctx), len(k.GetAllUnstakingUsersAllSpecs(ctx))))
		}
		k.SetBlockDeadlineForCallback(ctx, types.BlockDeadlineForCallback{Deadline: types.BlockNum{Num: minDeadaline}})
	}
	return nil
}
