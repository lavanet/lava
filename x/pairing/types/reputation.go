package types

import (
	"fmt"
	stdMath "math"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
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

// ShouldTruncate checks if the ReputationVarianceStabilizationPeriod has passed since
// the reputation's creation. If so, QoS score reports should be truncated before they're added to the
// reputation's epoch QoS score.
func (r Reputation) ShouldTruncate(stabilizationPeriod int64, currentTime int64) bool {
	return r.CreationTime+stabilizationPeriod < currentTime
}

// DecayFactor calculates the appropriate decay factor for a reputation from the time it was last updated
// the decay factor is: exp(-timeSinceLastUpdate/halfLifeFactor)
func (r Reputation) DecayFactor(halfLifeFactor int64, currentTime int64) math.LegacyDec {
	if halfLifeFactor <= 0 {
		utils.LavaFormatWarning("DecayFactor: calculate reputation decay factor failed, invalid half life factor",
			fmt.Errorf("half life factor is not positive"),
			utils.LogAttr("half_life_factor", halfLifeFactor),
		)
		return math.LegacyZeroDec()
	}

	timeSinceLastUpdate := currentTime - r.TimeLastUpdated
	if timeSinceLastUpdate < 0 {
		utils.LavaFormatError("DecayFactor: calculate reputation decay factor failed, invalid reputation",
			fmt.Errorf("reputation last update time is larger than current time"),
			utils.LogAttr("current_time", currentTime),
			utils.LogAttr("reputation_time_last_updated", r.TimeLastUpdated),
		)
		return math.LegacyZeroDec()
	}

	exponent := float64(timeSinceLastUpdate / halfLifeFactor)
	decayFactorFloat := stdMath.Exp(exponent)
	decayFactorString := fmt.Sprintf("%.18f", decayFactorFloat)
	decayFactor, err := math.LegacyNewDecFromStr(decayFactorString)
	if err != nil {
		utils.LavaFormatError("DecayFactor: calculate reputation decay factor failed, invalid decay factor string", err,
			utils.LogAttr("decay_factor_string", decayFactorString),
			utils.LogAttr("time_since_last_update", timeSinceLastUpdate),
			utils.LogAttr("half_life_factor", halfLifeFactor),
		)
		return math.LegacyZeroDec()
	}
	return decayFactor
}

// ReputationScoreKey returns a key for the reputations fixation store (reputationsFS)
func ReputationScoreKey(chainID string, cluster string, provider string) string {
	return chainID + " " + cluster + " " + provider
}
