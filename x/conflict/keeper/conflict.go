package keeper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils/sigs"
	"github.com/lavanet/lava/utils/slices"
	"github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

func (k Keeper) ValidateFinalizationConflict(ctx sdk.Context, conflictData *types.FinalizationConflict, clientAddr sdk.AccAddress) error {
	return nil
}

func (k Keeper) ValidateResponseConflict(ctx sdk.Context, conflictData *types.ResponseConflict, clientAddr sdk.AccAddress) error {
	// 1. validate mismatching data
	chainID := conflictData.ConflictRelayData0.Request.RelaySession.SpecId
	if chainID != conflictData.ConflictRelayData1.Request.RelaySession.SpecId {
		return fmt.Errorf("mismatching request parameters between providers %s, %s", chainID, conflictData.ConflictRelayData1.Request.RelaySession.SpecId)
	}

	block := conflictData.ConflictRelayData0.Request.RelaySession.Epoch
	if block != conflictData.ConflictRelayData1.Request.RelaySession.Epoch {
		return fmt.Errorf("mismatching request parameters between providers %d, %d", block, conflictData.ConflictRelayData1.Request.RelaySession.Epoch)
	}

	if conflictData.ConflictRelayData0.Request.RelayData.ConnectionType != conflictData.ConflictRelayData1.Request.RelayData.ConnectionType {
		return fmt.Errorf("mismatching request parameters between providers %s, %s", conflictData.ConflictRelayData0.Request.RelayData.ConnectionType, conflictData.ConflictRelayData1.Request.RelayData.ConnectionType)
	}

	if conflictData.ConflictRelayData0.Request.RelayData.ApiUrl != conflictData.ConflictRelayData1.Request.RelayData.ApiUrl {
		return fmt.Errorf("mismatching request parameters between providers %s, %s", conflictData.ConflictRelayData0.Request.RelayData.ApiUrl, conflictData.ConflictRelayData1.Request.RelayData.ApiUrl)
	}

	if !bytes.Equal(conflictData.ConflictRelayData0.Request.RelayData.Data, conflictData.ConflictRelayData1.Request.RelayData.Data) {
		return fmt.Errorf("mismatching request parameters between providers %s, %s", conflictData.ConflictRelayData0.Request.RelayData.Data, conflictData.ConflictRelayData1.Request.RelayData.Data)
	}

	if conflictData.ConflictRelayData0.Request.RelayData.RequestBlock != conflictData.ConflictRelayData1.Request.RelayData.RequestBlock {
		return fmt.Errorf("mismatching request parameters between providers %d, %d", conflictData.ConflictRelayData0.Request.RelayData.RequestBlock, conflictData.ConflictRelayData1.Request.RelayData.RequestBlock)
	}

	if conflictData.ConflictRelayData0.Request.RelayData.ApiInterface != conflictData.ConflictRelayData1.Request.RelayData.ApiInterface {
		return fmt.Errorf("mismatching request parameters between providers %s, %s", conflictData.ConflictRelayData0.Request.RelayData.ApiInterface, conflictData.ConflictRelayData1.Request.RelayData.ApiInterface)
	}

	// 1.5 validate params
	epochStart, _, err := k.epochstorageKeeper.GetEpochStartForBlock(ctx, uint64(block))
	if err != nil {
		return fmt.Errorf("could not find epoch for block %d", block)
	}

	if conflictData.ConflictRelayData0.Request.RelayData.RequestBlock < 0 {
		return fmt.Errorf("invalid request block height %d", conflictData.ConflictRelayData0.Request.RelayData.RequestBlock)
	}

	epochBlocks, err := k.epochstorageKeeper.EpochBlocks(ctx, uint64(block))
	if err != nil {
		return fmt.Errorf("could not get EpochBlocks param")
	}
	span := k.VoteStartSpan(ctx) * epochBlocks
	if uint64(ctx.BlockHeight())-epochStart >= span {
		return fmt.Errorf("conflict was received outside of the allowed span, current: %d, span %d - %d", ctx.BlockHeight(), epochStart, epochStart+span)
	}

	_, _, err = k.pairingKeeper.VerifyPairingData(ctx, chainID, epochStart)
	if err != nil {
		return err
	}

	_, err = k.pairingKeeper.GetProjectData(ctx, clientAddr, chainID, epochStart)
	if err != nil {
		return fmt.Errorf("did not find a project for %s on epoch %d, chainID %s error: %s", clientAddr, epochStart, chainID, err.Error())
	}

	verifyClientAddrFromSignatureOnRequest := func(conflictRelayData types.ConflictRelayData) error {
		pubKey, err := sigs.RecoverPubKey(*conflictRelayData.Request.RelaySession)
		if err != nil {
			return fmt.Errorf("invalid consumer signature in relay request %+v , error: %s", conflictRelayData.Request, err.Error())
		}
		derived_clientAddr, err := sdk.AccAddressFromHexUnsafe(pubKey.Address().String())
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
		return fmt.Errorf("conflict data 0: %s", err)
	}

	err = verifyClientAddrFromSignatureOnRequest(*conflictData.ConflictRelayData1)
	if err != nil {
		return fmt.Errorf("conflict data 1: %s", err)
	}

	// 3. validate providers signatures and stakeEntry for that epoch
	providerAddressFromRelayReplyAndVerifyStakeEntry := func(reply *types.ReplyMetadata, first bool) (providerAddress sdk.AccAddress, err error) {
		print_st := "first"
		if !first {
			print_st = "second"
		}
		pubKey, err := sigs.RecoverPubKey(reply)
		if err != nil {
			return nil, fmt.Errorf("RecoverPubKeyFromReplyMetadata %s provider: %w", print_st, err)
		}
		providerAddress, err = sdk.AccAddressFromHexUnsafe(pubKey.Address().String())
		if err != nil {
			return nil, fmt.Errorf("AccAddressFromHex %s provider: %w", print_st, err)
		}
		_, err = k.epochstorageKeeper.GetStakeEntryForProviderEpoch(ctx, chainID, providerAddress, epochStart)
		if err != nil {
			return nil, fmt.Errorf("did not find a stake entry for %s provider %s on epoch %d, chainID %s error: %s", print_st, providerAddress, epochStart, chainID, err.Error())
		}
		return providerAddress, nil
	}

	providerAccAddress0, err := providerAddressFromRelayReplyAndVerifyStakeEntry(conflictData.ConflictRelayData0.Reply, true)
	if err != nil {
		return err
	}

	providerAccAddress1, err := providerAddressFromRelayReplyAndVerifyStakeEntry(conflictData.ConflictRelayData1.Reply, false)
	if err != nil {
		return err
	}

	// 4. validate finalization
	validateResponseFinalizationData := func(expectedAddress sdk.AccAddress, replyMetadata *types.ReplyMetadata, request *pairingtypes.RelayRequest, first bool) (err error) {
		print_st := "first"
		if !first {
			print_st = "second"
		}

		metaData := types.NewRelayFinalizationMetaData(*replyMetadata, *request, clientAddr)
		pubKey, err := sigs.RecoverPubKey(metaData)
		if err != nil {
			return fmt.Errorf("RecoverPubKey %s provider ResponseFinalizationData: %w", print_st, err)
		}
		derived_providerAccAddress, err := sdk.AccAddressFromHexUnsafe(pubKey.Address().String())
		if err != nil {
			return fmt.Errorf("AccAddressFromHex %s provider ResponseFinalizationData: %w", print_st, err)
		}
		if !derived_providerAccAddress.Equals(expectedAddress) {
			return fmt.Errorf("mismatching %s provider address signature and responseFinalizationData %s , %s", print_st, derived_providerAccAddress, expectedAddress)
		}
		// validate the responses are finalized
		if !k.specKeeper.IsFinalizedBlock(ctx, chainID, request.RelayData.RequestBlock, replyMetadata.LatestBlock) {
			return fmt.Errorf("block isn't finalized on %s provider! %d,%d ", print_st, request.RelayData.RequestBlock, replyMetadata.LatestBlock)
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

	// 5. validate mismatching responses
	if bytes.Equal(conflictData.ConflictRelayData0.Reply.HashAllDataHash, conflictData.ConflictRelayData1.Reply.HashAllDataHash) {
		return fmt.Errorf("no conflict between providers data responses, its the same")
	}

	return nil
}

func (k Keeper) ValidateSameProviderConflict(ctx sdk.Context, conflictData *types.FinalizationConflict, clientAddr sdk.AccAddress) (mismatchingBlockHeight int64, mismatchingBlockHashes map[string]string, err error) {
	// Nil check
	if conflictData == nil {
		return 0, nil, fmt.Errorf("ValidateSameProviderConflict: Conflict data is nil")
	}

	if conflictData.RelayReply0 == nil || conflictData.RelayReply1 == nil {
		return 0, nil, fmt.Errorf("ValidateSameProviderConflict: Reply metadata is nil")
	}

	// Validate Sig of provider and compare addresses
	provider0PubKey, err := sigs.RecoverPubKey(conflictData.RelayReply0)
	if err != nil {
		return 0, nil, fmt.Errorf("ValidateSameProviderConflict: Failed to recover public key: %w", err)
	}

	providerAddress0, err := sdk.AccAddressFromHexUnsafe(provider0PubKey.Address().String())
	if err != nil {
		return 0, nil, fmt.Errorf("ValidateSameProviderConflict: Failed to get provider address: %w", err)
	}

	provider1PubKey, err := sigs.RecoverPubKey(conflictData.RelayReply1)
	if err != nil {
		return 0, nil, fmt.Errorf("ValidateSameProviderConflict: Failed to recover public key: %w", err)
	}

	providerAddress1, err := sdk.AccAddressFromHexUnsafe(provider1PubKey.Address().String())
	if err != nil {
		return 0, nil, fmt.Errorf("ValidateSameProviderConflict: Failed to get provider address: %w", err)
	}

	if !providerAddress0.Equals(providerAddress1) {
		return 0, nil, fmt.Errorf("ValidateSameProviderConflict: Mismatching provider addresses %s, %s", providerAddress0, providerAddress1)
	}

	// Validate client address
	if conflictData.RelayReply0.ConsumerAddress != clientAddr.String() || conflictData.RelayReply1.ConsumerAddress != clientAddr.String() {
		return 0, nil, fmt.Errorf("ValidateSameProviderConflict: Mismatching client addresses %s, %s", conflictData.RelayReply0.ConsumerAddress, conflictData.RelayReply1.ConsumerAddress)
	}

	// Validate matching spec and epoch
	if conflictData.RelayReply0.SpecId != conflictData.RelayReply1.SpecId {
		return 0, nil, fmt.Errorf("ValidateSameProviderConflict: Mismatching spec id %s, %s", conflictData.RelayReply0.SpecId, conflictData.RelayReply1.SpecId)
	}

	if conflictData.RelayReply0.Epoch != conflictData.RelayReply1.Epoch {
		return conflictData.RelayReply0.Epoch, nil, fmt.Errorf("ValidateSameProviderConflict: Mismatching epochs %d, %d", conflictData.RelayReply0.Epoch, conflictData.RelayReply1.Epoch)
	}

	specId := conflictData.RelayReply0.SpecId
	providerAddress := providerAddress0
	epoch := uint64(conflictData.RelayReply0.Epoch)

	// Validate pairing
	project, err := k.pairingKeeper.GetProjectData(ctx, clientAddr, specId, epoch)
	if err != nil {
		return 0, nil, err
	}

	_, _, _, err = k.pairingKeeper.ValidatePairingForClient(ctx, conflictData.RelayReply0.SpecId, providerAddress, epoch, project)
	if err != nil {
		return 0, nil, err
	}

	// Validate block nums are ordered && Finalization distance is right

	finalizedBlocksMap0, minBlock0, maxBlock0, err := k.validateBlockHeights(ctx, conflictData.RelayReply0)
	if err != nil {
		return 0, nil, err
	}

	finalizedBlocksMap1, minBlock1, maxBlock1, err := k.validateBlockHeights(ctx, conflictData.RelayReply0)
	if err != nil {
		return 0, nil, err
	}

	if err := k.validateFinalizedBlock(ctx, conflictData.RelayReply0, maxBlock0); err != nil {
		return 0, nil, err
	}

	if err := k.validateFinalizedBlock(ctx, conflictData.RelayReply1, maxBlock1); err != nil {
		return 0, nil, err
	}

	// Check the hashes between responses
	firstOverlappingBlock := int64(math.Max(float64(minBlock0), float64(minBlock1)))
	lastOverlappingBlock := int64(math.Min(float64(maxBlock0), float64(maxBlock1)))
	if firstOverlappingBlock > lastOverlappingBlock {
		return 0, nil, fmt.Errorf("ValidateSameProviderConflict: No overlapping blocks between providers: %d, %d", minBlock0, minBlock1)
	}

	mismatchingBlockHashes = map[string]string{}
	mismatchingBlockHeight = int64(0)
	for i := firstOverlappingBlock; i <= lastOverlappingBlock; i++ {
		if finalizedBlocksMap0[i] != finalizedBlocksMap1[i] {
			mismatchingBlockHashes[providerAddress0.String()] = finalizedBlocksMap0[i]
			mismatchingBlockHashes[providerAddress1.String()] = finalizedBlocksMap1[i]
			mismatchingBlockHeight = i
		}
	}

	if len(mismatchingBlockHashes) == 0 {
		return 0, mismatchingBlockHashes, fmt.Errorf("ValidateSameProviderConflict: All overlapping blocks are equal between providers")
	}

	return mismatchingBlockHeight, mismatchingBlockHashes, nil
}

func (k Keeper) validateBlockHeights(ctx sdk.Context, relayFinalization *types.RelayFinalization) (finalizedBlocksMarshalled map[int64]string, minBlock int64, maxBlock int64, err error) {
	EMPTY_MAP := map[int64]string{}

	// Unmarshall finalized blocks
	finalizedBlocks := map[int64]string{}
	err = json.Unmarshal(relayFinalization.FinalizedBlocksHashes, &finalizedBlocks)
	if err != nil {
		return EMPTY_MAP, 0, 0, fmt.Errorf("ValidateSameProviderConflict: Failed unmarshalling finalized blocks data: %w", err)
	}

	// Validate that finalized blocks are not empty
	if len(finalizedBlocks) == 0 {
		return EMPTY_MAP, 0, 0, fmt.Errorf("ValidateSameProviderConflict: No finalized blocks data")
	}

	// Sort block heights
	blockHeights := make([]int64, len(finalizedBlocks))
	idx := 0
	for blockNum := range finalizedBlocks {
		blockHeights[idx] = blockNum
		idx++
	}
	slices.SortInt64Slice(blockHeights)

	// Validate that blocks are consecutive
	_, isConsecutive := slices.IsInt64SliceConsecutive(blockHeights)
	if !isConsecutive {
		return EMPTY_MAP, 0, 0, fmt.Errorf("ValidateSameProviderConflict: Finalized blocks are not consecutive: %v", blockHeights)
	}

	// Validate that all finalized blocks are finalized
	for _, blockNum := range blockHeights {
		if !k.specKeeper.IsFinalizedBlock(ctx, relayFinalization.SpecId, blockNum, relayFinalization.GetLatestBlock()) {
			return EMPTY_MAP, 0, 0, fmt.Errorf("ValidateSameProviderConflict: Finalized block is not finalized: %d", blockNum)
		}
	}

	return finalizedBlocks, blockHeights[0], blockHeights[len(blockHeights)-1], nil
}

func (k Keeper) validateFinalizedBlock(ctx sdk.Context, relayFinalization *types.RelayFinalization, maxBlock int64) error {
	latestBlock := relayFinalization.GetLatestBlock()
	blockDistanceToFinalization := relayFinalization.GetBlockDistanceToFinalization()

	// Validate that finalization distance is right
	if maxBlock != latestBlock-blockDistanceToFinalization {
		return fmt.Errorf("ValidateSameProviderConflict: Finalization distance is not right: %d, %d", maxBlock, latestBlock-blockDistanceToFinalization)
	}

	if k.specKeeper.IsFinalizedBlock(ctx, relayFinalization.SpecId, maxBlock+1, latestBlock) {
		return fmt.Errorf("ValidateSameProviderConflict: Finalized block is not in FinalizedBlocksHashes map. Block height: %d", maxBlock+1)
	}

	return nil
}
