package types

import (
	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
)

var (
	ReputationPrefix = collections.NewPrefix([]byte("Reputation/"))
)

// ReputationKey returns a key to the reputations indexed map
func ReputationKey(chainID string, cluster string, provider string) collections.Triple[string, string, string] {
	return collections.Join3(chainID, cluster, provider)
}

func NewReputation(ctx sdk.Context) Reputation {
	timestamp := ctx.BlockTime().UTC().Unix()
	return Reputation{
		Score:           ZeroQosScore,
		EpochScore:      ZeroQosScore,
		TimeLastUpdated: timestamp,
		CreationTime:    timestamp,
		Stake:           sdk.NewCoin(commontypes.TokenDenom, sdk.ZeroInt()),
	}
}

func (r Reputation) Equal(other Reputation) bool {
	return r.Score.Equal(other.Score) && r.EpochScore.Equal(other.EpochScore) &&
		r.TimeLastUpdated == other.TimeLastUpdated && r.CreationTime == other.CreationTime &&
		r.Stake.IsEqual(other.Stake)
}

// ReputationScoreKey returns a key for the reputations fixation store (reputationsFS)
func ReputationScoreKey(chainID string, cluster string, provider string) string {
	return chainID + " " + cluster + " " + provider
}
