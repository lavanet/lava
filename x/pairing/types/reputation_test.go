package types

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

// TestShouldTruncate tests the should truncate method
func TestShouldTruncate(t *testing.T) {
	tests := []struct {
		name                string
		creationTime        int64
		stabilizationPeriod int64
		currentTime         int64
		truncate            bool
	}{
		{
			name:                "stabilization time not passed",
			creationTime:        1,
			stabilizationPeriod: 1,
			currentTime:         3,
			truncate:            true,
		},
		{
			name:                "stabilization time passed",
			creationTime:        3,
			stabilizationPeriod: 1,
			currentTime:         3,
			truncate:            false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reputation := Reputation{CreationTime: tt.creationTime}
			require.Equal(t, tt.truncate, reputation.ShouldTruncate(tt.stabilizationPeriod, tt.currentTime))
		})
	}
}

// TestDecayFactor tests the decay factor method. Note that upon error, the returned decay factor is zero
func TestDecayFactor(t *testing.T) {
	tests := []struct {
		name            string
		timeLastUpdated int64
		halfLifeFactor  int64
		currentTime     int64
		valid           bool
	}{
		{
			name:            "happy flow",
			timeLastUpdated: 1,
			halfLifeFactor:  1,
			currentTime:     2,
			valid:           true,
		},
		{
			name:            "invalid half life factor",
			timeLastUpdated: 1,
			halfLifeFactor:  -1,
			currentTime:     2,
			valid:           false,
		},
		{
			name:            "current time smaller than reputation last update",
			timeLastUpdated: 2,
			halfLifeFactor:  1,
			currentTime:     1,
			valid:           false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reputation := Reputation{TimeLastUpdated: tt.timeLastUpdated}
			decay := reputation.calcDecayFactor(tt.halfLifeFactor, tt.currentTime)
			if tt.valid {
				require.False(t, decay.Equal(math.LegacyZeroDec()))
			} else {
				require.True(t, decay.Equal(math.LegacyZeroDec()))
			}
		})
	}
}
