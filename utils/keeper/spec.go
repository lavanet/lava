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
	"github.com/lavanet/lava/v5/x/spec/client/utils"
	"github.com/lavanet/lava/v5/x/spec/keeper"
	"github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
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

func decodeProposal(path string) (utils.SpecAddProposalJSON, error) {
	proposal := utils.SpecAddProposalJSON{}
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
	specs, err := getAllSpecs(url)
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

	if parts[3] != "tree" || len(parts) <= 5 {
		return "", fmt.Errorf("invalid GitHub folder URL")
	}

	owner := parts[1]
	repo := parts[2]
	branch := parts[4]
	path := strings.Join(parts[5:], "/")

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, branch)
	return apiURL, nil
}

func getAllSpecs(url string) (map[string]types.Spec, error) {
	githubAPIURL, err := convertGitHubURLToAPI(url)
	if err != nil {
		return nil, err
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a new request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPIURL, nil)
	if err != nil {
		return nil, err
	}

	// Execute the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s", githubAPIURL)
	}

	// Parse the GitHub API response
	var files []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Url  string `json:"download_url"`
	}
	err = json.NewDecoder(resp.Body).Decode(&files)
	if err != nil {
		return nil, err
	}

	// Filter for .json files
	var specFiles []string
	for _, file := range files {
		if file.Type == "file" && strings.HasSuffix(file.Name, ".json") {
			specFiles = append(specFiles, file.Url)
		}
	}
	if len(specFiles) == 0 {
		return nil, fmt.Errorf("no spec files found")
	}

	specs := map[string]types.Spec{}

	// Test reading each spec file
	for _, specFile := range specFiles {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, specFile, nil)
		if err != nil {
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch %s", specFile)
		}

		content, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		// Parse the JSON content into a spec proposal
		var proposal utils.SpecAddProposalJSON
		err = json.Unmarshal(content, &proposal)
		if err != nil {
			return nil, err
		}

		for _, spec := range proposal.Proposal.Specs {
			specs[spec.Index] = spec
		}
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
		return nil, fmt.Errorf("spec not found for chainId: %s", index)
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
