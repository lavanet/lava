package rpcconsumer

import (
	"context"
	"strings"

	"github.com/lavanet/lava/v4/protocol/chainlib"
	"github.com/lavanet/lava/v4/protocol/chainlib/extensionslib"
	common "github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/utils"
	slices "github.com/lavanet/lava/v4/utils/lavaslices"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
)

type RetryHashCacheInf interface {
	CheckHashInCache(hash string) bool
	AddHashToCache(hash string)
}

type RelayParserInf interface {
	ParseRelay(
		ctx context.Context,
		url string,
		req string,
		connectionType string,
		dappID string,
		consumerIp string,
		metadata []pairingtypes.Metadata,
	) (protocolMessage chainlib.ProtocolMessage, err error)
}

type ArchiveStatus struct {
	isArchive    bool
	isUpgraded   bool
	isHashCached bool
}

type RelayState struct {
	archiveStatus   ArchiveStatus
	stateNumber     int
	protocolMessage chainlib.ProtocolMessage
	cache           RetryHashCacheInf
	relayParser     RelayParserInf
	ctx             context.Context
}

func NewRelayState(ctx context.Context, protocolMessage chainlib.ProtocolMessage, stateNumber int, cache RetryHashCacheInf, relayParser RelayParserInf, archiveInfo ArchiveStatus) *RelayState {
	relayRequestData := protocolMessage.RelayPrivateData()
	isArchive := false
	if slices.Contains(relayRequestData.Extensions, extensionslib.ArchiveExtension) {
		isArchive = true
	}
	return &RelayState{ctx: ctx, protocolMessage: protocolMessage, stateNumber: stateNumber, cache: cache, relayParser: relayParser, archiveStatus: ArchiveStatus{isArchive: isArchive, isUpgraded: archiveInfo.isUpgraded, isHashCached: archiveInfo.isHashCached}}
}

func (rs *RelayState) GetStateNumber() int {
	if rs == nil {
		return 0
	}
	return rs.stateNumber
}

func (rs *RelayState) GetProtocolMessage() chainlib.ProtocolMessage {
	return rs.protocolMessage
}

func (rs *RelayState) upgradeToArchiveIfNeeded(numberOfRetriesLaunched int, numberOfNodeErrors uint64) bool {
	hashes := rs.protocolMessage.GetRequestedBlocksHashes()
	// If we got upgraded and we still got a node error (>= 2) we know upgrade didn't work
	if rs.archiveStatus.isUpgraded && numberOfNodeErrors >= 2 {
		// Validate the following.
		// 1. That we have applied archive
		// 2. That we had more than one node error (meaning the 2nd was a successful archive [node error] 100%)
		// Now -
		// We know we have applied archive and failed.
		// 1. We can remove the archive, return to the original protocol message,
		// 2. Set all hashes as irrelevant for future queries.
		if !rs.archiveStatus.isHashCached {
			for _, hash := range hashes {
				rs.cache.AddHashToCache(hash)
			}
			rs.archiveStatus.isHashCached = true
		}
		// We do not want to send additional relays after archive attempt. return false.
		return false
	}
	if !rs.archiveStatus.isArchive && len(hashes) > 0 && numberOfNodeErrors > 0 {
		// Launch archive only on the second retry attempt.
		if numberOfRetriesLaunched == 1 {
			// Iterate over all hashes found in relay, if we don't have them in the cache we can try retry on archive.
			// If we are familiar with all, we don't want to allow archive.
			for _, hash := range hashes {
				if !rs.cache.CheckHashInCache(hash) {
					// If we didn't find the hash in the cache we can try archive relay.
					relayRequestData := rs.protocolMessage.RelayPrivateData()
					// We need to set archive.
					// Create a new relay private data containing the extension.
					userData := rs.protocolMessage.GetUserData()
					// add all existing extensions including archive split by "," so the override will work
					existingExtensionsPlusArchive := strings.Join(append(relayRequestData.Extensions, extensionslib.ArchiveExtension), ",")
					metaDataForArchive := []pairingtypes.Metadata{{Name: common.EXTENSION_OVERRIDE_HEADER_NAME, Value: existingExtensionsPlusArchive}}
					newProtocolMessage, err := rs.relayParser.ParseRelay(rs.ctx, relayRequestData.ApiUrl, string(relayRequestData.Data), relayRequestData.ConnectionType, userData.DappId, userData.ConsumerIp, metaDataForArchive)
					if err != nil {
						utils.LavaFormatError("Failed converting to archive message in shouldRetry", err, utils.LogAttr("relayRequestData", relayRequestData), utils.LogAttr("metadata", metaDataForArchive))
					}
					// Creating an archive protocol message, and set it to current protocol message
					rs.protocolMessage = newProtocolMessage
					// for future batches.
					rs.archiveStatus.isUpgraded = true
					rs.archiveStatus.isArchive = true
					break
				}
			}
			// We had node error, and we have a hash parsed.
		}
	}
	return true
}
