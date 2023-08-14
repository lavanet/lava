package scores

import (
	"cosmossdk.io/math"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
)

const qosReqName = "qos-req"

// QosReq implements the ScoreReq interface for provider staking requirement(s)
type QosReq struct {
	qos pairingtypes.QualityOfServiceReport
}

func (qr *QosReq) Init(policy planstypes.Policy) bool {
	return true
}

// Score calculates the the provider's qos score
func (qr *QosReq) Score(score PairingScore) math.Uint {
	return math.NewUint(1) // TBD
}

func (qr *QosReq) GetName() string {
	return qosReqName
}

// Equal used to compare slots to determine slot groups.
// Equal always returns true (QosReq score of an entry is the entry's qos)
func (qr *QosReq) Equal(other ScoreReq) bool {
	return true
}

func (qr *QosReq) GetReqForSlot(policy planstypes.Policy, slotIdx int) ScoreReq {
	return qr
}
