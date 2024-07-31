package updaters

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/utils"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

const (
	CallbackKeyForSpecUpdate = "spec-update"
)

type SpecGetter interface {
	GetSpec(ctx context.Context, chainID string) (*spectypes.Spec, error)
}

type SpecUpdatable interface {
	SetSpec(spectypes.Spec)
	Active() bool
	GetUniqueName() string
}

type SpecVerifier interface {
	VerifySpec(spectypes.Spec)
	GetUniqueName() string
}

type SpecUpdater struct {
	lock             sync.RWMutex
	eventTracker     *EventTracker
	chainId          string
	specGetter       SpecGetter
	blockLastUpdated uint64
	specUpdatables   map[string]*SpecUpdatable // key is api interface
	specVerifiers    map[string]*SpecVerifier
	spec             *spectypes.Spec
	shouldUpdate     bool
}

func NewSpecUpdater(chainId string, specGetter SpecGetter, eventTracker *EventTracker) *SpecUpdater {
	return &SpecUpdater{chainId: chainId, specGetter: specGetter, eventTracker: eventTracker, specUpdatables: map[string]*SpecUpdatable{}, specVerifiers: map[string]*SpecVerifier{}}
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

	updatableUniqueName := (*specUpdatable).GetUniqueName()
	key := strings.Join([]string{updatableUniqueName, endpoint.Key()}, "_")
	existingSpecUpdatable, found := su.specUpdatables[key]
	if found {
		if (*existingSpecUpdatable).Active() {
			return utils.LavaFormatError("panic level error Trying to register to spec updates on already registered updatable unique name + chain + API interface", nil,
				utils.Attribute{Key: "updatableUniqueName", Value: updatableUniqueName},
				utils.Attribute{Key: "endpoint", Value: endpoint},
				utils.Attribute{Key: "specUpdatable", Value: existingSpecUpdatable})
		}
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
		su.spec = spec
	}
	(*specUpdatable).SetSpec(*spec)
	su.specUpdatables[key] = specUpdatable
	return nil
}

func (su *SpecUpdater) RegisterSpecVerifier(ctx context.Context, specVerifier *SpecVerifier, chainId string) error {
	su.lock.Lock()
	defer su.lock.Unlock()

	// validating
	if su.chainId != chainId {
		return utils.LavaFormatError("panic level error Trying to register spec for wrong chain id stored in spec_updater", nil, utils.Attribute{Key: "chainId", Value: chainId}, utils.Attribute{Key: "stored_spec", Value: su.chainId})
	}

	verifierUniqueName := (*specVerifier).GetUniqueName()
	key := strings.Join([]string{verifierUniqueName, chainId}, "_")
	existingSpecVerifier, found := su.specVerifiers[key]
	if found {
		return utils.LavaFormatError("panic level error Trying to register to spec verifications on already registered verifier unique name + chain + API interface", nil,
			utils.Attribute{Key: "verifierUniqueName", Value: verifierUniqueName},
			utils.Attribute{Key: "chainId", Value: chainId},
			utils.Attribute{Key: "specVerifier", Value: existingSpecVerifier})
	}

	su.specVerifiers[key] = specVerifier
	return nil
}

func (su *SpecUpdater) setBlockLastUpdatedAtomically(block uint64) {
	atomic.StoreUint64(&su.blockLastUpdated, block)
}

// only call when locked
func (su *SpecUpdater) updateInner(latestBlock int64) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	spec, err := su.specGetter.GetSpec(timeoutCtx, su.chainId)
	if err != nil {
		utils.LavaFormatError("could not get spec when updated, did not update specs, will retry next block", err)
		return
	}
	if spec.BlockLastUpdated > su.blockLastUpdated {
		su.setBlockLastUpdatedAtomically(spec.BlockLastUpdated)
	}
	for _, specUpdatable := range su.specUpdatables {
		utils.LavaFormatDebug("SpecUpdater: updating spec for chainId",
			utils.LogAttr("chainId", su.chainId),
			utils.LogAttr("specUpdatable", (*specUpdatable).GetUniqueName()))
		(*specUpdatable).SetSpec(*spec)
	}
	for _, specVerifier := range su.specVerifiers {
		utils.LavaFormatDebug("SpecUpdater: updating spec for chainId",
			utils.LogAttr("chainId", su.chainId),
			utils.LogAttr("specVerifier", (*specVerifier).GetUniqueName()))
		go (*specVerifier).VerifySpec(*spec)
	}
	su.shouldUpdate = false // update was successful
}

func (su *SpecUpdater) Reset(latestBlock int64) {
	utils.LavaFormatDebug("Reset state called on Spec Updater", utils.LogAttr("block", latestBlock))
	su.lock.Lock()
	defer su.lock.Unlock()
	su.shouldUpdate = true
	su.updateInner(latestBlock)
}

func (su *SpecUpdater) Update(latestBlock int64) {
	su.lock.Lock()
	defer su.lock.Unlock()
	if su.shouldUpdate {
		su.updateInner(latestBlock)
	} else {
		specUpdated, err := su.eventTracker.getLatestSpecModifyEvents(latestBlock)
		if specUpdated || err != nil {
			su.shouldUpdate = true
			su.updateInner(latestBlock)
		}
	}
}
