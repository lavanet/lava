package updaters

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/utils"
	utilsspec "github.com/lavanet/lava/v5/x/spec/client/utils"
	"github.com/lavanet/lava/v5/x/spec/types"
)

func GetSpecFromGit(url string, index string) (types.Spec, error) {
	specs := getAllSpecs(url)
	spec, err := expandSpec(specs, index)
	if err != nil {
		return types.Spec{}, err
	}
	return *spec, nil
}

func convertGitHubURLToAPI(input string) (string, error) {
	parsedURL, err := url.Parse(input)
	if err != nil {
		return "", err
	}

	parts := strings.Split(parsedURL.Path, "/")

	if parts[3] != "tree" {
		return "", fmt.Errorf("invalid GitHub folder URL")
	}

	owner := parts[1]
	repo := parts[2]
	branch := parts[4]
	path := strings.Join(parts[5:], "/")

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, branch)
	return apiURL, nil
}

func getAllSpecs(url string) map[string]types.Spec {
	// githubAPIURL := "https://api.github.com/repos/lavanet/lava/contents/specs/mainnet-1/specs"
	githubAPIURL, err := convertGitHubURLToAPI(url)
	_ = err
	// githubAPIURL := "https://api.github.com/repos" + strings.TrimPrefix(url, "https://github.com") + "/contents/"

	// Fetch directory listing from GitHub API
	resp, err := http.Get(githubAPIURL)
	_ = err
	// require.NoError(t, err)
	defer resp.Body.Close()
	// require.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse the GitHub API response
	var files []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Url  string `json:"download_url"`
	}
	err = json.NewDecoder(resp.Body).Decode(&files)
	_ = err
	// require.NoError(t, err)

	// Filter for .json files
	var specFiles []string
	for _, file := range files {
		if file.Type == "file" && strings.HasSuffix(file.Name, ".json") {
			specFiles = append(specFiles, file.Url)
		}
	}
	// require.NotEmpty(t, specFiles, "No spec files found")

	specs := map[string]types.Spec{}

	// Test reading each spec file
	for _, specFile := range specFiles {

		// Fetch the file content directly from GitHub
		resp, err := http.Get(specFile)
		_ = err
		// require.NoError(t, err)

		defer resp.Body.Close()

		// require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to fetch %s", specFile)

		content, err := io.ReadAll(resp.Body)
		// require.NoError(t, err)

		// Parse the JSON content into a spec proposal
		var proposal utilsspec.SpecAddProposalJSON
		err = json.Unmarshal(content, &proposal)
		// require.NoError(t, err)

		for _, spec := range proposal.Proposal.Specs {
			specs[spec.Index] = spec
		}
	}
	return specs
}

func expandSpec(specs map[string]types.Spec, index string) (*types.Spec, error) {
	getBaseSpec := func(ctx sdk.Context, index string) (types.Spec, bool) {
		spec, ok := specs[index]
		return spec, ok
	}
	spec, ok := specs[index]
	if !ok {
		return nil, fmt.Errorf("spec not found for chainId: %s", index)
	}
	depends := map[string]bool{index: true}
	inherit := map[string]bool{}

	ctx := sdk.Context{}
	details, err := types.DoExpandSpec(ctx, &spec, depends, &inherit, spec.Index, getBaseSpec)
	_ = details
	if err != nil {
		return nil, utils.LavaFormatError("spec expand failed", err,
			utils.Attribute{Key: "imports", Value: details},
		)
	}
	return &spec, nil
}
