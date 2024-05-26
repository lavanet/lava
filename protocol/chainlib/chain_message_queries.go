package chainlib

import (
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/x/spec/types"
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

func IsOfFunctionType(chainMessage ChainMessageForSend, functionTag types.FUNCTION_TAG) bool {
	chainMessageApiName := chainMessage.GetApi().Name
	for _, parseDirective := range chainMessage.GetApiCollection().GetParseDirectives() {
		if parseDirective.ApiName == chainMessageApiName && parseDirective.FunctionTag == functionTag {
			return true
		}
	}
	return false
}
