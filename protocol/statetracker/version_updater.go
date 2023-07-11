package statetracker

import (
	"context"
	"fmt"
	"sync"

	"github.com/lavanet/lava/protocol/upgrade"
	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

const (
	CallbackKeyForVersionUpdate = "version-update"
)

type VersionGetter interface {
	GetVersion(ctx context.Context) (*protocoltypes.Version, error)
}

type VersionUpdatable interface {
	SetProtocolVersion(*upgrade.ProtocolVersion)
}

type VersionUpdater struct {
	lock              sync.RWMutex
	eventTracker      *EventTracker
	versionGetter     VersionGetter
	versionUpdatables []*VersionUpdatable
}

func NewVersionUpdater(versionGetter VersionGetter, eventTracker *EventTracker) *VersionUpdater {
	return &VersionUpdater{versionGetter: versionGetter, eventTracker: eventTracker, versionUpdatables: []*VersionUpdatable{}}
}

func (vu *VersionUpdater) UpdaterKey() string {
	return CallbackKeyForVersionUpdate
}

// deprecate this:
// Note: purpose of updatables is send callback
// call it InitializeVersionUpdatable : use it to fetch version when rpcprovider / rpcconsumer starts
// @audit validate version taken from protocol version and consensus
func (vu *VersionUpdater) RegisterVersionUpdatable(ctx context.Context, versionUpdatable VersionUpdatable) {
	vu.lock.Lock()
	defer vu.lock.Unlock()
	version, err := vu.versionGetter.GetVersion(ctx)
	if err != nil {
		utils.LavaFormatFatal("could not get version for updater", err)
	}
	fmt.Println("GetVersion - version: ", version)
	vu.versionUpdatables = append(vu.versionUpdatables, &versionUpdatable)
}

func (vu *VersionUpdater) Update(latestBlock int64) {
	vu.lock.RLock()
	defer vu.lock.RUnlock()
	fmt.Println("VERSION UPDATE ENTERED!")
	fmt.Println("initVer: ", upgrade.LavaProtocolVersion)

	version, err := vu.eventTracker.getLatestVersionEvents()
	fmt.Println("Update funct- version: ", version)
	if err != nil {
		return
	}
	if version != nil {
		// we already get the version with param parsing, but let's fetch it again from blockchain
		// version, err := vu.versionGetter.GetVersion(context.Background())
		// if err != nil {
		// 	utils.LavaFormatError("could not get version when updated, did not update protocol version and needed to", err)
		// 	return
		// }

		//@audit check version here again!

		// deprecate this:
		for _, versionUpdatable := range vu.versionUpdatables {
			if versionUpdatable == nil {
				continue
			}
			(*versionUpdatable).SetProtocolVersion(version)
		}

		// check if we are correlation with minimum version: fatal panic (throw enough info for lavaVisor / user to catch the cause)

	}
	// add print if target version doesn't matched, create a warning (every block): target version is not met, cur. version, target version.
}
