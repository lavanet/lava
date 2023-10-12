package statetracker

import (
	"context"
	"sync"

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
	paramsUpdated := dpu.eventTracker.getLatestDowntimeParamsUpdateEvents()
	if paramsUpdated {
		// fetch updated downtime params from consensus
		params, err := dpu.downtimeParamsStateQuery.GetDowntimeParams(context.Background())
		if err != nil {
			utils.LavaFormatError("could not get downtime params when updated, did not update downtime params and needed to", err)
			return
		}
		for _, downtimeParamsUpdatable := range dpu.downtimeParamsUpdatables {
			(*downtimeParamsUpdatable).SetDowntimeParams(*params)
		}
	}
}
