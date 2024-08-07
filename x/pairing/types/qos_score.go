package types

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/lavanet/lava/v2/utils"
)

var (
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
