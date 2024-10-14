package chainlib

import (
	"strings"

	"github.com/lavanet/lava/v3/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v3/protocol/common"
	"github.com/lavanet/lava/v3/utils"
	pairingtypes "github.com/lavanet/lava/v3/x/pairing/types"
	"golang.org/x/exp/slices"
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

func (bpm *BaseProtocolMessage) UpdateEarliestAndValidateExtensionRules(extensionParser *extensionslib.ExtensionParser, earliestBlockHashRequested int64, addon string, seenBlock int64) {
	if earliestBlockHashRequested >= 0 {
		success := bpm.UpdateEarliestInMessage(earliestBlockHashRequested)
		if success {
			extensionParser.ExtensionParsing(addon, bpm, uint64(seenBlock))
			currentProtocolExtensions := bpm.GetExtensions()
			currentPrivateDataExtensions := bpm.RelayPrivateData().Extensions
			utils.LavaFormatTrace("[Archive Debug] Trying to add extensions", utils.LogAttr("currentProtocolExtensions", currentProtocolExtensions), utils.LogAttr("currentPrivateDataExtensions", currentPrivateDataExtensions))
			if len(currentProtocolExtensions) > len(currentPrivateDataExtensions) {
				// we need to add the missing extension to the private data.
				for _, ext := range currentProtocolExtensions {
					if !slices.Contains(currentPrivateDataExtensions, ext.Name) {
						currentPrivateDataExtensions = append(currentPrivateDataExtensions, ext.Name)
						if len(currentProtocolExtensions) == len(currentPrivateDataExtensions) {
							break
						}
					}
				}
				bpm.RelayPrivateData().Extensions = currentPrivateDataExtensions
				utils.LavaFormatTrace("[Archive Debug] After Swap", utils.LogAttr("bpm.RelayPrivateData().Extensions", bpm.RelayPrivateData().Extensions))
			}
		}
	}
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
	UpdateEarliestAndValidateExtensionRules(extensionParser *extensionslib.ExtensionParser, earliestBlockHashRequested int64, addon string, seenBlock int64)
}
