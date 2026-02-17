package statetracker

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chaintracker"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	updaters "github.com/lavanet/lava/v5/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v5/utils"
	speckeeper "github.com/lavanet/lava/v5/utils/keeper"
	"github.com/lavanet/lava/v5/utils/specfetcher"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
)

const (
	BlocksToSaveLavaChainTracker   = 1 // we only need the latest block
	TendermintConsensusParamsQuery = "consensus_params"
	MAINNET_SPEC                   = "LAVA"
	TESTNET_SPEC                   = "LAV1"
)

var (
	lavaSpecName = ""
	// TODO: add a governance param change that indicates what spec id belongs to lava.
	LavaSpecOptions = []string{TESTNET_SPEC, MAINNET_SPEC}
)

type IStateTracker interface {
	LatestBlock() int64
	GetAverageBlockTime() time.Duration
	RegisterForUpdates(ctx context.Context, updater Updater) Updater
	GetEventTracker() *updaters.EventTracker
}

// ConsumerStateTracker CSTis a class for tracking consumer data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type StateTracker struct {
	chainTracker         chaintracker.IChainTracker
	registrationLock     sync.RWMutex
	newLavaBlockUpdaters map[string]Updater
	EventTracker         *updaters.EventTracker
	AverageBlockTime     time.Duration
}

type Updater interface {
	Update(int64)
	Reset(int64)
	UpdaterKey() string
}

type SpecUpdaterInf interface {
	RegisterForSpecUpdates(ctx context.Context, specUpdatable updaters.SpecUpdatable, endpoint lavasession.RPCEndpoint) error
}

// Either register for spec updates or set spec for offline spec, used in both consumer and provider process
// Deprecated: Use RegisterForSpecUpdatesOrSetStaticSpecsWithToken for multi-source support.
func RegisterForSpecUpdatesOrSetStaticSpec(ctx context.Context, chainParser chainlib.ChainParser, specPath string, rpcEndpoint lavasession.RPCEndpoint, specUpdaterInf SpecUpdaterInf) error {
	if specPath == "" {
		return specUpdaterInf.RegisterForSpecUpdates(ctx, chainParser, rpcEndpoint)
	}
	return RegisterForSpecUpdatesOrSetStaticSpecsWithToken(ctx, chainParser, []string{specPath}, rpcEndpoint, specUpdaterInf, "", "")
}

// RegisterForSpecUpdatesOrSetStaticSpecWithToken loads specs from a single source.
// Deprecated: Use RegisterForSpecUpdatesOrSetStaticSpecsWithToken for multi-source support.
func RegisterForSpecUpdatesOrSetStaticSpecWithToken(ctx context.Context, chainParser chainlib.ChainParser, specPath string, rpcEndpoint lavasession.RPCEndpoint, specUpdaterInf SpecUpdaterInf, githubToken string, gitlabToken string) error {
	if specPath == "" {
		return specUpdaterInf.RegisterForSpecUpdates(ctx, chainParser, rpcEndpoint)
	}
	return RegisterForSpecUpdatesOrSetStaticSpecsWithToken(ctx, chainParser, []string{specPath}, rpcEndpoint, specUpdaterInf, githubToken, gitlabToken)
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
func RegisterForSpecUpdatesOrSetStaticSpecsWithToken(ctx context.Context, chainParser chainlib.ChainParser, specPaths []string, rpcEndpoint lavasession.RPCEndpoint, specUpdaterInf SpecUpdaterInf, githubToken string, gitlabToken string) error {
	if len(specPaths) == 0 {
		return specUpdaterInf.RegisterForSpecUpdates(ctx, chainParser, rpcEndpoint)
	}

	// Expand comma-separated paths (for local files)
	expandedPaths := expandCommaSeparatedPaths(specPaths)

	if len(expandedPaths) == 0 {
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

func GetLavaSpecWithRetry(ctx context.Context, specQueryClient spectypes.QueryClient) (*spectypes.QueryGetSpecResponse, error) {
	var specResponse *spectypes.QueryGetSpecResponse
	var err error
	for i := 0; i < updaters.BlockResultRetry; i++ {
		if lavaSpecName == "" { // spec name is not initialized, try fetching specs.
			for _, specId := range LavaSpecOptions {
				specResponse, err = specQueryClient.Spec(ctx, &spectypes.QueryGetSpecRequest{
					ChainID: specId,
				})
				if err != nil {
					continue
				}
				utils.LavaFormatInfo("Lava Spec found on chain", utils.LogAttr("SpecId", specId))
				lavaSpecName = specId
				break
			}
		} else {
			specResponse, err = specQueryClient.Spec(ctx, &spectypes.QueryGetSpecRequest{
				ChainID: lavaSpecName,
			})
		}
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	return specResponse, err
}

func NewStateTracker(ctx context.Context, txFactory tx.Factory, stateQuery *updaters.StateQuery, chainFetcher chaintracker.ChainFetcher, blockNotFoundCallback func(latestBlockTime time.Time)) (ret *StateTracker, err error) {
	// validate chainId
	status, err := stateQuery.Status(ctx)
	if err != nil {
		return nil, utils.LavaFormatError("failed getting status", err)
	}
	if txFactory.ChainID() != status.NodeInfo.Network {
		return nil, utils.LavaFormatError("Chain ID mismatch", nil, utils.Attribute{Key: "--chain-id", Value: txFactory.ChainID()}, utils.Attribute{Key: "Node chainID", Value: status.NodeInfo.Network})
	}

	eventTracker := &updaters.EventTracker{StateQuery: stateQuery}
	for i := 0; i < updaters.BlockResultRetry; i++ {
		err = eventTracker.UpdateBlockResults(0)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond * time.Duration(i+1)) // need this so it doesn't just spam the attempts, and tendermint fails getting block results pretty often
	}
	if err != nil {
		return nil, utils.LavaFormatError("failed getting blockResults after retries", err)
	}
	specQueryClient := stateQuery.GetSpecQueryClient()
	specResponse, err := GetLavaSpecWithRetry(ctx, specQueryClient)
	if err != nil {
		utils.LavaFormatFatal("failed querying lava spec for state tracker", err)
	}
	cst := &StateTracker{newLavaBlockUpdaters: map[string]Updater{}, EventTracker: eventTracker}
	chainTrackerConfig := chaintracker.ChainTrackerConfig{
		NewLatestCallback:     cst.newLavaBlock,
		OldBlockCallback:      blockNotFoundCallback,
		BlocksToSave:          BlocksToSaveLavaChainTracker,
		AverageBlockTime:      time.Duration(specResponse.Spec.AverageBlockTime) * time.Millisecond,
		ServerBlockMemory:     25 + BlocksToSaveLavaChainTracker,
		PollingTimeMultiplier: chaintracker.LavaPollingMultiplierFrequency,
		ParseDirectiveEnabled: true,
	}
	cst.AverageBlockTime = chainTrackerConfig.AverageBlockTime
	cst.chainTracker, err = chaintracker.NewChainTracker(ctx, chainFetcher, chainTrackerConfig)
	cst.chainTracker.StartAndServe(ctx)
	cst.chainTracker.RegisterForBlockTimeUpdates(cst) // registering for block time updates.
	return cst, err
}

func (st *StateTracker) UpdateBlockTime(blockTime time.Duration) {
	st.registrationLock.Lock()
	defer st.registrationLock.Unlock()
	st.AverageBlockTime = blockTime
}

func (st *StateTracker) GetAverageBlockTime() time.Duration {
	st.registrationLock.RLock()
	defer st.registrationLock.RUnlock()
	return st.AverageBlockTime
}

func (st *StateTracker) newLavaBlock(blockFrom int64, blockTo int64, hash string) {
	// go over the registered updaters and trigger update
	st.registrationLock.RLock()
	defer st.registrationLock.RUnlock()
	// if we had a huge gap
	if time.Duration(blockTo-blockFrom)*st.AverageBlockTime > time.Hour { // if we are 1H behind
		// in case we have a huge gap we launch a reset on the state of all the updaters. as the state is no longer valid.
		// this can be caused by a huge catch up on blocks after a halt or a sync state on the node. sometimes pruning the blocks the protocol requires.
		// therefore we need to reset the state and fetch all information from the chain
		// first update the event tracker to latest block.
		err := st.EventTracker.UpdateBlockResults(blockTo)
		if err != nil {
			utils.LavaFormatError("failing to fetch latest result after gap", err, utils.LogAttr("blockFrom", blockFrom), utils.LogAttr("blockTo", blockTo))
		}
		// reset will try to reset the updaters. if it fails it will retry every update until it succeeds.
		for _, updater := range st.newLavaBlockUpdaters {
			updater.Reset(blockTo)
		}
		return // return after state has been reset.
	}

	for block := blockFrom + 1; block <= blockTo; block++ {
		// first update event tracker
		err := st.EventTracker.UpdateBlockResults(block)
		if err != nil {
			utils.LavaFormatWarning("calling update without updated events tracker", err)
		}
		// after events were updated we can trigger updaters
		for _, updater := range st.newLavaBlockUpdaters {
			updater.Update(block)
		}
	}
}

func (st *StateTracker) RegisterForUpdates(ctx context.Context, updater Updater) Updater {
	st.registrationLock.Lock()
	defer st.registrationLock.Unlock()
	existingUpdater, ok := st.newLavaBlockUpdaters[updater.UpdaterKey()]
	if !ok {
		st.newLavaBlockUpdaters[updater.UpdaterKey()] = updater
		existingUpdater = updater
	}
	return existingUpdater
}

// For lavavisor access
func (st *StateTracker) GetEventTracker() *updaters.EventTracker {
	return st.EventTracker
}

func IsLavaNativeSpec(checked string) bool {
	for _, nativeLavaChain := range LavaSpecOptions {
		if checked == nativeLavaChain {
			return true
		}
	}
	return false
}

func (st *StateTracker) LatestBlock() int64 {
	return st.chainTracker.GetAtomicLatestBlockNum()
}
