package statetracker

import (
	"sync"

	"golang.org/x/net/context"
)

const (
	CallbackKeyForEpochUpdate = "epoch-update"
)

type EpochUpdatable interface {
	UpdateEpoch(epoch uint64)
}

type EpochUpdater struct {
	lock            sync.RWMutex
	epochUpdatables []*EpochUpdatable
	currentEpoch    uint64
	stateQuery      *EpochStateQuery
}

func NewEpochUpdater(stateQuery *EpochStateQuery) *EpochUpdater {
	return &EpochUpdater{epochUpdatables: []*EpochUpdatable{}, stateQuery: stateQuery}
}

func (eu *EpochUpdater) RegisterEpochUpdatable(ctx context.Context, epochUpdatable EpochUpdatable) {
	eu.lock.Lock()
	defer eu.lock.Unlock()
	eu.epochUpdatables = append(eu.epochUpdatables, &epochUpdatable)
}

func (eu *EpochUpdater) UpdaterKey() string {
	return CallbackKeyForEpochUpdate
}

func (eu *EpochUpdater) Update(latestBlock int64) {
	eu.lock.RLock()
	defer eu.lock.RUnlock()
	ctx := context.Background()
	currentEpoch, err := eu.stateQuery.CurrentEpochStart(ctx)
	if err != nil {
		return // failed to get the current epoch
	}
	if currentEpoch <= eu.currentEpoch {
		return // still the same epoch
	}
	eu.currentEpoch = currentEpoch
	for _, epochUpdatable := range eu.epochUpdatables {
		if epochUpdatable == nil {
			continue
		}
		(*epochUpdatable).UpdateEpoch(currentEpoch)
	}
}
