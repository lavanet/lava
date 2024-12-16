package types_test

import (
	"testing"

	"github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestSortSpecsByHierarchy(t *testing.T) {
	tests := []struct {
		name          string
		specs         []types.Spec
		expectedOrder []string
		expectError   bool
	}{
		{
			name:          "empty specs",
			specs:         []types.Spec{},
			expectedOrder: []string{},
			expectError:   false,
		},
		{
			name: "single spec no imports",
			specs: []types.Spec{
				{Index: "spec1", Imports: []string{}},
			},
			expectedOrder: []string{"spec1"},
			expectError:   false,
		},
		{
			name: "linear dependency",
			specs: []types.Spec{
				{Index: "spec2", Imports: []string{"spec1"}},
				{Index: "spec1", Imports: []string{}},
				{Index: "spec3", Imports: []string{"spec2"}},
			},
			expectedOrder: []string{"spec1", "spec2", "spec3"},
			expectError:   false,
		},
		{
			name: "multiple independent specs",
			specs: []types.Spec{
				{Index: "spec1", Imports: []string{}},
				{Index: "spec2", Imports: []string{}},
				{Index: "spec3", Imports: []string{}},
			},
			expectedOrder: []string{"spec1", "spec2", "spec3"},
			expectError:   false,
		},
		{
			name: "complex dependencies",
			specs: []types.Spec{
				{Index: "spec3", Imports: []string{"spec1", "spec2"}},
				{Index: "spec2", Imports: []string{"spec1"}},
				{Index: "spec1", Imports: []string{}},
				{Index: "spec4", Imports: []string{"spec2", "spec3"}},
			},
			expectedOrder: []string{"spec1", "spec2", "spec3", "spec4"},
			expectError:   false,
		},
		{
			name: "circular dependency",
			specs: []types.Spec{
				{Index: "spec1", Imports: []string{"spec2"}},
				{Index: "spec2", Imports: []string{"spec1"}},
			},
			expectedOrder: nil,
			expectError:   true,
		},
		{
			name: "self dependency",
			specs: []types.Spec{
				{Index: "spec1", Imports: []string{"spec1"}},
			},
			expectedOrder: nil,
			expectError:   true,
		},
		{
			name: "complex circular dependency",
			specs: []types.Spec{
				{Index: "spec1", Imports: []string{"spec3"}},
				{Index: "spec2", Imports: []string{"spec1"}},
				{Index: "spec3", Imports: []string{"spec2"}},
			},
			expectedOrder: nil,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorted, err := types.SortSpecsByHierarchy(tt.specs)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, sorted)
			} else {
				require.NoError(t, err)
				require.NotNil(t, sorted)
				require.Equal(t, len(tt.expectedOrder), len(sorted))

				// Check if the order matches expected
				for i, expectedIndex := range tt.expectedOrder {
					require.Equal(t, expectedIndex, sorted[i].Index)
				}

				// Verify that dependencies are satisfied
				processed := make(map[string]bool)
				for _, spec := range sorted {
					// Check that all imports are already processed
					for _, imp := range spec.Imports {
						require.True(t, processed[imp],
							"Spec %s depends on %s which hasn't been processed yet",
							spec.Index, imp)
					}
					processed[spec.Index] = true
				}
			}
		})
	}
}
