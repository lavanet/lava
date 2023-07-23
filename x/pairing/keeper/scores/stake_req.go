package scores

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

const stakeReqName = "stake-req"

// StakeReq implements the ScoreReq interface for provider staking requirement(s)
type StakeReq struct{}

func (sr *StakeReq) Init() bool {
	return true
}

// Score calculates the the provider score as the normalized stake
func (sr *StakeReq) Score(stakeEntry epochstoragetypes.StakeEntry) sdk.Uint {
	return sdk.NewUint(stakeEntry.Stake.Amount.Uint64())
}

func (sr *StakeReq) GetName() string {
	return stakeReqName
}

// Equal used to compare slots to determine slot groups.
// Equal always returns true (StakeReq score of an entry is the entry's stake)
func (sr *StakeReq) Equal(other ScoreReq) bool {
	return true
}

func (sr *StakeReq) GetReqForSlot(slotIdx int) ScoreReq {
	return sr
}
