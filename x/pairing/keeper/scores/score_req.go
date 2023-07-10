package scores

import (
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

// when adding a new req, update CalcSlots()

// Score() calculates a provider's score according to the requirement
// GetName() gets the ScoreReq's name
type ScoreReq interface {
	Score(stakeEntry epochstoragetypes.StakeEntry, weight uint64) uint64
	GetName() string
}
