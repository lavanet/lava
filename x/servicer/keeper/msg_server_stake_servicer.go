package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/servicer/types"
)

func (k msgServer) StakeServicer(goCtx context.Context, msg *types.MsgStakeServicer) (*types.MsgStakeServicerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx)
	specName := msg.Spec
	err := specName.ValidateBasic() //TODO: basic validation, we dont want to read the entire spec list here
	if err != nil {
		return nil, err
	}

	foundAndActive, _, _ := k.Keeper.specKeeper.IsSpecFoundAndActive(ctx, specName.Name)
	if !foundAndActive {
		details := map[string]string{"spec": specName.Name}
		return nil, utils.LavaError(ctx, logger, "stake_servicer_spec", details, "spec not found or not active")
	}
	//if we get here, the spec is active and supported
	if msg.Amount.IsLT(k.Keeper.GetMinStake(ctx)) {
		details := map[string]string{"servicer": msg.Creator, "stake": msg.Amount.String(), "minStake": k.Keeper.GetMinStake(ctx).String()}
		return nil, utils.LavaError(ctx, logger, "stake_servicer_amount", details, "invalid servicer address")
	}
	senderAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		details := map[string]string{"servicer": msg.Creator, "error": err.Error()}
		return nil, utils.LavaError(ctx, logger, "stake_servicer_addr", details, "invalid servicer address")
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
	for _, operatorAddressData := range msg.OperatorAddresses {
		operatorAddr := types.OperatorAddress{Address: operatorAddressData}
		err := operatorAddr.ValidateBasic()
		if err != nil {
			details := map[string]string{"operator": operatorAddressData, "error": err.Error()}
			return nil, utils.LavaError(ctx, logger, "stake_servicer_operator", details, "invalid operator address")
		}
	}

	specStakeStorage, found := k.Keeper.GetSpecStakeStorage(ctx, specName.Name)
	if !found {

		//this is the first servicer for the supported spec
		stakeStorage := types.StakeStorage{
			Staked: []types.StakeMap{},
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
	for _, storageMap := range stakeStorage.Staked {
		if storageMap.Index == msg.Creator {
			// already exists
			details := map[string]string{"servicer": senderAddr.String(), "deadline": strconv.FormatUint(blockDeadline.Num, 10), "stake": msg.Amount.String(), "requestedDeadline": strconv.FormatUint(msg.Deadline.Num, 10)}
			if storageMap.Stake.IsLT(msg.Amount) {
				// increasing stake is allowed
				if storageMap.Deadline.Num >= blockDeadline.Num {
					//lowering the deadline is allowed
					valid, err := verifySufficientAmountAndSendToModule(ctx, k, senderAddr, msg.Amount.Sub(storageMap.Stake))
					if !valid {
						details["error"] = err.Error()
						details["needed_stake"] = msg.Amount.Sub(storageMap.Stake).String()
						return nil, utils.LavaError(ctx, logger, "stake_servicer_amount", details, "insufficient funds to pay for difference in stake")
					}
					storageMap.Stake = msg.Amount
					storageMap.Deadline = blockDeadline
					storageMap.OperatorAddresses = msg.OperatorAddresses
					entryExists = true

					utils.LogLavaEvent(ctx, logger, "servicer_stake_update", details, "Changing Staked Servicer")
					break
				}

				return nil, utils.LavaError(ctx, logger, "stake_servicer_deadline", details, "can't increase deadline for existing servicer")
			}
			return nil, utils.LavaError(ctx, logger, "stake_servicer_stake", details, "can't decrease stake for existing servicer")
		}
	}
	if !entryExists {
		// servicer isn't staked so add him
		// new staking takes effect from the next block
		if blockDeadline.Num <= uint64(ctx.BlockHeight())+1 {
			blockDeadline.Num = uint64(ctx.BlockHeight()) + 1
		}
		details := map[string]string{"servicer": senderAddr.String(), "deadline": strconv.FormatUint(blockDeadline.Num, 10), "stake": msg.Amount.String(), "requestedDeadline": strconv.FormatUint(msg.Deadline.Num, 10)}
		valid, err := verifySufficientAmountAndSendToModule(ctx, k, senderAddr, msg.Amount)
		if !valid {
			details["error"] = err.Error()
			return nil, utils.LavaError(ctx, logger, "stake_servicer_amount", details, "insufficient amount to pay for stake")
		}

		stakeStorage.Staked = append(stakeStorage.Staked, types.StakeMap{
			Index:             msg.Creator,
			Stake:             msg.Amount,
			Deadline:          blockDeadline,
			OperatorAddresses: msg.OperatorAddresses,
		})

		utils.LogLavaEvent(ctx, logger, "servicer_stake_new", details, "Adding Staked Servicer")
	}
	k.Keeper.SetSpecStakeStorage(ctx, specStakeStorage)
	return &types.MsgStakeServicerResponse{}, nil
}
