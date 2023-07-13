package scores

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

const stakeReqName = "stake-req"

// StakeReq implements the ScoreReq interface for provider staking requirement(s)
type StakeReq struct {
	MinStake sdk.Int
}

// Score calculates the the provider score as the normalized stake
func (sr StakeReq) Score(stakeEntry epochstoragetypes.StakeEntry) uint64 {
	return stakeEntry.Stake.Amount.Quo(sr.MinStake).Uint64()
}

func (sr StakeReq) GetName() string {
	return stakeReqName
}

// Equal used to compare slots to determine slot groups.
// Equal always returns true (StakeReq score of an entry is the entry's stake)
func (sr StakeReq) Equal(other ScoreReq) bool {
	return true
}
