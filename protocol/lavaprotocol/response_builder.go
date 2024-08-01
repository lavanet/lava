package lavaprotocol

import (
	"context"
	"sort"

	"github.com/goccy/go-json"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	"github.com/lavanet/lava/v2/utils/sigs"
	conflicttypes "github.com/lavanet/lava/v2/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
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
			utils.Attribute{Key: "request", Value: request}, utils.Attribute{Key: "reply", Value: reply})
	}
	reply.Sig = sig

	if signDataReliability {
		// update sig blocks signature
		relayFinalization := pairingtypes.NewRelayFinalization(pairingtypes.NewRelayExchange(request, *reply), consumerAddress)
		sigBlocks, err := sigs.Sign(pkey, relayFinalization)
		if err != nil {
			return nil, utils.LavaFormatError("failed signing finalization data", err,
				utils.Attribute{Key: "request", Value: request}, utils.Attribute{Key: "reply", Value: reply}, utils.Attribute{Key: "userAddr", Value: consumerAddress})
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
		return utils.LavaFormatError("reply server address mismatch ", ProviderFinzalizationDataError, utils.LogAttr("GUID", ctx), utils.Attribute{Key: "parsed Address", Value: serverAddr.String()}, utils.Attribute{Key: "expected address", Value: addr}, utils.Attribute{Key: "requestedBlock", Value: relayRequest.RelayData.RequestBlock}, utils.Attribute{Key: "latestBlock", Value: reply.GetLatestBlock()})
	}

	return nil
}

func VerifyFinalizationData(reply *pairingtypes.RelayReply, relayRequest *pairingtypes.RelayRequest, providerAddr string, consumerAcc sdk.AccAddress, latestSessionBlock int64, blockDistanceForfinalization uint32) (finalizedBlocks map[int64]string, finalizationConflict *conflicttypes.FinalizationConflict, errRet error) {
	relayFinalization := pairingtypes.NewRelayFinalization(pairingtypes.NewRelayExchange(*relayRequest, *reply), consumerAcc)
	serverKey, err := sigs.RecoverPubKey(relayFinalization)
	if err != nil {
		return nil, nil, err
	}

	serverAddr, err := sdk.AccAddressFromHexUnsafe(serverKey.Address().String())
	if err != nil {
		return nil, nil, err
	}

	if serverAddr.String() != providerAddr {
		return nil, nil, utils.LavaFormatError("reply server address mismatch in finalization data ", ProviderFinzalizationDataError, utils.Attribute{Key: "parsed Address", Value: serverAddr.String()}, utils.Attribute{Key: "expected address", Value: providerAddr})
	}

	finalizedBlocks = map[int64]string{} // TODO:: define struct in relay response
	err = json.Unmarshal(reply.FinalizedBlocksHashes, &finalizedBlocks)
	if err != nil {
		return nil, nil, utils.LavaFormatError("failed in unmarshalling finalized blocks data", ProviderFinzalizationDataError, utils.Attribute{Key: "FinalizedBlocksHashes", Value: string(reply.FinalizedBlocksHashes)}, utils.Attribute{Key: "errMsg", Value: err.Error()})
	}

	finalizationConflict, err = verifyFinalizationDataIntegrity(reply, latestSessionBlock, finalizedBlocks, blockDistanceForfinalization, providerAddr)
	if err != nil {
		return nil, finalizationConflict, err
	}
	providerLatestBlock := reply.LatestBlock
	seenBlock := relayRequest.RelayData.SeenBlock
	requestBlock := relayRequest.RelayData.RequestBlock
	if providerLatestBlock < lavaslices.Min([]int64{seenBlock, requestBlock}) {
		return nil, nil, utils.LavaFormatError("provider response does not meet consistency requirements", ProviderFinzalizationDataError, utils.LogAttr("ProviderAddress", relayRequest.RelaySession.Provider), utils.LogAttr("providerLatestBlock", providerLatestBlock), utils.LogAttr("seenBlock", seenBlock), utils.LogAttr("requestBlock", requestBlock), utils.Attribute{Key: "provider address", Value: providerAddr})
	}
	return finalizedBlocks, finalizationConflict, errRet
}

func verifyFinalizationDataIntegrity(reply *pairingtypes.RelayReply, latestSessionBlock int64, finalizedBlocks map[int64]string, blockDistanceForfinalization uint32, providerAddr string) (finalizationConflict *conflicttypes.FinalizationConflict, err error) {
	latestBlock := reply.LatestBlock
	sorted := make([]int64, len(finalizedBlocks))
	idx := 0
	maxBlockNum := int64(0)
	// TODO: compare finalizedBlocks len vs chain parser len to validate (get from same place as blockDistanceForfinalization arrives)

	for blockNum := range finalizedBlocks {
		if !spectypes.IsFinalizedBlock(blockNum, latestBlock, blockDistanceForfinalization) {
			finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: reply}
			return finalizationConflict, utils.LavaFormatError("Simulation: provider returned non finalized block reply for reliability", ProviderFinzalizationDataAccountabilityError, utils.Attribute{Key: "blockNum", Value: blockNum}, utils.Attribute{Key: "latestBlock", Value: latestBlock}, utils.Attribute{Key: "Provider", Value: providerAddr}, utils.Attribute{Key: "finalizedBlocks", Value: finalizedBlocks})
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
			finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: reply}
			return finalizationConflict, utils.LavaFormatError("Simulation: provider returned non consecutive finalized blocks reply", ProviderFinzalizationDataAccountabilityError, utils.Attribute{Key: "curr block", Value: sorted[index]}, utils.Attribute{Key: "prev block", Value: sorted[index-1]}, utils.Attribute{Key: "Provider", Value: providerAddr}, utils.Attribute{Key: "finalizedBlocks", Value: finalizedBlocks})
		}
	}

	// check that latest finalized block address + 1 points to a non finalized block
	if spectypes.IsFinalizedBlock(maxBlockNum+1, latestBlock, blockDistanceForfinalization) {
		finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: reply}
		return finalizationConflict, utils.LavaFormatError("Simulation: provider returned finalized hashes for an older latest block", ProviderFinzalizationDataAccountabilityError,
			utils.Attribute{Key: "maxBlockNum", Value: maxBlockNum},
			utils.Attribute{Key: "latestBlock", Value: latestBlock}, utils.Attribute{Key: "Provider", Value: providerAddr}, utils.Attribute{Key: "finalizedBlocks", Value: finalizedBlocks})
	}

	// New reply should have blocknum >= from block same provider
	if latestSessionBlock > latestBlock {
		finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: reply}
		return finalizationConflict, utils.LavaFormatError("Simulation: Provider supplied an older latest block than it has previously", ProviderFinzalizationDataAccountabilityError,
			utils.Attribute{Key: "session.LatestBlock", Value: latestSessionBlock},
			utils.Attribute{Key: "latestBlock", Value: latestBlock}, utils.Attribute{Key: "Provider", Value: providerAddr})
	}

	return finalizationConflict, nil
}
