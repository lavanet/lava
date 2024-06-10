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
