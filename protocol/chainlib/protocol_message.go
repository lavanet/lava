package chainlib

import (
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
)

type BaseProtocolMessage struct {
	ChainMessage
	directiveHeaders map[string]string
	relayRequestData *pairingtypes.RelayPrivateData
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

func NewProtocolMessage(chainMessage ChainMessage, directiveHeaders map[string]string, relayRequestData *pairingtypes.RelayPrivateData) ProtocolMessage {
	return &BaseProtocolMessage{
		ChainMessage:     chainMessage,
		directiveHeaders: directiveHeaders,
		relayRequestData: relayRequestData,
	}
}

type ProtocolMessage interface {
	ChainMessage
	GetDirectiveHeaders() map[string]string
	RelayPrivateData() *pairingtypes.RelayPrivateData
	HashCacheRequest(chainId string) ([]byte, func([]byte) []byte, error)
}
