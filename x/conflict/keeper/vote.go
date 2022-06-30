package keeper

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/conflict/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"golang.org/x/exp/slices"
)

func (k Keeper) AllocateNewConflictVote(ctx sdk.Context) string {
	found := false
	var index uint64 = 0
	var sIndex string
	for !found {
		index++
		sIndex = strconv.FormatUint(index, 10)
		_, found = k.GetConflictVote(ctx, sIndex)
	}
	return sIndex
}

func (k Keeper) HandleAndCloseVote(ctx sdk.Context, ConflictVote types.ConflictVote) {
	logger := k.Logger(ctx)
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
	votersStake := map[string]sdk.Int{}

	//count votes and punish jury that didnt vote
	for address, vote := range ConflictVote.VotersHash {
		accAddress, err := sdk.AccAddressFromBech32(address)
		if err != nil {
			utils.LavaError(ctx, logger, "invalid_address", map[string]string{"error": err.Error()}, "")
			continue
		}
		entry, err := k.epochstorageKeeper.GetStakeEntryForProviderEpoch(ctx, ConflictVote.ChainID, accAddress, ConflictVote.VoteStartBlock)
		if err != nil {
			utils.LavaError(ctx, logger, "voter_stake_entry", map[string]string{"error": err.Error(), "voteStart": strconv.FormatUint(ConflictVote.VoteStartBlock, 10)}, "failed to get stake entry for provider in voter list")
		}
		stake := entry.Stake.Amount
		totalVotes.Add(stake)
		votersStake[address] = stake
		switch vote.Result {
		case types.Provider0:
			firstProviderVotes.Add(stake)
		case types.Provider1:
			firstProviderVotes.Add(stake)
		case types.None:
			noneProviderVotes.Add(stake)
		default:
			providersWithoutVote = append(providersWithoutVote, address)
			bail := stake
			bail.Quo(sdk.NewIntFromUint64(5)) //20%
			k.pairingKeeper.JailEntry(ctx, accAddress, true, ConflictVote.ChainID, uint64(ConflictVote.VoteStartBlock), ConflictVote.VoteStartBlock+k.epochstorageKeeper.BlocksToSave(ctx), sdk.NewCoin(epochstoragetypes.TokenDenom, bail))
			slashed, err := k.pairingKeeper.SlashEntry(ctx, accAddress, true, ConflictVote.ChainID, sdk.NewDecWithPrec(5, 2))
			rewardPool.Add(slashed)
			if err != nil {
				utils.LavaError(ctx, logger, "slash_failed_vote", map[string]string{"error": err.Error()}, "slashing failed at vote conflict")
			}
		}
	}

	halfTotalVotes := totalVotes
	halfTotalVotes.Quo(sdk.NewIntFromUint64(2))
	if firstProviderVotes.GT(halfTotalVotes) || secondProviderVotes.GT(halfTotalVotes) || noneProviderVotes.GT(halfTotalVotes) {
		//we have enough votes for a valid vote
		//find the winner
		var winner int64
		var winnersAddr string
		var winnerVotersStake sdk.Int
		if firstProviderVotes.GT(secondProviderVotes) && firstProviderVotes.GT(noneProviderVotes) {
			winner = types.Provider0
			winnersAddr = ConflictVote.FirstProvider.Account
			winnerVotersStake = firstProviderVotes
		} else if secondProviderVotes.GT(noneProviderVotes) {
			winner = types.Provider1
			winnersAddr = ConflictVote.SecondProvider.Account
			winnerVotersStake = secondProviderVotes
		} else {
			winner = types.None
			winnerVotersStake = noneProviderVotes
		}

		//punish the frauds and fill the reward pool
		//we need to finish the punishment before rewarding to fill up the reward pool
		for address, vote := range ConflictVote.VotersHash {
			if vote.Result != winner && !slices.Contains(providersWithoutVote, address) {
				accAddress, err := sdk.AccAddressFromBech32(address)
				if err != nil {
					utils.LavaError(ctx, logger, "invalid_address", map[string]string{"error": err.Error()}, "")
					continue
				}
				slashed, err := k.pairingKeeper.SlashEntry(ctx, accAddress, true, ConflictVote.ChainID, sdk.NewDecWithPrec(1, 0))
				rewardPool.Add(slashed)
				if err != nil {
					utils.LavaError(ctx, logger, "slash_failed_vote", map[string]string{"error": err.Error()}, "slashing failed at vote conflict")
				}

				err = k.pairingKeeper.UnstakeEntry(ctx, true, ConflictVote.ChainID, address)
				if err != nil {
					utils.LavaError(ctx, logger, "unstake_fraud_failed", map[string]string{"error": err.Error()}, "unstaking fraud voter failed")
				}
			}
		}

		//give reward to voters
		votersRewardPoolPrecentage := sdk.NewDecWithPrec(15, 2)                        //TODO param
		rewardAllWinningVoters := votersRewardPoolPrecentage.MulInt(rewardPool.Amount) //15% / len(voters)
		for address, vote := range ConflictVote.VotersHash {
			if vote.Result == winner {
				//calculate the reward for the voter relative part (rewardpool*stake/stakesum)
				rewardVoter := rewardAllWinningVoters.MulInt(votersStake[address]).QuoInt(winnerVotersStake)
				accAddress, err := sdk.AccAddressFromBech32(address)
				if err != nil {
					utils.LavaError(ctx, logger, "invalid_address", map[string]string{"error": err.Error()}, "")
					continue
				}
				k.pairingKeeper.CreditStakeEntry(ctx, ConflictVote.ChainID, accAddress, sdk.NewCoin(epochstoragetypes.TokenDenom, rewardVoter.TruncateInt()), true)
			}
		}

		//reward winner provider
		if winner != types.None {
			winnerRewardPoolPrecentage := sdk.NewDecWithPrec(15, 2)              //TODO param
			winnerReward := winnerRewardPoolPrecentage.MulInt(rewardPool.Amount) //15%
			accWinnerAddress, err := sdk.AccAddressFromBech32(winnersAddr)
			if err != nil {
				utils.LavaError(ctx, logger, "invalid_address", map[string]string{"error": err.Error()}, "")
			} else {

				entry, found, indexFound := k.epochstorageKeeper.StakeEntryByAddress(ctx, epochstoragetypes.ProviderKey, ConflictVote.ChainID, accWinnerAddress)
				if found {
					entry.Stake = entry.Stake.AddAmount(winnerReward.TruncateInt())
					k.epochstorageKeeper.ModifyStakeEntry(ctx, epochstoragetypes.ProviderKey, ConflictVote.ChainID, entry, indexFound)
				}
			}
		}
	}

	//reward client
	clientRewardPoolPrecentage := sdk.NewDecWithPrec(50, 2)              //TODO param
	clientReward := clientRewardPoolPrecentage.MulInt(rewardPool.Amount) //50%
	accClientAddress, err := sdk.AccAddressFromBech32(ConflictVote.ClientAddress)
	if err != nil {
		utils.LavaError(ctx, logger, "invalid_address", map[string]string{"error": err.Error()}, "")
	} else {
		entry, found, indexFound := k.epochstorageKeeper.StakeEntryByAddress(ctx, epochstoragetypes.ClientKey, ConflictVote.ChainID, accClientAddress)
		if found {
			entry.Stake = entry.Stake.AddAmount(clientReward.TruncateInt())
			k.epochstorageKeeper.ModifyStakeEntry(ctx, epochstoragetypes.ClientKey, ConflictVote.ChainID, entry, indexFound)
		}
	}

	k.RemoveConflictVote(ctx, ConflictVote.Index)

	eventData := map[string]string{"voteID": ConflictVote.Index}
	utils.LogLavaEvent(ctx, logger, types.ConflictVoteResolvedEventName, eventData, "conflict detection resolved")
}

func (k Keeper) TransitionVoteToReveal(ctx sdk.Context, conflictVote types.ConflictVote) {
	logger := k.Logger(ctx)
	conflictVote.VoteState = types.StateReveal
	conflictVote.VoteDeadline = conflictVote.VoteDeadline + k.VotePeriod(ctx)*k.epochstorageKeeper.EpochBlocks(ctx)
	k.SetConflictVote(ctx, conflictVote)

	eventData := map[string]string{}
	eventData["voteID"] = conflictVote.Index
	eventData["voteDeadline"] = strconv.FormatUint(conflictVote.VoteDeadline, 10)
	utils.LogLavaEvent(ctx, logger, types.ConflictVoteRevealEventName, eventData, "Vote is now in reveal state")
}
