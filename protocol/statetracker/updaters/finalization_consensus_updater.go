package updaters

import (
	"context"
	"sync"
	"time"

	"github.com/lavanet/lava/v2/protocol/lavaprotocol"
	"github.com/lavanet/lava/v2/utils"
)

const (
	CallbackKeyForFinalizationConsensusUpdate = "finalization-consensus-update"
)

type FinalizationConsensusUpdater struct {
	lock                              sync.RWMutex
	registeredFinalizationConsensuses []*lavaprotocol.FinalizationConsensus
	nextBlockForUpdate                uint64
	stateQuery                        *ConsumerStateQuery
	specId                            string
}

func NewFinalizationConsensusUpdater(stateQuery *ConsumerStateQuery, specId string) *FinalizationConsensusUpdater {
	return &FinalizationConsensusUpdater{registeredFinalizationConsensuses: []*lavaprotocol.FinalizationConsensus{}, stateQuery: stateQuery, specId: specId}
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

func (fcu *FinalizationConsensusUpdater) updateInner(latestBlock int64) {
	fcu.lock.RLock()
	defer fcu.lock.RUnlock()
	if int64(fcu.nextBlockForUpdate) > latestBlock {
		return
	}
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, epoch, nextBlockForUpdate, err := fcu.stateQuery.GetPairing(timeoutCtx, fcu.specId, latestBlock)
	if err != nil {
		utils.LavaFormatError("could not get block stats for finalization consensus updater, trying again next block", err, utils.Attribute{Key: "latestBlock", Value: latestBlock})
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

func (fcu *FinalizationConsensusUpdater) Reset(latestBlock int64) {
	utils.LavaFormatDebug("Reset Triggered for FinalizationConsensusUpdater", utils.LogAttr("block", latestBlock))
	fcu.updateInner(latestBlock)
}

func (fcu *FinalizationConsensusUpdater) Update(latestBlock int64) {
	fcu.updateInner(latestBlock)
}
