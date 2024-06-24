package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestQosFracValidation(t *testing.T) {
	tests := []struct {
		name  string
		qf    types.QosFrac
		valid bool
	}{
		{"valid", types.QosFrac{Num: math.LegacyOneDec(), Denom: math.LegacyOneDec()}, true},
		{"valid - zero num", types.QosFrac{Num: math.LegacyZeroDec(), Denom: math.LegacyOneDec()}, true},
		{"invalid - negative num", types.QosFrac{Num: math.LegacyNewDec(-1), Denom: math.LegacyOneDec()}, false},
		{"invalid - zero denom", types.QosFrac{Num: math.LegacyOneDec(), Denom: math.LegacyZeroDec()}, false},
		{"invalid - negative denom", types.QosFrac{Num: math.LegacyOneDec(), Denom: math.LegacyNewDec(-1)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.valid, tt.qf.IsValid())
		})
	}
}
