package types

import (
	"fmt"

	"cosmossdk.io/math"
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

func NewQosScore(score Frac, variance Frac) QosScore {
	return QosScore{Score: score, Variance: variance}
}

func (qs QosScore) Equal(other QosScore) bool {
	return qs.Score.Equal(other.Score) && qs.Variance.Equal(other.Variance)
}
