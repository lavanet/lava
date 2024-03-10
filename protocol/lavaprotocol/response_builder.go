package lavaprotocol

import (
	"context"
	"encoding/json"
	"sort"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	"github.com/lavanet/lava/utils/slices"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

func SignRelayResponse(consumerAddress sdk.AccAddress, request pairingtypes.RelayRequest, pkey *btcSecp256k1.PrivateKey, reply *pairingtypes.RelayReply, signDataReliability bool) (*pairingtypes.RelayReply, error) {
	// request is a copy of the original request, but won't modify it
	// update relay request requestedBlock to the provided one in case it was arbitrary
	UpdateRequestedBlock(request.RelayData, reply)
	// Update signature,
	relayExchange := pairingtypes.NewRelayExchange(request, *reply)
	sig, err := sigs.Sign(pkey, relayExchange)
	if err != nil {
		return nil, utils.LavaFormatError("failed signing relay response", err,
			utils.LogAttr("request", request),
			utils.LogAttr("reply", reply),
		)
	}
	reply.Sig = sig

	if signDataReliability {
		// update sig blocks signature
		relayFinalization := conflicttypes.NewRelayFinalization(request.RelaySession, reply, consumerAddress)
		sigBlocks, err := sigs.Sign(pkey, relayFinalization)
		if err != nil {
			return nil, utils.LavaFormatError("failed signing finalization data", err,
				utils.LogAttr("request", request),
				utils.LogAttr("reply", reply),
				utils.LogAttr("userAddr", consumerAddress),
			)
		}
		reply.SigBlocks = sigBlocks
	}
	return reply, nil
}

func VerifyRelayReply(ctx context.Context, reply *pairingtypes.RelayReply, relayRequest *pairingtypes.RelayRequest, addr string) error {
	relayExchange := pairingtypes.NewRelayExchange(*relayRequest, *reply)
	serverKey, err := sigs.RecoverPubKey(relayExchange)
	if err != nil {
		return err
	}
	serverAddr, err := sdk.AccAddressFromHexUnsafe(serverKey.Address().String())
	if err != nil {
		return err
	}
	if serverAddr.String() != addr {
		return utils.LavaFormatError("reply server address mismatch ", ProviderFinalizationDataError,
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("parsed Address", serverAddr.String()),
			utils.LogAttr("expected address", addr),
			utils.LogAttr("requestedBlock", relayRequest.RelayData.RequestBlock),
			utils.LogAttr("latestBlock", reply.GetLatestBlock()),
		)
	}

	return nil
}

func VerifyFinalizationData(reply *pairingtypes.RelayReply, relayRequest *pairingtypes.RelayRequest, providerAddr string, consumerAcc sdk.AccAddress, latestSessionBlock int64, blockDistanceForFinalization uint32) (finalizedBlocks map[int64]string, finalizationConflict *conflicttypes.FinalizationConflict, errRet error) {
	relayFinalization := conflicttypes.NewRelayFinalization(relayRequest.RelaySession, reply, consumerAcc)
	serverKey, err := sigs.RecoverPubKey(relayFinalization)
	if err != nil {
		return nil, nil, err
	}

	serverAddr, err := sdk.AccAddressFromHexUnsafe(serverKey.Address().String())
	if err != nil {
		return nil, nil, err
	}

	if serverAddr.String() != providerAddr {
		return nil, nil, utils.LavaFormatError("reply server address mismatch in finalization data ", ProviderFinalizationDataError,
			utils.LogAttr("parsed Address", serverAddr.String()),
			utils.LogAttr("expected address", providerAddr),
		)
	}

	finalizedBlocks = map[int64]string{} // TODO:: define struct in relay response
	err = json.Unmarshal(reply.FinalizedBlocksHashes, &finalizedBlocks)
	if err != nil {
		return nil, nil, utils.LavaFormatError("failed in unmarshalling finalized blocks data", ProviderFinalizationDataError,
			utils.LogAttr("FinalizedBlocksHashes", string(reply.FinalizedBlocksHashes)),
			utils.LogAttr("errMsg", err.Error()),
		)
	}

	finalizationConflict, err = verifyFinalizationDataIntegrity(relayRequest.RelaySession, reply, latestSessionBlock, finalizedBlocks, blockDistanceForFinalization, consumerAcc, providerAddr)
	if err != nil {
		return nil, finalizationConflict, err
	}
	providerLatestBlock := reply.LatestBlock
	seenBlock := relayRequest.RelayData.SeenBlock
	requestBlock := relayRequest.RelayData.RequestBlock
	if providerLatestBlock < slices.Min([]int64{seenBlock, requestBlock}) {
		return nil, nil, utils.LavaFormatError("provider response does not meet consistency requirements", ProviderFinalizationDataError,
			utils.LogAttr("ProviderAddress", relayRequest.RelaySession.Provider),
			utils.LogAttr("providerLatestBlock", providerLatestBlock),
			utils.LogAttr("seenBlock", seenBlock),
			utils.LogAttr("requestBlock", requestBlock),
			utils.LogAttr("provider address", providerAddr),
		)
	}
	return finalizedBlocks, finalizationConflict, errRet
}

func verifyFinalizationDataIntegrity(relaySession *pairingtypes.RelaySession, reply *pairingtypes.RelayReply, latestSessionBlock int64, finalizedBlocks map[int64]string, blockDistanceForFinalization uint32, consumerAddr sdk.AccAddress, providerAddr string) (finalizationConflict *conflicttypes.FinalizationConflict, err error) {
	latestBlock := reply.LatestBlock
	sorted := make([]int64, len(finalizedBlocks))
	idx := 0
	maxBlockNum := int64(0)
	// TODO: compare finalizedBlocks len vs chain parser len to validate (get from same place as blockDistanceForFinalization arrives)

	replyFinalization := conflicttypes.NewRelayFinalization(relaySession, reply, consumerAddr)

	for blockNum := range finalizedBlocks {
		if !spectypes.IsFinalizedBlock(blockNum, latestBlock, blockDistanceForFinalization) {
			finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: &replyFinalization}
			return finalizationConflict, utils.LavaFormatError("Simulation: provider returned non finalized block reply for reliability", ProviderFinalizationDataAccountabilityError,
				utils.LogAttr("blockNum", blockNum),
				utils.LogAttr("latestBlock", latestBlock),
				utils.LogAttr("Provider", providerAddr),
				utils.LogAttr("finalizedBlocks", finalizedBlocks),
			)
		}

		sorted[idx] = blockNum

		if blockNum > maxBlockNum {
			maxBlockNum = blockNum
		}
		idx++
		// TODO: check blockhash length and format
	}

	// check for consecutive blocks
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	for index := range sorted {
		if index != 0 && sorted[index]-1 != sorted[index-1] {
			// log.Println("provider returned non consecutive finalized blocks reply.\n Provider: %s", providerAcc)
			finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: &replyFinalization}
			return finalizationConflict, utils.LavaFormatError("Simulation: provider returned non consecutive finalized blocks reply", ProviderFinalizationDataAccountabilityError,
				utils.LogAttr("curr block", sorted[index]),
				utils.LogAttr("prev block", sorted[index-1]),
				utils.LogAttr("Provider", providerAddr),
				utils.LogAttr("finalizedBlocks", finalizedBlocks),
			)
		}
	}

	// check that latest finalized block address + 1 points to a non finalized block
	if spectypes.IsFinalizedBlock(maxBlockNum+1, latestBlock, blockDistanceForFinalization) {
		finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: &replyFinalization}
		return finalizationConflict, utils.LavaFormatError("Simulation: provider returned finalized hashes for an older latest block", ProviderFinalizationDataAccountabilityError,
			utils.LogAttr("maxBlockNum", maxBlockNum),
			utils.LogAttr("latestBlock", latestBlock),
			utils.LogAttr("Provider", providerAddr),
			utils.LogAttr("finalizedBlocks", finalizedBlocks),
		)
	}

	// New reply should have blocknum >= from block same provider
	if latestSessionBlock > latestBlock {
		finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: &replyFinalization}
		return finalizationConflict, utils.LavaFormatError("Simulation: Provider supplied an older latest block than it has previously", ProviderFinalizationDataAccountabilityError,
			utils.LogAttr("session.LatestBlock", latestSessionBlock),
			utils.LogAttr("latestBlock", latestBlock),
			utils.LogAttr("Provider", providerAddr),
		)
	}

	return finalizationConflict, nil
}
