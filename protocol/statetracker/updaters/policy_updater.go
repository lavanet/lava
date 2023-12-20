package updaters

import (
	"context"
	"sync"
	"time"

	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	plantypes "github.com/lavanet/lava/x/plans/types"
)

const (
	CallbackKeyForPolicyUpdate = "policy-update"
)

type PolicySetter interface {
	SetPolicy(policyInformation map[string]struct{}) error
	BuildMapFromPolicyQuery(policy *plantypes.Policy, chainId string, apiInterface string) (map[string]struct{}, error)
}

type PolicyFetcher interface {
	GetEffectivePolicy(ctx context.Context, consumerAddress, chainID string) (*plantypes.Policy, error)
}

type PolicyUpdater struct {
	lock                  sync.RWMutex
	eventTracker          *EventTracker
	chainId               string
	consumerAddress       string
	lastTimeUpdatedPolicy uint64
	policyFetcher         PolicyFetcher
	policyUpdatables      map[string]PolicySetter // key is apiInterface.
}

func NewPolicyUpdater(chainId string, policyFetcher PolicyFetcher, eventTracker *EventTracker, consumerAddress string) *PolicyUpdater {
	return &PolicyUpdater{
		chainId:               chainId,
		policyFetcher:         policyFetcher,
		eventTracker:          eventTracker,
		policyUpdatables:      make(map[string]PolicySetter),
		consumerAddress:       consumerAddress,
		lastTimeUpdatedPolicy: 0,
	}
}

func (pu *PolicyUpdater) AddPolicySetter(policyUpdatable PolicySetter, endpoint lavasession.RPCEndpoint) error {
	pu.lock.Lock()
	defer pu.lock.Unlock()
	existingPolicySetter, found := pu.policyUpdatables[endpoint.ApiInterface]
	if found {
		return utils.LavaFormatError("panic level error Trying to register to policy updates on already registered", nil,
			utils.Attribute{Key: "endpoint", Value: endpoint},
			utils.Attribute{Key: "policyUpdatable", Value: existingPolicySetter})
	}
	pu.policyUpdatables[endpoint.ApiInterface] = policyUpdatable
	return nil
}

func (pu *PolicyUpdater) UpdaterKey() string {
	return CallbackKeyForPolicyUpdate + pu.chainId
}

func (pu *PolicyUpdater) BuildPolicyMapAndSetPolicy(policyUpdatable PolicySetter, policy *plantypes.Policy, apiInterface string) error {
	policyMap, err := policyUpdatable.BuildMapFromPolicyQuery(policy, pu.chainId, apiInterface)
	if err != nil {
		return utils.LavaFormatError("panic level error, failed building policy map from query result", err, utils.LogAttr("policy_result", policy))
	}
	err = policyUpdatable.SetPolicy(policyMap)
	if err != nil {
		return utils.LavaFormatError("panic level error, failed setting policy on policy updatable", err, utils.LogAttr("chainId", pu.chainId), utils.LogAttr("api_interface", apiInterface))
	}
	return nil
}

func (pu *PolicyUpdater) UpdateEpoch(epoch uint64) {
	pu.lock.Lock()
	defer pu.lock.Unlock()
	if pu.lastTimeUpdatedPolicy < epoch {
		// update policy now
		utils.LavaFormatInfo("Epoch Changed, fetching current policy and updating the effective policy")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		policy, err := pu.policyFetcher.GetEffectivePolicy(ctx, pu.consumerAddress, pu.chainId)
		if err != nil {
			utils.LavaFormatError("could not get spec when updated, did not update specs and needed to", err)
			return
		}
		for apiInterface, policyUpdatable := range pu.policyUpdatables {
			err = pu.BuildPolicyMapAndSetPolicy(policyUpdatable, policy, apiInterface)
			if err != nil {
				utils.LavaFormatError("Failed Updating policy", err, utils.LogAttr("apiInterface", apiInterface), utils.LogAttr("chainId", pu.chainId))
			}
		}
	}
}
