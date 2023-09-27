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
	spec             *spectypes.Spec
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

	// validating
	if su.chainId != endpoint.ChainID {
		return utils.LavaFormatError("panic level error Trying to register spec for wrong chain id stored in spec_updater", nil, utils.Attribute{Key: "endpoint", Value: endpoint}, utils.Attribute{Key: "stored_spec", Value: su.chainId})
	}
	_, found := su.specUpdatables[endpoint.Key()]
	if found {
		return utils.LavaFormatError("panic level error Trying to register to spec updates on already registered chain + API interfcae", nil, utils.Attribute{Key: "endpoint", Value: endpoint})
	}

	var spec *spectypes.Spec
	if su.spec != nil {
		spec = su.spec
	} else { // we don't have spec stored so we need to fetch it
		var err error
		spec, err = su.specGetter.GetSpec(ctx, su.chainId)
		if err != nil {
			return utils.LavaFormatError("panic level error could not get chain spec failed registering", err, utils.Attribute{Key: "chainID", Value: su.chainId})
		}
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
