package keeper

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/conflict/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"golang.org/x/exp/slices"
)

func (k Keeper) ValidateFinalizationConflict(ctx sdk.Context, conflictData *types.FinalizationConflict, clientAddr sdk.AccAddress) error {
	return nil
}

func (k Keeper) ValidateResponseConflict(ctx sdk.Context, conflictData *types.ResponseConflict, clientAddr sdk.AccAddress) error {
	//1. validate mismatching data
	chainID := conflictData.ConflictRelayData0.Request.ChainID
	if chainID != conflictData.ConflictRelayData1.Request.ChainID {
		return fmt.Errorf("mismatching request parameters between providers %s, %s", chainID, conflictData.ConflictRelayData1.Request.ChainID)
	}
	block := conflictData.ConflictRelayData0.Request.BlockHeight
	if block != conflictData.ConflictRelayData1.Request.BlockHeight {
		return fmt.Errorf("mismatching request parameters between providers %d, %d", block, conflictData.ConflictRelayData1.Request.BlockHeight)
	}
	if conflictData.ConflictRelayData0.Request.ApiId != conflictData.ConflictRelayData1.Request.ApiId {
		return fmt.Errorf("mismatching request parameters between providers %d, %d", conflictData.ConflictRelayData0.Request.ApiId, conflictData.ConflictRelayData1.Request.ApiId)
	}
	if conflictData.ConflictRelayData0.Request.ApiUrl != conflictData.ConflictRelayData1.Request.ApiUrl {
		return fmt.Errorf("mismatching request parameters between providers %s, %s", conflictData.ConflictRelayData0.Request.ApiUrl, conflictData.ConflictRelayData1.Request.ApiUrl)
	}
	if !bytes.Equal(conflictData.ConflictRelayData0.Request.Data, conflictData.ConflictRelayData1.Request.Data) {
		return fmt.Errorf("mismatching request parameters between providers %s, %s", conflictData.ConflictRelayData0.Request.Data, conflictData.ConflictRelayData1.Request.Data)
	}
	if conflictData.ConflictRelayData0.Request.ApiUrl != conflictData.ConflictRelayData1.Request.ApiUrl {
		return fmt.Errorf("mismatching request parameters between providers %s, %s", conflictData.ConflictRelayData0.Request.ApiUrl, conflictData.ConflictRelayData1.Request.ApiUrl)
	}
	if conflictData.ConflictRelayData0.Request.RequestBlock != conflictData.ConflictRelayData1.Request.RequestBlock {
		return fmt.Errorf("mismatching request parameters between providers %d, %d", conflictData.ConflictRelayData0.Request.RequestBlock, conflictData.ConflictRelayData1.Request.RequestBlock)
	}
	epochStart, _ := k.epochstorageKeeper.GetEpochStartForBlock(ctx, uint64(block))
	k.pairingKeeper.VerifyPairingData(ctx, chainID, clientAddr, epochStart)
	//2. validate signer
	clientEntry, err := k.epochstorageKeeper.GetStakeEntryForClientEpoch(ctx, chainID, clientAddr, epochStart)
	if err != nil || clientEntry == nil {
		return fmt.Errorf("did not find a stake entry for consumer %s on epoch %d, chainID %s error: %s", clientAddr, epochStart, chainID, err.Error())
	}
	verifyClientAddrFromSignatureOnRequest := func(conflictRelayData types.ConflictRelayData) error {
		pubKey, err := sigs.RecoverPubKeyFromRelay(*conflictRelayData.Request)
		if err != nil {
			return fmt.Errorf("invalid consumer signature in relay request %+v , error: %s", conflictRelayData.Request, err.Error())
		}
		derived_clientAddr, err := sdk.AccAddressFromHex(pubKey.Address().String())
		if err != nil {
			return fmt.Errorf("invalid consumer address from signature in relay request %+v , error: %s", conflictRelayData.Request, err.Error())
		}
		if !derived_clientAddr.Equals(clientAddr) {
			return fmt.Errorf("mismatching consumer address signature and msg.Creator in relay request %s , %s", derived_clientAddr, clientAddr)
		}
		return nil
	}
	err = verifyClientAddrFromSignatureOnRequest(*conflictData.ConflictRelayData0)
	if err != nil {
		return err
	}
	err = verifyClientAddrFromSignatureOnRequest(*conflictData.ConflictRelayData1)
	if err != nil {
		return err
	}
	//3. validate providers signatures and stakeEntry for that epoch
	providerAddressFromRelayReplyAndVerifyStakeEntry := func(request *pairingtypes.RelayRequest, reply *pairingtypes.RelayReply, first bool) (providerAddress sdk.AccAddress, err error) {
		print_st := "first"
		if !first {
			print_st = "second"
		}
		pubKey, err := sigs.RecoverPubKeyFromRelayReply(reply, request)
		if err != nil {
			return nil, fmt.Errorf("RecoverProviderPubKeyFromQueryAndAllDataHash %s provider: %w", print_st, err)
		}
		providerAddress, err = sdk.AccAddressFromHex(pubKey.Address().String())
		if err != nil {
			return nil, fmt.Errorf("AccAddressFromHex %s provider: %w", print_st, err)
		}
		_, err = k.epochstorageKeeper.GetStakeEntryForProviderEpoch(ctx, chainID, providerAddress, epochStart)
		if err != nil {
			return nil, fmt.Errorf("did not find a stake entry for %s provider %s on epoch %d, chainID %s error: %s", print_st, providerAddress, epochStart, chainID, err.Error())
		}
		return providerAddress, nil
	}
	providerAccAddress0, err := providerAddressFromRelayReplyAndVerifyStakeEntry(conflictData.ConflictRelayData0.Request, conflictData.ConflictRelayData0.Reply, true)
	if err != nil {
		return err
	}
	providerAccAddress1, err := providerAddressFromRelayReplyAndVerifyStakeEntry(conflictData.ConflictRelayData1.Request, conflictData.ConflictRelayData1.Reply, false)
	if err != nil {
		return err
	}
	//4. validate finalization
	validateResponseFinalizationData := func(expectedAddress sdk.AccAddress, response *pairingtypes.RelayReply, request *pairingtypes.RelayRequest, first bool) (err error) {
		print_st := "first"
		if !first {
			print_st = "second"
		}

		pubKey, err := sigs.RecoverPubKeyFromResponseFinalizationData(response, request, clientAddr)
		if err != nil {
			return fmt.Errorf("RecoverPubKey %s provider ResponseFinalizationData: %w", print_st, err)
		}
		derived_providerAccAddress, err := sdk.AccAddressFromHex(pubKey.Address().String())
		if err != nil {
			return fmt.Errorf("AccAddressFromHex %s provider ResponseFinalizationData: %w", print_st, err)
		}
		if !derived_providerAccAddress.Equals(expectedAddress) {
			return fmt.Errorf("mismatching %s provider address signature and responseFinazalizationData %s , %s", print_st, derived_providerAccAddress, expectedAddress)
		}
		//validate the responses are finalized
		if !k.specKeeper.IsFinalizedBlock(ctx, chainID, request.RequestBlock, response.LatestBlock) {
			return fmt.Errorf("block isn't finalized on %s provider! %d,%d ", print_st, request.RequestBlock, response.LatestBlock)
		}
		return nil
	}
	err = validateResponseFinalizationData(providerAccAddress0, conflictData.ConflictRelayData0.Reply, conflictData.ConflictRelayData0.Request, true)
	if err != nil {
		return err
	}
	err = validateResponseFinalizationData(providerAccAddress1, conflictData.ConflictRelayData1.Reply, conflictData.ConflictRelayData1.Request, true)
	if err != nil {
		return err
	}
	//5. validate mismatching responses
	if bytes.Equal(conflictData.ConflictRelayData0.Reply.Data, conflictData.ConflictRelayData1.Reply.Data) {
		return fmt.Errorf("no conflict between providers data responses, its the same")
	}
	return nil
}

func (k Keeper) ValidateSameProviderConflict(ctx sdk.Context, conflictData *types.FinalizationConflict, clientAddr sdk.AccAddress) error {
	return nil
}

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
	//all wrong voters are punished
	//add stake as wieght
	//votecounts is bigint
	//valid only if one of the votes is bigger than 50% from total
	//punish providers that didnt vote - discipline/jail + bail = 20%stake + slash 5%stake
	//(dont add jailed providers to voters)
	//if strong majority punish wrong providers - jail from start of memory to end + slash 100%stake
	//reward pool is the slashed amount from all punished providers
	//reward to stake - client 50%, the original provider 10%, 20% the voters

	totalVotes := big.NewInt(0)
	firstProviderVotes := big.NewInt(0)
	secondProviderVotes := big.NewInt(0)
	noneProviderVotes := big.NewInt(0)
	var providersWithoutVote []string
	rewardPool := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.ZeroInt())

	//count votes and punish jury that didnt vote
	for address, vote := range ConflictVote.VotersHash {
		stake := big.NewInt(0) //find the real stake for each provider
		totalVotes.Add(totalVotes, stake)
		switch vote.Result {
		case types.Provider0:
			firstProviderVotes.Add(firstProviderVotes, stake)
		case types.Provider1:
			firstProviderVotes.Add(secondProviderVotes, stake)
		case types.None:
			noneProviderVotes.Add(noneProviderVotes, stake)
		default:
			providersWithoutVote = append(providersWithoutVote, address)
			bail := big.NewInt(0)
			bail.Div(stake, big.NewInt(5))
			k.pairingKeeper.JailEntry(ctx, sdk.AccAddress(address), true, ConflictVote.ChainID, 0, 0, sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewIntFromBigInt(bail)))
			slashed, err := k.pairingKeeper.SlashEntry(ctx, sdk.AccAddress(address), true, ConflictVote.ChainID, sdk.NewDecWithPrec(5, 2))
			rewardPool.Add(slashed)
		}
	}

	halfTotalVotes := big.NewInt(0)
	halfTotalVotes.Div(totalVotes, big.NewInt(2))
	if firstProviderVotes.Cmp(halfTotalVotes) > 0 || secondProviderVotes.Cmp(halfTotalVotes) > 0 || noneProviderVotes.Cmp(halfTotalVotes) > 0 {
		//we have enough votes for a valid vote
		//find the winner
		var winner int64
		if firstProviderVotes.Cmp(secondProviderVotes) > 0 && firstProviderVotes.Cmp(noneProviderVotes) > 0 {
			winner = types.Provider0
		} else if secondProviderVotes.Cmp(noneProviderVotes) > 0 {
			winner = types.Provider1
		} else {
			winner = types.None
		}

		//punish the frauds and fill the reward pool
		for address, vote := range ConflictVote.VotersHash {
			if vote.Result != winner && !slices.Contains(providersWithoutVote, address) {
				slashed, err := k.pairingKeeper.SlashEntry(ctx, sdk.AccAddress(address), true, ConflictVote.ChainID, sdk.NewDecWithPrec(1, 0))
				rewardPool.Add(slashed)
			}
			//unstake provider??
		}

		for address, vote := range ConflictVote.VotersHash {
			if vote.Result == winner {
				slashed, err := k.pairingKeeper.SlashEntry(ctx, sdk.AccAddress(address), true, ConflictVote.ChainID, sdk.NewDecWithPrec(1, 0))
				rewardPool.Add(slashed)
			}
			//unstake provider??
		}
		//4) reward voters and providers
	}

	k.RemoveConflictVote(ctx, ConflictVote.Index)

	logger := k.Logger(ctx)
	eventData := map[string]string{"voteID": ConflictVote.Index}
	utils.LogLavaEvent(ctx, logger, "conflict_detection_vote_resolved", eventData, "conflict detection resolved")
}
