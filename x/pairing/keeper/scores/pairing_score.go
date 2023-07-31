package scores

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

const (
	// TODO: temp strategy weight until we implement taking it out of the policy
	UNIFORM_WEIGHT uint64 = 1
)

// PairingScore holds a provider's score with respect to a set of requirements (ScoreReq), indexed by their unique name.
type PairingScore struct {
	Provider          *epochstoragetypes.StakeEntry
	Score             sdk.Uint
	ScoreComponents   map[string]sdk.Uint
	ValidForSelection bool
}

func NewPairingScore(stakeEntry *epochstoragetypes.StakeEntry) *PairingScore {
	score := PairingScore{
		Provider:          stakeEntry,
		Score:             sdk.OneUint(),
		ScoreComponents:   map[string]sdk.Uint{},
		ValidForSelection: true,
	}
	return &score
}

// map: key: ScoreReq name, value: weight in the final pairing score
type ScoreStrategy map[string]uint64
