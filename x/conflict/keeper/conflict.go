package keeper

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils/sigs"
	"github.com/lavanet/lava/v2/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
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
		stakeEntry, found := k.epochstorageKeeper.GetStakeEntryForProviderEpoch(ctx, chainID, providerAddress.String(), epochStart)
		if !found {
			return nil, fmt.Errorf("did not find a stake entry for %s provider %s on epoch %d, chainID %s", print_st, providerAddress, epochStart, chainID)
		}

		if stakeEntry.IsAddressVaultAndNotProvider(providerAddress.String()) {
			return nil, fmt.Errorf("provider vault address should not be used in conflict. vault: %s, provider: %s, chainID: %s, epoch: %d", stakeEntry.Vault, stakeEntry.Address, chainID, epochStart)
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
	validateResponseFinalizationData := func(expectedAddress sdk.AccAddress, response *types.ReplyMetadata, request *pairingtypes.RelayRequest, first bool) (err error) {
		print_st := "first"
		if !first {
			print_st = "second"
		}

		metaData := types.NewRelayFinalizationMetaData(*response, *request, clientAddr)
		pubKey, err := sigs.RecoverPubKey(metaData)
		if err != nil {
			return fmt.Errorf("RecoverPubKey %s provider ResponseFinalizationData: %w", print_st, err)
		}
		derived_providerAccAddress, err := sdk.AccAddressFromHexUnsafe(pubKey.Address().String())
		if err != nil {
			return fmt.Errorf("AccAddressFromHex %s provider ResponseFinalizationData: %w", print_st, err)
		}
		if !derived_providerAccAddress.Equals(expectedAddress) {
			return fmt.Errorf("mismatching %s provider address signature and responseFinazalizationData %s , %s", print_st, derived_providerAccAddress, expectedAddress)
		}
		// validate the responses are finalized
		if !k.specKeeper.IsFinalizedBlock(ctx, chainID, request.RelayData.RequestBlock, response.LatestBlock) {
			return fmt.Errorf("block isn't finalized on %s provider! %d,%d ", print_st, request.RelayData.RequestBlock, response.LatestBlock)
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

func (k Keeper) ValidateSameProviderConflict(ctx sdk.Context, conflictData *types.FinalizationConflict, clientAddr sdk.AccAddress) error {
	return nil
}
