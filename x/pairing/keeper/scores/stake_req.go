package scores

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

const stake_req_name = "stake-req"

// stake requirement that implements the ScoreReq interface
type StakeReq struct {
	MinStake sdk.Int
}

// calculates the stake score of a provider (which is simply the normalized stake)
func (sr StakeReq) Score(stakeEntry epochstoragetypes.StakeEntry, weight uint64) uint64 {
	normalizedStake := stakeEntry.Stake.Amount.Quo(sr.MinStake)
	return normalizedStake.ToDec().Power(weight).BigInt().Uint64()
}

func (sr StakeReq) GetName() string {
	return stake_req_name
}
