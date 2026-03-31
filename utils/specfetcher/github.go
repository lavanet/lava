package specfetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lavanet/lava/v5/utils"
	types "github.com/lavanet/lava/v5/types/spec"
)

// fetchFromGitHub fetches all specs from a GitHub repository.
func (f *Fetcher) fetchFromGitHub(ctx context.Context, info *RepoInfo) (map[string]types.Spec, error) {
	// Build the API URL for listing directory contents
	apiURL := f.buildGitHubAPIURL(info)

	utils.LavaFormatInfo("Fetching spec file list from GitHub",
		utils.LogAttr("api_url", apiURL))

	if f.config.Token != "" {
		utils.LavaFormatInfo("Using GitHub token authentication",
			utils.LogAttr("token_prefix", f.config.Token[:min(4, len(f.config.Token))]))
	} else {
		utils.LavaFormatInfo("Using unauthenticated GitHub access",
			utils.LogAttr("rate_limit", "60 req/hour"))
	}

	// Fetch the directory listing
	apiCtx, cancel := context.WithTimeout(ctx, f.config.APITimeout)
	defer cancel()

	resp, err := f.doRequest(apiCtx, http.MethodGet, apiURL, f.setGitHubHeaders)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error (status: %d, body: %s)", resp.StatusCode, string(body))
	}

	// Parse directory listing
	fileURLs, err := f.parseGitHubDirectoryListing(resp.Body, info)
	if err != nil {
		return nil, err
	}

	if len(fileURLs) == 0 {
		return nil, fmt.Errorf("no .json spec files found in repository")
	}

	utils.LavaFormatInfo("Found spec files to fetch",
		utils.LogAttr("file_count", len(fileURLs)))

	// Fetch all spec files in parallel
	return f.fetchFilesParallel(ctx, fileURLs, f.setGitHubHeaders)
}

// buildGitHubAPIURL constructs the GitHub API URL for listing directory contents.
func (f *Fetcher) buildGitHubAPIURL(info *RepoInfo) string {
	// Split project path into owner/repo
	parts := strings.SplitN(info.ProjectPath, "/", 2)
	owner, repo := parts[0], parts[1]

	return fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		owner, repo, info.Path, info.Branch)
}

// buildGitHubRawURL constructs a raw.githubusercontent.com URL for file content.
func (f *Fetcher) buildGitHubRawURL(info *RepoInfo, filename string) string {
	parts := strings.SplitN(info.ProjectPath, "/", 2)
	owner, repo := parts[0], parts[1]

	if info.Path != "" {
		return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s/%s",
			owner, repo, info.Branch, info.Path, filename)
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s",
		owner, repo, info.Branch, filename)
}

// setGitHubHeaders sets the appropriate headers for GitHub API requests.
func (f *Fetcher) setGitHubHeaders(req *http.Request) {
	if f.config.Token != "" {
		req.Header.Set("Authorization", "token "+f.config.Token)
	}
}

// gitHubFileEntry represents a file entry in GitHub API response.
type gitHubFileEntry struct {
	Name string `json:"name"`
	Type string `json:"type"` // "file" or "dir"
}

// parseGitHubDirectoryListing parses the GitHub API response and returns URLs for JSON files.
func (f *Fetcher) parseGitHubDirectoryListing(body io.Reader, info *RepoInfo) ([]string, error) {
	var files []gitHubFileEntry
	if err := json.NewDecoder(body).Decode(&files); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	var urls []string
	for _, file := range files {
		if file.Type == "file" && strings.HasSuffix(file.Name, ".json") {
			urls = append(urls, f.buildGitHubRawURL(info, file.Name))
		}
	}
	return urls, nil
}
