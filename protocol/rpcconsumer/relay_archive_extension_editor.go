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

func (raee *RelayArchiveExtensionEditor) SetArchiveExtensionAsOriginal() {
	raee.AddArchiveExtensionToMessage()

	if !raee.originalRelayRequestDataHadArchive {
		raee.relayRequestData.Extensions = append(raee.relayRequestData.Extensions, extensionslib.ExtensionTypeArchive)
	}

	raee.originalChainMessageHadArchive = true
	raee.originalRelayRequestDataHadArchive = true
}

func (raee *RelayArchiveExtensionEditor) IsOriginallyArchiveExtension() bool {
	return raee.originalChainMessageHadArchive || raee.originalRelayRequestDataHadArchive
}

func (raee *RelayArchiveExtensionEditor) AddArchiveExtensionToMessage() {
	if !raee.originalRelayRequestDataHadArchive {
		raee.relayRequestData.Extensions = append(raee.relayRequestData.Extensions, extensionslib.ExtensionTypeArchive)
	}

	if !raee.originalChainMessageHadArchive {
		raee.chainMessage.SetExtension(raee.archiveExtensions)
	}
}

func (raee *RelayArchiveExtensionEditor) RemoveArchiveExtensionFromMessage() {
	if !raee.originalRelayRequestDataHadArchive {
		raee.relayRequestData.Extensions = raee.originalRelayRequestDataExtensions
	}

	if !raee.originalChainMessageHadArchive {
		raee.chainMessage.RemoveExtension(raee.archiveExtensions.Name)
	}
}
