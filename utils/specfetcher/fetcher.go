// Package specfetcher provides functionality to fetch Lava specs from remote Git repositories.
// It supports both GitHub and GitLab (including self-hosted instances) with optional authentication.
package specfetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/x/spec/types"
)

// Default timeouts and concurrency settings
const (
	DefaultAPITimeout       = 5 * time.Second
	DefaultFileFetchTimeout = 45 * time.Second
	DefaultMaxConcurrency   = 10
)

// ProviderType identifies the Git hosting provider.
type ProviderType string

const (
	ProviderGitHub ProviderType = "github"
	ProviderGitLab ProviderType = "gitlab"
)

// RepoInfo contains parsed information about a Git repository URL.
type RepoInfo struct {
	Provider    ProviderType
	Host        string // e.g., "https://github.com" or "https://gitlab.example.com"
	ProjectPath string // e.g., "owner/repo" or "group/subgroup/repo"
	Branch      string
	Path        string // path within the repository
}

// Config holds configuration for the spec fetcher.
type Config struct {
	// Token for authentication (GitHub PAT or GitLab PAT)
	Token string

	// Timeouts
	APITimeout       time.Duration
	FileFetchTimeout time.Duration

	// MaxConcurrency limits parallel file fetches
	MaxConcurrency int

	// HTTPClient allows custom HTTP client (useful for testing)
	HTTPClient *http.Client
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		APITimeout:       DefaultAPITimeout,
		FileFetchTimeout: DefaultFileFetchTimeout,
		MaxConcurrency:   DefaultMaxConcurrency,
		HTTPClient:       http.DefaultClient,
	}
}

// Fetcher handles fetching specs from remote Git repositories.
type Fetcher struct {
	config Config
}

// New creates a new Fetcher with the given configuration.
func New(config Config) *Fetcher {
	if config.APITimeout == 0 {
		config.APITimeout = DefaultAPITimeout
	}
	if config.FileFetchTimeout == 0 {
		config.FileFetchTimeout = DefaultFileFetchTimeout
	}
	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = DefaultMaxConcurrency
	}
	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}
	return &Fetcher{config: config}
}

// FetchSpec fetches a single spec by chain ID from a remote repository.
func (f *Fetcher) FetchSpec(ctx context.Context, repoURL, chainID string) (types.Spec, error) {
	specs, err := f.FetchAllSpecs(ctx, repoURL)
	if err != nil {
		return types.Spec{}, err
	}

	spec, err := expandSpec(specs, chainID)
	if err != nil {
		return types.Spec{}, err
	}
	return *spec, nil
}

// FetchAllSpecs fetches all specs from a remote repository.
func (f *Fetcher) FetchAllSpecs(ctx context.Context, repoURL string) (map[string]types.Spec, error) {
	info, err := ParseRepoURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository URL: %w", err)
	}

	switch info.Provider {
	case ProviderGitHub:
		return f.fetchFromGitHub(ctx, info)
	case ProviderGitLab:
		return f.fetchFromGitLab(ctx, info)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", info.Provider)
	}
}

// ParseRepoURL parses a GitHub or GitLab URL and extracts repository information.
//
// Supported URL formats:
//   - GitHub: https://github.com/{owner}/{repo}/tree/{branch}/{path}
//   - GitLab: https://gitlab.com/{owner}/{repo}/-/tree/{branch}/{path}
//   - GitLab (self-hosted): https://gitlab.example.com/{group}/{repo}/-/tree/{branch}/{path}
func ParseRepoURL(rawURL string) (*RepoInfo, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	parts := splitPath(parsed.Path)
	host := parsed.Scheme + "://" + parsed.Host

	// Detect provider based on URL structure
	if parsed.Host == "github.com" {
		return parseGitHubURL(host, parts)
	}

	// GitLab URLs contain "/-/" separator
	if containsGitLabSeparator(parts) {
		return parseGitLabURL(host, parts)
	}

	return nil, fmt.Errorf("unrecognized repository URL format: %s", rawURL)
}

// parseGitHubURL parses a GitHub repository URL.
// Expected format: owner/repo/tree/branch/path...
func parseGitHubURL(host string, parts []string) (*RepoInfo, error) {
	if len(parts) < 4 || parts[2] != "tree" {
		return nil, fmt.Errorf("invalid GitHub URL: expected format https://github.com/owner/repo/tree/branch/path")
	}

	return &RepoInfo{
		Provider:    ProviderGitHub,
		Host:        host,
		ProjectPath: parts[0] + "/" + parts[1],
		Branch:      parts[3],
		Path:        strings.Join(parts[4:], "/"),
	}, nil
}

// parseGitLabURL parses a GitLab repository URL.
// Expected format: owner/repo/-/tree/branch/path... or group/subgroup/repo/-/tree/branch/path...
func parseGitLabURL(host string, parts []string) (*RepoInfo, error) {
	// Find the position of "-" which separates project path from tree/branch info
	dashIdx := -1
	for i, part := range parts {
		if part == "-" {
			dashIdx = i
			break
		}
	}

	if dashIdx < 1 || dashIdx+2 >= len(parts) || parts[dashIdx+1] != "tree" {
		return nil, fmt.Errorf("invalid GitLab URL: expected format https://gitlab.com/owner/repo/-/tree/branch/path")
	}

	var path string
	if len(parts) > dashIdx+3 {
		path = strings.Join(parts[dashIdx+3:], "/")
	}

	return &RepoInfo{
		Provider:    ProviderGitLab,
		Host:        host,
		ProjectPath: strings.Join(parts[:dashIdx], "/"),
		Branch:      parts[dashIdx+2],
		Path:        path,
	}, nil
}

// splitPath splits a URL path into non-empty components.
func splitPath(path string) []string {
	var parts []string
	for _, part := range strings.Split(path, "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

// containsGitLabSeparator checks if the path contains the GitLab "/-/" separator.
func containsGitLabSeparator(parts []string) bool {
	for _, part := range parts {
		if part == "-" {
			return true
		}
	}
	return false
}

// fetchResult holds the result of fetching a single spec file.
type fetchResult struct {
	specs  map[string]types.Spec
	errors []string
}

// doRequest performs an HTTP request with the configured client and timeout.
func (f *Fetcher) doRequest(ctx context.Context, method, url string, setHeaders func(*http.Request)) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if setHeaders != nil {
		setHeaders(req)
	}

	return f.config.HTTPClient.Do(req)
}

// fetchFilesParallel fetches multiple files in parallel and parses them as specs.
func (f *Fetcher) fetchFilesParallel(ctx context.Context, fileURLs []string, setHeaders func(*http.Request)) (map[string]types.Spec, error) {
	resultChan := make(chan fetchResult, len(fileURLs))
	semaphore := make(chan struct{}, f.config.MaxConcurrency)

	for _, fileURL := range fileURLs {
		go func(url string) {
			semaphore <- struct{}{}        // acquire
			defer func() { <-semaphore }() // release

			result := fetchResult{specs: make(map[string]types.Spec)}

			fetchCtx, cancel := context.WithTimeout(ctx, f.config.FileFetchTimeout)
			defer cancel()

			resp, err := f.doRequest(fetchCtx, http.MethodGet, url, setHeaders)
			if err != nil {
				result.errors = append(result.errors, fmt.Sprintf("%s: %v", url, err))
				resultChan <- result
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				result.errors = append(result.errors, fmt.Sprintf("%s: HTTP %d", url, resp.StatusCode))
				resultChan <- result
				return
			}

			content, err := io.ReadAll(resp.Body)
			if err != nil {
				result.errors = append(result.errors, fmt.Sprintf("%s: failed to read body: %v", url, err))
				resultChan <- result
				return
			}

			var proposal types.SpecAddProposalJSON
			if err := json.Unmarshal(content, &proposal); err != nil {
				result.errors = append(result.errors, fmt.Sprintf("%s: failed to parse JSON: %v", url, err))
				resultChan <- result
				return
			}

			for _, spec := range proposal.Proposal.Specs {
				result.specs[spec.Index] = spec
			}
			resultChan <- result
		}(fileURL)
	}

	// Collect results
	specs := make(map[string]types.Spec)
	var fetchErrors []string

	for i := 0; i < len(fileURLs); i++ {
		result := <-resultChan
		for k, v := range result.specs {
			specs[k] = v
		}
		fetchErrors = append(fetchErrors, result.errors...)
	}

	if len(specs) == 0 {
		if len(fetchErrors) > 0 {
			return nil, fmt.Errorf("failed to fetch specs: %s", strings.Join(fetchErrors, "; "))
		}
		return nil, fmt.Errorf("no specs found")
	}

	// Log any fetch errors
	if len(fetchErrors) > 0 {
		utils.LavaFormatWarning("Some spec files failed to fetch", nil,
			utils.LogAttr("error_count", len(fetchErrors)),
			utils.LogAttr("errors", strings.Join(fetchErrors, "; ")))
	}

	// Log loaded specs
	specIDs := make([]string, 0, len(specs))
	for id := range specs {
		specIDs = append(specIDs, id)
	}
	utils.LavaFormatInfo("Loaded specs from remote repository",
		utils.LogAttr("spec_count", len(specs)),
		utils.LogAttr("spec_ids", strings.Join(specIDs, ", ")))

	return specs, nil
}
