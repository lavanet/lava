package specfetcher

import (
	"context"

	"github.com/lavanet/lava/v5/x/spec/types"
)

// FetchSpecFromGitHub fetches a spec from a GitHub repository.
// This is a convenience function that creates a new Fetcher with the provided token.
//
// URL format: https://github.com/{owner}/{repo}/tree/{branch}/{path}
// Example: https://github.com/lavanet/lava/tree/main/specs/mainnet-1/specs
func FetchSpecFromGitHub(ctx context.Context, repoURL, chainID, token string) (types.Spec, error) {
	config := DefaultConfig()
	config.Token = token
	fetcher := New(config)
	return fetcher.FetchSpec(ctx, repoURL, chainID)
}

// FetchSpecFromGitLab fetches a spec from a GitLab repository.
// This is a convenience function that creates a new Fetcher with the provided token.
//
// URL format: https://gitlab.com/{owner}/{repo}/-/tree/{branch}/{path}
// Example: https://gitlab.com/myorg/specs/-/tree/main/specs
//
// Note: For private repositories, the token must have at least "Reporter" role
// with "read_repository" scope.
func FetchSpecFromGitLab(ctx context.Context, repoURL, chainID, token string) (types.Spec, error) {
	config := DefaultConfig()
	config.Token = token
	fetcher := New(config)
	return fetcher.FetchSpec(ctx, repoURL, chainID)
}

// FetchSpec automatically detects the provider (GitHub or GitLab) and fetches the spec.
// Use this when you want automatic provider detection based on the URL structure.
func FetchSpec(ctx context.Context, repoURL, chainID, token string) (types.Spec, error) {
	config := DefaultConfig()
	config.Token = token
	fetcher := New(config)
	return fetcher.FetchSpec(ctx, repoURL, chainID)
}

// FetchAllSpecsFromRemote fetches all specs from a remote repository without expansion.
// This is useful for aggregating specs from multiple sources before expanding.
// The returned map contains unexpanded specs keyed by their chain ID (Index).
func FetchAllSpecsFromRemote(ctx context.Context, repoURL, token string) (map[string]types.Spec, error) {
	config := DefaultConfig()
	config.Token = token
	fetcher := New(config)
	return fetcher.FetchAllSpecs(ctx, repoURL)
}

// IsGitHubURL returns true if the URL is a GitHub repository URL.
func IsGitHubURL(rawURL string) bool {
	info, err := ParseRepoURL(rawURL)
	if err != nil {
		return false
	}
	return info.Provider == ProviderGitHub
}

// IsGitLabURL returns true if the URL is a GitLab repository URL.
func IsGitLabURL(rawURL string) bool {
	info, err := ParseRepoURL(rawURL)
	if err != nil {
		return false
	}
	return info.Provider == ProviderGitLab
}

// IsRemoteRepoURL returns true if the URL is a supported remote repository URL.
func IsRemoteRepoURL(rawURL string) bool {
	_, err := ParseRepoURL(rawURL)
	return err == nil
}
