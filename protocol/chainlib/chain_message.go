package chainlib

import (
	"github.com/lavanet/lava/protocol/parser"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type updatableRPCInput interface {
	parser.RPCInput
	UpdateLatestBlockInMessage(latestBlock uint64, modifyContent bool) (success bool)
	AppendHeader(metadata []pairingtypes.Metadata)
}

type parsedMessage struct {
	api            *spectypes.Api
	requestedBlock int64
	msg            updatableRPCInput
	apiCollection  *spectypes.ApiCollection
	extensions     []*spectypes.Extension
}

func (pm parsedMessage) AppendHeader(metadata []pairingtypes.Metadata) {
	pm.msg.AppendHeader(metadata)
}

func (pm parsedMessage) GetApi() *spectypes.Api {
	return pm.api
}

func (pm parsedMessage) GetApiCollection() *spectypes.ApiCollection {
	return pm.apiCollection
}

func (pm parsedMessage) RequestedBlock() int64 {
	return pm.requestedBlock
}

func (pm parsedMessage) GetRPCMessage() parser.RPCInput {
	return pm.msg
}

func (pm *parsedMessage) UpdateLatestBlockInMessage(latestBlock int64, modifyContent bool) (modifiedOnLatestReq bool) {
	if latestBlock <= spectypes.NOT_APPLICABLE || pm.RequestedBlock() != spectypes.LATEST_BLOCK {
		return false
	}
	success := pm.msg.UpdateLatestBlockInMessage(uint64(latestBlock), modifyContent)
	if success {
		pm.requestedBlock = latestBlock
		return true
	}
	return false
}

func (pm *parsedMessage) GetExtensions() []*spectypes.Extension {
	return pm.extensions
}

func (pm *parsedMessage) SetExtension(extension *spectypes.Extension) {
	if len(pm.extensions) > 0 {
		pm.extensions = append(pm.extensions, extension)
	} else {
		pm.extensions = []*spectypes.Extension{extension}
	}
}

type CraftData struct {
	Path           string
	Data           []byte
	ConnectionType string
}

func CraftChainMessage(parsing *spectypes.ParseDirective, connectionType string, chainParser ChainParser, craftData *CraftData, metadata []pairingtypes.Metadata) (ChainMessageForSend, error) {
	return chainParser.CraftMessage(parsing, connectionType, craftData, metadata)
}
