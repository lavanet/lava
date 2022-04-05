package keeper

import (
	"context"
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
		details := map[string]string{"spec": specName.Name}
		return nil, utils.LavaError(ctx, logger, "stake_user_spec", details, "spec not found or not active")
	}
	//if we get here, the spec is active and supported
	if msg.Amount.IsLT(k.Keeper.GetMinStake(ctx)) {
		details := map[string]string{"user": msg.Creator, "stake": msg.Amount.String(), "minStake": k.Keeper.GetMinStake(ctx).String()}
		return nil, utils.LavaError(ctx, logger, "stake_user_amount", details, "invalid user address")
	}
	senderAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		details := map[string]string{"user": msg.Creator, "error": err.Error()}
		return nil, utils.LavaError(ctx, logger, "stake_user_addr", details, "invalid user address")
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
	blockDeadline := msg.Deadline.Num
	//TODO: improve the finding logic and the way its saved looping a list is slow and bad
	for _, userStake := range stakeStorage.StakedUsers {
		if userStake.Index == msg.Creator {
			// already exists
			details := map[string]string{"user": senderAddr.String(), "deadline": strconv.FormatUint(blockDeadline, 10), "stake": msg.Amount.String(), "requestedDeadline": strconv.FormatUint(msg.Deadline.Num, 10)}
			if userStake.Stake.IsLT(msg.Amount) {
				// increasing stake is allowed
				if userStake.Deadline.Num >= blockDeadline {
					//lowering the deadline is allowed
					valid, err := verifySufficientAmountAndSendToModule(ctx, k, senderAddr, msg.Amount.Sub(userStake.Stake))
					if !valid {
						details["error"] = err.Error()
						details["needed_stake"] = msg.Amount.Sub(userStake.Stake).String()
						return nil, utils.LavaError(ctx, logger, "stake_user_amount", details, "insufficient funds to pay for difference in stake")
					}
					userStake.Stake = msg.Amount
					userStake.Deadline = types.BlockNum{Num: blockDeadline}
					entryExists = true
					utils.LogLavaEvent(ctx, logger, "user_stake_update", details, "Existing Staked User modified")

					break
				}
				return nil, utils.LavaError(ctx, logger, "stake_user_deadline", details, "can't increase deadline for existing user")
			}
			return nil, utils.LavaError(ctx, logger, "stake_user_stake", details, "can't decrease stake for existing User")
		}
	}
	if !entryExists {
		// User isn't staked so add him

		// staking takes effect from the next block
		if blockDeadline <= uint64(ctx.BlockHeight())+1 {
			blockDeadline = uint64(ctx.BlockHeight()) + 1
		}
		details := map[string]string{"user": senderAddr.String(), "deadline": strconv.FormatUint(blockDeadline, 10), "stake": msg.Amount.String(), "requestedDeadline": strconv.FormatUint(msg.Deadline.Num, 10)}

		valid, err := verifySufficientAmountAndSendToModule(ctx, k, senderAddr, msg.Amount)
		if !valid {
			details["error"] = err.Error()
			return nil, utils.LavaError(ctx, logger, "stake_user_amount", details, "insufficient amount to pay for stake")
		}

		stakeStorage.StakedUsers = append(stakeStorage.StakedUsers, types.UserStake{
			Index:    msg.Creator,
			Stake:    msg.Amount,
			Deadline: types.BlockNum{Num: blockDeadline},
		})
		utils.LogLavaEvent(ctx, logger, "user_stake_new", details, "Adding Staked User")
	}
	k.Keeper.SetSpecStakeStorage(ctx, specStakeStorage)
	_ = ctx

	return &types.MsgStakeUserResponse{}, nil
}
