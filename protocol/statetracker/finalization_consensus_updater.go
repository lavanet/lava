package statetracker

import (
	"context"
	"sync"

	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/utils"
)

const (
	CallbackKeyForFinalizationConsensusUpdate = "finalization-consensus-update"
)

type FinalizationConsensusUpdater struct {
	lock                              sync.RWMutex
	registeredFinalizationConsensuses []*lavaprotocol.FinalizationConsensus
	nextBlockForUpdate                uint64
	stateQuery                        *ConsumerStateQuery
}

func NewFinalizationConsensusUpdater(stateQuery *ConsumerStateQuery) *FinalizationConsensusUpdater {
	return &FinalizationConsensusUpdater{registeredFinalizationConsensuses: []*lavaprotocol.FinalizationConsensus{}, stateQuery: stateQuery}
}

func (fcu *FinalizationConsensusUpdater) RegisterFinalizationConsensus(finalizationConsensus *lavaprotocol.FinalizationConsensus) {
	// TODO: also update here for the first time
	fcu.lock.Lock()
	defer fcu.lock.Unlock()
	fcu.registeredFinalizationConsensuses = append(fcu.registeredFinalizationConsensuses, finalizationConsensus)
}

func (fcu *FinalizationConsensusUpdater) UpdaterKey() string {
	return CallbackKeyForFinalizationConsensusUpdate
}

func (fcu *FinalizationConsensusUpdater) Update(latestBlock int64) {
	fcu.lock.RLock()
	defer fcu.lock.RUnlock()
	ctx := context.Background()
	if int64(fcu.nextBlockForUpdate) > latestBlock {
		return
	}
	_, epoch, nextBlockForUpdate, err := fcu.stateQuery.GetPairing(ctx, "", latestBlock)
	if err != nil {
		utils.LavaFormatError("could not get block stats for finzalizationConsensus, trying again later", err, utils.Attribute{Key: "latestBlock", Value: latestBlock})
		fcu.nextBlockForUpdate += 1
		return
	}
	fcu.nextBlockForUpdate = nextBlockForUpdate
	for _, finalizationConsensus := range fcu.registeredFinalizationConsensuses {
		if finalizationConsensus == nil {
			continue
		}
		finalizationConsensus.NewEpoch(epoch)
	}
}
