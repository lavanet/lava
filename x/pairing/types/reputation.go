package types

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
)

var (
	ReputationPrefix                = collections.NewPrefix([]byte("Reputation/"))
	ReputationRefKeysPrefix         = collections.NewPrefix([]byte("ReputationRefKeys/"))
	ReputationRefKeysIteratorPrefix = collections.NewPrefix([]byte("ReputationRefKeysIterator/"))
)

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

func ReputationKey(chainID string, cluster string, provider string) collections.Triple[string, string, string] {
	return collections.Join3(chainID, cluster, provider)
}

func NewReputation(ctx sdk.Context) (Reputation, error) {
	timestamp := ctx.BlockTime().UTC().Unix()
	zeroFrac, err := NewFrac(math.LegacyZeroDec(), math.LegacyOneDec())
	if err != nil {
		return Reputation{}, utils.LavaFormatError("NewReputation: zero Frac creation failed", err)
	}
	return Reputation{
		Score:           NewQosScore(zeroFrac, zeroFrac),
		EpochScore:      NewQosScore(zeroFrac, zeroFrac),
		TimeLastUpdated: uint64(timestamp),
		CreationTime:    uint64(timestamp),
	}, nil
}

func (r Reputation) Equal(other Reputation) bool {
	return r.Score.Equal(other.Score) && r.EpochScore.Equal(other.EpochScore) &&
		r.TimeLastUpdated == other.TimeLastUpdated && r.CreationTime == other.CreationTime
}
