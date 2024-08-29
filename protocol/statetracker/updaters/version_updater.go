package updaters

import (
	"context"
	"sync"
	"time"

	"github.com/lavanet/lava/v2/utils"
	protocoltypes "github.com/lavanet/lava/v2/x/protocol/types"
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
	shouldUpdate         bool
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

// call when locked only
func (vu *VersionUpdater) updateInner(latestBlock int64) {
	// fetch updated version from consensus
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	version, err := vu.VersionStateQuery.GetProtocolVersion(timeoutCtx)
	if err != nil {
		utils.LavaFormatError("could not get version when updated, did not update protocol version and needed to", err)
		return
	}
	utils.LavaFormatDebug("Protocol version has been fetched successfully!",
		utils.Attribute{Key: "old_version", Value: vu.LastKnownVersion},
		utils.Attribute{Key: "new_version", Value: version})
	// if no error, set the last known version.
	vu.LastKnownVersion = version
	vu.shouldUpdate = false // updated successfully
}

func (vu *VersionUpdater) Reset(latestBlock int64) {
	utils.LavaFormatDebug("Reset Triggered for Version Updater", utils.LogAttr("block", latestBlock))
	vu.Lock.Lock()
	defer vu.Lock.Unlock()
	vu.shouldUpdate = true
	vu.updateInner(latestBlock)
}

func (vu *VersionUpdater) Update(latestBlock int64) {
	vu.Lock.Lock()
	defer vu.Lock.Unlock()
	if vu.shouldUpdate {
		vu.updateInner(latestBlock)
	} else {
		versionUpdated, err := vu.eventTracker.getLatestVersionEvents(latestBlock)
		if versionUpdated || err != nil {
			vu.shouldUpdate = true
			vu.updateInner(latestBlock)
		}
	}
	// monitor protocol version on each new block even if it was not updated (used for logging purposes)
	err := vu.ValidateProtocolVersion(vu.LastKnownVersion)
	if err != nil {
		utils.LavaFormatError("Validate Protocol Version Error", err)
	}
}
