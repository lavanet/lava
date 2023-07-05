package score

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

// stake requirement that implements the ScoreReq interface
type StakeReq struct {
	stake sdk.Coin
}

// calculates the stake score of a provider (which is simply the normalized stake)
func (sr StakeReq) Score(provider epochstoragetypes.StakeEntry, weight uint64) uint64 {
	return provider.Stake.Amount.ToDec().Power(weight).BigInt().Uint64()
}
