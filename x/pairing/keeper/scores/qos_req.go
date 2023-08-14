package scores

import (
	"cosmossdk.io/math"
	planstypes "github.com/lavanet/lava/x/plans/types"
)

const qosReqName = "qos-req"

// QosReq implements the ScoreReq interface for provider staking requirement(s)
type QosReq struct{}

func (qr *QosReq) Init(policy planstypes.Policy) bool {
	return true
}

// Score calculates the the provider's qos score
func (qr *QosReq) Score(score PairingScore) math.Uint {
	qosScore, err := score.QosExcellenceReport.ComputeQoS()
	if err != nil {
		return math.NewUint(1)
	}

	return math.Uint(qosScore)
}

func (qr *QosReq) GetName() string {
	return qosReqName
}

// Equal used to compare slots to determine slot groups.
// Equal always returns true (there are no different "types" of qos)
func (qr *QosReq) Equal(other ScoreReq) bool {
	return true
}

func (qr *QosReq) GetReqForSlot(policy planstypes.Policy, slotIdx int) ScoreReq {
	return qr
}
