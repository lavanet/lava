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
}

type VersionUpdater struct {
	lock              sync.RWMutex
	eventTracker      *EventTracker
	versionStateQuery VersionStateQuery
}

func NewVersionUpdater(versionStateQuery VersionStateQuery, eventTracker *EventTracker) *VersionUpdater {
	return &VersionUpdater{versionStateQuery: versionStateQuery, eventTracker: eventTracker}
}

func (vu *VersionUpdater) UpdaterKey() string {
	return CallbackKeyForVersionUpdate
}

func (vu *VersionUpdater) RegisterVersionUpdatable() {
	// currently no behavior is needed, in case we need to add a behavior in the future
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

		// validate version and potentially throw panic due to version mismatch - this will trigger LavaVisor!
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
