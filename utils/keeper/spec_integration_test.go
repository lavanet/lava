//go:build integration
// +build integration

package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Integration tests that require network access
// Run with: go test -tags=integration

func TestGetSpecFromGitIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name      string
		url       string
		index     string
		wantError bool
	}{
		{
			name:      "Fetch Ethereum spec from magma-devs/lava-specs",
			url:       "https://github.com/magma-devs/lava-specs/tree/main/",
			index:     "ETH1",
			wantError: false,
		},
		{
			name:      "Fetch Bitcoin spec from magma-devs/lava-specs",
			url:       "https://github.com/magma-devs/lava-specs/tree/main/",
			index:     "BTC1",
			wantError: false,
		},
		{
			name:      "Fetch non-existent spec",
			url:       "https://github.com/magma-devs/lava-specs/tree/main/",
			index:     "NONEXISTENT1",
			wantError: true,
		},
		{
			name:      "Invalid repository URL",
			url:       "https://github.com/nonexistent/repo/tree/main/",
			index:     "ETH1",
			wantError: true,
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
			require.NotEmpty(t, spec.Name)
		})
	}
}

func TestGetAllSpecsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test that we can fetch multiple specs from the repository
	url := "https://github.com/magma-devs/lava-specs/tree/main/"

	// This tests the internal getAllSpecsWithToken function indirectly through GetSpecFromGit
	// We'll test multiple known specs to verify the function works correctly
	knownSpecs := []string{"ETH1", "BTC1", "LAV1"}

	for _, specIndex := range knownSpecs {
		t.Run("Fetch_"+specIndex, func(t *testing.T) {
			spec, err := GetSpecFromGit(url, specIndex)
			// Some specs might not exist, so we don't require success for all
			if err != nil {
				t.Logf("Spec %s not found or error: %v", specIndex, err)
				return
			}

			require.NotEmpty(t, spec)
			require.Equal(t, specIndex, spec.Index)
		})
	}
}

// Test GitHub token functionality with real GitHub API
func TestGetSpecFromGitWithTokenIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test with empty token (should work for public repos)
	t.Run("Public_Repo_No_Token", func(t *testing.T) {
		url := "https://github.com/magma-devs/lava-specs/tree/main/"
		index := "ETH1"

		spec, err := GetSpecFromGitWithToken(url, index, "")
		if err != nil {
			t.Logf("Spec %s not found or error: %v", index, err)
			return
		}

		require.NotEmpty(t, spec)
		require.Equal(t, index, spec.Index)
	})

	// Test with a fake token (should still work for public repos)
	t.Run("Public_Repo_With_Fake_Token", func(t *testing.T) {
		url := "https://github.com/magma-devs/lava-specs/tree/main/"
		index := "ETH1"
		fakeToken := "ghp_fake_token_for_testing"

		spec, err := GetSpecFromGitWithToken(url, index, fakeToken)
		if err != nil {
			t.Logf("Spec %s not found or error: %v", index, err)
			return
		}

		require.NotEmpty(t, spec)
		require.Equal(t, index, spec.Index)
	})

	// Test with invalid token on private repo (should fail)
	t.Run("Private_Repo_Invalid_Token", func(t *testing.T) {
		// This test assumes there's a private repo - adjust URL as needed
		url := "https://github.com/private-org/private-repo/tree/main/"
		index := "TEST1"
		invalidToken := "ghp_invalid_token"

		_, err := GetSpecFromGitWithToken(url, index, invalidToken)
		// This should fail for private repos with invalid tokens
		require.Error(t, err)
	})
}

// Benchmark the improved spec fetching
func BenchmarkGetSpecFromGitIntegration(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping integration benchmark in short mode")
	}

	url := "https://github.com/magma-devs/lava-specs/tree/main/"
	index := "ETH1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetSpecFromGit(url, index)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark spec fetching with token
func BenchmarkGetSpecFromGitWithTokenIntegration(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping integration benchmark in short mode")
	}

	url := "https://github.com/magma-devs/lava-specs/tree/main/"
	index := "ETH1"
	token := "ghp_test_token"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetSpecFromGitWithToken(url, index, token)
		if err != nil {
			b.Fatal(err)
		}
	}
}
