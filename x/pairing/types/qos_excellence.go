package types

import (
	"fmt"

	"cosmossdk.io/math"
)

var DefaultQosFrac = QosFrac{Num: math.LegacyNewDec(5), Denom: math.LegacyNewDec(4)}

// NewQosFrac creates a new QosFrac object and verifies it's valid
func NewQosFrac(num math.LegacyDec, denom math.LegacyDec) (QosFrac, error) {
	qf := QosFrac{Num: num, Denom: denom}
	if !qf.IsValid() {
		return QosFrac{}, fmt.Errorf("invalid QosFrac. Num: %s, Denom: %s", num.String(), denom.String())
	}

	return qf, nil
}

// IsValid is a QosFrac method that validates that the num is not negative and the denom is larger than zero
func (qf QosFrac) IsValid() bool {
	return qf.Num.GTE(math.LegacyZeroDec()) && qf.Denom.GT(math.LegacyZeroDec())
}
