package keeper

import (
	"errors"
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
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
	if deadline.Deadline.Num != uint64(currentBlock) { // didn't reach the first deadline
		return nil
	}
	err := k.creditUnstakingServicersAndRemoveFromCallback(ctx, deadline.Deadline)
	return err
}

func (k Keeper) creditUnstakingServicersAndRemoveFromCallback(ctx sdk.Context, deadline *types.BlockNum) error {
	unstakingServicers := k.GetAllUnstakingServicersAllSpecs(ctx)
	minDeadaline := uint64(math.MaxUint64)
	indexesForDelete := make([]uint64, 0)
	//handlng an entry needs a few things done:
	//A1. remove the entry from SpecStakeStorage.StakeStorage.Unstaking
	//A2. remove the entry from UnstakingServicersAllSpecs
	//A3. transfer money to the servicer account
	//A4. repeat for all entries with deadline
	//A5. set new deadline for next callback

	verifySufficientAmountAndSendFromModuleToAddress := func(ctx sdk.Context, k Keeper, addr sdk.AccAddress, neededAmount sdk.Coin) (bool, error) {
		moduleBalance := k.bankKeeper.GetBalance(ctx, k.accountKeeper.GetModuleAddress(types.ModuleName), "stake")
		if moduleBalance.IsLT(neededAmount) {
			return false, errors.New(fmt.Sprintf("insufficient balance for unstaking %s current balance: %s", neededAmount, moduleBalance))
		}
		err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, []sdk.Coin{neededAmount})
		if err != nil {
			return false, errors.New(fmt.Sprintf("invalid transfer coins from module, %s to account %s", err, addr))
		}
		return true, nil
	}

	for idx, unstakingEntry := range unstakingServicers {
		//A4. repeat for all entries with deadline
		if unstakingEntry.Unstaking.Deadline.Num == deadline.Num {
			// found an entry that needs handling
			found_matching_entry := false
			indexesForDelete = append(indexesForDelete, uint64(idx))
			//TODO: when this list is sorted just check the first element, instead of looping on it
			var receiverAddr sdk.AccAddress
			var err error
			//A1. remove the entry from SpecStakeStorage.StakeStorage.Unstaking
			for idx2, currentStakeMap := range unstakingEntry.SpecStakeStorage.StakeStorage.Unstaking {
				if currentStakeMap == *unstakingEntry.Unstaking {
					found_matching_entry = true
					// effeciently delete storageMap from stakeStorage.Staked
					receiverAddr, err = sdk.AccAddressFromBech32(currentStakeMap.Index)
					if err != nil {
						panic(fmt.Sprintf("invalid address for storage %s error: %s", currentStakeMap.Index, err))
					}
					unstakingEntry.SpecStakeStorage.StakeStorage.Unstaking[idx2] = unstakingEntry.SpecStakeStorage.StakeStorage.Unstaking[len(unstakingEntry.SpecStakeStorage.StakeStorage.Unstaking)-1] // replace the element at delete index with the last one
					unstakingEntry.SpecStakeStorage.StakeStorage.Unstaking = unstakingEntry.SpecStakeStorage.StakeStorage.Unstaking[:len(unstakingEntry.SpecStakeStorage.StakeStorage.Unstaking)-1]      // remove last element
					break
				}
			}
			if !found_matching_entry {
				panic("mismatch in data structures: didnt find a stake map in the container that fits the unstaking entry")
			}

			//A3. transfer stake money to the servicer account
			valid, err := verifySufficientAmountAndSendFromModuleToAddress(ctx, k, receiverAddr, unstakingEntry.Unstaking.Stake)
			if !valid {
				panic(fmt.Sprintf("error unstaking : %s", err))
			}

			//update the removal in keeper
			k.SetSpecStakeStorage(ctx, *unstakingEntry.SpecStakeStorage)
		} else {
			// found an entry that isn't handled now, but later because its deadline isnt current block
			entryDeadline := unstakingEntry.Unstaking.Deadline.Num
			if entryDeadline < minDeadaline {
				minDeadaline = entryDeadline
			}
		}
	}
	//A2. remove the entry from UnstakingServicersAllSpecs, remove all the processed entries together (from the end not to affect the indexes, because we might remove more than 1)
	for idx := len(indexesForDelete) - 1; idx >= 0; idx-- {
		k.RemoveUnstakingServicersAllSpecs(ctx, indexesForDelete[idx])
	}

	//A5. set new deadline for next callback
	if k.GetUnstakingServicersAllSpecsCount(ctx) == 0 {
		//no more deadlines, resolved all unstaking
		k.SetBlockDeadlineForCallback(ctx, types.BlockDeadlineForCallback{Deadline: &types.BlockNum{Num: 0}})
	} else {
		// still some deadlines to go over, so set the closest one
		// and check sanity that deadlines are in the future
		if minDeadaline < uint64(ctx.BlockHeight()) || minDeadaline == uint64(math.MaxUint64) {
			panic(fmt.Sprintf("trying to set invalid next deadline! %d block height: %d", minDeadaline, uint64(ctx.BlockHeight())))
		}
		k.SetBlockDeadlineForCallback(ctx, types.BlockDeadlineForCallback{Deadline: &types.BlockNum{Num: minDeadaline}})
	}
	return nil
}
