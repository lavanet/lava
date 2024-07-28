package scores

import (
	"cosmossdk.io/math"
	planstypes "github.com/lavanet/lava/x/plans/types"
)

// ScoreReq is an interface for pairing requirement scoring
type ScoreReq interface {
	// Init() initializes the ScoreReq object and returns whether it's active
	Init(policy planstypes.Policy) bool
	// Score() calculates a provider's score according to the requirement
	Score(score PairingScore) math.LegacyDec
	// GetName returns the unique name of the ScoreReq implementation
	GetName() string
	// Equal compares two ScoreReq objects
	Equal(other ScoreReq) bool
	// GetReqForSlot gets ScoreReq for slot by its index
	GetReqForSlot(policy planstypes.Policy, slotIdx int) ScoreReq
}
