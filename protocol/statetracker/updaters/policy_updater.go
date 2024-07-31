package updaters

import (
	"context"
	"sync"
	"time"

	chainlib "github.com/lavanet/lava/v2/protocol/chainlib"
	"github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/utils"
	plantypes "github.com/lavanet/lava/v2/x/plans/types"
)

const (
	CallbackKeyForPolicyUpdate = "policy-update"
)

type PolicySetter interface {
	SetPolicy(policy chainlib.PolicyInf, chainId string, apiInterface string) error
}

type PolicyFetcher interface {
	GetConsumerPolicy(ctx context.Context, consumerAddress, chainID string) (*plantypes.Policy, error)
}

type PolicyUpdater struct {
	lock                  sync.RWMutex
	chainId               string
	consumerAddress       string
	lastTimeUpdatedPolicy uint64
	policyFetcher         PolicyFetcher
	policyUpdatables      map[string]PolicySetter // key is apiInterface.
}

func NewPolicyUpdater(chainId string, policyFetcher PolicyFetcher, consumerAddress string, policyUpdatable PolicySetter, endpoint lavasession.RPCEndpoint) *PolicyUpdater {
	return &PolicyUpdater{
		chainId:               chainId,
		policyFetcher:         policyFetcher,
		policyUpdatables:      map[string]PolicySetter{endpoint.ApiInterface: policyUpdatable},
		consumerAddress:       consumerAddress,
		lastTimeUpdatedPolicy: 0,
	}
}

func (pu *PolicyUpdater) AddPolicySetter(policyUpdatable PolicySetter, endpoint lavasession.RPCEndpoint) error {
	pu.lock.Lock()
	defer pu.lock.Unlock()
	existingPolicySetter, found := pu.policyUpdatables[endpoint.ApiInterface]
	if found {
		return utils.LavaFormatError("Trying to register to policy updates on already registered api interface", nil,
			utils.Attribute{Key: "endpoint", Value: endpoint},
			utils.Attribute{Key: "policyUpdatable", Value: existingPolicySetter})
	}
	pu.policyUpdatables[endpoint.ApiInterface] = policyUpdatable
	return nil
}

func (pu *PolicyUpdater) UpdaterKey() string {
	return CallbackKeyForPolicyUpdate + pu.chainId
}

func (pu *PolicyUpdater) setPolicy(policyUpdatable PolicySetter, policy *plantypes.Policy, apiInterface string) error {
	err := policyUpdatable.SetPolicy(policy, pu.chainId, apiInterface)
	if err != nil {
		return utils.LavaFormatError("panic level error, failed setting policy on policy updatable", err, utils.LogAttr("chainId", pu.chainId), utils.LogAttr("api_interface", apiInterface))
	}
	return nil
}

func (pu *PolicyUpdater) UpdateEpoch(epoch uint64) {
	pu.lock.Lock()
	defer pu.lock.Unlock()
	// update policy now
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	policy, err := pu.policyFetcher.GetConsumerPolicy(ctx, pu.consumerAddress, pu.chainId)
	if err != nil {
		utils.LavaFormatError("could not get GetConsumerPolicy updated, did not update policy", err, utils.LogAttr("epoch", epoch))
		return
	}
	for apiInterface, policyUpdatable := range pu.policyUpdatables {
		err = pu.setPolicy(policyUpdatable, policy, apiInterface)
		if err != nil {
			utils.LavaFormatError("Failed Updating policy", err, utils.LogAttr("apiInterface", apiInterface), utils.LogAttr("chainId", pu.chainId))
		}
	}
}
