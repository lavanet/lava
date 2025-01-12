package utils

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"golang.org/x/exp/constraints"
)

const (
	DecayFactorNaturalBaseString = "2.71828182845904523536028747135266249775724709369995957496696763"
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

	// Step 1: Calculate e^a
	eToA := e.Power(numeratorUint64)

	// Step 2: Take the bth root
	result, err := eToA.ApproxRoot(denominatorUint64)
	if err != nil {
		panic(err)
	}

	if negative {
		// Step 3: Take the reciprocal
		result = sdk.OneDec().Quo(result)
	}

	return result
}
