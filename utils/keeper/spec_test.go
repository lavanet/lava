package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetSpecFromGit(t *testing.T) {
	t.SkipNow()

	tests := []struct {
		name      string
		url       string
		index     string
		wantError bool
	}{
		{
			name:      "Invalid URL format",
			url:       "https://github.com/invalid/url",
			index:     "test-index",
			wantError: true,
		},
		{
			name:      "Valid URL but non-existent repository",
			url:       "https://github.com/lavanet/nonexistent-repo/tree/main/specs",
			index:     "test-index",
			wantError: true,
		},
		{
			name:      "Valid URL with non-existent spec index",
			url:       "https://github.com/lavanet/lava/tree/main/specs",
			index:     "nonexistent-index",
			wantError: true,
		},
		{
			name:      "Valid URL and existing spec index",
			url:       "https://github.com/lavanet/lava/tree/main/specs/mainnet-1/specs",
			index:     "LAV1", // Replace with a known valid spec index
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := GetSpecFromGit(tt.url, tt.index)

			if tt.wantError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, spec)
			require.Equal(t, tt.index, spec.Index)
			// Add more specific assertions about the spec content if needed
		})
	}
}
