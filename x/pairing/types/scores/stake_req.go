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
	STAKE_NORMALIZATION_FACTOR = 16 // normalize the stake so we won't overflow the final score uint64
)

// calculates the stake score of a provider (which is simply the normalized stake)
func (sr StakeReq) Score(provider epochstoragetypes.StakeEntry, weight uint64) uint64 {
	return provider.Stake.Amount.Uint64() ^ weight
}

func (sr StakeReq) GetBitmapValue() uint64 {
	return STAKE_REQ
}
