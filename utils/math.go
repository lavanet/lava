package utils

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"golang.org/x/exp/constraints"
)

const (
	DecayFactorNaturalBaseString = "2.718281828459045235"
)

func Min[T constraints.Ordered](x, y T) T {
	if x < y {
		return x
	}
	return y
}

func Max[T constraints.Ordered](x, y T) T {
	if x > y {
		return x
	}
	return y
}

// NaturalBaseExponentFraction calculates an exponent of the
// natural base e using a sdk.Dec. using the formula: e^(numerator / denominator)
// since it is not possible to directly calculate a power of a fraction,
// we're doing it in three steps:
// 1. Calculate e^numerator
// 2. Take the denominatorth root
// 3. Take the reciprocal (if negative=true)
func NaturalBaseExponentFraction(numerator, denominator int64, negative bool) math.LegacyDec {
	numeratorUint64 := uint64(numerator)
	denominatorUint64 := uint64(denominator)

	e := sdk.MustNewDecFromStr(DecayFactorNaturalBaseString)

	// Step 1: Take the bth root of e
	eRoot, err := e.ApproxRoot(denominatorUint64)
	if err != nil {
		panic(err)
	}

	// Step 2: Calculate (e^(1/b))^a
	result := eRoot.Power(numeratorUint64)

	if negative {
		// Step 3: Take the reciprocal
		result = sdk.OneDec().Quo(result)
	}

	return result
}
