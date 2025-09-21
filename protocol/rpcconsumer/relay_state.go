package rpcconsumer

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
	common "github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/utils"
	slices "github.com/lavanet/lava/v5/utils/lavaslices"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
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
	isArchive      atomic.Bool
	isUpgraded     atomic.Bool
	isHashCached   atomic.Bool
	isEarliestUsed atomic.Bool
}

func (as *ArchiveStatus) Copy() *ArchiveStatus {
	archiveStatus := &ArchiveStatus{}
	archiveStatus.isArchive.Store(as.isArchive.Load())
	archiveStatus.isUpgraded.Store(as.isUpgraded.Load())
	archiveStatus.isHashCached.Store(as.isHashCached.Load())
	archiveStatus.isEarliestUsed.Store(as.isEarliestUsed.Load())
	return archiveStatus
}

type RelayState struct {
	archiveStatus   *ArchiveStatus
	stateNumber     int
	protocolMessage chainlib.ProtocolMessage
	cache           RetryHashCacheInf
	relayParser     RelayParserInf
	ctx             context.Context
	lock            sync.RWMutex
}

func GetEmptyRelayState(ctx context.Context, protocolMessage chainlib.ProtocolMessage) *RelayState {
	archiveStatus := &ArchiveStatus{}
	archiveStatus.isEarliestUsed.Store(true)
	return &RelayState{
		ctx:             ctx,
		protocolMessage: protocolMessage,
		archiveStatus:   archiveStatus,
	}
}

func NewRelayState(ctx context.Context, protocolMessage chainlib.ProtocolMessage, stateNumber int, cache RetryHashCacheInf, relayParser RelayParserInf, archiveStatus *ArchiveStatus) *RelayState {
	relayRequestData := protocolMessage.RelayPrivateData()
	if archiveStatus == nil {
		utils.LavaFormatError("misuse detected archiveStatus is nil", nil, utils.Attribute{Key: "protocolMessage.GetApi", Value: protocolMessage.GetApi()})
		archiveStatus = &ArchiveStatus{}
	}
	rs := &RelayState{
		ctx:             ctx,
		protocolMessage: protocolMessage,
		stateNumber:     stateNumber,
		cache:           cache,
		relayParser:     relayParser,
		archiveStatus:   archiveStatus,
	}
	rs.archiveStatus.isArchive.Store(rs.CheckIsArchive(relayRequestData))
	return rs
}

func (rs *RelayState) CheckIsArchive(relayRequestData *pairingtypes.RelayPrivateData) bool {
	return relayRequestData != nil && slices.Contains(relayRequestData.Extensions, extensionslib.ArchiveExtension)
}

func (rs *RelayState) GetIsEarliestUsed() bool {
	if rs == nil || rs.archiveStatus == nil {
		return true
	}
	return rs.archiveStatus.isEarliestUsed.Load()
}

func (rs *RelayState) GetIsArchive() bool {
	if rs == nil {
		return false
	}
	return rs.archiveStatus.isArchive.Load()
}

func (rs *RelayState) GetIsUpgraded() bool {
	if rs == nil {
		return false
	}
	return rs.archiveStatus.isUpgraded.Load()
}

func (rs *RelayState) SetIsEarliestUsed() {
	if rs == nil || rs.archiveStatus == nil {
		return
	}
	rs.archiveStatus.isEarliestUsed.Store(true)
}

func (rs *RelayState) SetIsArchive(isArchive bool) {
	if rs == nil || rs.archiveStatus == nil {
		return
	}
	rs.archiveStatus.isArchive.Store(isArchive)
}

func (rs *RelayState) GetStateNumber() int {
	if rs == nil {
		return 0
	}
	return rs.stateNumber
}

func (rs *RelayState) GetProtocolMessage() chainlib.ProtocolMessage {
	if rs == nil {
		return nil
	}
	rs.lock.RLock()
	defer rs.lock.RUnlock()
	return rs.protocolMessage
}

func (rs *RelayState) SetProtocolMessage(protocolMessage chainlib.ProtocolMessage) {
	if rs == nil {
		return
	}
	rs.lock.Lock()
	defer rs.lock.Unlock()
	rs.protocolMessage = protocolMessage
}

// Static function to determine if archive upgrade is needed and return the appropriate protocol message
// This doesn't require a RelayState object, avoiding the need to create it twice
func upgradeToArchiveIfNeeded(ctx context.Context, protocolMessage chainlib.ProtocolMessage, archiveStatus *ArchiveStatus, relayParser RelayParserInf, cache RetryHashCacheInf, numberOfRetriesLaunched int, numberOfNodeErrors uint64) chainlib.ProtocolMessage {
	if archiveStatus == nil {
		return protocolMessage
	}

	hashes := protocolMessage.GetRequestedBlocksHashes()
	// If we got upgraded and we still got a node error (>= 2) we know upgrade didn't work
	if archiveStatus.isUpgraded.Load() && numberOfNodeErrors >= 2 {
		// Validate the following.
		// 1. That we have applied archive
		// 2. That we had more than one node error (meaning the 2nd was a successful archive [node error] 100%)
		// Now -
		// We know we have applied archive and failed.
		// 1. We can remove the archive, return to the original protocol message,
		// 2. Set all hashes as irrelevant for future queries.
		if !archiveStatus.isHashCached.Load() {
			for _, hash := range hashes {
				cache.AddHashToCache(hash)
			}
			archiveStatus.isHashCached.Store(true)
		}
		return protocolMessage
	}

	relayRequestData := protocolMessage.RelayPrivateData()
	if relayRequestData == nil {
		utils.LavaFormatError("Relay request data is nil", nil, utils.LogAttr("GUID", ctx))
		return protocolMessage
	}
	userData := protocolMessage.GetUserData()
	if !archiveStatus.isArchive.Load() && numberOfRetriesLaunched == 1 {
		utils.LavaFormatTrace("Launching archive on first retry", utils.LogAttr("GUID", ctx))
		// Launch archive only on the second retry attempt.
		// Create a new relay private data containing the extension.
		// add all existing extensions including archive split by "," so the override will work
		existingExtensionsPlusArchive := strings.Join(append(relayRequestData.Extensions, extensionslib.ArchiveExtension), ",")
		metaDataForArchive := []pairingtypes.Metadata{{Name: common.EXTENSION_OVERRIDE_HEADER_NAME, Value: existingExtensionsPlusArchive}}
		utils.LavaFormatTrace("[Archive Debug] Calling ParseRelay with archive extension", utils.LogAttr("existingExtensionsPlusArchive", existingExtensionsPlusArchive), utils.LogAttr("metaDataForArchive", metaDataForArchive), utils.LogAttr("apiUrl", relayRequestData.ApiUrl), utils.LogAttr("relayRequestData", relayRequestData), utils.LogAttr("GUID", ctx))
		newProtocolMessage, err := relayParser.ParseRelay(ctx, relayRequestData.ApiUrl, string(relayRequestData.Data), relayRequestData.ConnectionType, userData.DappId, userData.ConsumerIp, metaDataForArchive)
		if err != nil {
			utils.LavaFormatError("Failed converting to archive message in shouldRetry", err, utils.LogAttr("relayRequestData", relayRequestData), utils.LogAttr("metadata", metaDataForArchive))
			return protocolMessage
		} else {
			// Update archive status
			archiveStatus.isUpgraded.Store(true)
			archiveStatus.isArchive.Store(true)
			utils.LavaFormatTrace("[Archive Debug] Archive retry successful", utils.LogAttr("extensions", newProtocolMessage.GetExtensions()), utils.LogAttr("relayPrivateData", newProtocolMessage.RelayPrivateData()), utils.LogAttr("GUID", ctx))
			return newProtocolMessage
		}
	} else if archiveStatus.isUpgraded.Load() && numberOfRetriesLaunched == 2 {
		utils.LavaFormatTrace("Removing archive on second retry", utils.LogAttr("GUID", ctx))
		// Remove archive extension when on second retry
		filteredExtensions := make([]string, 0, len(relayRequestData.Extensions))
		for _, ext := range relayRequestData.Extensions {
			if ext != extensionslib.ArchiveExtension {
				filteredExtensions = append(filteredExtensions, ext)
			}
		}
		existingExtensions := strings.Join(filteredExtensions, ",")
		metaDataForArchive := []pairingtypes.Metadata{{Name: common.EXTENSION_OVERRIDE_HEADER_NAME, Value: existingExtensions}}
		newProtocolMessage, err := relayParser.ParseRelay(ctx, relayRequestData.ApiUrl, string(relayRequestData.Data), relayRequestData.ConnectionType, userData.DappId, userData.ConsumerIp, metaDataForArchive)
		if err != nil {
			utils.LavaFormatError("Failed converting to regular message in shouldRetry", err, utils.LogAttr("relayRequestData", relayRequestData), utils.LogAttr("metadata", metaDataForArchive))
			return protocolMessage
		} else {
			archiveStatus.isArchive.Store(false)
			return newProtocolMessage
		}
	}
	return protocolMessage
}

// Legacy method wrapper for backward compatibility
func (rs *RelayState) upgradeToArchiveIfNeeded(numberOfRetriesLaunched int, numberOfNodeErrors uint64) {
	if rs == nil || rs.archiveStatus == nil {
		return
	}

	// Use the static function to get the upgraded protocol message
	upgradedProtocolMessage := upgradeToArchiveIfNeeded(rs.ctx, rs.GetProtocolMessage(), rs.archiveStatus, rs.relayParser, rs.cache, numberOfRetriesLaunched, numberOfNodeErrors)

	// Update the RelayState with the new protocol message
	rs.SetProtocolMessage(upgradedProtocolMessage)
}
