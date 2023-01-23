package statetracker

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/lavaprotocol"
)

const (
	CallbackKeyForFinalizationConsensusUpdate = "finalization-consensus-update"
)

type FinalizationConsensusUpdater struct {
	registeredFinalizationConsensuses []*lavaprotocol.FinalizationConsensus
	nextBlockForUpdate                uint64
	stateQuery                        StateQuery
}

func NewFinalizationConsensusUpdater(consumerAddress sdk.AccAddress, stateQuery StateQuery) *FinalizationConsensusUpdater {
	return &FinalizationConsensusUpdater{registeredFinalizationConsensuses: []*lavaprotocol.FinalizationConsensus{}, stateQuery: stateQuery}
}

func (fcu *FinalizationConsensusUpdater) RegisterFinalizationConsensus(finalizationConsensus *lavaprotocol.FinalizationConsensus) {
	// TODO: also update here for the first time
	fcu.registeredFinalizationConsensuses = append(fcu.registeredFinalizationConsensuses, finalizationConsensus)
}

func (fcu *FinalizationConsensusUpdater) UpdaterKey() string {
	return CallbackKeyForFinalizationConsensusUpdate
}

func (fcu *FinalizationConsensusUpdater) Update(latestBlock int64) error {
	if int64(fcu.nextBlockForUpdate) > latestBlock {
		return fmt.Errorf("%d is not latest block", latestBlock)
	}
	_, epoch, nextBlockForUpdate, err := fcu.stateQuery.GetPairing(latestBlock)
	if err != nil {
		return err
	}
	fcu.nextBlockForUpdate = nextBlockForUpdate
	for _, finalizationConsensus := range fcu.registeredFinalizationConsensuses {
		finalizationConsensus.NewEpoch(epoch)
	}

	return nil
}
