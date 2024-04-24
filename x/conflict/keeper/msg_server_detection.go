package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/conflict/types"
	pairingfilters "github.com/lavanet/lava/x/pairing/keeper/filters"
	"golang.org/x/exp/slices"
)

func DetectionIndex(creatorAddr string, conflict *types.ResponseConflict, epochStart uint64) string {
	return creatorAddr + conflict.ConflictRelayData0.Request.RelaySession.Provider + conflict.ConflictRelayData1.Request.RelaySession.Provider + strconv.FormatUint(epochStart, 10)
}

func (k msgServer) Detection(goCtx context.Context, msg *types.MsgDetection) (*types.MsgDetectionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Keeper.Logger(ctx)
	clientAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, utils.LavaFormatWarning("invalid client address", err,
			utils.Attribute{Key: "client", Value: msg.Creator},
		)
	}
	switch msg.Conflict.(type) {
	case *types.MsgDetection_FinalizationConflict:
		conflict := msg.GetFinalizationConflict()
		if conflict == nil {
			return nil, utils.LavaFormatWarning("finalization conflict is nil", nil)
		}

		if conflict.RelayFinalization_0 == nil || conflict.RelayFinalization_1 == nil {
			return nil, utils.LavaFormatWarning("conflict finalization is nil", nil)
		}

		if conflict.RelayFinalization_0.RelaySession == nil || conflict.RelayFinalization_1.RelaySession == nil {
			return nil, utils.LavaFormatWarning("conflict relay session is nil", nil)
		}

		if conflict.RelayFinalization_0.RelaySession.Provider == conflict.RelayFinalization_1.RelaySession.Provider {
			eventData, err := k.handleSameProviderFinalizationConflict(ctx, conflict, clientAddr)
			if err != nil {
				return nil, err
			}

			utils.LogLavaEvent(ctx, logger, types.ConflictDetectionSameProviderEventName, eventData, "Simulation: Got a new valid conflict detection from consumer on same provider")
		} else {
			eventData, err := k.handleTwoProvidersFinalizationConflict(ctx, conflict, clientAddr)
			if err != nil {
				return nil, err
			}

			utils.LogLavaEvent(ctx, logger, types.ConflictDetectionTwoProvidersEventName, eventData, "Simulation: Got a new valid conflict detection from consumer on two providers")
		}

		return &types.MsgDetectionResponse{}, nil
	case *types.MsgDetection_ResponseConflict:
		eventData, err := k.handleResponseConflict(ctx, goCtx, msg.GetResponseConflict(), clientAddr)
		if err != nil {
			return nil, err
		}

		utils.LogLavaEvent(ctx, logger, types.ConflictVoteDetectionEventName, eventData, "Simulation: Got a new valid conflict detection from consumer, starting new vote")
		return &types.MsgDetectionResponse{}, nil
	default:
		return nil, utils.LavaFormatWarning("invalid conflict type", nil, utils.LogAttr("conflict", fmt.Sprintf("%+v", msg.Conflict)))
	}
}

func (k Keeper) LotteryVoters(goCtx context.Context, epoch uint64, chainID string, exemptions []string) []string {
	ctx := sdk.UnwrapSDKContext(goCtx)
	entries, err := k.epochstorageKeeper.GetStakeEntryForAllProvidersEpoch(ctx, chainID, epoch)
	if err != nil {
		return make([]string, 0)
	}

	freezeFilter := pairingfilters.FrozenProvidersFilter{}
	frozenProviders := freezeFilter.Filter(ctx, *entries, epoch) // bool array -> true == not frozen

	voters := make([]string, 0)
	for i, entry := range *entries {
		if !slices.Contains(exemptions, entry.Address) && frozenProviders[i] {
			voters = append(voters, entry.Address)
		}
	}

	return voters
}

func (k msgServer) handleTwoProvidersFinalizationConflict(ctx sdk.Context, conflict *types.FinalizationConflict, clientAddr sdk.AccAddress) (eventData map[string]string, err error) {
	err = k.Keeper.ValidateFinalizationConflict(ctx, conflict, clientAddr)
	if err != nil {
		return nil, utils.LavaFormatWarning("Simulation: invalid finalization conflict detection", err,
			utils.LogAttr("client", clientAddr.String()),
		)
	}

	eventData = map[string]string{"client": clientAddr.String()}
	eventData["chainID"] = conflict.RelayFinalization_0.RelaySession.SpecId
	eventData["provider0"] = fmt.Sprintf("%+v", conflict.RelayFinalization_0.RelaySession.Provider)
	eventData["provider1"] = fmt.Sprintf("%+v", conflict.RelayFinalization_1.RelaySession.Provider)
	// eventData["mismatching_block_height"] = fmt.Sprintf("%+v", mismatchingBlockHeight)
	// eventData["mismatching_block_hashes"] = fmt.Sprintf("%+v", mismatchingBlockHashes)

	return eventData, nil
}

func (k msgServer) handleSameProviderFinalizationConflict(ctx sdk.Context, conflict *types.FinalizationConflict, clientAddr sdk.AccAddress) (eventData map[string]string, err error) {
	providerAddress, mismatchingBlockHeight, mismatchingBlockHashes, err := k.Keeper.ValidateSameProviderConflict(ctx, conflict, clientAddr)
	if err != nil {
		return nil, utils.LavaFormatWarning("Simulation: invalid same provider conflict detection", err,
			utils.LogAttr("client", clientAddr.String()),
		)
	}

	eventData = map[string]string{"client": clientAddr.String()}
	eventData["chainID"] = conflict.RelayFinalization_0.RelaySession.SpecId
	eventData["provider"] = fmt.Sprintf("%+v", providerAddress)
	eventData["mismatching_block_height"] = fmt.Sprintf("%+v", mismatchingBlockHeight)
	eventData["mismatching_block_hashes"] = fmt.Sprintf("%+v", mismatchingBlockHashes)

	return eventData, nil
}

func (k msgServer) handleResponseConflict(ctx sdk.Context, goCtx context.Context, conflict *types.ResponseConflict, clientAddr sdk.AccAddress) (eventData map[string]string, err error) {
	err = k.Keeper.ValidateResponseConflict(ctx, conflict, clientAddr)
	if err != nil {
		return nil, utils.LavaFormatWarning("Simulation: invalid response conflict detection", err,
			utils.LogAttr("client", clientAddr.String()),
		)
	}

	// the conflict detection transaction is valid!, start a vote
	// TODO: 1. start a vote, with vote ID (unique, list index isn't good because its changing, use a map)
	// 2. create an event to declare vote
	// 3. accept incoming commit transactions for this vote,
	// 4. after vote ends, accept reveal transactions, strike down every provider that voted (only valid if there was a commit)
	// 5. majority wins, minority gets penalised
	epochStart, _, err := k.epochstorageKeeper.GetEpochStartForBlock(ctx, uint64(conflict.ConflictRelayData0.Request.RelaySession.Epoch))
	if err != nil {
		return nil, utils.LavaFormatWarning("Simulation: could not get EpochStart for specific block", err,
			utils.LogAttr("client", clientAddr.String()),
			utils.LogAttr("provider0", conflict.ConflictRelayData0.Request.RelaySession.Provider),
			utils.LogAttr("provider1", conflict.ConflictRelayData1.Request.RelaySession.Provider),
		)
	}
	index := DetectionIndex(clientAddr.String(), conflict, epochStart)
	found := k.Keeper.AllocateNewConflictVote(ctx, index)
	if found {
		return nil, utils.LavaFormatWarning("Simulation: conflict with is already open for this client and providers in this epoch", err,
			utils.LogAttr("client", clientAddr.String()),
			utils.LogAttr("provider0", conflict.ConflictRelayData0.Request.RelaySession.Provider),
			utils.LogAttr("provider1", conflict.ConflictRelayData1.Request.RelaySession.Provider),
		)
	}
	conflictVote := types.ConflictVote{}
	conflictVote.Index = index
	conflictVote.VoteState = types.StateCommit
	conflictVote.VoteStartBlock = uint64(conflict.ConflictRelayData0.Request.RelaySession.Epoch)
	epochBlocks, err := k.epochstorageKeeper.EpochBlocks(ctx, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, utils.LavaFormatError("Simulation: could not get epochblocks", err,
			utils.LogAttr("client", clientAddr.String()),
			utils.LogAttr("provider0", conflict.ConflictRelayData0.Request.RelaySession.Provider),
			utils.LogAttr("provider1", conflict.ConflictRelayData1.Request.RelaySession.Provider),
		)
	}

	voteDeadline, err := k.Keeper.epochstorageKeeper.GetNextEpoch(ctx, uint64(ctx.BlockHeight())+k.VotePeriod(ctx)*epochBlocks)
	if err != nil {
		return nil, utils.LavaFormatError("Simulation: could not get NextEpoch", err,
			utils.LogAttr("client", clientAddr.String()),
			utils.LogAttr("provider0", conflict.ConflictRelayData0.Request.RelaySession.Provider),
			utils.LogAttr("provider1", conflict.ConflictRelayData1.Request.RelaySession.Provider),
		)
	}
	conflictVote.VoteDeadline = voteDeadline
	conflictVote.ApiUrl = conflict.ConflictRelayData0.Request.RelayData.ApiUrl
	conflictVote.ClientAddress = clientAddr.String()
	conflictVote.ChainID = conflict.ConflictRelayData0.Request.RelaySession.SpecId
	conflictVote.RequestBlock = uint64(conflict.ConflictRelayData0.Request.RelayData.RequestBlock)
	conflictVote.RequestData = conflict.ConflictRelayData0.Request.RelayData.Data

	conflictVote.FirstProvider.Account = conflict.ConflictRelayData0.Request.RelaySession.Provider
	conflictVote.FirstProvider.Response = conflict.ConflictRelayData0.Reply.HashAllDataHash
	conflictVote.SecondProvider.Account = conflict.ConflictRelayData1.Request.RelaySession.Provider
	conflictVote.SecondProvider.Response = conflict.ConflictRelayData1.Reply.HashAllDataHash
	conflictVote.Votes = []types.Vote{}
	voters := k.Keeper.LotteryVoters(goCtx, epochStart, conflictVote.ChainID, []string{conflictVote.FirstProvider.Account, conflictVote.SecondProvider.Account})
	for _, voter := range voters {
		conflictVote.Votes = append(conflictVote.Votes, types.Vote{Address: voter, Hash: []byte{}, Result: types.NoVote})
	}
	metadataBytes, err := json.Marshal(conflict.ConflictRelayData0.Request.RelayData.Metadata)
	if err != nil {
		return nil, utils.LavaFormatError("could not marshal metadata in the event", err,
			utils.LogAttr("client", clientAddr.String()),
			utils.LogAttr("provider0", conflict.ConflictRelayData0.Request.RelaySession.Provider),
			utils.LogAttr("provider1", conflict.ConflictRelayData1.Request.RelaySession.Provider),
		)
	}

	k.SetConflictVote(ctx, conflictVote)

	eventData = map[string]string{"client": clientAddr.String()}
	eventData["voteID"] = conflictVote.Index
	eventData["chainID"] = conflictVote.ChainID
	eventData["connectionType"] = conflict.ConflictRelayData0.Request.RelayData.ConnectionType
	eventData["apiURL"] = conflictVote.ApiUrl
	eventData["requestData"] = string(conflictVote.RequestData)
	eventData["requestBlock"] = strconv.FormatUint(conflictVote.RequestBlock, 10)
	eventData["voteDeadline"] = strconv.FormatUint(conflictVote.VoteDeadline, 10)
	eventData["voters"] = strings.Join(voters, ",")
	eventData["apiInterface"] = conflict.ConflictRelayData0.Request.RelayData.ApiInterface
	eventData["metadata"] = string(metadataBytes)

	return eventData, nil
}
