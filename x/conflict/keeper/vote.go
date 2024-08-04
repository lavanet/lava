package keeper

import (
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/conflict/types"
	"golang.org/x/exp/slices"
)

const (
	BailStakeDiv = 5 // 20% - Can't be 0!
	MajorityDiv  = 2 // 50% - Can't be 0!
)

var SlashStakePercent = sdk.NewDecWithPrec(5, 2) // 0.05

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

func (k Keeper) HandleAndCloseVote(ctx sdk.Context, conflictVote types.ConflictVote) {
	logger := k.Logger(ctx)
	eventData := []utils.Attribute{{Key: "voteID", Value: conflictVote.Index}}
	var eventName string
	// all wrong voters are punished
	// add stake as wieght
	// valid only if one of the votes is bigger than 50% from total
	// punish providers that didnt vote - discipline/jail + bail = 20%stake + slash 5%stake
	// (dont add jailed providers to voters)
	// if strong majority punish wrong providers - jail from start of memory to end + slash 100%stake
	// reward pool is the slashed amount from all punished providers
	// reward to stake - client 50%, the original provider 10%, 20% the voters
	totalVotes := sdk.ZeroInt()
	firstProviderVotes := sdk.ZeroInt()
	secondProviderVotes := sdk.ZeroInt()
	noneProviderVotes := sdk.ZeroInt()
	var providersWithoutVote []string
	rewardPool := sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), sdk.ZeroInt())
	rewardCount := math.ZeroInt()
	votersStake := map[string]math.Int{} // this is needed in order to give rewards for each voter according to their stake(so we dont take this data twice from the keeper)
	ConsensusVote := true
	var majorityMet bool

	var winner int64
	var winnersAddr string
	var winnerVotersStake math.Int

	// count votes and punish jury that didnt vote
	epochVoteStart, _, err := k.epochstorageKeeper.GetEpochStartForBlock(ctx, conflictVote.VoteStartBlock) // TODO check if we need to check for overlap
	if err != nil {
		k.CleanUpVote(ctx, conflictVote.Index)
		utils.LavaFormatWarning("failed to get epoch start", err,
			utils.Attribute{Key: "voteID", Value: conflictVote.Index},
			utils.Attribute{Key: "voteStartBlock", Value: conflictVote.VoteStartBlock},
		)
		return
	}

	blocksToSave, err := k.epochstorageKeeper.BlocksToSave(ctx, conflictVote.VoteStartBlock)
	if err != nil {
		k.CleanUpVote(ctx, conflictVote.Index)
		utils.LavaFormatWarning("failed to get blocks to save", err,
			utils.Attribute{Key: "voteID", Value: conflictVote.Index},
			utils.Attribute{Key: "voteStartBlock", Value: conflictVote.VoteStartBlock},
		)
		return
	}

	for _, vote := range conflictVote.Votes {
		entry, found := k.epochstorageKeeper.GetStakeEntryForProviderEpoch(ctx, conflictVote.ChainID, vote.Address, epochVoteStart)
		if !found {
			utils.LavaFormatWarning("failed to get stake entry for provider in voter list", fmt.Errorf("stake entry not found"),
				utils.Attribute{Key: "voteID", Value: conflictVote.Index},
				utils.Attribute{Key: "voteChainID", Value: conflictVote.ChainID},
				utils.Attribute{Key: "voteStartBlock", Value: conflictVote.VoteStartBlock},
			)
			ConsensusVote = false
			continue
		}
		stake := entry.EffectiveStake()
		totalVotes = totalVotes.Add(stake) // count all the stake in the vote
		votersStake[vote.Address] = stake  // save stake for reward weight
		switch vote.Result {               // count vote for each provider
		case types.Provider0:
			firstProviderVotes = firstProviderVotes.Add(stake)
		case types.Provider1:
			secondProviderVotes = secondProviderVotes.Add(stake)
		case types.NoneOfTheProviders:
			noneProviderVotes = noneProviderVotes.Add(stake)
		default:
			// punish providers that didnt vote
			providersWithoutVote = append(providersWithoutVote, vote.Address)
			bail := stake
			bail.Quo(sdk.NewIntFromUint64(BailStakeDiv))
			err = k.pairingKeeper.JailEntry(ctx, vote.Address, conflictVote.ChainID, conflictVote.VoteStartBlock, blocksToSave, sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), bail))
			if err != nil {
				utils.LavaFormatWarning("jailing failed at vote conflict", err)
				// not skipping to continue to slash
			}
			slashed, err := k.pairingKeeper.SlashEntry(ctx, vote.Address, conflictVote.ChainID, SlashStakePercent)
			rewardPool = rewardPool.Add(slashed)
			if err != nil {
				utils.LavaFormatWarning("slashing failed at vote conflict", err)
				continue
			}
		}
	}
	eventData = append(eventData, utils.Attribute{Key: "NumOfNoVoters", Value: len(providersWithoutVote)})
	eventData = append(eventData, utils.Attribute{Key: "NumOfVoters", Value: len(conflictVote.Votes) - len(providersWithoutVote)})
	eventData = append(eventData, utils.Attribute{Key: "TotalVotes", Value: totalVotes})
	eventData = append(eventData, utils.Attribute{Key: "FirstProviderVotes", Value: firstProviderVotes})
	eventData = append(eventData, utils.Attribute{Key: "SecondProviderVotes", Value: secondProviderVotes})
	eventData = append(eventData, utils.Attribute{Key: "NoneProviderVotes", Value: noneProviderVotes})

	halfTotalVotes := totalVotes.Quo(sdk.NewIntFromUint64(MajorityDiv))
	majorityMet = firstProviderVotes.GT(halfTotalVotes) || secondProviderVotes.GT(halfTotalVotes) || noneProviderVotes.GT(halfTotalVotes)
	if majorityMet {
		eventName = types.ConflictVoteResolvedEventName
		// we have enough votes for a valid vote
		// find the winner
		if firstProviderVotes.GT(secondProviderVotes) && firstProviderVotes.GT(noneProviderVotes) {
			winner = types.Provider0
			winnersAddr = conflictVote.FirstProvider.Account
			winnerVotersStake = firstProviderVotes
		} else if secondProviderVotes.GT(noneProviderVotes) {
			winner = types.Provider1
			winnersAddr = conflictVote.SecondProvider.Account
			winnerVotersStake = secondProviderVotes
		} else {
			winner = types.NoneOfTheProviders
			winnerVotersStake = noneProviderVotes
			winnersAddr = "None"
		}

		if totalVotes.IsZero() {
			utils.LavaFormatWarning("totalVotes is zero", fmt.Errorf("critical: Attempt to divide by zero"),
				utils.LogAttr("totalVotes", totalVotes),
				utils.LogAttr("winnerVotersStake", winnerVotersStake),
			)
			return
		}
		eventData = append(eventData, utils.Attribute{Key: "winner", Value: winnersAddr})
		eventData = append(eventData, utils.Attribute{Key: "winnerVotes%", Value: sdk.NewDecFromInt(winnerVotersStake).QuoInt(totalVotes)})

		// punish the frauds(the provider that was found lying and all the voters that voted for him) and fill the reward pool
		// we need to finish the punishment before rewarding to fill up the reward pool
		if ConsensusVote && sdk.NewDecFromInt(winnerVotersStake).QuoInt(totalVotes).GTE(k.MajorityPercent(ctx)) {
			for _, vote := range conflictVote.Votes {
				if vote.Result != winner && !slices.Contains(providersWithoutVote, vote.Address) { // punish those who voted wrong, voters that didnt vote already got punished
					slashed, err := k.pairingKeeper.SlashEntry(ctx, vote.Address, conflictVote.ChainID, sdk.NewDecWithPrec(1, 0))
					rewardPool = rewardPool.Add(slashed)
					if err != nil {
						utils.LavaFormatWarning("slashing failed at vote conflict", err)
					}

					// TODO: uncomment this when conflict is more stable
					// err = k.pairingKeeper.FreezeProvider(ctx, vote.Address, []string{conflictVote.ChainID}, types.UnstakeDescriptionFraudVote)
					// if err != nil {
					// 	utils.LavaFormatWarning("unstaking fraud voter failed", err)
					// 	continue
					// }
				}
			}
		}
	} else {
		eventName = types.ConflictVoteUnresolvedEventName
		eventData = append(eventData, utils.Attribute{Key: "voteFailed", Value: "not_enough_voters"})
	}

	// reward client
	clientRewardPoolPercentage := k.Rewards(ctx).ClientRewardPercent
	clientReward := clientRewardPoolPercentage.MulInt(rewardPool.Amount)
	rewardCount = rewardCount.Add(clientReward.TruncateInt())
	if rewardCount.GT(rewardPool.Amount) {
		utils.LavaFormatError("Reward overflow from the reward pool", err, eventData...)
		k.RemoveConflictVote(ctx, conflictVote.Index)
		return
	}

	// TODO: add consumer rewards

	if majorityMet {
		// reward winner provider
		if winner != types.NoneOfTheProviders {
			winnerRewardPoolPercentage := k.Rewards(ctx).WinnerRewardPercent
			winnerReward := winnerRewardPoolPercentage.MulInt(rewardPool.Amount)
			rewardCount = rewardCount.Add(winnerReward.TruncateInt())
			if rewardCount.GT(rewardPool.Amount) {
				utils.LavaFormatError("Reward overflow from the reward pool", err, eventData...)
				k.RemoveConflictVote(ctx, conflictVote.Index)
				return
			}
			accWinnerAddress, err := sdk.AccAddressFromBech32(winnersAddr)
			if err != nil {
				utils.LavaFormatWarning("invalid winner address", err,
					utils.Attribute{Key: "voteAddress", Value: winnersAddr},
				)
			} else {
				ok, err := k.pairingKeeper.CreditStakeEntry(ctx, conflictVote.ChainID, accWinnerAddress, sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), winnerReward.TruncateInt()))
				if !ok {
					utils.LavaFormatWarning("failed to credit client", err)
				}
			}
		}

		// give reward to voters
		votersRewardPoolPercentage := k.Rewards(ctx).VotersRewardPercent
		rewardAllWinningVoters := votersRewardPoolPercentage.MulInt(rewardPool.Amount)
		for _, vote := range conflictVote.Votes {
			if vote.Result == winner {
				// calculate the reward for the voter relative part (rewardpool*stake/stakesum)

				if winnerVotersStake.IsZero() {
					utils.LavaFormatWarning("winnerVotersStake is zero", fmt.Errorf("critical: Attempt to divide by zero"),
						utils.LogAttr("winnerVotersStake", winnerVotersStake),
						utils.LogAttr("votersStake[vote.Address]", votersStake[vote.Address]),
					)
					return
				}

				rewardVoter := rewardAllWinningVoters.MulInt(votersStake[vote.Address]).QuoInt(winnerVotersStake)
				rewardCount = rewardCount.Add(rewardVoter.TruncateInt())
				if rewardCount.GT(rewardPool.Amount) {
					utils.LavaFormatError("Reward overflow from the reward pool", err, eventData...)
					k.RemoveConflictVote(ctx, conflictVote.Index)
					return
				}
				accAddress, err := sdk.AccAddressFromBech32(vote.Address)
				if err != nil {
					utils.LavaFormatWarning("invalid vote address", err,
						utils.Attribute{Key: "voteAddress", Value: vote.Address},
					)
					continue
				}
				ok, err := k.pairingKeeper.CreditStakeEntry(ctx, conflictVote.ChainID, accAddress, sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), rewardVoter.TruncateInt()))
				if !ok {
					details := map[string]string{}
					if err != nil {
						details["error"] = err.Error()
					}
					utils.LavaFormatWarning("failed to credit client", err)
					continue
				}
			}
		}
	}

	eventData = append(eventData, utils.Attribute{Key: "RewardPool", Value: rewardPool.Amount})

	k.RemoveConflictVote(ctx, conflictVote.Index)

	eventDataMap := map[string]string{}
	for _, attribute := range eventData {
		eventDataMap[attribute.Key] = fmt.Sprint(attribute.Value)
	}
	utils.LogLavaEvent(ctx, logger, eventName, eventDataMap, "conflict detection resolved")
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
