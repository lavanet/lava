package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test for the double-slash bug fix in convertGitHubURLToRaw
func TestConvertGitHubURLToRaw_NoDoubleSlash(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:     "Root directory with trailing slash",
			input:    "https://github.com/magma-devs/lava-specs/tree/main/",
			expected: "https://raw.githubusercontent.com/magma-devs/lava-specs/main",
		},
		{
			name:     "Root directory without trailing slash",
			input:    "https://github.com/magma-devs/lava-specs/tree/main",
			expected: "https://raw.githubusercontent.com/magma-devs/lava-specs/main",
		},
		{
			name:     "Subdirectory with trailing slash",
			input:    "https://github.com/owner/repo/tree/branch/subdir/",
			expected: "https://raw.githubusercontent.com/owner/repo/branch/subdir",
		},
		{
			name:     "Subdirectory without trailing slash",
			input:    "https://github.com/owner/repo/tree/branch/subdir",
			expected: "https://raw.githubusercontent.com/owner/repo/branch/subdir",
		},
		{
			name:     "Deep subdirectory",
			input:    "https://github.com/owner/repo/tree/main/path/to/specs",
			expected: "https://raw.githubusercontent.com/owner/repo/main/path/to/specs",
		},
		{
			name:        "Invalid URL - no tree",
			input:       "https://github.com/owner/repo/blob/main/file.json",
			expectError: true,
		},
		{
			name:        "Invalid URL - not enough parts",
			input:       "https://github.com/owner/repo",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertGitHubURLToRaw(tt.input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)

				// Ensure no double slashes in the result (except in https://)
				cleanResult := result[8:] // Remove "https://"
				require.NotContains(t, cleanResult, "//", "URL should not contain double slashes")
			}
		})
	}
}

func TestConvertGitHubURLToAPI_NoDoubleSlash(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:     "Root directory",
			input:    "https://github.com/magma-devs/lava-specs/tree/main/",
			expected: "https://api.github.com/repos/magma-devs/lava-specs/contents/?ref=main",
		},
		{
			name:     "Subdirectory",
			input:    "https://github.com/owner/repo/tree/branch/specs",
			expected: "https://api.github.com/repos/owner/repo/contents/specs?ref=branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertGitHubURLToAPI(tt.input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

// Test that when appending filenames, we don't get double slashes
func TestURLConstruction_NoDoubleSlash(t *testing.T) {
	baseURLs := []string{
		"https://github.com/magma-devs/lava-specs/tree/main/",
		"https://github.com/magma-devs/lava-specs/tree/main",
	}

	for _, baseURL := range baseURLs {
		t.Run(baseURL, func(t *testing.T) {
			rawURL, err := convertGitHubURLToRaw(baseURL)
			require.NoError(t, err)

			// Simulate appending a filename (as done in getAllSpecsWithToken)
			fullURL := rawURL + "/" + "ethereum.json"

			// Check for double slashes (excluding https://)
			cleanURL := fullURL[8:] // Remove "https://"
			require.NotContains(t, cleanURL, "//", "Constructed URL should not have double slashes")

			// Verify correct structure
			require.Contains(t, fullURL, "https://raw.githubusercontent.com/magma-devs/lava-specs/main/ethereum.json")
		})
	}
}
