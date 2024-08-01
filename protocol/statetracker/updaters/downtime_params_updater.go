package updaters

import (
	"context"
	"sync"
	"time"

	"github.com/lavanet/lava/v2/utils"
	downtimev1 "github.com/lavanet/lava/v2/x/downtime/v1"
)

const (
	CallbackKeyForDowntimeParamsUpdate = "downtime-params-update"
)

type DowntimeParamsStateQuery interface {
	GetDowntimeParams(ctx context.Context) (*downtimev1.Params, error)
}

type DowntimeParamsUpdatable interface {
	SetDowntimeParams(downtimev1.Params)
}

// DowntimeParamsUpdater update downtime params for registered updatables after params change proposals
type DowntimeParamsUpdater struct {
	lock                     sync.RWMutex
	eventTracker             *EventTracker
	downtimeParamsStateQuery DowntimeParamsStateQuery
	downtimeParams           *downtimev1.Params
	downtimeParamsUpdatables []*DowntimeParamsUpdatable
	shouldUpdate             bool
}

func NewDowntimeParamsUpdater(downtimeParamsStateQuery DowntimeParamsStateQuery, eventTracker *EventTracker) *DowntimeParamsUpdater {
	return &DowntimeParamsUpdater{downtimeParamsStateQuery: downtimeParamsStateQuery, eventTracker: eventTracker, downtimeParamsUpdatables: []*DowntimeParamsUpdatable{}}
}

func (dpu *DowntimeParamsUpdater) UpdaterKey() string {
	return CallbackKeyForDowntimeParamsUpdate
}

func (dpu *DowntimeParamsUpdater) RegisterDowntimeParamsUpdatable(ctx context.Context, downtimeParamsUpdatable *DowntimeParamsUpdatable) error {
	dpu.lock.Lock()
	defer dpu.lock.Unlock()

	var params *downtimev1.Params
	if dpu.downtimeParams != nil {
		params = dpu.downtimeParams
	} else {
		var err error
		params, err = dpu.downtimeParamsStateQuery.GetDowntimeParams(ctx)
		if err != nil {
			return utils.LavaFormatError("panic level error could not get downtime params, failed registering", err)
		}

		dpu.downtimeParams = params
	}

	(*downtimeParamsUpdatable).SetDowntimeParams(*params)
	dpu.downtimeParamsUpdatables = append(dpu.downtimeParamsUpdatables, downtimeParamsUpdatable)
	return nil
}

func (dpu *DowntimeParamsUpdater) fetchResourcesAndUpdateHandlers() error {
	// fetch updated downtime params from consensus
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	params, err := dpu.downtimeParamsStateQuery.GetDowntimeParams(timeoutCtx)
	if err != nil {
		return utils.LavaFormatError("Failed fetching latest Downtime params from chain", err)
	}

	for _, downtimeParamsUpdatable := range dpu.downtimeParamsUpdatables {
		// iterate over all updaters and execute their updatable
		(*downtimeParamsUpdatable).SetDowntimeParams(*params)
	}
	return nil
}

// call only when locked.
func (dpu *DowntimeParamsUpdater) updateInner(latestBlock int64) {
	err := dpu.fetchResourcesAndUpdateHandlers()
	if err == nil {
		utils.LavaFormatDebug("Updated Downtime params successfully")
		dpu.shouldUpdate = false
	} else {
		utils.LavaFormatError("Failed updating downtime parameters", err, utils.LogAttr("block", latestBlock))
	}
}

func (dpu *DowntimeParamsUpdater) Reset(latestBlock int64) {
	dpu.lock.Lock()
	defer dpu.lock.Unlock()
	utils.LavaFormatDebug("Reset Triggered for Downtime Updater", utils.LogAttr("block", latestBlock))
	dpu.shouldUpdate = true
	dpu.updateInner(latestBlock)
}

func (dpu *DowntimeParamsUpdater) Update(latestBlock int64) {
	dpu.lock.Lock()
	defer dpu.lock.Unlock()
	if dpu.shouldUpdate {
		dpu.updateInner(latestBlock)
	} else {
		paramsUpdated, err := dpu.eventTracker.getLatestDowntimeParamsUpdateEvents(latestBlock)
		if paramsUpdated || err != nil {
			dpu.shouldUpdate = true // in case we fail to update now. remember to update next block update
			dpu.updateInner(latestBlock)
		}
	}
}
