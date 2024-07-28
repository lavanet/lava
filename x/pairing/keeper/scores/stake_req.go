package scores

import (
	"cosmossdk.io/math"
	planstypes "github.com/lavanet/lava/x/plans/types"
)

const stakeReqName = "stake-req"

// StakeReq implements the ScoreReq interface for provider staking requirement(s)
type StakeReq struct{}

func (sr *StakeReq) Init(policy planstypes.Policy) bool {
	return true
}

// Score calculates the the provider score as the normalized stake
func (sr *StakeReq) Score(score PairingScore) math.LegacyDec {
	effectiveStake := score.Provider.EffectiveStake()
	if !effectiveStake.IsPositive() {
		return math.LegacyOneDec()
	}
	return math.LegacyNewDecFromInt(effectiveStake)
}

func (sr *StakeReq) GetName() string {
	return stakeReqName
}

// Equal used to compare slots to determine slot groups.
// Equal always returns true (StakeReq score of an entry is the entry's stake)
func (sr *StakeReq) Equal(other ScoreReq) bool {
	return true
}

func (sr *StakeReq) GetReqForSlot(policy planstypes.Policy, slotIdx int) ScoreReq {
	return sr
}
