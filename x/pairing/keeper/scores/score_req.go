package scores

import (
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

// ScoreReq is an interface for pairing requirement scoring
type ScoreReq interface {
	// Score() calculates a provider's score according to the requirement
	Score(stakeEntry epochstoragetypes.StakeEntry) uint64
	// GetName returns the unique name of the ScoreReq implementation
	GetName() string
}
