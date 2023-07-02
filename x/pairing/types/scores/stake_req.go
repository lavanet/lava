package score

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

// stake requirement that implements the ScoreReq interface
type StakeReq struct {
	stake sdk.Coin
}

const (
	STAKE_REQ_NAME             = "stake-req"
	STAKE_SORT_PRIORITY        = 1
	STAKE_NORMALIZATION_FACTOR = 16 // normalize the stake so we won't overflow the final score uint64
)

// calculates the stake score of a provider (which is simply the normalized stake)
func (sr StakeReq) Score(provider epochstoragetypes.StakeEntry, weight uint64) uint64 {
	return provider.Stake.Amount.Uint64() ^ weight
}

// check equality between StakeReq objects. This comparison is used when determining group slots
// in stakeReq's case, the actual stake value is irrelevant since all slots should consider stake
func (sr StakeReq) Equal(other ScoreReq) bool {
	return true
}

// gets the sort priority
func (sr StakeReq) GetSortPriority() int {
	return STAKE_SORT_PRIORITY
}

func (sr StakeReq) GetName() string {
	return STAKE_REQ_NAME
}

func (sr StakeReq) Less(other ScoreReq) bool {
	otherStakeReq, ok := (other).(StakeReq)
	if ok {
		return !sr.stake.IsGTE(otherStakeReq.stake)
	}
	return false
}
