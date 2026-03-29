package statetracker

import (
	"context"
	"strings"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	updaters "github.com/lavanet/lava/v5/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v5/utils"
	speckeeper "github.com/lavanet/lava/v5/utils/keeper"
	"github.com/lavanet/lava/v5/utils/specfetcher"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
)

const (
	MAINNET_SPEC = "LAVA"
	TESTNET_SPEC = "LAV1"
)

var (
	// LavaSpecOptions lists the chain IDs that identify the Lava native chain.
	LavaSpecOptions = []string{TESTNET_SPEC, MAINNET_SPEC}
)

// SpecUpdaterInf is implemented by state-tracker types that can register a
// SpecUpdatable for live on-chain spec updates.  Callers that do not need live
// updates (e.g. the smart router using static specs) may pass nil.
type SpecUpdaterInf interface {
	RegisterForSpecUpdates(ctx context.Context, specUpdatable updaters.SpecUpdatable, endpoint lavasession.RPCEndpoint) error
}

// expandCommaSeparatedPaths takes a slice of paths (from StringArray flag) and expands
// any comma-separated values within each element. This allows users to specify multiple
// local files either as separate flags or as comma-separated values:
//
//	--use-static-spec file1.json --use-static-spec file2.json
//	--use-static-spec file1.json,file2.json
//	--use-static-spec file1.json,file2.json --use-static-spec file3.json
//
// Note: Comma-separated values are only supported for local files, not for remote URLs.
func expandCommaSeparatedPaths(specPaths []string) []string {
	var expanded []string
	for _, path := range specPaths {
		// Skip empty paths
		if path == "" {
			continue
		}

		// If it's a remote URL, don't split by comma (URLs may contain commas in query params)
		if specfetcher.IsRemoteRepoURL(path) {
			expanded = append(expanded, path)
			continue
		}

		// Split by comma for local paths
		parts := strings.Split(path, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				expanded = append(expanded, trimmed)
			}
		}
	}
	return expanded
}

// RegisterForSpecUpdatesOrSetStaticSpecsWithToken loads specs from multiple sources and aggregates them.
// Sources are processed in order; later sources override earlier ones for the same chain ID.
//
// It supports:
//   - Remote repositories (GitHub/GitLab) - detected automatically by URL pattern
//   - Local directories - all JSON files in the directory are loaded
//   - Local files - single JSON file or comma-separated list of files
//
// Local files can be specified either as separate flags or comma-separated:
//
//	--use-static-spec file1.json --use-static-spec file2.json
//	--use-static-spec file1.json,file2.json
//
// For remote repositories, the appropriate token (githubToken or gitlabToken) is used for authentication.
// When specPaths is empty and specUpdaterInf is non-nil, it falls back to blockchain registration.
func RegisterForSpecUpdatesOrSetStaticSpecsWithToken(ctx context.Context, chainParser chainlib.ChainParser, specPaths []string, rpcEndpoint lavasession.RPCEndpoint, specUpdaterInf SpecUpdaterInf, githubToken string, gitlabToken string) error {
	if len(specPaths) == 0 {
		if specUpdaterInf == nil {
			return utils.LavaFormatError("no spec paths provided and no spec updater registered", nil,
				utils.LogAttr("chain_id", rpcEndpoint.ChainID))
		}
		return specUpdaterInf.RegisterForSpecUpdates(ctx, chainParser, rpcEndpoint)
	}

	// Expand comma-separated paths (for local files)
	expandedPaths := expandCommaSeparatedPaths(specPaths)

	if len(expandedPaths) == 0 {
		if specUpdaterInf == nil {
			return utils.LavaFormatError("no valid spec paths after expansion and no spec updater registered", nil,
				utils.LogAttr("chain_id", rpcEndpoint.ChainID))
		}
		return specUpdaterInf.RegisterForSpecUpdates(ctx, chainParser, rpcEndpoint)
	}

	// Aggregate all specs from all sources
	aggregatedSpecs := make(map[string]spectypes.Spec)

	for i, specPath := range expandedPaths {
		utils.LavaFormatInfo("Loading specs from source",
			utils.LogAttr("source_index", i+1),
			utils.LogAttr("total_sources", len(expandedPaths)),
			utils.LogAttr("source", specPath))

		specs, err := loadAllSpecsFromSource(ctx, specPath, githubToken, gitlabToken)
		if err != nil {
			return utils.LavaFormatError("failed loading specs from source", err,
				utils.LogAttr("source_index", i+1),
				utils.LogAttr("source", specPath))
		}

		// Merge into aggregated specs (later sources override earlier)
		for chainID, spec := range specs {
			if _, exists := aggregatedSpecs[chainID]; exists {
				utils.LavaFormatInfo("Overriding spec from later source",
					utils.LogAttr("chain_id", chainID),
					utils.LogAttr("source", specPath))
			}
			aggregatedSpecs[chainID] = spec
		}
	}

	if len(aggregatedSpecs) == 0 {
		return utils.LavaFormatError("no specs loaded from any source", nil,
			utils.LogAttr("sources", expandedPaths))
	}

	// Expand the requested spec with its dependencies
	spec, err := speckeeper.ExpandSpecWithDependencies(aggregatedSpecs, rpcEndpoint.ChainID)
	if err != nil {
		return utils.LavaFormatError("failed expanding spec", err,
			utils.LogAttr("chain_id", rpcEndpoint.ChainID),
			utils.LogAttr("available_specs", len(aggregatedSpecs)))
	}

	utils.LavaFormatInfo("Successfully loaded and expanded spec from aggregated sources",
		utils.LogAttr("chain_id", spec.Index),
		utils.LogAttr("total_sources", len(specPaths)),
		utils.LogAttr("total_specs_loaded", len(aggregatedSpecs)))

	chainParser.SetSpec(*spec)
	return nil
}

// loadAllSpecsFromSource loads all specs from a single source without expansion.
// Returns a map of specs keyed by their chain ID.
func loadAllSpecsFromSource(ctx context.Context, source, githubToken, gitlabToken string) (map[string]spectypes.Spec, error) {
	// Check if it's a remote repository URL
	if specfetcher.IsRemoteRepoURL(source) {
		return loadAllSpecsFromRemoteRepo(ctx, source, githubToken, gitlabToken)
	}

	// Local file or directory
	return speckeeper.GetAllSpecsFromPath(source)
}

// loadAllSpecsFromRemoteRepo fetches all specs from a GitHub or GitLab repository.
func loadAllSpecsFromRemoteRepo(ctx context.Context, repoURL, githubToken, gitlabToken string) (map[string]spectypes.Spec, error) {
	// Determine which token to use based on the provider
	var token string
	if specfetcher.IsGitHubURL(repoURL) {
		token = githubToken
	} else if specfetcher.IsGitLabURL(repoURL) {
		token = gitlabToken
	}

	specs, err := specfetcher.FetchAllSpecsFromRemote(ctx, repoURL, token)
	if err != nil {
		return nil, utils.LavaFormatError("failed fetching specs from remote repository", err,
			utils.LogAttr("repo_url", repoURL))
	}

	return specs, nil
}

// IsLavaNativeSpec returns true when the given chain ID is the Lava native chain.
func IsLavaNativeSpec(checked string) bool {
	for _, nativeLavaChain := range LavaSpecOptions {
		if checked == nativeLavaChain {
			return true
		}
	}
	return false
}
