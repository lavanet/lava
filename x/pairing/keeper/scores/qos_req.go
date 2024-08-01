package scores

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
)

const qosReqName = "qos-req"

type QosGetter interface {
	GetQos(ctx sdk.Context, chainID string, cluster string, provider string) (pairingtypes.QualityOfServiceReport, error)
}

// QosReq implements the ScoreReq interface for provider staking requirement(s)
type QosReq struct{}

func (qr QosReq) Init(policy planstypes.Policy) bool {
	return true
}

// Score calculates the the provider's qos score
func (qr QosReq) Score(score PairingScore) math.Uint {
	// TODO: update Qos in providerQosFS properly and uncomment this code below
	// Also, the qos score should range between 0.5-2

	// qosScore, err := score.QosExcellenceReport.ComputeQoS()
	// if err != nil {
	// 	return math.NewUint(1)
	// }

	// return math.Uint(qosScore)
	return math.NewUint(1)
}

func (qr QosReq) GetName() string {
	return qosReqName
}

// Equal used to compare slots to determine slot groups.
// Equal always returns true (there are no different "types" of qos)
func (qr QosReq) Equal(other ScoreReq) bool {
	return true
}

func (qr QosReq) GetReqForSlot(policy planstypes.Policy, slotIdx int) ScoreReq {
	return qr
}
