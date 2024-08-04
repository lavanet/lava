package types

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	ReputationPrefix                = collections.NewPrefix([]byte("Reputation/"))
	ReputationRefKeysPrefix         = collections.NewPrefix([]byte("ReputationRefKeys/"))
	ReputationRefKeysIteratorPrefix = collections.NewPrefix([]byte("ReputationRefKeysIterator/"))
)

// ReputationRefIndexes are defined as a "multi" index that can reference several reputations
// The ref indices are chain+cluser. The primary indices are chain+cluster+provider.
type ReputationRefIndexes struct {
	Index *indexes.Multi[collections.Pair[string, string], collections.Triple[string, string, string], Reputation]
}

func (rri ReputationRefIndexes) IndexesList() []collections.Index[collections.Triple[string, string, string], Reputation] {
	return []collections.Index[collections.Triple[string, string, string], Reputation]{rri.Index}
}

func NewReputationRefIndexes(sb *collections.SchemaBuilder) ReputationRefIndexes {
	return ReputationRefIndexes{
		Index: indexes.NewMulti(sb, ReputationRefKeysIteratorPrefix, "reputation_ref_keys_iterator",
			collections.PairKeyCodec(collections.StringKey, collections.StringKey),
			collections.TripleKeyCodec(collections.StringKey, collections.StringKey, collections.StringKey),
			func(pk collections.Triple[string, string, string], _ Reputation) (collections.Pair[string, string], error) {
				return collections.Join(pk.K1(), pk.K2()), nil
			},
		),
	}
}

// ReputationKey returns a key to the reputations indexed map
func ReputationKey(chainID string, cluster string, provider string) collections.Triple[string, string, string] {
	return collections.Join3(chainID, cluster, provider)
}

func NewReputation(ctx sdk.Context) Reputation {
	timestamp := ctx.BlockTime().UTC().Unix()
	return Reputation{
		Score:           DefaultQosScore,
		EpochScore:      DefaultQosScore,
		TimeLastUpdated: timestamp,
		CreationTime:    timestamp,
	}
}

func (r Reputation) Equal(other Reputation) bool {
	return r.Score.Equal(other.Score) && r.EpochScore.Equal(other.EpochScore) &&
		r.TimeLastUpdated == other.TimeLastUpdated && r.CreationTime == other.CreationTime
}

// ReputationScoreKey returns a key for the reputations fixation store (reputationsFS)
func ReputationScoreKey(chainID string, cluster string, provider string) string {
	return chainID + " " + cluster + " " + provider
}
