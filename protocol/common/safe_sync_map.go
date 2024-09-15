package common

import (
	"sync"

	"github.com/lavanet/lava/v3/utils"
)

type SafeSyncMap[K, V any] struct {
	localMap sync.Map
}

func (ssm *SafeSyncMap[K, V]) Store(key K, toSet V) {
	ssm.localMap.Store(key, toSet)
}

func (ssm *SafeSyncMap[K, V]) Load(key K) (ret V, ok bool, err error) {
	value, ok := ssm.localMap.Load(key)
	if !ok {
		return ret, ok, nil
	}
	ret, ok = value.(V)
	if !ok {
		return ret, false, utils.LavaFormatError("invalid usage of syncmap, could not cast result into a PolicyUpdater", nil)
	}
	return ret, true, nil
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
// The function returns the value that was loaded or stored.
func (ssm *SafeSyncMap[K, V]) LoadOrStore(key K, value V) (ret V, loaded bool, err error) {
	actual, loaded := ssm.localMap.LoadOrStore(key, value)
	if loaded {
		// loaded from map
		var ok bool
		ret, ok = actual.(V)
		if !ok {
			return ret, false, utils.LavaFormatError("invalid usage of sync map, could not cast result into a PolicyUpdater", nil)
		}
		return ret, true, nil
	}

	// stored in map
	return value, false, nil
}

func (ssm *SafeSyncMap[K, V]) Range(f func(key, value any) bool) {
	ssm.localMap.Range(f)
}
