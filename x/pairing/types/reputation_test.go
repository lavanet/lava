package types

import (
	"testing"

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
