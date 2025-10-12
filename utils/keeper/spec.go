package keeper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tmdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
	utils "github.com/lavanet/lava/v5/utils"
	specutils "github.com/lavanet/lava/v5/x/spec/client/utils"
	"github.com/lavanet/lava/v5/x/spec/keeper"
	"github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

const (
	// GitHub API request timeout
	githubAPIRequestTimeout = 5 * time.Second
	// Individual spec file fetch timeout when fetching in parallel
	specFileFetchTimeout = 45 * time.Second
	// Maximum number of concurrent workers for parallel spec file fetching
	maxConcurrentSpecFetches = 10
)

func SpecKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
	keep, ctx, err := specKeeper()
	require.NoError(t, err)
	return keep, ctx
}

func specKeeper() (*keeper.Keeper, sdk.Context, error) {
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	err := stateStore.LoadLatestVersion()
	if err != nil {
		return nil, sdk.Context{}, err
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	paramsSubspace := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"SpecParams",
	)
	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		paramsSubspace,
		nil,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx, nil
}

func decodeProposal(path string) (specutils.SpecAddProposalJSON, error) {
	proposal := specutils.SpecAddProposalJSON{}
	contents, err := os.ReadFile(path)
	if err != nil {
		return proposal, err
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields() // This will make the unmarshal fail if there are unused fields

	err = decoder.Decode(&proposal)
	return proposal, err
}

func GetSpecsFromPath(path string, specIndex string, ctxArg *sdk.Context, keeper *keeper.Keeper) (specRet types.Spec, err error) {
	var ctx sdk.Context
	if keeper == nil || ctxArg == nil {
		keeper, ctx, err = specKeeper()
		if err != nil {
			return types.Spec{}, err
		}
	} else {
		ctx = *ctxArg
	}

	// Split the string by "," if we have a spec with dependencies we need to first load the dependencies.. for example:
	// ibc.json, cosmossdk.json, lava.json.
	files := strings.Split(path, ",")
	for _, fileName := range files {
		trimmedFileName := strings.TrimSpace(fileName)
		spec, err := GetSpecFromPath(trimmedFileName, specIndex, &ctx, keeper)
		if err == nil {
			return spec, nil
		}
	}
	return types.Spec{}, fmt.Errorf("spec not found %s", specIndex)
}

func GetSpecFromPath(path string, specIndex string, ctxArg *sdk.Context, keeper *keeper.Keeper) (specRet types.Spec, err error) {
	var ctx sdk.Context
	if keeper == nil || ctxArg == nil {
		keeper, ctx, err = specKeeper()
		if err != nil {
			return types.Spec{}, err
		}
	} else {
		ctx = *ctxArg
	}

	proposal, err := decodeProposal(path)
	if err != nil {
		return types.Spec{}, err
	}

	for _, spec := range proposal.Proposal.Specs {
		keeper.SetSpec(ctx, spec)
		if specIndex != spec.Index {
			continue
		}
		fullspec, err := keeper.ExpandSpec(ctx, spec)
		if err != nil {
			return types.Spec{}, err
		}
		return fullspec, nil
	}
	return types.Spec{}, fmt.Errorf("spec not found %s", path)
}

func GetASpec(specIndex, getToTopMostPath string, ctxArg *sdk.Context, keeper *keeper.Keeper) (specRet types.Spec, err error) {
	var ctx sdk.Context
	if keeper == nil || ctxArg == nil {
		keeper, ctx, err = specKeeper()
		if err != nil {
			return types.Spec{}, err
		}
	} else {
		ctx = *ctxArg
	}

	proposalDirectories := []string{
		"specs/mainnet-1/specs/",
		"specs/testnet-2/specs/",
	}
	baseProposalFiles := []string{
		"ibc.json", "cosmoswasm.json", "tendermint.json", "cosmossdk.json",
		"cosmossdkv45.json", "cosmossdkv50.json", "ethereum.json", "ethermint.json", "solana.json",
	}

	// Create a map of base files for quick lookup
	baseFiles := make(map[string]struct{})
	for _, f := range baseProposalFiles {
		baseFiles[f] = struct{}{}
	}

	// Try each directory
	for _, proposalDirectory := range proposalDirectories {
		// Try base proposal files first
		for _, fileName := range baseProposalFiles {
			spec, err := GetSpecFromPath(getToTopMostPath+proposalDirectory+fileName, specIndex, &ctx, keeper)
			if err == nil {
				return spec, nil
			}
		}

		// Read all files from the proposal directory
		files, err := os.ReadDir(getToTopMostPath + proposalDirectory)
		if err != nil {
			continue // Skip to next directory if this one fails
		}

		// Try additional JSON files that aren't in baseProposalFiles
		for _, file := range files {
			fileName := file.Name()
			// Skip if not a JSON file or if it's in baseProposalFiles
			if !strings.HasSuffix(fileName, ".json") {
				continue
			}
			if _, exists := baseFiles[fileName]; exists {
				continue
			}

			spec, err := GetSpecFromPath(getToTopMostPath+proposalDirectory+fileName, specIndex, &ctx, keeper)
			if err == nil {
				return spec, nil
			}
		}
	}

	return types.Spec{}, fmt.Errorf("spec not found %s", specIndex)
}

func GetSpecFromGit(url string, index string) (types.Spec, error) {
	return GetSpecFromGitWithToken(url, index, "")
}

func GetSpecFromGitWithToken(url string, index string, githubToken string) (types.Spec, error) {
	specs, err := getAllSpecsWithToken(url, githubToken)
	if err != nil {
		return types.Spec{}, err
	}
	spec, err := expandSpec(specs, index)
	if err != nil {
		return types.Spec{}, err
	}
	return *spec, nil
}

func GetSpecFromLocalDir(specPath string, index string) (types.Spec, error) {
	specs := map[string]types.Spec{}
	var errs []error

	// Walk through all files and subdirectories in the specPath
	err := filepath.WalkDir(specPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			errs = append(errs, fmt.Errorf("error accessing path %s: %w", path, err))
			return nil // Continue walking, but record the error
		}

		if d.IsDir() {
			return nil // Skip directories
		}

		// Attempt to decode the proposal from the file
		proposal, err := decodeProposal(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("error decoding proposal from %s: %w", path, err))
			return nil // Continue walking, but record the error
		}

		// Extract specs from the proposal and add them to the map
		for _, spec := range proposal.Proposal.Specs {
			specs[spec.Index] = spec
		}
		return nil
	})
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return types.Spec{}, fmt.Errorf("multiple errors occurred: %v", errs)
	}

	// Log loaded specs for debugging
	if len(specs) > 0 {
		specIDs := make([]string, 0, len(specs))
		for id := range specs {
			specIDs = append(specIDs, id)
		}
		utils.LavaFormatInfo("Loaded specs from local directory",
			utils.LogAttr("spec_count", len(specs)),
			utils.LogAttr("directory", specPath),
			utils.LogAttr("spec_ids", strings.Join(specIDs, ", ")))
	}

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

	// Validate that this is a GitHub URL
	if parsedURL.Host != "github.com" {
		return "", fmt.Errorf("invalid GitHub folder URL")
	}

	parts := strings.Split(parsedURL.Path, "/")

	// Remove empty parts from the beginning
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}

	// Expected format: owner/repo/tree/branch/path...
	if len(cleanParts) < 4 || cleanParts[2] != "tree" {
		return "", fmt.Errorf("invalid GitHub folder URL")
	}

	owner := cleanParts[0]
	repo := cleanParts[1]
	branch := cleanParts[3]

	var path string
	if len(cleanParts) > 4 {
		path = strings.Join(cleanParts[4:], "/")
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, branch)
	return apiURL, nil
}

func convertGitHubURLToRaw(input string) (string, error) {
	parsedURL, err := url.Parse(input)
	if err != nil {
		return "", err
	}

	// Validate that this is a GitHub URL
	if parsedURL.Host != "github.com" {
		return "", fmt.Errorf("invalid GitHub folder URL")
	}

	parts := strings.Split(parsedURL.Path, "/")

	// Remove empty parts from the beginning
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}

	// Expected format: owner/repo/tree/branch/path...
	if len(cleanParts) < 4 || cleanParts[2] != "tree" {
		return "", fmt.Errorf("invalid GitHub folder URL")
	}

	owner := cleanParts[0]
	repo := cleanParts[1]
	branch := cleanParts[3]

	var path string
	if len(cleanParts) > 4 {
		path = strings.Join(cleanParts[4:], "/")
	}

	// Use raw.githubusercontent.com instead of GitHub API
	// Don't add trailing slash if path is empty to avoid double slashes when appending filenames
	if path != "" {
		rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, branch, path)
		return rawURL, nil
	}
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", owner, repo, branch)
	return rawURL, nil
}

func getAllSpecsWithToken(url string, githubToken string) (map[string]types.Spec, error) {
	// First, get the list of files using GitHub API (only 1 API call)
	githubAPIURL, err := convertGitHubURLToAPI(url)
	if err != nil {
		return nil, fmt.Errorf("failed to convert GitHub URL to API format: %w", err)
	}

	utils.LavaFormatInfo("Fetching spec file list from GitHub",
		utils.LogAttr("url", githubAPIURL))
	if githubToken != "" {
		utils.LavaFormatInfo("Using GitHub token authentication",
			utils.LogAttr("token_prefix", githubToken[:4]))
	} else {
		utils.LavaFormatInfo("Using unauthenticated GitHub access",
			utils.LogAttr("rate_limit", "60 req/hour"))
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), githubAPIRequestTimeout)
	defer cancel()

	// Create a new request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add GitHub token authentication if provided
	if githubToken != "" {
		req.Header.Set("Authorization", "token "+githubToken)
	}

	// Execute the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from GitHub API: %w (check network connectivity)", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned error (status: %d, url: %s, body: %s)", resp.StatusCode, githubAPIURL, string(body))
	}

	// Parse the GitHub API response
	var files []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	err = json.NewDecoder(resp.Body).Decode(&files)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	// Filter for .json files and convert to raw URLs
	var specFiles []string
	rawBaseURL, err := convertGitHubURLToRaw(url)
	if err != nil {
		return nil, fmt.Errorf("failed to convert GitHub URL to raw format: %w", err)
	}

	for _, file := range files {
		if file.Type == "file" && strings.HasSuffix(file.Name, ".json") {
			specFiles = append(specFiles, rawBaseURL+"/"+file.Name)
		}
	}
	if len(specFiles) == 0 {
		return nil, fmt.Errorf("no .json spec files found in repository")
	}

	utils.LavaFormatInfo("Found spec files to fetch",
		utils.LogAttr("file_count", len(specFiles)))

	// Fetch files in parallel with worker pool (10 concurrent workers)
	type fetchResult struct {
		specs  map[string]types.Spec
		errors []string
	}

	resultChan := make(chan fetchResult, len(specFiles))
	workerSemaphore := make(chan struct{}, maxConcurrentSpecFetches) // Limit concurrent requests

	for _, specFile := range specFiles {
		go func(url string) {
			workerSemaphore <- struct{}{}        // Acquire
			defer func() { <-workerSemaphore }() // Release

			result := fetchResult{specs: map[string]types.Spec{}}

			var content []byte

			// Single attempt with longer timeout (parallel fetching is faster overall)
			ctx, cancel := context.WithTimeout(context.Background(), specFileFetchTimeout)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				result.errors = append(result.errors, fmt.Sprintf("%s: failed to create request: %v", url, err))
				resultChan <- result
				return
			}

			// Add GitHub token authentication for raw URLs if provided
			if githubToken != "" {
				req.Header.Set("Authorization", "token "+githubToken)
			}

			resp, err := http.DefaultClient.Do(req)
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

			content, err = io.ReadAll(resp.Body)
			if err != nil {
				result.errors = append(result.errors, fmt.Sprintf("%s: failed to read body: %v", url, err))
				resultChan <- result
				return
			}

			// Parse the JSON content into a spec proposal
			var proposal specutils.SpecAddProposalJSON
			err = json.Unmarshal(content, &proposal)
			if err != nil {
				result.errors = append(result.errors, fmt.Sprintf("%s: failed to parse JSON: %v", url, err))
				resultChan <- result
				return
			}

			for _, spec := range proposal.Proposal.Specs {
				result.specs[spec.Index] = spec
			}

			resultChan <- result
		}(specFile)
	}

	// Collect results
	specs := map[string]types.Spec{}
	var fetchErrors []string

	for i := 0; i < len(specFiles); i++ {
		result := <-resultChan
		for k, v := range result.specs {
			specs[k] = v
		}
		fetchErrors = append(fetchErrors, result.errors...)
	}

	if len(specs) == 0 {
		return nil, fmt.Errorf("no specs found")
	}

	// Log any fetch errors
	if len(fetchErrors) > 0 {
		utils.LavaFormatWarning("Errors occurred while fetching spec files", nil,
			utils.LogAttr("error_count", len(fetchErrors)),
			utils.LogAttr("errors", strings.Join(fetchErrors, "; ")))
	}

	// Log loaded specs for debugging
	if len(specs) > 0 {
		specIDs := make([]string, 0, len(specs))
		for id := range specs {
			specIDs = append(specIDs, id)
		}
		utils.LavaFormatInfo("Loaded specs from GitHub",
			utils.LogAttr("spec_count", len(specs)),
			utils.LogAttr("spec_ids", strings.Join(specIDs, ", ")))
	}

	return specs, nil
}

func expandSpec(specs map[string]types.Spec, index string) (*types.Spec, error) {
	getBaseSpec := func(ctx sdk.Context, index string) (types.Spec, bool) {
		spec, ok := specs[index]
		return spec, ok
	}
	spec, ok := specs[index]
	if !ok {
		// List available specs for better debugging
		availableSpecs := make([]string, 0, len(specs))
		for id := range specs {
			availableSpecs = append(availableSpecs, id)
		}
		return nil, fmt.Errorf("spec not found for chainId: %s (available specs: %v)", index, availableSpecs)
	}
	depends := map[string]bool{index: true}
	inherit := map[string]bool{}

	ctx := sdk.Context{}
	details, err := types.DoExpandSpec(ctx, &spec, depends, &inherit, spec.Index, getBaseSpec)
	_ = details
	if err != nil {
		return nil, fmt.Errorf("spec expand failed: %w", err)
	}
	return &spec, nil
}
