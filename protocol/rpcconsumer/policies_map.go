package rpcconsumer

import (
	"sync"

	"github.com/lavanet/lava/v2/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v2/utils"
)

type syncMapPolicyUpdaters struct {
	localMap sync.Map
}

func (sm *syncMapPolicyUpdaters) Store(key string, toSet *updaters.PolicyUpdater) {
	sm.localMap.Store(key, toSet)
}

func (sm *syncMapPolicyUpdaters) Load(key string) (ret *updaters.PolicyUpdater, ok bool) {
	value, ok := sm.localMap.Load(key)
	if !ok {
		return nil, ok
	}
	ret, ok = value.(*updaters.PolicyUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid usage of syncmap, could not cast result into a PolicyUpdater", nil)
	}
	return ret, true
}
