package scores

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
)

const reputationReqName = "reputation-req"

type ReputationGetter interface {
	GetReputationScoreForBlock(ctx sdk.Context, chainID string, cluster string, provider string, block uint64) (val math.LegacyDec, entryBlock uint64, found bool)
}

// ReputationReq implements the ScoreReq interface for provider staking requirement(s)
type ReputationReq struct{}

func (rr ReputationReq) Init(policy planstypes.Policy) bool {
	return true
}

// Score gets the provider's reputation pairing score using the reputationFS fixation store
func (rr ReputationReq) Score(score PairingScore) math.LegacyDec {
	if score.reputationScore.GT(types.MaxReputationPairingScore) {
		return types.MaxReputationPairingScore
	} else if score.reputationScore.LT(types.MinReputationPairingScore) {
		return types.MinReputationPairingScore
	}

	return score.reputationScore
}

func (rr ReputationReq) GetName() string {
	return reputationReqName
}

// Equal used to compare slots to determine slot groups.
// Equal always returns true (there are no different "types" of reputation)
func (rr ReputationReq) Equal(other ScoreReq) bool {
	return true
}

func (rr ReputationReq) GetReqForSlot(policy planstypes.Policy, slotIdx int) ScoreReq {
	return rr
}
