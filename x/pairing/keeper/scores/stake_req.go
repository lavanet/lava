package scores

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
)

const stakeReqName = "stake-req"

// StakeReq implements the ScoreReq interface for provider staking requirement(s)
type StakeReq struct{}

func (sr *StakeReq) Init(policy planstypes.Policy) bool {
	return true
}

// Score calculates the the provider score as the normalized stake
func (sr *StakeReq) Score(score PairingScore) math.Uint {
	effectiveStake := score.Provider.EffectiveStake()
	if !effectiveStake.IsPositive() {
		return math.OneUint()
	}
	return sdk.NewUint(effectiveStake.Uint64())
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
