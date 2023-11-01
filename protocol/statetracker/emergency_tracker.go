package statetracker

import (
	"sync"
	"time"

	"github.com/lavanet/lava/utils"
	downtimev1 "github.com/lavanet/lava/x/downtime/v1"
)

type EmergencyTracker struct {
	lock             sync.RWMutex
	downtimeParams   downtimev1.Params
	virtualEpochsMap map[uint64]uint64
	latestEpoch      uint64
	latestEpochTime  time.Time
}

func (cs *EmergencyTracker) SetDowntimeParams(params downtimev1.Params) {
	cs.lock.Lock()
	defer cs.lock.Unlock()
	cs.downtimeParams = params
}

func (cs *EmergencyTracker) GetDowntimeParams() downtimev1.Params {
	cs.lock.RLock()
	defer cs.lock.RUnlock()
	return cs.downtimeParams
}

func (cs *EmergencyTracker) UpdateEpoch(epoch uint64) {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.latestEpoch = epoch
	cs.latestEpochTime = time.Now().UTC()
	cs.virtualEpochsMap[epoch] = 0
}

func (cs *EmergencyTracker) blockNotFound(latestBlockTime time.Time) {
	cs.lock.Lock()
	defer cs.lock.Unlock()
	if time.Since(latestBlockTime) > cs.downtimeParams.DowntimeDuration {
		epochDuration := cs.downtimeParams.EpochDuration.Milliseconds()
		if epochDuration == 0 {
			utils.LavaFormatWarning("Emergency Tracker: downtime params are not initialized", nil)
			return
		}

		virtualEpoch := uint64(time.Since(cs.latestEpochTime).Milliseconds() + epochDuration - 1/epochDuration)
		if virtualEpoch > 0 && cs.virtualEpochsMap[cs.latestEpoch] != virtualEpoch {
			utils.LavaFormatDebug("Emergency Tracker: emergency mode enabled", utils.Attribute{
				Key:   "virtual_epoch",
				Value: virtualEpoch,
			})
		}
		if virtualEpoch < cs.virtualEpochsMap[cs.latestEpoch] {
			utils.LavaFormatError("Current virtual epoch is lower than stored", nil)
			return
		}

		cs.virtualEpochsMap[cs.latestEpoch] = virtualEpoch
	}
}

func (cs *EmergencyTracker) GetVirtualEpoch(epoch uint64) uint64 {
	cs.lock.RLock()
	defer cs.lock.RUnlock()
	return cs.virtualEpochsMap[epoch]
}

func NewEmergencyTracker() (emergencyTracker *EmergencyTracker, emergencyCallback func(latestBlockTime time.Time)) {
	emergencyTracker = &EmergencyTracker{
		virtualEpochsMap: map[uint64]uint64{},
		latestEpoch:      0,
	}

	return emergencyTracker, emergencyTracker.blockNotFound
}
