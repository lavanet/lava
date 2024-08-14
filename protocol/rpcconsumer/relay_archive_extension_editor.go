package rpcconsumer

import (
	"github.com/lavanet/lava/v2/protocol/chainlib"
	"github.com/lavanet/lava/v2/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

type ArchiveMessageManager struct {
	chainMessage     chainlib.ChainMessage
	relayRequestData *pairingtypes.RelayPrivateData

	originalChainMessageHadArchive     bool
	originalRelayRequestDataHadArchive bool
	originalRelayRequestDataExtensions []string
	archiveExtensions                  *spectypes.Extension
}

func NewArchiveMessageManager(chainMessage chainlib.ChainMessage, relayRequestData *pairingtypes.RelayPrivateData, archiveExtensions *spectypes.Extension) *ArchiveMessageManager {
	archiveMessageManager := &ArchiveMessageManager{
		chainMessage:                       chainMessage,
		relayRequestData:                   relayRequestData,
		originalChainMessageHadArchive:     false,
		originalRelayRequestDataHadArchive: false,
		originalRelayRequestDataExtensions: relayRequestData.GetExtensions(),
		archiveExtensions:                  archiveExtensions,
	}

	if lavaslices.ContainsPredicate(chainMessage.GetExtensions(), isArchiveExtension) {
		archiveMessageManager.originalChainMessageHadArchive = true
	}

	if lavaslices.Contains(relayRequestData.Extensions, extensionslib.ExtensionTypeArchive) {
		archiveMessageManager.originalRelayRequestDataHadArchive = true
	}

	return archiveMessageManager
}

func isArchiveExtension(extension *spectypes.Extension) bool {
	return extension.Name == extensionslib.ExtensionTypeArchive
}

func (rmm *ArchiveMessageManager) SetArchiveExtensionAsOriginal() {
	rmm.AddArchiveExtensionToMessage()

	if !rmm.originalRelayRequestDataHadArchive {
		rmm.relayRequestData.Extensions = append(rmm.relayRequestData.Extensions, extensionslib.ExtensionTypeArchive)
	}

	rmm.originalChainMessageHadArchive = true
	rmm.originalRelayRequestDataHadArchive = true
}

func (rmm *ArchiveMessageManager) IsOriginallyArchiveExtension() bool {
	return rmm.originalChainMessageHadArchive || rmm.originalRelayRequestDataHadArchive
}

func (rmm *ArchiveMessageManager) AddArchiveExtensionToMessage() {
	if !rmm.originalRelayRequestDataHadArchive {
		rmm.relayRequestData.Extensions = append(rmm.relayRequestData.Extensions, extensionslib.ExtensionTypeArchive)
	}

	if !rmm.originalChainMessageHadArchive {
		rmm.chainMessage.SetExtension(rmm.archiveExtensions)
	}
}

func (rmm *ArchiveMessageManager) RemoveArchiveExtensionFromMessage() {
	if !rmm.originalRelayRequestDataHadArchive {
		rmm.relayRequestData.Extensions = rmm.originalRelayRequestDataExtensions
	}

	if !rmm.originalChainMessageHadArchive {
		rmm.chainMessage.RemoveExtension(rmm.archiveExtensions.Name)
	}
}
