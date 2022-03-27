package keeper

import (
	"context"
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/user/types"
)

func (k msgServer) UnstakeUser(goCtx context.Context, msg *types.MsgUnstakeUser) (*types.MsgUnstakeUserResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	specName := msg.Spec
	err := specName.ValidateBasic() //TODO: basic validation, we dont want to read the entire spec list here
	if err != nil {
		return nil, err
	}

	// we can unstake disabled specs, but not missing ones
	_, found, _ := k.Keeper.specKeeper.IsSpecFoundAndActive(ctx, specName.Name)
	if !found {
		return nil, errors.New("spec not found, can't unstake")
	}
	unstakingUser := msg.Creator
	err = k.Keeper.UnstakeUser(ctx, *specName, unstakingUser, *msg.Deadline)
	return &types.MsgUnstakeUserResponse{}, err
}
