package types

import (
	"fmt"
	stdMath "math"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
)

var (
	ReputationPrefix                              = collections.NewPrefix([]byte("Reputation/"))
	ReputationPairingScoreBenchmarkStakeThreshold = sdk.NewDecWithPrec(1, 1) // 0.1 = 10%
	MaxReputationPairingScore                     = sdk.NewDec(2)
	MinReputationPairingScore                     = sdk.NewDecWithPrec(5, 1)   // 0.5
	DefaultReputationPairingScore                 = sdk.NewDecWithPrec(125, 2) // 1.25
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

// ShouldTruncate checks if the ReputationVarianceStabilizationPeriod has passed since
// the reputation's creation. If so, QoS score reports should be truncated before they're added to the
// reputation's epoch QoS score.
func (r Reputation) ShouldTruncate(stabilizationPeriod int64, currentTime int64) bool {
	return r.CreationTime+stabilizationPeriod < currentTime
}

// calcDecayFactor calculates the appropriate decay factor for a reputation from the time it was last updated
// the decay factor is: exp(-timeSinceLastUpdate/halfLifeFactor)
func (r Reputation) calcDecayFactor(halfLifeFactor int64, currentTime int64) math.LegacyDec {
	if halfLifeFactor <= 0 {
		utils.LavaFormatWarning("calcDecayFactor: calculate reputation decay factor failed, invalid half life factor",
			fmt.Errorf("half life factor is not positive"),
			utils.LogAttr("half_life_factor", halfLifeFactor),
		)
		return math.LegacyZeroDec()
	}

	timeSinceLastUpdate := currentTime - r.TimeLastUpdated
	if timeSinceLastUpdate < 0 {
		utils.LavaFormatError("calcDecayFactor: calculate reputation decay factor failed, invalid reputation",
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
		utils.LavaFormatError("calcDecayFactor: calculate reputation decay factor failed, invalid decay factor string", err,
			utils.LogAttr("decay_factor_string", decayFactorString),
			utils.LogAttr("time_since_last_update", timeSinceLastUpdate),
			utils.LogAttr("half_life_factor", halfLifeFactor),
		)
		return math.LegacyZeroDec()
	}
	return decayFactor
}

func (r Reputation) ApplyTimeDecayAndUpdateScore(halfLifeFactor int64, currentTime int64) (Reputation, error) {
	decayFactor := r.calcDecayFactor(halfLifeFactor, currentTime)
	r.Score.Score.Num = (r.Score.Score.Num.Mul(decayFactor)).Add(r.EpochScore.Score.Num)
	r.Score.Score.Denom = (r.Score.Score.Denom.Mul(decayFactor)).Add(r.EpochScore.Score.Denom)
	r.Score.Variance.Num = (r.Score.Variance.Num.Mul(decayFactor)).Add(r.EpochScore.Variance.Num)
	r.Score.Variance.Denom = (r.Score.Variance.Denom.Mul(decayFactor)).Add(r.EpochScore.Variance.Denom)
	if !r.Validate() {
		return Reputation{}, utils.LavaFormatError("ApplyTimeDecayAndUpdateScore: cannot update reputation",
			fmt.Errorf("reputation result is invalid"),
			utils.LogAttr("reputation_result", r.String()),
			utils.LogAttr("decay_factor", decayFactor.String()),
			utils.LogAttr("half_life_factor", halfLifeFactor),
		)
	}
	return r, nil
}

func (r Reputation) Validate() bool {
	if r.CreationTime <= 0 || r.TimeLastUpdated <= 0 || r.TimeLastUpdated < r.CreationTime ||
		r.Stake.Denom != commontypes.TokenDenom || !r.Score.Validate() || !r.EpochScore.Validate() {
		return false
	}

	return true
}

// ReputationScoreKey returns a key for the reputations fixation store (reputationsFS)
func ReputationScoreKey(chainID string, cluster string, provider string) string {
	return chainID + " " + cluster + " " + provider
}
