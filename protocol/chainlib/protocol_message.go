package chainlib

import (
	"strings"

	"github.com/lavanet/lava/v4/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/utils"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	"github.com/lavanet/lava/v4/x/spec/types"
)

type UserData struct {
	ConsumerIp string
	DappId     string
}

type BaseProtocolMessage struct {
	ChainMessage
	directiveHeaders map[string]string
	relayRequestData *pairingtypes.RelayPrivateData
	userData         common.UserData
}

func (bpm *BaseProtocolMessage) GetUserData() common.UserData {
	return bpm.userData
}

func (bpm *BaseProtocolMessage) GetDirectiveHeaders() map[string]string {
	return bpm.directiveHeaders
}

func (bpm *BaseProtocolMessage) RelayPrivateData() *pairingtypes.RelayPrivateData {
	return bpm.relayRequestData
}

func (bpm *BaseProtocolMessage) HashCacheRequest(chainId string) ([]byte, func([]byte) []byte, error) {
	return HashCacheRequest(bpm.relayRequestData, chainId)
}

// addMissingExtensions adds any extensions from updatedProtocolExtensions that are not in currentPrivateDataExtensions
func (bpm *BaseProtocolMessage) addMissingExtensions(updatedProtocolExtensions []*types.Extension, currentPrivateDataExtensions []string) []string {
	// Create a map for O(1) lookups
	existingExtensions := make(map[string]struct{}, len(currentPrivateDataExtensions))
	for _, ext := range currentPrivateDataExtensions {
		existingExtensions[ext] = struct{}{}
	}

	// Add missing extensions
	for _, ext := range updatedProtocolExtensions {
		if _, exists := existingExtensions[ext.Name]; !exists {
			currentPrivateDataExtensions = append(currentPrivateDataExtensions, ext.Name)
			if len(updatedProtocolExtensions) == len(currentPrivateDataExtensions) {
				break
			}
		}
	}
	return currentPrivateDataExtensions
}

func (bpm *BaseProtocolMessage) UpdateEarliestAndValidateExtensionRules(extensionParser *extensionslib.ExtensionParser, earliestBlockHashRequested int64, addon string, seenBlock int64) bool {
	if earliestBlockHashRequested >= 0 {
		success := bpm.UpdateEarliestInMessage(earliestBlockHashRequested)
		// check if we successfully updated the earliest block in the message
		if success {
			// parse the extensions for the new updated earliest block
			extensionParser.ExtensionParsing(addon, bpm, uint64(seenBlock))
			updatedProtocolExtensions := bpm.GetExtensions()
			currentPrivateDataExtensions := bpm.RelayPrivateData().Extensions
			utils.LavaFormatTrace("[Archive Debug] Trying to add extensions", utils.LogAttr("currentProtocolExtensions", updatedProtocolExtensions), utils.LogAttr("currentPrivateDataExtensions", currentPrivateDataExtensions))
			if len(updatedProtocolExtensions) > len(currentPrivateDataExtensions) {
				// we need to add the missing extension to the private data.
				currentPrivateDataExtensions = bpm.addMissingExtensions(updatedProtocolExtensions, currentPrivateDataExtensions)
				bpm.RelayPrivateData().Extensions = currentPrivateDataExtensions
				utils.LavaFormatTrace("[Archive Debug] After Swap", utils.LogAttr("bpm.RelayPrivateData().Extensions", bpm.RelayPrivateData().Extensions))
				return true
			}
		}
	}
	return false
}

func (bpm *BaseProtocolMessage) GetBlockedProviders() []string {
	if bpm.directiveHeaders == nil {
		return nil
	}
	blockedProviders, ok := bpm.directiveHeaders[common.BLOCK_PROVIDERS_ADDRESSES_HEADER_NAME]
	if ok {
		blockProviders := strings.Split(blockedProviders, ",")
		if len(blockProviders) <= 2 {
			return blockProviders
		}
	}
	return nil
}

func NewProtocolMessage(chainMessage ChainMessage, directiveHeaders map[string]string, relayRequestData *pairingtypes.RelayPrivateData, dappId, consumerIp string) ProtocolMessage {
	return &BaseProtocolMessage{
		ChainMessage:     chainMessage,
		directiveHeaders: directiveHeaders,
		relayRequestData: relayRequestData,
		userData:         common.UserData{DappId: dappId, ConsumerIp: consumerIp},
	}
}

type ProtocolMessage interface {
	ChainMessage
	GetDirectiveHeaders() map[string]string
	RelayPrivateData() *pairingtypes.RelayPrivateData
	HashCacheRequest(chainId string) ([]byte, func([]byte) []byte, error)
	GetBlockedProviders() []string
	GetUserData() common.UserData
	UpdateEarliestAndValidateExtensionRules(extensionParser *extensionslib.ExtensionParser, earliestBlockHashRequested int64, addon string, seenBlock int64) bool
}
