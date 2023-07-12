package statetracker

import (
	"context"
	"sync"

	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

const (
	CallbackKeyForVersionUpdate = "version-update"
)

type VersionStateQuery interface {
	GetProtocolVersion(ctx context.Context) (*protocoltypes.Version, error)
	CheckProtocolVersion(cx context.Context) error
}

type VersionUpdatable interface {
	SetProtocolVersion(*protocoltypes.Version)
}

type VersionUpdater struct {
	lock              sync.RWMutex
	eventTracker      *EventTracker
	versionStateQuery VersionStateQuery
	versionUpdatables []*VersionUpdatable
}

func NewVersionUpdater(versionStateQuery VersionStateQuery, eventTracker *EventTracker) *VersionUpdater {
	return &VersionUpdater{versionStateQuery: versionStateQuery, eventTracker: eventTracker, versionUpdatables: []*VersionUpdatable{}}
}

func (vu *VersionUpdater) UpdaterKey() string {
	return CallbackKeyForVersionUpdate
}

func (vu *VersionUpdater) RegisterVersionUpdatable(ctx context.Context, versionUpdatable VersionUpdatable) {
	vu.lock.Lock()
	defer vu.lock.Unlock()
	vu.versionUpdatables = append(vu.versionUpdatables, &versionUpdatable)
}

func (vu *VersionUpdater) Update(latestBlock int64) {
	vu.lock.RLock()
	defer vu.lock.RUnlock()
	versionUpdated := vu.eventTracker.getLatestVersionEvents()
	if versionUpdated {
		// fetch updated version from consensus
		version, err := vu.versionStateQuery.GetProtocolVersion(context.Background())
		if err != nil {
			utils.LavaFormatError("could not get version when updated, did not update protocol version and needed to", err)
			return
		}
		// set version
		for _, versionUpdatable := range vu.versionUpdatables {
			if versionUpdatable == nil {
				continue
			}
			(*versionUpdatable).SetProtocolVersion(version)
		}
		// validate version
		err = vu.versionStateQuery.CheckProtocolVersion(context.Background())
		if err != nil {
			utils.LavaFormatError("checkVersion failed", err)
			return
		}
		utils.LavaFormatInfo("Protocol version has been successfully updated!", utils.Attribute{Key: "version", Value: version})
	}
	// monitor protocol version on each new block
	err := vu.versionStateQuery.CheckProtocolVersion(context.Background())
	if err != nil {
		utils.LavaFormatError("checkVersion failed", err)
		return
	}
}
