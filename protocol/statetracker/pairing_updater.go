package statetracker

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/lavasession"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

const (
	CallbackKeyForPairingUpdate = "pairing-update"
)

type PairingUpdater struct {
	consumerSessionManagers []*lavasession.ConsumerSessionManager
	nextBlockForUpdate      uint64
	stateQuery              *StateQuery
}

func NewPairingUpdater(consumerAddress sdk.AccAddress, stateQuery *StateQuery) *PairingUpdater {
	return &PairingUpdater{consumerSessionManagers: []*lavasession.ConsumerSessionManager{}, stateQuery: stateQuery}
}

func (pu *PairingUpdater) RegisterPairing(consumerSessionManager *lavasession.ConsumerSessionManager) {
	// TODO: also update here for the first time
	pu.consumerSessionManagers = append(pu.consumerSessionManagers, consumerSessionManager)
}

func (pu *PairingUpdater) UpdaterKey() string {
	return CallbackKeyForPairingUpdate
}

func (pu *PairingUpdater) Update(latestBlock int64) {
	if int64(pu.nextBlockForUpdate) > latestBlock {
		return
	}
	pairingList, epoch, nextBlockForUpdate := pu.stateQuery.GetPairing(latestBlock)

	for _, consumerSessionManager := range pu.consumerSessionManagers {
		pairingListForThisCSM := filterPairingListByEndpoint(pairingList, consumerSessionManager.RPCEndpoint())
		consumerSessionManager.UpdateAllProviders(epoch, pairingListForThisCSM)
	}
	pu.nextBlockForUpdate = nextBlockForUpdate
}

func filterPairingListByEndpoint(pairingList []epochstoragetypes.StakeEntry, rpcEndpoint lavasession.RPCEndpoint) (filteredList []*lavasession.ConsumerSessionsWithProvider) {
	// go over stake entries, and filter endpoints that match geolocation and api interface
	return
}
