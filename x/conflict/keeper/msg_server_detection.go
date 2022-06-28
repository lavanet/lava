package keeper

import (
	"context"
	"strconv"
	"strings"

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
	} else if msg.FinalizationConflict == nil && msg.ResponseConflict == nil && msg.SameProviderConflict != nil {
		err := k.Keeper.ValidateSameProviderConflict(ctx, msg.SameProviderConflict, clientAddr)
		if err != nil {
			return nil, utils.LavaError(ctx, logger, "same_provider_conflict_detection", map[string]string{"client": msg.Creator, "error": err.Error()}, "same provider conflict detection error")
		}
	} else if msg.FinalizationConflict == nil && msg.ResponseConflict != nil && msg.SameProviderConflict == nil {
		err := k.Keeper.ValidateResponseConflict(ctx, msg.ResponseConflict, clientAddr)
		if err != nil {
			return nil, utils.LavaError(ctx, logger, "response_conflict_detection", map[string]string{"client": msg.Creator, "error": err.Error()}, "response conflict detection error")
		}
	}

	//the conflict detection transaction is valid!, start a vote
	//TODO: 1. start a vote, with vote ID (unique, list index isn't good because its changing, use a map)
	//2. create an event to declare vote
	//3. accept incoming commit transactions for this vote,
	//4. after vote ends, accept reveal transactions, strike down every provider that voted (only valid if there was a commit)
	//5. majority wins, minority gets penalised
	index := k.Keeper.AllocateNewConflictVote(ctx)
	conflictVote := types.ConflictVote{}
	conflictVote.Index = index
	conflictVote.VoteState = types.Commit
	conflictVote.VoteStartBlock = ctx.BlockHeight()
	//conflictVote.VoteDeadline = ??
	conflictVote.ApiUrl = msg.ResponseConflict.ConflictRelayData0.Request.ApiUrl
	conflictVote.ChainID = msg.ResponseConflict.ConflictRelayData0.Request.ChainID
	conflictVote.RequestBlock = msg.ResponseConflict.ConflictRelayData0.Request.RequestBlock
	conflictVote.RequestData = msg.ResponseConflict.ConflictRelayData0.Request.Data

	conflictVote.FirstProvider.Account = msg.ResponseConflict.ConflictRelayData0.Request.Provider
	conflictVote.FirstProvider.Response = msg.ResponseConflict.ConflictRelayData0.Reply.Data
	conflictVote.SecondProvider.Account = msg.ResponseConflict.ConflictRelayData1.Request.Provider
	conflictVote.SecondProvider.Response = msg.ResponseConflict.ConflictRelayData1.Reply.Data
	conflictVote.VotersHash = map[string]types.Vote{}
	voters := k.Keeper.LotteryVoters(goCtx, conflictVote.ChainID)
	for _, voter := range voters {
		conflictVote.VotersHash[voter] = types.Vote{Hash: []byte{}}
	}

	k.SetConflictVote(ctx, conflictVote)

	eventData := map[string]string{"client": msg.Creator}
	eventData["voteID"] = conflictVote.Index
	eventData["chainID"] = conflictVote.ChainID
	eventData["apiURL"] = conflictVote.ApiUrl
	eventData["requestData"] = string(conflictVote.RequestData)
	eventData["requestBlock"] = strconv.FormatInt(conflictVote.RequestBlock, 10)
	eventData["voteDeadline"] = strconv.FormatInt(conflictVote.VoteDeadline, 10)
	eventData["voters"] = strings.Join(voters, ",")

	utils.LogLavaEvent(ctx, logger, "response_conflict_detection", eventData, "Got a new valid conflict detection from consumer, starting new vote")
	return &types.MsgDetectionResponse{}, nil
}

func (k Keeper) LotteryVoters(goCtx context.Context, chainID string) []string {
	ctx := sdk.UnwrapSDKContext(goCtx)
	epochStart, _ := k.epochstorageKeeper.GetEpochStartForBlock(ctx, uint64(ctx.BlockHeight()))
	entries, err := k.epochstorageKeeper.GetStakeEntryForAllProvidersEpoch(ctx, chainID, epochStart)

	if err != nil {
		return make([]string, 0)
	}

	voters := make([]string, 0)
	for _, entry := range *entries {
		voters = append(voters, entry.Address)
	}
	return voters
}
