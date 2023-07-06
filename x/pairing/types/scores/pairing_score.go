package score

import (
	"reflect"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

const (
	// TODO: temp strategy weight until we implement taking it out of the policy
	UNIFORM_WEIGHT uint64 = 1
)

// object to hold a provider's score and the requirements that construct the score (map[scoreReqName] -> score from that req)
type PairingScore struct {
	Provider        *epochstoragetypes.StakeEntry
	Score           uint64
	ScoreComponents map[reflect.Type]uint64
}

func NewPairingScore(provider *epochstoragetypes.StakeEntry) *PairingScore {
	score := PairingScore{
		Provider:        provider,
		Score:           1,
		ScoreComponents: map[reflect.Type]uint64{},
	}
	return &score
}

// map: key: ScoreReq name, value: weight in the final pairing score
type ScoreStrategy map[reflect.Type]uint64
