package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/conflict/types"
)

func (k msgServer) Detection(goCtx context.Context, msg *types.MsgDetection) (*types.MsgDetectionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Keeper.Logger(ctx)
	clientAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, utils.LavaError(ctx, logger, "conflict_detection", map[string]string{"client": msg.Creator, "error": err.Error()}, "parsing client address")
	}
	if msg.FinalizationConflict != nil && msg.ResponseConflict == nil && msg.SameProviderConflict == nil {
		err := k.Keeper.ValidateFinalizationConflict(ctx, msg.FinalizationConflict, clientAddr)
		if err != nil {
			return nil, utils.LavaError(ctx, logger, "Finalization_conflict_detection", map[string]string{"client": msg.Creator, "error": err.Error()}, "finalization conflict detection error")
		}
	} else if msg.FinalizationConflict == nil && msg.ResponseConflict != nil && msg.SameProviderConflict == nil {
		err := k.Keeper.ValidateResponseConflict(ctx, msg.ResponseConflict, clientAddr)
		if err != nil {
			return nil, utils.LavaError(ctx, logger, "response_conflict_detection", map[string]string{"client": msg.Creator, "error": err.Error()}, "response conflict detection error")
		}
		//the conflict detection transaction is valid!, start a vote
		//TODO: 1. start a vote, with vote ID (unique, list index isn't good because its changing, use a map)
		//2. create an event to declare vote
		//3. accept incoming commit transactions for this vote,
		//4. after vote ends, accept reveal transactions, strike down every provider that voted (only valid if there was a commit)
		//5. majority wins, minority gets penalised
	} else if msg.FinalizationConflict == nil && msg.ResponseConflict == nil && msg.SameProviderConflict != nil {
		err := k.Keeper.ValidateSameProviderConflict(ctx, msg.SameProviderConflict, clientAddr)
		if err != nil {
			return nil, utils.LavaError(ctx, logger, "same_provider_conflict_detection", map[string]string{"client": msg.Creator, "error": err.Error()}, "same provider conflict detection error")
		}
	}
	utils.LogLavaEvent(ctx, logger, "conflict_detection", map[string]string{"client": msg.Creator}, "Got a new valid conflict detection from consumer")
	return &types.MsgDetectionResponse{}, nil
}
