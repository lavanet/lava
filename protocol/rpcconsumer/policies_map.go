package rpcconsumer

import (
	"sync"

	"github.com/lavanet/lava/v3/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v3/utils"
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

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
// The function returns the value that was loaded or stored.
func (sm *syncMapPolicyUpdaters) LoadOrStore(key string, value *updaters.PolicyUpdater) (ret *updaters.PolicyUpdater, loaded bool) {
	actual, loaded := sm.localMap.LoadOrStore(key, value)
	if loaded {
		// loaded from map
		ret, loaded = actual.(*updaters.PolicyUpdater)
		if !loaded {
			utils.LavaFormatFatal("invalid usage of syncmap, could not cast result into a PolicyUpdater", nil)
		}
		return ret, loaded
	}

	// stored in map
	return value, false
}
