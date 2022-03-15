package keeper

import (
	"context"
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

func (k msgServer) StakeServicer(goCtx context.Context, msg *types.MsgStakeServicer) (*types.MsgStakeServicerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	specName := msg.Spec
	err := specName.ValidateBasic() //TODO: basic validation, we dont want to read the entire spec list here
	if err != nil {
		return nil, err
	}

	foundAndActive := k.Keeper.specKeeper.IsSpecFoundAndActive(ctx, specName.Name)
	if foundAndActive != true {
		return nil, errors.New("spec not found or not enabled")
	}
	//if we get here, the spec is active and supported
	//TODO: transfer stake from servicer
	specStakeStorage, found := k.Keeper.GetSpecStakeStorage(ctx, specName.Name)
	if found != true {
		//this is the first servicer for the supported spec
		// newSpecStakeStorage := types.SpecStakeStorage{
		// 	Index:        specName.Name,
		// 	StakeStorage: stakeStorage,
		// }
		// k.Keeper.SetSpecStakeStorage(ctx, newSpecStakeStorage)
		return nil, errors.New("specName not found in SpecStakeStorage")
	} else {
		stakeStorage := specStakeStorage.StakeStorage
		//TODO: find if it already exists
		// stakeStorage.Staked, types.StakeMap{
		// 	Index:    msg.Creator,
		// 	Stake:    msg.Amount,
		// 	Deadline: msg.Deadline,
		// }
	}
	// TODO: Handling the message
	_ = ctx

	return &types.MsgStakeServicerResponse{}, nil
}
