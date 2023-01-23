package statetracker

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/lavasession"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

const (
	CallbackKeyForPairingUpdate = "pairing-update"
)

type PairingUpdater interface {
	RegisterPairing(consumerSessionManager *lavasession.ConsumerSessionManager)
	UpdaterKey() string
	Update(int64) error
}

type pairingUpdater struct {
	consumerSessionManagers []*lavasession.ConsumerSessionManager
	nextBlockForUpdate      uint64
	stateQuery              StateQuery
}

func NewPairingUpdater(consumerAddress sdk.AccAddress, stateQuery StateQuery) PairingUpdater {
	return &pairingUpdater{consumerSessionManagers: []*lavasession.ConsumerSessionManager{}, stateQuery: stateQuery}
}

func (pu *pairingUpdater) RegisterPairing(consumerSessionManager *lavasession.ConsumerSessionManager) {
	// TODO: also update here for the first time
	pu.consumerSessionManagers = append(pu.consumerSessionManagers, consumerSessionManager)
}

func (pu *pairingUpdater) UpdaterKey() string {
	return CallbackKeyForPairingUpdate
}

func (pu *pairingUpdater) Update(latestBlock int64) error {
	if int64(pu.nextBlockForUpdate) > latestBlock {
		return fmt.Errorf("%d is not latest block", latestBlock)
	}

	pairingList, epoch, nextBlockForUpdate, err := pu.stateQuery.GetPairing(latestBlock)
	if err != nil {
		return err
	}

	for _, consumerSessionManager := range pu.consumerSessionManagers {
		pairingListForThisCSM := filterPairingListByEndpoint(pairingList, consumerSessionManager.RPCEndpoint())
		consumerSessionManager.UpdateAllProviders(epoch, pairingListForThisCSM)
	}
	pu.nextBlockForUpdate = nextBlockForUpdate

	return nil
}

func filterPairingListByEndpoint(pairingList []epochstoragetypes.StakeEntry, rpcEndpoint lavasession.RPCEndpoint) (filteredList []*lavasession.ConsumerSessionsWithProvider) {
	// go over stake entries, and filter endpoints that match geolocation and api interface
	return
}
