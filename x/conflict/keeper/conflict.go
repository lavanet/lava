package keeper

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
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
	if conflictData.ConflictRelayData0.Request.ConnectionType != conflictData.ConflictRelayData1.Request.ConnectionType {
		return fmt.Errorf("mismatching request parameters between providers %s, %s", conflictData.ConflictRelayData0.Request.ConnectionType, conflictData.ConflictRelayData1.Request.ConnectionType)
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

	//1.5 validate params
	epochStart, _, err := k.epochstorageKeeper.GetEpochStartForBlock(ctx, uint64(block))
	if err != nil {
		return fmt.Errorf("could not find epoch for block %d", block)
	}
	if conflictData.ConflictRelayData0.Request.RequestBlock < 0 {
		return fmt.Errorf("invalid request block height %d", conflictData.ConflictRelayData0.Request.RequestBlock)
	}

	epochBlocks, err := k.epochstorageKeeper.EpochBlocks(ctx, uint64(block))
	if err != nil {
		return fmt.Errorf("could not get EpochBlocks param")
	}
	span := k.VoteStartSpan(ctx) * epochBlocks
	if uint64(ctx.BlockHeight())-epochStart >= span {
		return fmt.Errorf("conflict was recieved outside of the allowed span, current: %d, span %d - %d", ctx.BlockHeight(), epochStart, epochStart+span)
	}

	k.pairingKeeper.VerifyPairingData(ctx, chainID, clientAddr, epochStart)
	//2. validate signer
	_, err = k.epochstorageKeeper.GetStakeEntryForClientEpoch(ctx, chainID, clientAddr, epochStart)
	if err != nil {
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
