package updaters

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
	GetProtocolVersion(ctx context.Context) (*ProtocolVersionResponse, error)
}

type VersionValidationInf interface {
	ValidateProtocolVersion(lastKnownVersion *ProtocolVersionResponse) error
}

type VersionUpdater struct {
	Lock                 sync.RWMutex
	eventTracker         *EventTracker
	VersionStateQuery    VersionStateQuery
	LastKnownVersion     *ProtocolVersionResponse
	VersionValidationInf // embedding the interface, this tells: VersionUpdater has ValidateProtocolVersion method
}

func NewVersionUpdater(versionStateQuery VersionStateQuery, eventTracker *EventTracker, version *protocoltypes.Version, versionValidator VersionValidationInf) *VersionUpdater {
	return &VersionUpdater{VersionStateQuery: versionStateQuery, eventTracker: eventTracker, LastKnownVersion: &ProtocolVersionResponse{Version: version, BlockNumber: "uninitialized"}, VersionValidationInf: versionValidator}
}

func (vu *VersionUpdater) UpdaterKey() string {
	return CallbackKeyForVersionUpdate
}

func (vu *VersionUpdater) RegisterVersionUpdatable() {
	vu.Lock.RLock()
	defer vu.Lock.RUnlock()
	err := vu.ValidateProtocolVersion(vu.LastKnownVersion)
	if err != nil {
		utils.LavaFormatError("Protocol Version Error", err)
	}
}

func (vu *VersionUpdater) Update(latestBlock int64) {
	vu.Lock.Lock()
	defer vu.Lock.Unlock()
	versionUpdated, err := vu.eventTracker.getLatestVersionEvents(latestBlock)
	if versionUpdated || err != nil {
		// fetch updated version from consensus
		version, err := vu.VersionStateQuery.GetProtocolVersion(context.Background())
		if err != nil {
			utils.LavaFormatError("could not get version when updated, did not update protocol version and needed to", err)
			return
		}
		utils.LavaFormatInfo("Protocol version has been fetched successfully!",
			utils.Attribute{Key: "old_version", Value: vu.LastKnownVersion},
			utils.Attribute{Key: "new_version", Value: version})
		// if no error, set the last known version.
		vu.LastKnownVersion = version
	}
	// monitor protocol version on each new block
	err = vu.ValidateProtocolVersion(vu.LastKnownVersion)
	if err != nil {
		utils.LavaFormatError("Validate Protocol Version Error", err)
	}
}
