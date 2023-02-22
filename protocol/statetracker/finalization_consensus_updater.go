package statetracker

import (
	"context"
	"strconv"

	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

const (
	CallbackKeyForFinalizationConsensusUpdate = "finalization-consensus-update"
)

type ConsumerStateQueryInterface interface {
	GetPairing(ctx context.Context, chainID string, latestBlock int64) (pairingList []epochstoragetypes.StakeEntry, epoch uint64, nextBlockForUpdate uint64, errRet error)
}

type FinalizationConsensusUpdater struct {
	registeredFinalizationConsensuses []*lavaprotocol.FinalizationConsensus
	nextBlockForUpdate                uint64
	stateQuery                        ConsumerStateQueryInterface
}

func NewFinalizationConsensusUpdater(stateQuery ConsumerStateQueryInterface) *FinalizationConsensusUpdater {
	return &FinalizationConsensusUpdater{registeredFinalizationConsensuses: []*lavaprotocol.FinalizationConsensus{}, stateQuery: stateQuery}
}

func (fcu *FinalizationConsensusUpdater) RegisterFinalizationConsensus(finalizationConsensus *lavaprotocol.FinalizationConsensus) {
	// TODO: also update here for the first time
	fcu.registeredFinalizationConsensuses = append(fcu.registeredFinalizationConsensuses, finalizationConsensus)
}

func (fcu *FinalizationConsensusUpdater) UpdaterKey() string {
	return CallbackKeyForFinalizationConsensusUpdate
}

func (fcu *FinalizationConsensusUpdater) Update(latestBlock int64) {
	ctx := context.Background()

	if int64(fcu.nextBlockForUpdate) > latestBlock {
		return
	}
	_, epoch, nextBlockForUpdate, err := fcu.stateQuery.GetPairing(ctx, "", latestBlock)
	if err != nil {
		utils.LavaFormatError("could not get block stats for finzalizationConsensus, trying again later", err, &map[string]string{"latestBlock": strconv.FormatInt(latestBlock, 10)})
		fcu.nextBlockForUpdate += 1
		return
	}
	fcu.nextBlockForUpdate = nextBlockForUpdate
	for _, finalizationConsensus := range fcu.registeredFinalizationConsensuses {
		finalizationConsensus.NewEpoch(epoch)
	}
}
