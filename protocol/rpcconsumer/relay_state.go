package rpcconsumer

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"

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

func (rs *RelayState) upgradeToArchiveIfNeeded(numberOfRetriesLaunched int, numberOfNodeErrors uint64) {
	if rs == nil || rs.archiveStatus == nil {
		return
	}
	hashes := rs.GetProtocolMessage().GetRequestedBlocksHashes()
	// If we got upgraded and we still got a node error (>= 2) we know upgrade didn't work
	if rs.archiveStatus.isUpgraded.Load() && numberOfNodeErrors >= 2 {
		// Validate the following.
		// 1. That we have applied archive
		// 2. That we had more than one node error (meaning the 2nd was a successful archive [node error] 100%)
		// Now -
		// We know we have applied archive and failed.
		// 1. We can remove the archive, return to the original protocol message,
		// 2. Set all hashes as irrelevant for future queries.
		if !rs.archiveStatus.isHashCached.Load() {
			for _, hash := range hashes {
				rs.cache.AddHashToCache(hash)
			}
			rs.archiveStatus.isHashCached.Store(true)
		}
		return
	}
	if !rs.archiveStatus.isArchive.Load() && len(hashes) > 0 {
		// Launch archive only on the second retry attempt.
		if numberOfRetriesLaunched == 1 {
			// Iterate over all hashes found in relay, if we don't have them in the cache we can try retry on archive.
			// If we are familiar with all, we don't want to allow archive.
			for _, hash := range hashes {
				if !rs.cache.CheckHashInCache(hash) {
					// If we didn't find the hash in the cache we can try archive relay.
					protocolMessage := rs.GetProtocolMessage()
					relayRequestData := protocolMessage.RelayPrivateData()
					// We need to set archive.
					// Create a new relay private data containing the extension.
					userData := protocolMessage.GetUserData()
					// add all existing extensions including archive split by "," so the override will work
					existingExtensionsPlusArchive := strings.Join(append(relayRequestData.Extensions, extensionslib.ArchiveExtension), ",")
					metaDataForArchive := []pairingtypes.Metadata{{Name: common.EXTENSION_OVERRIDE_HEADER_NAME, Value: existingExtensionsPlusArchive}}
					newProtocolMessage, err := rs.relayParser.ParseRelay(rs.ctx, relayRequestData.ApiUrl, string(relayRequestData.Data), relayRequestData.ConnectionType, userData.DappId, userData.ConsumerIp, metaDataForArchive)
					if err != nil {
						utils.LavaFormatError("Failed converting to archive message in shouldRetry", err, utils.LogAttr("relayRequestData", relayRequestData), utils.LogAttr("metadata", metaDataForArchive))
					} else {
						// Creating an archive protocol message, and set it to current protocol message
						rs.SetProtocolMessage(newProtocolMessage)
						// for future batches.
						rs.archiveStatus.isUpgraded.Store(true)
						rs.archiveStatus.isArchive.Store(true)
					}
					break
				}
			}
			// We had node error, and we have a hash parsed.
		}
	}
}
