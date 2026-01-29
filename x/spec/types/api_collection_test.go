package types_test

import (
	"testing"

	"github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestSpecCategoryCombine(t *testing.T) {
	tests := []struct {
		name     string
		first    types.SpecCategory
		second   types.SpecCategory
		expected types.SpecCategory
	}{
		{
			name: "two stateless categories",
			first: types.SpecCategory{
				Deterministic: true,
				Local:         false,
				Subscription:  false,
				Stateful:      0,
				HangingApi:    false,
			},
			second: types.SpecCategory{
				Deterministic: true,
				Local:         false,
				Subscription:  false,
				Stateful:      0,
				HangingApi:    false,
			},
			expected: types.SpecCategory{
				Deterministic: true,
				Local:         false,
				Subscription:  false,
				Stateful:      0, // max(0, 0) = 0
				HangingApi:    false,
			},
		},
		{
			name: "one stateful and one stateless",
			first: types.SpecCategory{
				Deterministic: true,
				Local:         false,
				Subscription:  false,
				Stateful:      1, // stateful
				HangingApi:    false,
			},
			second: types.SpecCategory{
				Deterministic: true,
				Local:         false,
				Subscription:  false,
				Stateful:      0, // stateless
				HangingApi:    false,
			},
			expected: types.SpecCategory{
				Deterministic: true,
				Local:         false,
				Subscription:  false,
				Stateful:      1, // max(1, 0) = 1, should be treated as stateful
				HangingApi:    false,
			},
		},
		{
			name: "two stateful categories",
			first: types.SpecCategory{
				Deterministic: true,
				Local:         false,
				Subscription:  false,
				Stateful:      1,
				HangingApi:    false,
			},
			second: types.SpecCategory{
				Deterministic: true,
				Local:         false,
				Subscription:  false,
				Stateful:      1,
				HangingApi:    false,
			},
			expected: types.SpecCategory{
				Deterministic: true,
				Local:         false,
				Subscription:  false,
				Stateful:      1, // max(1, 1) = 1, should remain stateful (not 2!)
				HangingApi:    false,
			},
		},
		{
			name: "deterministic combines with AND",
			first: types.SpecCategory{
				Deterministic: true,
				Stateful:      0,
			},
			second: types.SpecCategory{
				Deterministic: false,
				Stateful:      0,
			},
			expected: types.SpecCategory{
				Deterministic: false, // true && false = false
				Stateful:      0,
			},
		},
		{
			name: "local combines with OR",
			first: types.SpecCategory{
				Local:    true,
				Stateful: 0,
			},
			second: types.SpecCategory{
				Local:    false,
				Stateful: 0,
			},
			expected: types.SpecCategory{
				Local:    true, // true || false = true
				Stateful: 0,
			},
		},
		{
			name: "subscription combines with OR",
			first: types.SpecCategory{
				Subscription: false,
				Stateful:     0,
			},
			second: types.SpecCategory{
				Subscription: true,
				Stateful:     0,
			},
			expected: types.SpecCategory{
				Subscription: true, // false || true = true
				Stateful:     0,
			},
		},
		{
			name: "hangingApi combines with OR",
			first: types.SpecCategory{
				HangingApi: false,
				Stateful:   0,
			},
			second: types.SpecCategory{
				HangingApi: true,
				Stateful:   0,
			},
			expected: types.SpecCategory{
				HangingApi: true, // false || true = true
				Stateful:   0,
			},
		},
		{
			name: "batch with multiple stateful methods should remain stateful",
			first: types.SpecCategory{
				Deterministic: false,
				Local:         false,
				Subscription:  false,
				Stateful:      1,
				HangingApi:    false,
			},
			second: types.SpecCategory{
				Deterministic: false,
				Local:         false,
				Subscription:  false,
				Stateful:      1,
				HangingApi:    false,
			},
			expected: types.SpecCategory{
				Deterministic: false,
				Local:         false,
				Subscription:  false,
				Stateful:      1, // Critical: must be 1, not 2!
				HangingApi:    false,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.first.Combine(tc.second)
			require.Equal(t, tc.expected.Deterministic, result.Deterministic, "Deterministic mismatch")
			require.Equal(t, tc.expected.Local, result.Local, "Local mismatch")
			require.Equal(t, tc.expected.Subscription, result.Subscription, "Subscription mismatch")
			require.Equal(t, tc.expected.Stateful, result.Stateful, "Stateful mismatch")
			require.Equal(t, tc.expected.HangingApi, result.HangingApi, "HangingApi mismatch")
		})
	}
}

// TestSpecCategoryCombineStatefulBatchScenarios tests specific batch scenarios
// to ensure stateful batches are handled correctly
func TestSpecCategoryCombineStatefulBatchScenarios(t *testing.T) {
	const (
		STATELESS = 0
		STATEFUL  = 1 // CONSISTENCY_SELECT_ALL_PROVIDERS
	)

	t.Run("batch of 3 stateless methods", func(t *testing.T) {
		cat1 := types.SpecCategory{Stateful: STATELESS}
		cat2 := types.SpecCategory{Stateful: STATELESS}
		cat3 := types.SpecCategory{Stateful: STATELESS}

		combined := cat1.Combine(cat2).Combine(cat3)
		require.Equal(t, uint32(STATELESS), combined.Stateful, "3 stateless should result in stateless")
	})

	t.Run("batch of 1 stateful and 2 stateless methods", func(t *testing.T) {
		cat1 := types.SpecCategory{Stateful: STATEFUL}
		cat2 := types.SpecCategory{Stateful: STATELESS}
		cat3 := types.SpecCategory{Stateful: STATELESS}

		combined := cat1.Combine(cat2).Combine(cat3)
		require.Equal(t, uint32(STATEFUL), combined.Stateful, "1 stateful + 2 stateless should result in stateful")
	})

	t.Run("batch of 2 stateful and 1 stateless methods", func(t *testing.T) {
		cat1 := types.SpecCategory{Stateful: STATEFUL}
		cat2 := types.SpecCategory{Stateful: STATEFUL}
		cat3 := types.SpecCategory{Stateful: STATELESS}

		combined := cat1.Combine(cat2).Combine(cat3)
		require.Equal(t, uint32(STATEFUL), combined.Stateful, "2 stateful + 1 stateless should result in stateful (not 2!)")
	})

	t.Run("batch of 3 stateful methods", func(t *testing.T) {
		cat1 := types.SpecCategory{Stateful: STATEFUL}
		cat2 := types.SpecCategory{Stateful: STATEFUL}
		cat3 := types.SpecCategory{Stateful: STATEFUL}

		combined := cat1.Combine(cat2).Combine(cat3)
		require.Equal(t, uint32(STATEFUL), combined.Stateful, "3 stateful should result in stateful (not 3!)")
	})
}
