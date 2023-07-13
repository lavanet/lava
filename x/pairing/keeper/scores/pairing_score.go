package scores

import (
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

const (
	// TODO: temp strategy weight until we implement taking it out of the policy
	UNIFORM_WEIGHT uint64 = 1
)

// PairingScore holds a provider's score with respect to a set of requirements (ScoreReq), indexed by their unique name.
type PairingScore struct {
	Provider        *epochstoragetypes.StakeEntry
	Score           uint64
	ScoreComponents map[string]uint64
}

func NewPairingScore(stakeEntry *epochstoragetypes.StakeEntry) *PairingScore {
	score := PairingScore{
		Provider:        stakeEntry,
		Score:           1,
		ScoreComponents: map[string]uint64{},
	}
	return &score
}

// map: key: ScoreReq name, value: weight in the final pairing score
type ScoreStrategy map[string]uint64
