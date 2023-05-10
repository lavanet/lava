package statetracker

import (
	"context"

	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	CallbackKeyForSpecUpdate = "spec-update"
)

type SpecGetter interface {
	GetSpec(ctx context.Context, chainID string) (*spectypes.Spec, error)
}

type SpecUpdater struct {
	eventTracker     *EventTracker
	chainParser      chainlib.ChainParser
	chainId          string
	apiInterface     string
	specGetter       SpecGetter
	blockLastUpdated uint64
}

func NewSpecUpdater(chainId string, chainParser chainlib.ChainParser, specGetter SpecGetter, eventTracker *EventTracker, apiInterface string) *SpecUpdater {
	return &SpecUpdater{chainId: chainId, chainParser: chainParser, specGetter: specGetter, eventTracker: eventTracker, apiInterface: apiInterface}
}

func (su *SpecUpdater) UpdaterKey() string {
	utils.LavaFormatDebug("UpdaterKey: " + CallbackKeyForSpecUpdate + su.chainId + su.apiInterface)
	return CallbackKeyForSpecUpdate + su.chainId + su.apiInterface
}

func (su *SpecUpdater) Update(latestBlock int64) {
	specUpdated := su.eventTracker.getLatestSpecModifyEvents()
	if specUpdated {
		spec, err := su.specGetter.GetSpec(context.Background(), su.chainId)
		if err != nil {
			utils.LavaFormatError("Failed getting spec", err)
			return
		}
		if su.blockLastUpdated < spec.BlockLastUpdated {
			utils.LavaFormatDebug("Spec has been modified, updating spec", utils.Attribute{Key: "Chain", Value: su.chainId}, utils.Attribute{Key: "ApiInterface", Value: su.apiInterface})
			su.setSpec(spec)
		}
	}
}

func (su *SpecUpdater) setSpec(spec *spectypes.Spec) {
	// set blockLastUpdated
	su.blockLastUpdated = spec.BlockLastUpdated
	su.chainParser.SetSpec(*spec)
}

func (su *SpecUpdater) InitSpec(ctx context.Context) error {
	spec, err := su.specGetter.GetSpec(ctx, su.chainId)
	if err != nil {
		return err
	}
	su.setSpec(spec)
	return nil
}
