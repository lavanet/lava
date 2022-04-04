package keeper

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/user/types"
)

func (k msgServer) StakeUser(goCtx context.Context, msg *types.MsgStakeUser) (*types.MsgStakeUserResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Keeper.Logger(ctx)
	specName := msg.Spec
	err := specName.ValidateBasic() //TODO: basic validation, we dont want to read the entire spec list here
	if err != nil {
		return nil, err
	}

	foundAndActive, _, _ := k.Keeper.specKeeper.IsSpecFoundAndActive(ctx, specName.Name)
	if !foundAndActive {
		return nil, errors.New("spec not found or not enabled")
	}
	//if we get here, the spec is active and supported
	if msg.Amount.IsLT(k.Keeper.GetMinStake(ctx)) {
		return nil, fmt.Errorf("insufficient stake amount, amount must be %s or greater", k.Keeper.GetMinStake(ctx))
	}
	senderAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address %s error: %s", msg.Creator, err)
	}
	//define the function here for later use
	verifySufficientAmountAndSendToModule := func(ctx sdk.Context, k msgServer, addr sdk.AccAddress, neededAmount sdk.Coin) (bool, error) {
		if k.Keeper.bankKeeper.GetBalance(ctx, addr, "stake").IsLT(neededAmount) {
			return false, fmt.Errorf("insufficient balance for staking %s current balance: %s", neededAmount, k.Keeper.bankKeeper.GetBalance(ctx, addr, "stake"))
		}
		err := k.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, []sdk.Coin{neededAmount})
		if err != nil {
			return false, fmt.Errorf("invalid transfer coins to module, %s", err)
		}
		return true, nil
	}

	// }
	specStakeStorage, found := k.Keeper.GetSpecStakeStorage(ctx, specName.Name)
	if !found {

		//this is the first User for the supported spec
		stakeStorage := types.StakeStorage{
			StakedUsers: []types.UserStake{},
		}
		specStakeStorage = types.SpecStakeStorage{
			Index:        specName.Name,
			StakeStorage: &stakeStorage,
		}
		// newSpecStakeStorage :=
		// k.Keeper.SetSpecStakeStorage(ctx, newSpecStakeStorage)
	}
	stakeStorage := specStakeStorage.StakeStorage
	entryExists := false
	blockDeadline := msg.Deadline
	//TODO: improve the finding logic and the way its saved looping a list is slow and bad
	for _, userStake := range stakeStorage.StakedUsers {
		if userStake.Index == msg.Creator {
			// already exists
			if userStake.Stake.IsLT(msg.Amount) {
				// increasing stake is allowed
				if userStake.Deadline.Num >= blockDeadline.Num {
					//lowering the deadline is allowed
					valid, err := verifySufficientAmountAndSendToModule(ctx, k, senderAddr, msg.Amount.Sub(userStake.Stake))
					if !valid {
						return nil, fmt.Errorf("error updating stake: %s", err)
					}
					userStake.Stake = msg.Amount
					userStake.Deadline = blockDeadline
					entryExists = true
					details := map[string]string{"user": senderAddr.String(), "deadline": strconv.FormatUint(blockDeadline.Num, 10), "stake": msg.Amount.String(), "requestedDeadline": strconv.FormatUint(msg.Deadline.Num, 10)}
					utils.LogLavaEvent(ctx, logger, "user_stake_update", details, "Existing Staked User modified")

					break
				}
				return nil, errors.New("can't increase deadline for existing User")
			}
			return nil, errors.New("can't decrease stake for existing User")
		}
	}
	if !entryExists {
		// User isn't staked so add him

		// staking takes effect from the next block
		if blockDeadline.Num <= uint64(ctx.BlockHeight())+1 {
			blockDeadline.Num = uint64(ctx.BlockHeight()) + 1
		}

		valid, err := verifySufficientAmountAndSendToModule(ctx, k, senderAddr, msg.Amount)
		if !valid {
			return nil, fmt.Errorf("error staking: %s", err)
		}

		stakeStorage.StakedUsers = append(stakeStorage.StakedUsers, types.UserStake{
			Index:    msg.Creator,
			Stake:    msg.Amount,
			Deadline: blockDeadline,
		})
		details := map[string]string{"user": senderAddr.String(), "deadline": strconv.FormatUint(blockDeadline.Num, 10), "stake": msg.Amount.String(), "requestedDeadline": strconv.FormatUint(msg.Deadline.Num, 10)}
		utils.LogLavaEvent(ctx, logger, "user_stake_new", details, "Adding Staked User")
	}
	k.Keeper.SetSpecStakeStorage(ctx, specStakeStorage)
	_ = ctx

	return &types.MsgStakeUserResponse{}, nil
}
