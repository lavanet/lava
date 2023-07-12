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
	SetProtocolVersion(*protocoltypes.Version)
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

func (vu *VersionUpdater) RegisterVersionUpdatable(ctx context.Context, versionUpdatable VersionUpdatable) {
	vu.lock.Lock()
	defer vu.lock.Unlock()
	vu.versionUpdatables = append(vu.versionUpdatables, &versionUpdatable)
}

func (vu *VersionUpdater) Update(latestBlock int64) {
	vu.lock.RLock()
	defer vu.lock.RUnlock()

	fmt.Println("initVer: ", upgrade.LavaProtocolVersion)

	versionUpdated := vu.eventTracker.getLatestVersionEvents()
	if versionUpdated {
		// fetch updated version from consensus
		version, err := vu.versionGetter.GetVersion(context.Background())
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
		err = vu.CheckProtocolVersion()
		if err != nil {
			utils.LavaFormatError("checkVersion failed", err)
			return
		}
		utils.LavaFormatInfo("Protocol version updated", utils.Attribute{Key: "version", Value: version})
	}
}

func (vu *VersionUpdater) CheckProtocolVersion() error {
	networkVersion, err := vu.versionGetter.GetVersion(context.Background())
	if err != nil {
		return utils.LavaFormatError("could not get protocol version from network", err)

	}
	currentProtocolVersion := upgrade.LavaProtocolVersion
	// check min version
	if networkVersion.ConsumerMin != currentProtocolVersion.ConsumerMin || networkVersion.ProviderMin != currentProtocolVersion.ProviderMin {
		return utils.LavaFormatError("minimum protocol version mismatch!", nil)
	}
	if networkVersion.ConsumerTarget != currentProtocolVersion.ConsumerTarget || networkVersion.ProviderTarget != currentProtocolVersion.ProviderTarget {
		utils.LavaFormatWarning("target protocol version mismatch", nil)
	}
	return err
}
