package rpcconsumer

import (
	"github.com/lavanet/lava/v2/protocol/chainlib"
	"github.com/lavanet/lava/v2/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

type RelayArchiveExtensionEditor struct {
	chainMessage     chainlib.ChainMessage
	relayRequestData *pairingtypes.RelayPrivateData

	originalChainMessageHadArchive     bool
	originalRelayRequestDataHadArchive bool
	originalRelayRequestDataExtensions []string

	archiveExtensions *spectypes.Extension
}

func NewRelayArchiveExtensionEditor(chainMessage chainlib.ChainMessage, relayRequestData *pairingtypes.RelayPrivateData, archiveExtensions *spectypes.Extension) *RelayArchiveExtensionEditor {
	newMessageArchiveExtensionEditor := &RelayArchiveExtensionEditor{
		chainMessage:                       chainMessage,
		relayRequestData:                   relayRequestData,
		originalChainMessageHadArchive:     false,
		originalRelayRequestDataHadArchive: false,
		originalRelayRequestDataExtensions: relayRequestData.GetExtensions(),
		archiveExtensions:                  archiveExtensions,
	}

	if lavaslices.ContainsPredicate(chainMessage.GetExtensions(), isArchiveExtension) {
		newMessageArchiveExtensionEditor.originalChainMessageHadArchive = true
	}

	if lavaslices.Contains(relayRequestData.Extensions, extensionslib.ExtensionTypeArchive) {
		newMessageArchiveExtensionEditor.originalRelayRequestDataHadArchive = true
	}

	return newMessageArchiveExtensionEditor
}

func isArchiveExtension(extension *spectypes.Extension) bool {
	return extension.Name == extensionslib.ExtensionTypeArchive
}

func (maee *RelayArchiveExtensionEditor) AddArchiveExtensionToMessage() {
	if !maee.originalRelayRequestDataHadArchive {
		maee.relayRequestData.Extensions = append(maee.relayRequestData.Extensions, extensionslib.ExtensionTypeArchive)
	}

	if !maee.originalChainMessageHadArchive {
		maee.chainMessage.SetExtension(maee.archiveExtensions)
	}
}

func (maee *RelayArchiveExtensionEditor) RemoveArchiveExtensionFromMessage() {
	if !maee.originalRelayRequestDataHadArchive {
		maee.relayRequestData.Extensions = maee.originalRelayRequestDataExtensions
	}

	if !maee.originalChainMessageHadArchive {
		maee.chainMessage.RemoveExtension(maee.archiveExtensions.Name)
	}
}
