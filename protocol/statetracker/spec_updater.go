package statetracker

import (
	"context"
	"sync"

	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	CallbackKeyForSpecUpdate = "spec-update"
)

type SpecGetter interface {
	GetSpec(ctx context.Context, chainID string) (*spectypes.Spec, error)
}

type SpecUpdatable interface {
	SetSpec(spectypes.Spec)
}

type SpecUpdater struct {
	lock             sync.RWMutex
	eventTracker     *EventTracker
	chainId          string
	specGetter       SpecGetter
	blockLastUpdated uint64
	specUpdatables   map[string]*SpecUpdatable
}

func NewSpecUpdater(chainId string, specGetter SpecGetter, eventTracker *EventTracker) *SpecUpdater {
	return &SpecUpdater{chainId: chainId, specGetter: specGetter, eventTracker: eventTracker, specUpdatables: map[string]*SpecUpdatable{}}
}

func (su *SpecUpdater) UpdaterKey() string {
	return CallbackKeyForSpecUpdate + su.chainId
}

func (su *SpecUpdater) RegisterSpecUpdatable(ctx context.Context, specUpdatable *SpecUpdatable, endpoint lavasession.RPCEndpoint) error {
	su.lock.Lock()
	defer su.lock.Unlock()
	spec, err := su.specGetter.GetSpec(ctx, su.chainId)
	if err != nil {
		utils.LavaFormatFatal("could not get chain spec failed registering", err, utils.Attribute{Key: "chainID", Value: su.chainId})
	}
	(*specUpdatable).SetSpec(*spec)
	su.specUpdatables[endpoint.Key()] = specUpdatable
	return nil
}

func (su *SpecUpdater) Update(latestBlock int64) {
	su.lock.RLock()
	defer su.lock.RUnlock()
	specUpdated := su.eventTracker.getLatestSpecModifyEvents()
	if specUpdated {
		spec, err := su.specGetter.GetSpec(context.Background(), su.chainId)
		if err != nil {
			utils.LavaFormatError("could not get spec when updated, did not update specs and needed to", err)
			return
		}
		if spec.BlockLastUpdated > su.blockLastUpdated {
			su.blockLastUpdated = spec.BlockLastUpdated
		}
		for _, specUpdatable := range su.specUpdatables {
			(*specUpdatable).SetSpec(*spec)
		}
	}
}
