package finalizationverification

import (
	"encoding/json"
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/protocol/lavaprotocol/protocolerrors"
	"github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	"github.com/lavanet/lava/v2/utils/sigs"
	conflicttypes "github.com/lavanet/lava/v2/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

func VerifyFinalizationData(reply *pairingtypes.RelayReply, relayRequest *pairingtypes.RelayRequest, providerAddr string, consumerAcc sdk.AccAddress, latestSessionBlock, blockDistanceForFinalization, blocksInFinalizationProof int64) (finalizedBlocks map[int64]string, errRet error) {
	relayFinalization := conflicttypes.NewRelayFinalizationFromRelaySessionAndRelayReply(relayRequest.RelaySession, reply, consumerAcc)
	recoveredProviderPubKey, err := sigs.RecoverPubKey(relayFinalization)
	if err != nil {
		return nil, utils.LavaFormatWarning("Finalization data verification failed, RecoverPubKey returned error", err)
	}

	recoveredProviderAddr, err := sdk.AccAddressFromHexUnsafe(recoveredProviderPubKey.Address().String())
	if err != nil {
		return nil, utils.LavaFormatWarning("Finalization data verification failed, AccAddressFromHexUnsafe returned error", err)
	}

	if recoveredProviderAddr.String() != providerAddr {
		return nil, utils.LavaFormatError("provider address mismatch in finalization data ",
			errors.Join(common.ProviderFinalizationDataAccountabilityError, lavasession.BlockProviderError, lavasession.SessionOutOfSyncError),
			utils.LogAttr("parsed Address", recoveredProviderAddr.String()),
			utils.LogAttr("expected address", providerAddr),
		)
	}

	finalizedBlocks = map[int64]string{}
	err = json.Unmarshal(reply.FinalizedBlocksHashes, &finalizedBlocks)
	if err != nil {
		return nil, utils.LavaFormatError("failed in unmarshalling finalized blocks data",
			errors.Join(common.ProviderFinalizationDataAccountabilityError, lavasession.BlockProviderError, lavasession.SessionOutOfSyncError),
			utils.LogAttr("FinalizedBlocksHashes", string(reply.FinalizedBlocksHashes)),
			utils.LogAttr("errMsg", err.Error()),
		)
	}

	err = verifyFinalizationDataIntegrity(reply, latestSessionBlock, finalizedBlocks, blockDistanceForFinalization, blocksInFinalizationProof, providerAddr)
	if err != nil {
		return nil, err
	}

	providerLatestBlock := reply.LatestBlock
	seenBlock := relayRequest.RelayData.SeenBlock
	requestBlock := relayRequest.RelayData.RequestBlock

	if providerLatestBlock < lavaslices.Min([]int64{seenBlock, requestBlock}) {
		return nil, utils.LavaFormatError("provider response does not meet consistency requirements",
			errors.Join(common.ProviderFinalizationDataAccountabilityError, lavasession.SessionOutOfSyncError),
			utils.LogAttr("ProviderAddress", relayRequest.RelaySession.Provider),
			utils.LogAttr("providerLatestBlock", providerLatestBlock),
			utils.LogAttr("seenBlock", seenBlock),
			utils.LogAttr("requestBlock", requestBlock),
			utils.LogAttr("provider address", providerAddr),
		)
	}
	return finalizedBlocks, errRet
}

func verifyFinalizationDataIntegrity(reply *pairingtypes.RelayReply, latestSessionBlock int64, finalizedBlocks map[int64]string, blockDistanceForFinalization, blocksInFinalizationProof int64, providerAddr string) (err error) {
	latestBlock := reply.LatestBlock
	sorted := make([]int64, len(finalizedBlocks))
	idx := 0
	maxBlockNum := int64(0)

	if int64(len(finalizedBlocks)) != blocksInFinalizationProof {
		return utils.LavaFormatError("Simulation: provider returned incorrect number of finalized blocks",
			errors.Join(common.ProviderFinalizationDataAccountabilityError, lavasession.BlockProviderError, lavasession.SessionOutOfSyncError),
			utils.LogAttr("Provider", providerAddr),
			utils.LogAttr("blocksInFinalizationProof", blocksInFinalizationProof),
			utils.LogAttr("len(finalizedBlocks)", len(finalizedBlocks)),
			utils.LogAttr("finalizedBlocks", finalizedBlocks),
		)
	}

	for blockNum := range finalizedBlocks {
		// Check if finalized
		if !spectypes.IsFinalizedBlock(blockNum, latestBlock, blockDistanceForFinalization) {
			return utils.LavaFormatError("Simulation: provider returned non finalized block reply for reliability",
				errors.Join(common.ProviderFinalizationDataAccountabilityError, lavasession.BlockProviderError, lavasession.SessionOutOfSyncError),
				utils.LogAttr("blockNum", blockNum),
				utils.LogAttr("latestBlock", latestBlock),
				utils.LogAttr("Provider", providerAddr),
				utils.LogAttr("finalizedBlocks", finalizedBlocks),
			)
		}

		// Add to sorted
		sorted[idx] = blockNum

		// Update maxBlockNum
		if blockNum > maxBlockNum {
			maxBlockNum = blockNum
		}
		idx++
	}

	lavaslices.SortStable(sorted)

	// Check for consecutive blocks
	nonConsecutiveIndex, isConsecutive := lavaslices.IsSliceConsecutive(sorted)
	if !isConsecutive {
		return utils.LavaFormatError("Simulation: provider returned non consecutive finalized blocks reply", protocolerrors.ProviderFinalizationDataAccountabilityError,
			utils.LogAttr("currBlock", sorted[nonConsecutiveIndex]),
			utils.LogAttr("prevBlock", sorted[nonConsecutiveIndex-1]),
			utils.LogAttr("providerAddr", providerAddr),
			utils.LogAttr("finalizedBlocks", finalizedBlocks),
		)
	}

	// Check that latest finalized block address + 1 points to a non finalized block
	if spectypes.IsFinalizedBlock(maxBlockNum+1, latestBlock, blockDistanceForFinalization) {
		return utils.LavaFormatError("Simulation: provider returned finalized hashes for an older latest block", protocolerrors.ProviderFinalizationDataAccountabilityError,
			utils.LogAttr("maxBlockNum", maxBlockNum),
			utils.LogAttr("latestBlock", latestBlock),
			utils.LogAttr("Provider", providerAddr),
			utils.LogAttr("finalizedBlocks", finalizedBlocks),
		)
	}

	// New reply should have blocknum >= from block same provider
	if latestSessionBlock > latestBlock {
		return utils.LavaFormatError("Simulation: Provider supplied an older latest block than it has previously", protocolerrors.ProviderFinalizationDataAccountabilityError,
			utils.LogAttr("session.LatestBlock", latestSessionBlock),
			utils.LogAttr("latestBlock", latestBlock),
			utils.LogAttr("Provider", providerAddr),
		)
	}

	return nil
}
