package chainlib

import "github.com/lavanet/lava/protocol/common"

func ShouldSendToAllProviders(chainMessage ChainMessage) bool {
	return chainMessage.GetApi().Category.Stateful == common.CONSISTENCY_SELECT_ALLPROVIDERS
}

func GetAddon(chainMessage ChainMessage) string {
	return chainMessage.GetApiCollection().CollectionData.AddOn
}

func IsSubscription(chainMessage ChainMessage) bool {
	return chainMessage.GetApi().Category.Subscription
}

func IsHangingApi(chainMessage ChainMessage) bool {
	return chainMessage.GetApi().Category.HangingApi
}

func GetComputeUnits(chainMessage ChainMessage) uint64 {
	return chainMessage.GetApi().ComputeUnits
}

func GetStateful(chainMessage ChainMessage) uint32 {
	return chainMessage.GetApi().Category.Stateful
}
