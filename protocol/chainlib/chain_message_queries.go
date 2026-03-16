package chainlib

import (
	"github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/x/spec/types"
)

func ShouldSendToAllProviders(chainMessage ChainMessage) bool {
	return chainMessage.GetApi().Category.Stateful == common.CONSISTENCY_SELECT_ALL_PROVIDERS
}

func GetAddon(chainMessage ChainMessageForSend) string {
	return chainMessage.GetApiCollection().CollectionData.AddOn
}

func IsHangingApi(chainMessage ChainMessageForSend) bool {
	return chainMessage.GetApi().Category.HangingApi
}

func GetComputeUnits(chainMessage ChainMessageForSend) uint64 {
	return chainMessage.GetApi().ComputeUnits
}

func GetStateful(chainMessage ChainMessageForSend) uint32 {
	return chainMessage.GetApi().Category.Stateful
}

func GetParseDirective(api *types.Api, apiCollection *types.ApiCollection) *types.ParseDirective {
	chainMessageApiName := api.Name
	for _, parseDirective := range apiCollection.GetParseDirectives() {
		if parseDirective.ApiName == chainMessageApiName {
			return parseDirective
		}
	}
	return nil
}

func IsFunctionTagOfType(chainMessage ChainMessageForSend, functionTag types.FUNCTION_TAG) bool {
	parseDirective := chainMessage.GetParseDirective()
	if parseDirective != nil {
		return parseDirective.FunctionTag == functionTag
	}
	return false
}

// IsArchiveRequest returns true if the chain message carries the archive extension.
func IsArchiveRequest(chainMessage ChainMessage) bool {
	for _, ext := range chainMessage.GetExtensions() {
		if ext.Name == extensionslib.ArchiveExtension {
			return true
		}
	}
	return false
}

// IsDebugOrTraceRequest returns true if the request belongs to a debug or trace addon,
// as classified by the chain spec (AddOn = "debug" or "trace").
func IsDebugOrTraceRequest(chainMessage ChainMessageForSend) bool {
	addon := GetAddon(chainMessage)
	return addon == "debug" || addon == "trace"
}

// IsBatchRequest returns true if the chain message is a batch request (e.g., JSON-RPC batch).
func IsBatchRequest(chainMessage ChainMessage) bool {
	return chainMessage.IsBatch()
}
