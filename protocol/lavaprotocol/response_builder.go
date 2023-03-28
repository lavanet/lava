package lavaprotocol

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

func SignRelayResponse(consumerAddress sdk.AccAddress, request pairingtypes.RelayRequest, pkey *btcSecp256k1.PrivateKey, reply *pairingtypes.RelayReply, signDataReliability bool) (*pairingtypes.RelayReply, error) {
	// request is a copy of the original request, but won't modify it
	// update relay request requestedBlock to the provided one in case it was arbitrary
	UpdateRequestedBlock(request.RelayData, reply)
	// Update signature,
	sig, err := sigs.SignRelayResponse(pkey, reply, &request)
	if err != nil {
		return nil, utils.LavaFormatError("failed signing relay response", err,
			&map[string]string{"request": fmt.Sprintf("%v", request), "reply": fmt.Sprintf("%v", reply)})
	}
	reply.Sig = sig

	if signDataReliability {
		// update sig blocks signature
		sigBlocks, err := sigs.SignResponseFinalizationData(pkey, reply, &request, consumerAddress)
		if err != nil {
			return nil, utils.LavaFormatError("failed signing finalization data", err,
				&map[string]string{"request": fmt.Sprintf("%v", request), "reply": fmt.Sprintf("%v", reply), "userAddr": consumerAddress.String()})
		}
		reply.SigBlocks = sigBlocks
	}
	return reply, nil
}

func VerifyRelayReply(reply *pairingtypes.RelayReply, relayRequest *pairingtypes.RelayRequest, addr string) error {
	serverKey, err := sigs.RecoverPubKeyFromRelayReply(reply, relayRequest)
	if err != nil {
		return err
	}
	serverAddr, err := sdk.AccAddressFromHex(serverKey.Address().String())
	if err != nil {
		return err
	}
	if serverAddr.String() != addr {
		return utils.LavaFormatError("reply server address mismatch ", ProviderFinzalizationDataError, &map[string]string{"parsed Address": serverAddr.String(), "expected address": addr})
	}

	return nil
}

func VerifyFinalizationData(reply *pairingtypes.RelayReply, relayRequest *pairingtypes.RelayRequest, addr string, latestSessionBlock int64, blockDistanceForfinalization uint32) (finalizedBlocks map[int64]string, finalizationConflict *conflicttypes.FinalizationConflict, errRet error) {
	strAdd, err := sdk.AccAddressFromBech32(addr)
	if err != nil {
		return nil, nil, err
	}
	serverKey, err := sigs.RecoverPubKeyFromResponseFinalizationData(reply, relayRequest, strAdd)
	if err != nil {
		return nil, nil, err
	}

	serverAddr, err := sdk.AccAddressFromHex(serverKey.Address().String())
	if err != nil {
		return nil, nil, err
	}

	if serverAddr.String() != addr {
		return nil, nil, utils.LavaFormatError("reply server address mismatch in finalization data ", ProviderFinzalizationDataError, &map[string]string{"parsed Address": serverAddr.String(), "expected address": addr})
	}

	finalizedBlocks = map[int64]string{} // TODO:: define struct in relay response
	err = json.Unmarshal(reply.FinalizedBlocksHashes, &finalizedBlocks)
	if err != nil {
		return nil, nil, utils.LavaFormatError("failed in unmarshalling finalized blocks data", ProviderFinzalizationDataError, &map[string]string{"FinalizedBlocksHashes": string(reply.FinalizedBlocksHashes), "errMsg": err.Error()})
	}

	finalizationConflict, err = verifyFinalizationDataIntegrity(reply, latestSessionBlock, finalizedBlocks, blockDistanceForfinalization, addr)
	if err != nil {
		return nil, finalizationConflict, err
	}
	return
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
			return finalizationConflict, utils.LavaFormatError("Simulation: provider returned non finalized block reply for reliability", ProviderFinzalizationDataAccountabilityError, &map[string]string{"blockNum": strconv.FormatInt(blockNum, 10), "latestBlock": strconv.FormatInt(latestBlock, 10), "Provider": providerAddr, "finalizedBlocks": fmt.Sprintf("%+v", finalizedBlocks)})
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
			return finalizationConflict, utils.LavaFormatError("Simulation: provider returned non consecutive finalized blocks reply", ProviderFinzalizationDataAccountabilityError, &map[string]string{"curr block": strconv.FormatInt(sorted[index], 10), "prev block": strconv.FormatInt(sorted[index-1], 10), "Provider": providerAddr, "finalizedBlocks": fmt.Sprintf("%+v", finalizedBlocks)})
		}
	}

	// check that latest finalized block address + 1 points to a non finalized block
	if spectypes.IsFinalizedBlock(maxBlockNum+1, latestBlock, blockDistanceForfinalization) {
		finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: reply}
		return finalizationConflict, utils.LavaFormatError("Simulation: provider returned finalized hashes for an older latest block", ProviderFinzalizationDataAccountabilityError, &map[string]string{
			"maxBlockNum": strconv.FormatInt(maxBlockNum, 10),
			"latestBlock": strconv.FormatInt(latestBlock, 10), "Provider": providerAddr, "finalizedBlocks": fmt.Sprintf("%+v", finalizedBlocks),
		})
	}

	// New reply should have blocknum >= from block same provider
	if latestSessionBlock > latestBlock {
		finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: reply}
		return finalizationConflict, utils.LavaFormatError("Simulation: Provider supplied an older latest block than it has previously", ProviderFinzalizationDataAccountabilityError, &map[string]string{
			"session.LatestBlock": strconv.FormatInt(latestSessionBlock, 10),
			"latestBlock":         strconv.FormatInt(latestBlock, 10), "Provider": providerAddr,
		})
	}

	return finalizationConflict, nil
}
