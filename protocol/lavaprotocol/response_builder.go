package lavaprotocol

import (
	"context"
	"sort"

	"github.com/goccy/go-json"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/lavaslices"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

func CraftEmptyRPCResponseFromGenericMessage(message rpcInterfaceMessages.GenericMessage) (*rpcInterfaceMessages.RPCResponse, error) {
	createRPCResponse := func(rawId json.RawMessage) (*rpcInterfaceMessages.RPCResponse, error) {
		jsonRpcId, err := rpcInterfaceMessages.IdFromRawMessage(rawId)
		if err != nil {
			return nil, utils.LavaFormatError("failed creating jsonrpc id", err)
		}

		jsonResponse := &rpcInterfaceMessages.RPCResponse{
			JSONRPC: "2.0",
			ID:      jsonRpcId,
			Result:  nil,
			Error:   nil,
		}

		return jsonResponse, nil
	}

	var err error
	var rpcResponse *rpcInterfaceMessages.RPCResponse
	if hasID, ok := message.(interface{ GetID() json.RawMessage }); ok {
		rpcResponse, err = createRPCResponse(hasID.GetID())
		if err != nil {
			return nil, utils.LavaFormatError("failed creating jsonrpc id", err)
		}
	} else {
		rpcResponse, err = createRPCResponse([]byte("1"))
		if err != nil {
			return nil, utils.LavaFormatError("failed creating jsonrpc id", err)
		}
	}

	return rpcResponse, nil
}

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
		return utils.LavaFormatWarning("Relay reply verification failed, RecoverPubKey returned error", err, utils.LogAttr("GUID", ctx))
	}
	serverAddr, err := sdk.AccAddressFromHexUnsafe(serverKey.Address().String())
	if err != nil {
		return utils.LavaFormatWarning("Relay reply verification failed, AccAddressFromHexUnsafe returned error", err, utils.LogAttr("GUID", ctx))
	}
	if serverAddr.String() != addr {
		return utils.LavaFormatError("reply server address mismatch", ProviderFinalizationDataError,
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("parsedAddress", serverAddr.String()),
			utils.LogAttr("expectedAddress", addr),
			utils.LogAttr("requestedBlock", relayRequest.RelayData.RequestBlock),
			utils.LogAttr("latestBlock", reply.GetLatestBlock()),
		)
	}

	return nil
}

func VerifyFinalizationData(reply *pairingtypes.RelayReply, relayRequest *pairingtypes.RelayRequest, providerAddr string, consumerAcc sdk.AccAddress, latestSessionBlock int64, blockDistanceForfinalization uint32) (finalizedBlocks map[int64]string, finalizationConflict *conflicttypes.FinalizationConflict, errRet error) {
	relayFinalization := pairingtypes.NewRelayFinalization(pairingtypes.NewRelayExchange(*relayRequest, *reply), consumerAcc)
	serverKey, err := sigs.RecoverPubKey(relayFinalization)
	if err != nil {
		return nil, nil, utils.LavaFormatWarning("Finalization data verification failed, RecoverPubKey returned error", err)
	}

	serverAddr, err := sdk.AccAddressFromHexUnsafe(serverKey.Address().String())
	if err != nil {
		return nil, nil, utils.LavaFormatWarning("Finalization data verification failed, AccAddressFromHexUnsafe returned error", err)
	}

	if serverAddr.String() != providerAddr {
		return nil, nil, utils.LavaFormatError("reply server address mismatch in finalization data ", ProviderFinalizationDataError, utils.Attribute{Key: "parsed Address", Value: serverAddr.String()}, utils.Attribute{Key: "expected address", Value: providerAddr})
	}

	finalizedBlocks = map[int64]string{} // TODO:: define struct in relay response
	err = json.Unmarshal(reply.FinalizedBlocksHashes, &finalizedBlocks)
	if err != nil {
		return nil, nil, utils.LavaFormatError("failed in unmarshalling finalized blocks data", ProviderFinalizationDataError, utils.Attribute{Key: "FinalizedBlocksHashes", Value: string(reply.FinalizedBlocksHashes)}, utils.Attribute{Key: "errMsg", Value: err.Error()})
	}

	finalizationConflict, err = verifyFinalizationDataIntegrity(reply, latestSessionBlock, finalizedBlocks, blockDistanceForfinalization, providerAddr)
	if err != nil {
		return nil, finalizationConflict, err
	}
	providerLatestBlock := reply.LatestBlock
	seenBlock := relayRequest.RelayData.SeenBlock
	requestBlock := relayRequest.RelayData.RequestBlock
	if providerLatestBlock < lavaslices.Min([]int64{seenBlock, requestBlock}) {
		return nil, nil, utils.LavaFormatError("provider response does not meet consistency requirements", ProviderFinalizationDataError, utils.LogAttr("ProviderAddress", relayRequest.RelaySession.Provider), utils.LogAttr("providerLatestBlock", providerLatestBlock), utils.LogAttr("seenBlock", seenBlock), utils.LogAttr("requestBlock", requestBlock), utils.Attribute{Key: "provider address", Value: providerAddr})
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
			return finalizationConflict, utils.LavaFormatError("Simulation: provider returned non finalized block reply for reliability", ProviderFinalizationDataAccountabilityError, utils.Attribute{Key: "blockNum", Value: blockNum}, utils.Attribute{Key: "latestBlock", Value: latestBlock}, utils.Attribute{Key: "Provider", Value: providerAddr}, utils.Attribute{Key: "finalizedBlocks", Value: finalizedBlocks})
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
			return finalizationConflict, utils.LavaFormatError("Simulation: provider returned non consecutive finalized blocks reply", ProviderFinalizationDataAccountabilityError, utils.Attribute{Key: "curr block", Value: sorted[index]}, utils.Attribute{Key: "prev block", Value: sorted[index-1]}, utils.Attribute{Key: "Provider", Value: providerAddr}, utils.Attribute{Key: "finalizedBlocks", Value: finalizedBlocks})
		}
	}

	// check that latest finalized block address + 1 points to a non finalized block
	if spectypes.IsFinalizedBlock(maxBlockNum+1, latestBlock, blockDistanceForfinalization) {
		finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: reply}
		return finalizationConflict, utils.LavaFormatError("Simulation: provider returned finalized hashes for an older latest block", ProviderFinalizationDataAccountabilityError,
			utils.Attribute{Key: "maxBlockNum", Value: maxBlockNum},
			utils.Attribute{Key: "latestBlock", Value: latestBlock}, utils.Attribute{Key: "Provider", Value: providerAddr}, utils.Attribute{Key: "finalizedBlocks", Value: finalizedBlocks})
	}

	// New reply should have blocknum >= from block same provider
	if latestSessionBlock > latestBlock {
		finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: reply}
		return finalizationConflict, utils.LavaFormatError("Simulation: Provider supplied an older latest block than it has previously", ProviderFinalizationDataAccountabilityError,
			utils.Attribute{Key: "session.LatestBlock", Value: latestSessionBlock},
			utils.Attribute{Key: "latestBlock", Value: latestBlock}, utils.Attribute{Key: "Provider", Value: providerAddr})
	}

	return finalizationConflict, nil
}
