package updaters

import (
	"context"
	"sync"
	"time"

	"github.com/lavanet/lava/utils"
	downtimev1 "github.com/lavanet/lava/x/downtime/v1"
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

func (dpu *DowntimeParamsUpdater) Update(latestBlock int64) {
	dpu.lock.Lock()
	defer dpu.lock.Unlock()
	paramsUpdated, err := dpu.eventTracker.getLatestDowntimeParamsUpdateEvents(latestBlock)
	if paramsUpdated || err != nil {
		var params *downtimev1.Params
		// fetch updated downtime params from consensus
		for i := 0; i < BlockResultRetry; i++ {
			params, err = dpu.downtimeParamsStateQuery.GetDowntimeParams(context.Background())
			if err == nil {
				break
			}
			time.Sleep(50 * time.Millisecond * time.Duration(i+1))
		}

		if params == nil {
			return
		}

		for _, downtimeParamsUpdatable := range dpu.downtimeParamsUpdatables {
			// iterate over all updaters and execute their updatable
			(*downtimeParamsUpdatable).SetDowntimeParams(*params)
		}
	}
}
