package chainlib

import (
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type updatableRPCInput interface {
	rpcInterfaceMessages.GenericMessage
	UpdateLatestBlockInMessage(latestBlock uint64, modifyContent bool) (success bool)
	AppendHeader(metadata []pairingtypes.Metadata)
}

type parsedMessage struct {
	api                    *spectypes.Api
	latestRequestedBlock   int64
	earliestRequestedBlock int64
	msg                    updatableRPCInput
	apiCollection          *spectypes.ApiCollection
	extensions             []*spectypes.Extension
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

func (pm parsedMessage) RequestedBlock() (latest int64, earliest int64) {
	if pm.earliestRequestedBlock == 0 {
		// earliest is optional and not set here
		return pm.latestRequestedBlock, pm.latestRequestedBlock
	}
	return pm.latestRequestedBlock, pm.earliestRequestedBlock
}

func (pm parsedMessage) GetRPCMessage() rpcInterfaceMessages.GenericMessage {
	return pm.msg
}

func (pm *parsedMessage) UpdateLatestBlockInMessage(latestBlock int64, modifyContent bool) (modifiedOnLatestReq bool) {
	requestedBlock, _ := pm.RequestedBlock()
	if latestBlock <= spectypes.NOT_APPLICABLE || requestedBlock != spectypes.LATEST_BLOCK {
		return false
	}
	success := pm.msg.UpdateLatestBlockInMessage(uint64(latestBlock), modifyContent)
	if success {
		pm.latestRequestedBlock = latestBlock
		return true
	}
	return false
}

func (pm *parsedMessage) GetExtensions() []*spectypes.Extension {
	return pm.extensions
}

func (pm *parsedMessage) SetExtension(extension *spectypes.Extension) {
	if len(pm.extensions) > 0 {
		for _, ext := range pm.extensions {
			if ext.Name == extension.Name {
				// already existing, no need to add
				return
			}
		}
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
