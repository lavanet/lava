package chainlib

import (
	"strings"

	"github.com/lavanet/lava/v2/protocol/common"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
)

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
}
