package types

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/lavanet/lava/v2/utils"
)

var (
	FailureCost       int64 = 3000 // failed relay cost for QoS excellence report computation in milliseconds
	TruncateStdMargin int64 = 3    // number of standard deviations that determine the truncation limit

	// zero QoS score is: score = 0, var = 0
	ZeroQosScore = QosScore{
		Score:    Frac{Num: math.LegacyZeroDec(), Denom: math.LegacySmallestDec()},
		Variance: Frac{Num: math.LegacyZeroDec(), Denom: math.LegacySmallestDec()},
	}
)

func NewFrac(num math.LegacyDec, denom math.LegacyDec) (Frac, error) {
	if denom.Equal(math.LegacyZeroDec()) {
		return Frac{}, fmt.Errorf("cannot create Frac with zero denomination")
	}
	return Frac{Num: num, Denom: denom}, nil
}

func (f Frac) Equal(other Frac) bool {
	return f.Num.Equal(other.Num) && f.Denom.Equal(other.Denom)
}

func (f Frac) Resolve() (math.LegacyDec, error) {
	if f.Denom.IsZero() {
		return math.LegacyZeroDec(), utils.LavaFormatError("Frac Resolve: resolve failed", fmt.Errorf("frac has zero denom"))
	}

	return f.Num.Quo(f.Denom), nil
}

func NewQosScore(score Frac, variance Frac) QosScore {
	return QosScore{Score: score, Variance: variance}
}

func (qs QosScore) Equal(other QosScore) bool {
	return qs.Score.Equal(other.Score) && qs.Variance.Equal(other.Variance)
}

// Validate validates that both nums are non-negative and both denoms are strictly positive (non-zero)
func (qs QosScore) Validate() bool {
	if qs.Score.Num.IsNegative() || !qs.Score.Denom.IsPositive() || qs.Variance.Num.IsNegative() ||
		!qs.Variance.Denom.IsPositive() {
		return false
	}
	return true
}

// Update updates a QosScore with a new score from the QoS excellence report. The new score is truncated by the
// current variance. Then, it's updated using the weight (which is currently the relay num)
func (qs QosScore) Update(score math.LegacyDec, truncate bool, weight int64) QosScore {
	if truncate {
		score = qs.truncate(score)
	}

	// updated_score_num = qos_score_num + score * weight
	// updated_score_denom = qos_score_denom + weight
	qs.Score.Num = qs.Score.Num.Add(score.MulInt64(weight))
	qs.Score.Denom = qs.Score.Denom.Add(math.LegacyNewDec(weight))

	// updated_variance_num = qos_variance_num + (qos_score_num - score)^2 * weight
	// updated_score_denom = qos_score_denom + weight
	qs.Variance.Num = qs.Variance.Num.Add((qs.Score.Num.Sub(score)).Power(2).MulInt64(weight))
	qs.Variance.Denom = qs.Variance.Denom.Add(math.LegacyNewDec(weight))

	return qs
}

// Truncate truncates the QoS excellece report score by the current QoS score variance
func (qs QosScore) truncate(score math.LegacyDec) math.LegacyDec {
	std, err := qs.Variance.Num.ApproxSqrt()
	if err != nil {
		utils.LavaFormatError("QosScore truncate: truncate failed, could not calculate qos variance sqrt", err,
			utils.LogAttr("qos_score_variance", qs.Variance.String()),
		)
		return score
	}

	// truncated score = max(min(score, qos_score_num + 3*std), qos_score_num - 3*std)
	return math.LegacyMaxDec(math.LegacyMinDec(score, qs.Score.Num.Add(std.MulInt64(TruncateStdMargin))),
		qs.Score.Num.Sub(std.MulInt64(TruncateStdMargin)))
}
