package keeper

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/conflict/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"golang.org/x/exp/slices"
)

const (
	BailStakeDiv = 5 //20%
	MajorityDiv  = 2 //50%
)

var SlashStakePercent = sdk.NewDecWithPrec(5, 2) //0.05

func (k Keeper) AllocateNewConflictVote(ctx sdk.Context, key string) bool {
	_, found := k.GetConflictVote(ctx, key)

	return found
}

func (k Keeper) CheckAndHandleAllVotes(ctx sdk.Context) {
	if k.IsEpochStart(ctx) {
		conflictVotes := k.GetAllConflictVote(ctx)
		for _, conflictVote := range conflictVotes {
			if conflictVote.VoteDeadline <= uint64(ctx.BlockHeight()) {
				switch conflictVote.VoteState {
				case types.StateCommit:
					k.TransitionVoteToReveal(ctx, conflictVote)
				case types.StateReveal:
					k.HandleAndCloseVote(ctx, conflictVote)
				}
			}
		}
	}
}

func (k Keeper) HandleAndCloseVote(ctx sdk.Context, ConflictVote types.ConflictVote) {
	logger := k.Logger(ctx)
	eventData := map[string]string{"voteID": ConflictVote.Index}
	var eventName string
	//all wrong voters are punished
	//add stake as wieght
	//valid only if one of the votes is bigger than 50% from total
	//punish providers that didnt vote - discipline/jail + bail = 20%stake + slash 5%stake
	//(dont add jailed providers to voters)
	//if strong majority punish wrong providers - jail from start of memory to end + slash 100%stake
	//reward pool is the slashed amount from all punished providers
	//reward to stake - client 50%, the original provider 10%, 20% the voters
	totalVotes := sdk.ZeroInt()
	firstProviderVotes := sdk.ZeroInt()
	secondProviderVotes := sdk.ZeroInt()
	noneProviderVotes := sdk.ZeroInt()
	var providersWithoutVote []string
	rewardPool := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.ZeroInt())
	rewardCount := sdk.ZeroInt()
	votersStake := map[string]sdk.Int{} //this is needed in order to give rewards for each voter according to their stake(so we dont take this data twice from the keeper)
	ConsensusVote := true
	var majorityMet bool

	var winner int64
	var winnersAddr string
	var winnerVotersStake sdk.Int

	//count votes and punish jury that didnt vote
	epochVoteStart, _, err := k.epochstorageKeeper.GetEpochStartForBlock(ctx, ConflictVote.VoteStartBlock) //TODO check if we need to check for overlap
	if err != nil {
		k.CleanUpVote(ctx, ConflictVote.Index)
		utils.LavaError(ctx, logger, "vote_epoch_get_fail", map[string]string{"voteID": ConflictVote.Index, "block": strconv.FormatUint(ConflictVote.VoteStartBlock, 10)}, "failed to get epoch start")
		return
	}

	blocksToSave, err := k.epochstorageKeeper.BlocksToSave(ctx, ConflictVote.VoteStartBlock)
	if err != nil {
		k.CleanUpVote(ctx, ConflictVote.Index)
		utils.LavaError(ctx, logger, "vote_blockstosave_get_fail", map[string]string{"voteID": ConflictVote.Index, "block": strconv.FormatUint(ConflictVote.VoteStartBlock, 10)}, "failed to get blocks to save")
		return
	}

	for _, vote := range ConflictVote.Votes {
		accAddress, err := sdk.AccAddressFromBech32(vote.Address)
		if err != nil {
			utils.LavaError(ctx, logger, "invalid_address", map[string]string{"error": err.Error()}, "")
			ConsensusVote = false
			continue
		}
		entry, err := k.epochstorageKeeper.GetStakeEntryForProviderEpoch(ctx, ConflictVote.ChainID, accAddress, epochVoteStart)
		if err != nil {
			utils.LavaError(ctx, logger, "voter_stake_entry", map[string]string{"error": err.Error(), "voteStart": strconv.FormatUint(ConflictVote.VoteStartBlock, 10)}, "failed to get stake entry for provider in voter list")
			ConsensusVote = false
			continue
		}
		stake := entry.Stake.Amount
		totalVotes = totalVotes.Add(stake) //count all the stake in the vote
		votersStake[vote.Address] = stake  //save stake for reward weight
		switch vote.Result {               //count vote for each provider
		case types.Provider0:
			firstProviderVotes = firstProviderVotes.Add(stake)
		case types.Provider1:
			secondProviderVotes = secondProviderVotes.Add(stake)
		case types.NoneOfTheProviders:
			noneProviderVotes = noneProviderVotes.Add(stake)
		default:
			//punish providers that didnt vote
			providersWithoutVote = append(providersWithoutVote, vote.Address)
			bail := stake
			bail.Quo(sdk.NewIntFromUint64(BailStakeDiv))
			k.pairingKeeper.JailEntry(ctx, accAddress, true, ConflictVote.ChainID, uint64(ConflictVote.VoteStartBlock), blocksToSave, sdk.NewCoin(epochstoragetypes.TokenDenom, bail))
			slashed, err := k.pairingKeeper.SlashEntry(ctx, accAddress, true, ConflictVote.ChainID, SlashStakePercent)
			rewardPool = rewardPool.Add(slashed)
			if err != nil {
				utils.LavaError(ctx, logger, "slash_failed_vote", map[string]string{"error": err.Error()}, "slashing failed at vote conflict")
				continue
			}
		}
	}
	eventData["NumOfNoVoters"] = strconv.FormatInt(int64(len(providersWithoutVote)), 10)
	eventData["NumOfVoters"] = strconv.FormatInt(int64(len(ConflictVote.Votes)-len(providersWithoutVote)), 10)

	eventData["TotalVotes"] = totalVotes.String()
	eventData["FirstProviderVotes"] = firstProviderVotes.String()
	eventData["SecondProviderVotes"] = secondProviderVotes.String()
	eventData["NoneProviderVotes"] = noneProviderVotes.String()

	halfTotalVotes := totalVotes.Quo(sdk.NewIntFromUint64(MajorityDiv))
	majorityMet = firstProviderVotes.GT(halfTotalVotes) || secondProviderVotes.GT(halfTotalVotes) || noneProviderVotes.GT(halfTotalVotes)
	if majorityMet {
		eventName = types.ConflictVoteResolvedEventName
		//we have enough votes for a valid vote
		//find the winner
		if firstProviderVotes.GT(secondProviderVotes) && firstProviderVotes.GT(noneProviderVotes) {
			winner = types.Provider0
			winnersAddr = ConflictVote.FirstProvider.Account
			winnerVotersStake = firstProviderVotes
		} else if secondProviderVotes.GT(noneProviderVotes) {
			winner = types.Provider1
			winnersAddr = ConflictVote.SecondProvider.Account
			winnerVotersStake = secondProviderVotes
		} else {
			winner = types.NoneOfTheProviders
			winnerVotersStake = noneProviderVotes
			winnersAddr = "None"
		}

		eventData["winner"] = winnersAddr
		eventData["winnerVotes%"] = winnerVotersStake.ToDec().QuoInt(totalVotes).String()

		//punish the frauds(the provider that was found lying and all the voters that voted for him) and fill the reward pool
		//we need to finish the punishment before rewarding to fill up the reward pool
		if ConsensusVote && winnerVotersStake.ToDec().QuoInt(totalVotes).GTE(k.MajorityPercent(ctx)) {
			for _, vote := range ConflictVote.Votes {
				if vote.Result != winner && !slices.Contains(providersWithoutVote, vote.Address) { //punish those who voted wrong, voters that didnt vote already got punished
					accAddress, err := sdk.AccAddressFromBech32(vote.Address)
					if err != nil {
						utils.LavaError(ctx, logger, "invalid_address", map[string]string{"error": err.Error()}, "")
						continue
					}
					slashed, err := k.pairingKeeper.SlashEntry(ctx, accAddress, true, ConflictVote.ChainID, sdk.NewDecWithPrec(1, 0))
					rewardPool = rewardPool.Add(slashed)
					if err != nil {
						utils.LavaError(ctx, logger, "slash_failed_vote", map[string]string{"error": err.Error()}, "slashing failed at vote conflict")
					}

					err = k.pairingKeeper.UnstakeEntry(ctx, true, ConflictVote.ChainID, vote.Address)
					if err != nil {
						utils.LavaError(ctx, logger, "unstake_fraud_failed", map[string]string{"error": err.Error()}, "unstaking fraud voter failed")
						continue
					}
				}
			}
		}

	} else {
		eventName = types.ConflictVoteUnresolvedEventName
		eventData["voteFailed"] = "not_enough_voters"
	}

	//reward client
	clientRewardPoolPercentage := k.Rewards(ctx).ClientRewardPercent
	clientReward := clientRewardPoolPercentage.MulInt(rewardPool.Amount)
	rewardCount = rewardCount.Add(clientReward.TruncateInt())
	if rewardCount.GT(rewardPool.Amount) {
		utils.LavaError(ctx, logger, "VoteFail", eventData, "Reward overflow from the reward pool")
		k.RemoveConflictVote(ctx, ConflictVote.Index)
		return
	}
	accClientAddress, err := sdk.AccAddressFromBech32(ConflictVote.ClientAddress)
	if err != nil {
		utils.LavaError(ctx, logger, "invalid_address", map[string]string{"error": err.Error()}, "")
	} else {
		ok, err := k.pairingKeeper.CreditStakeEntry(ctx, ConflictVote.ChainID, accClientAddress, sdk.NewCoin(epochstoragetypes.TokenDenom, clientReward.TruncateInt()), false)
		if !ok {
			details := map[string]string{}
			if err != nil {
				details["error"] = err.Error()
			}
			utils.LavaError(ctx, logger, "failed_credit", details, "failed to credit client")
		}
	}

	if majorityMet {
		//reward winner provider
		if winner != types.NoneOfTheProviders {
			winnerRewardPoolPercentage := k.Rewards(ctx).WinnerRewardPercent
			winnerReward := winnerRewardPoolPercentage.MulInt(rewardPool.Amount)
			rewardCount = rewardCount.Add(winnerReward.TruncateInt())
			if rewardCount.GT(rewardPool.Amount) {
				utils.LavaError(ctx, logger, "VoteFail", eventData, "Reward overflow from the reward pool")
				k.RemoveConflictVote(ctx, ConflictVote.Index)
				return
			}
			accWinnerAddress, err := sdk.AccAddressFromBech32(winnersAddr)
			if err != nil {
				utils.LavaError(ctx, logger, "invalid_address", map[string]string{"error": err.Error()}, "")
			} else {
				ok, err := k.pairingKeeper.CreditStakeEntry(ctx, ConflictVote.ChainID, accWinnerAddress, sdk.NewCoin(epochstoragetypes.TokenDenom, winnerReward.TruncateInt()), true)
				if !ok {
					details := map[string]string{}
					if err != nil {
						details["error"] = err.Error()
					}
					utils.LavaError(ctx, logger, "failed_credit", details, "failed to credit provider")
				}
			}
		}

		//give reward to voters
		votersRewardPoolPercentage := k.Rewards(ctx).VotersRewardPercent
		rewardAllWinningVoters := votersRewardPoolPercentage.MulInt(rewardPool.Amount)
		for _, vote := range ConflictVote.Votes {
			if vote.Result == winner {
				//calculate the reward for the voter relative part (rewardpool*stake/stakesum)
				rewardVoter := rewardAllWinningVoters.MulInt(votersStake[vote.Address]).QuoInt(winnerVotersStake)
				rewardCount = rewardCount.Add(rewardVoter.TruncateInt())
				if rewardCount.GT(rewardPool.Amount) {
					utils.LavaError(ctx, logger, "VoteFail", eventData, "Reward overflow from the reward pool")
					k.RemoveConflictVote(ctx, ConflictVote.Index)
					return
				}
				accAddress, err := sdk.AccAddressFromBech32(vote.Address)
				if err != nil {
					utils.LavaError(ctx, logger, "invalid_address", map[string]string{"error": err.Error()}, "")
					continue
				}
				ok, err := k.pairingKeeper.CreditStakeEntry(ctx, ConflictVote.ChainID, accAddress, sdk.NewCoin(epochstoragetypes.TokenDenom, rewardVoter.TruncateInt()), true)
				if !ok {
					details := map[string]string{}
					if err != nil {
						details["error"] = err.Error()
					}
					utils.LavaError(ctx, logger, "failed_credit", details, "failed to credit voter")
					continue
				}
			}
		}
	}

	eventData["RewardPool"] = rewardPool.Amount.String()

	k.RemoveConflictVote(ctx, ConflictVote.Index)

	utils.LogLavaEvent(ctx, logger, eventName, eventData, "conflict detection resolved")
}

func (k Keeper) TransitionVoteToReveal(ctx sdk.Context, conflictVote types.ConflictVote) {
	logger := k.Logger(ctx)
	conflictVote.VoteState = types.StateReveal
	epochBlocks, err := k.epochstorageKeeper.EpochBlocks(ctx, uint64(ctx.BlockHeight()))
	if err != nil {
		k.CleanUpVote(ctx, conflictVote.Index)
	}

	conflictVote.VoteDeadline = uint64(ctx.BlockHeight()) + k.VotePeriod(ctx)*epochBlocks
	k.SetConflictVote(ctx, conflictVote)

	eventData := map[string]string{}
	eventData["voteID"] = conflictVote.Index
	eventData["voteDeadline"] = strconv.FormatUint(conflictVote.VoteDeadline, 10)
	utils.LogLavaEvent(ctx, logger, types.ConflictVoteRevealEventName, eventData, "Vote is now in reveal state")
}

func (k Keeper) CleanUpVote(ctx sdk.Context, index string) {
	k.RemoveConflictVote(ctx, index)
}

func FindVote(votes *[]types.Vote, address string) (int, bool) {
	for index, vote := range *votes {
		if vote.Address == address {
			return index, true
		}
	}
	return -1, false
}
