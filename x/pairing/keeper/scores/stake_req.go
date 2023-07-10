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
func (sr StakeReq) Score(stakeEntry epochstoragetypes.StakeEntry, weight uint64) uint64 {
	normalizedStake := stakeEntry.Stake.Amount.Quo(sr.MinStake)
	return normalizedStake.ToDec().Power(weight).BigInt().Uint64()
}

func (sr StakeReq) GetName() string {
	return stakeReqName
}
