package updaters

import (
	"sync"
	"time"

	"github.com/lavanet/lava/v2/utils"
	"golang.org/x/net/context"
)

const (
	CallbackKeyForEpochUpdate = "epoch-update"
)

type EpochUpdatable interface {
	UpdateEpoch(epoch uint64)
}

type EpochUpdatableWithBlockDelay struct {
	EpochUpdatable
	delay                int64              // amount of blocks to delay
	triggerUpdateOnBlock map[int64]struct{} // when to launch the updates
}

// Add a method to EpochUpdatableWithBlockDelay to update based on block delay.
func (euwbd *EpochUpdatableWithBlockDelay) UpdateOnBlock(currentEpoch uint64, latestBlock int64) {
	// utils.LavaFormatDebug("UpdateOnBlock", utils.Attribute{Key: "epoch", Value: currentEpoch}, utils.Attribute{Key: "latestBlock", Value: latestBlock})
	keysToDelete := []int64{}
	for triggerBlock := range euwbd.triggerUpdateOnBlock {
		if triggerBlock <= latestBlock { // making sure we didn't miss any updates by comparing to <= instead of ==
			keysToDelete = append(keysToDelete, triggerBlock)
			euwbd.EpochUpdatable.UpdateEpoch(currentEpoch)
		}
	}
	// deleting all blocks that were updated.
	for _, key := range keysToDelete {
		delete(euwbd.triggerUpdateOnBlock, key)
	}
}

type EpochStateQueryInterface interface {
	CurrentEpochStart(ctx context.Context) (uint64, error)
}

type EpochUpdater struct {
	lock            sync.RWMutex
	epochUpdatables []*EpochUpdatableWithBlockDelay
	currentEpoch    uint64
	stateQuery      EpochStateQueryInterface
}

func NewEpochUpdater(stateQuery EpochStateQueryInterface) *EpochUpdater {
	return &EpochUpdater{epochUpdatables: []*EpochUpdatableWithBlockDelay{}, stateQuery: stateQuery}
}

func (eu *EpochUpdater) RegisterEpochUpdatable(ctx context.Context, epochUpdatable EpochUpdatable, blocksUpdateDelay int64) {
	eu.lock.Lock()
	defer eu.lock.Unlock()
	// initialize with the current epoch
	currentEpoch, err := eu.stateQuery.CurrentEpochStart(ctx)
	if err != nil {
		utils.LavaFormatFatal("epoch updatable failed registering for epoch updates", err)
	}
	eu.currentEpoch = currentEpoch
	epochUpdatable.UpdateEpoch(currentEpoch)
	updatableWithDelay := &EpochUpdatableWithBlockDelay{
		delay:                blocksUpdateDelay,
		triggerUpdateOnBlock: make(map[int64]struct{}),
		EpochUpdatable:       epochUpdatable,
	}
	eu.epochUpdatables = append(eu.epochUpdatables, updatableWithDelay)
}

func (eu *EpochUpdater) UpdaterKey() string {
	return CallbackKeyForEpochUpdate
}

// our epoch updater always fetches the latest epoch start params.
func (eu *EpochUpdater) updateInner(latestBlock int64) {
	eu.lock.Lock()
	defer eu.lock.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	currentEpoch, err := eu.stateQuery.CurrentEpochStart(ctx)
	if err != nil {
		return // failed to get the current epoch
	}
	addTrigger := false
	if currentEpoch > eu.currentEpoch {
		addTrigger = true
		eu.currentEpoch = currentEpoch // update the current epoch
	}

	for _, epochUpdatable := range eu.epochUpdatables {
		if epochUpdatable == nil {
			continue
		}
		if addTrigger {
			// add the delayed updates
			epochUpdatable.triggerUpdateOnBlock[int64(currentEpoch)+epochUpdatable.delay] = struct{}{}
		}
		// iterate over all the delayed updates and execute their updatable. if delay is 0 it will execute immediately
		epochUpdatable.UpdateOnBlock(currentEpoch, latestBlock)
	}
}

func (eu *EpochUpdater) Reset(latestBlock int64) {
	utils.LavaFormatDebug("Reset triggered for Epoch Updater", utils.LogAttr("block", latestBlock))
	eu.updateInner(latestBlock)
}

func (eu *EpochUpdater) Update(latestBlock int64) {
	eu.updateInner(latestBlock)
}
