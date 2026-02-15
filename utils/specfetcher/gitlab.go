package specfetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/x/spec/types"
)

// fetchFromGitLab fetches all specs from a GitLab repository.
func (f *Fetcher) fetchFromGitLab(ctx context.Context, info *RepoInfo) (map[string]types.Spec, error) {
	// Build the API URL for listing directory contents
	apiURL := f.buildGitLabTreeAPIURL(info)

	utils.LavaFormatInfo("Fetching spec file list from GitLab",
		utils.LogAttr("api_url", apiURL))

	if f.config.Token != "" {
		utils.LavaFormatInfo("Using GitLab token authentication",
			utils.LogAttr("token_prefix", f.config.Token[:min(4, len(f.config.Token))]))
	} else {
		utils.LavaFormatInfo("Using unauthenticated GitLab access",
			utils.LogAttr("note", "private repos require authentication"))
	}

	// Fetch the directory listing
	apiCtx, cancel := context.WithTimeout(ctx, f.config.APITimeout)
	defer cancel()

	resp, err := f.doRequest(apiCtx, http.MethodGet, apiURL, f.setGitLabHeaders)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from GitLab API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitLab API error (status: %d, body: %s)", resp.StatusCode, string(body))
	}

	// Parse directory listing
	fileURLs, err := f.parseGitLabDirectoryListing(resp.Body, info)
	if err != nil {
		return nil, err
	}

	if len(fileURLs) == 0 {
		return nil, fmt.Errorf("no .json spec files found in repository")
	}

	utils.LavaFormatInfo("Found spec files to fetch",
		utils.LogAttr("file_count", len(fileURLs)))

	// Fetch all spec files in parallel
	return f.fetchFilesParallel(ctx, fileURLs, f.setGitLabHeaders)
}

// buildGitLabTreeAPIURL constructs the GitLab API URL for listing directory contents.
// Format: /api/v4/projects/{project_path}/repository/tree?ref={branch}&path={path}
func (f *Fetcher) buildGitLabTreeAPIURL(info *RepoInfo) string {
	encodedProject := url.PathEscape(info.ProjectPath)
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/tree?ref=%s",
		info.Host, encodedProject, url.QueryEscape(info.Branch))

	if info.Path != "" {
		apiURL += "&path=" + url.QueryEscape(info.Path)
	}
	return apiURL
}

// buildGitLabFileAPIURL constructs the GitLab API URL for fetching raw file content.
// Format: /api/v4/projects/{project_path}/repository/files/{file_path}/raw?ref={branch}
func (f *Fetcher) buildGitLabFileAPIURL(info *RepoInfo, filePath string) string {
	encodedProject := url.PathEscape(info.ProjectPath)
	encodedFilePath := url.PathEscape(filePath)

	return fmt.Sprintf("%s/api/v4/projects/%s/repository/files/%s/raw?ref=%s",
		info.Host, encodedProject, encodedFilePath, url.QueryEscape(info.Branch))
}

// setGitLabHeaders sets the appropriate headers for GitLab API requests.
func (f *Fetcher) setGitLabHeaders(req *http.Request) {
	if f.config.Token != "" {
		req.Header.Set("PRIVATE-TOKEN", f.config.Token)
	}
}

// gitLabFileEntry represents a file entry in GitLab API response.
type gitLabFileEntry struct {
	Name string `json:"name"`
	Type string `json:"type"` // "blob" for files, "tree" for directories
	Path string `json:"path"` // full path within repository
}

// parseGitLabDirectoryListing parses the GitLab API response and returns URLs for JSON files.
func (f *Fetcher) parseGitLabDirectoryListing(body io.Reader, info *RepoInfo) ([]string, error) {
	var files []gitLabFileEntry
	if err := json.NewDecoder(body).Decode(&files); err != nil {
		return nil, fmt.Errorf("failed to parse GitLab API response: %w", err)
	}

	var urls []string
	for _, file := range files {
		if file.Type == "blob" && strings.HasSuffix(file.Name, ".json") {
			urls = append(urls, f.buildGitLabFileAPIURL(info, file.Path))
		}
	}
	return urls, nil
}
