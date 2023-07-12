package score

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

// stake requirement that implements the ScoreReq interface
type StakeReq struct {
	MinStake sdk.Int
}

const (
	STAKE_REQ_NAME = "stake-req"
)

// calculates the stake score of a provider (which is simply the normalized stake)
func (sr StakeReq) Score(stakeEntry epochstoragetypes.StakeEntry, weight uint64) uint64 {
	return stakeEntry.Stake.Amount.Quo(sr.MinStake).Uint64()
}

func (sr StakeReq) GetName() string {
	return STAKE_REQ_NAME
}

// Equal() used to compare slots to determine slot groups.
// Usually Equal() compares the internal values of the ScoreReq,
// but StakeReq returns always true because the stake is the score itself
func (sr StakeReq) Equal(other ScoreReq) bool {
	return true
}
