package rpcsmartrouter

import (
	"context"
	"sync"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chaintracker"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/utils"
)

const (
	// DefaultBlocksToSave is the number of finalized blocks to keep in memory for fork detection
	DefaultBlocksToSave = 10

	// MinPollingInterval prevents too aggressive polling for fast chains
	MinPollingInterval = 100 * time.Millisecond
)

// EndpointChainTrackerManager manages per-endpoint ChainTrackers for the Smart Router.
// Each endpoint gets its own ChainTracker that continuously polls for block data,
// enabling accurate pre-request consistency validation and sync scoring.
type EndpointChainTrackerManager struct {
	mu sync.RWMutex

	// Map from endpoint URL to ChainTracker
	trackers map[string]chaintracker.IChainTracker

	// Map from endpoint URL to ChainFetcher (needed to access fetcher methods)
	fetchers map[string]*EndpointChainFetcher

	// Map from endpoint URL to cancel function for per-tracker context cancellation
	// This enables stopping individual trackers without affecting others
	cancelFuncs map[string]context.CancelFunc

	// Shared configuration
	chainParser  chainlib.ChainParser
	chainID      string
	apiInterface string
	pmetrics     *metrics.ProviderMetricsManager

	// Chain-specific timing
	averageBlockTime time.Duration
	blocksToSave     uint64

	// Callbacks for events (optional)
	onFork        func(endpointURL string, blockNum int64)
	onNewBlock    func(endpointURL string, fromBlock, toBlock int64)
	onConsistency func(endpointURL string, oldBlock, newBlock int64)

	// Context for managing goroutines (parent context for all trackers)
	ctx    context.Context
	cancel context.CancelFunc
}

// EndpointChainTrackerConfig holds configuration for the manager.
type EndpointChainTrackerConfig struct {
	ChainParser      chainlib.ChainParser
	ChainID          string
	ApiInterface     string
	AverageBlockTime time.Duration
	BlocksToSave     uint64
	Pmetrics         *metrics.ProviderMetricsManager

	// Optional callbacks
	OnFork        func(endpointURL string, blockNum int64)
	OnNewBlock    func(endpointURL string, fromBlock, toBlock int64)
	OnConsistency func(endpointURL string, oldBlock, newBlock int64)
}

// NewEndpointChainTrackerManager creates a new manager for per-endpoint ChainTrackers.
func NewEndpointChainTrackerManager(ctx context.Context, config EndpointChainTrackerConfig) *EndpointChainTrackerManager {
	blocksToSave := config.BlocksToSave
	if blocksToSave == 0 {
		blocksToSave = DefaultBlocksToSave
	}

	avgBlockTime := config.AverageBlockTime
	if avgBlockTime == 0 {
		avgBlockTime = 12 * time.Second // Default to Ethereum-like timing
	}

	ctxWithCancel, cancel := context.WithCancel(ctx)

	manager := &EndpointChainTrackerManager{
		trackers:         make(map[string]chaintracker.IChainTracker),
		fetchers:         make(map[string]*EndpointChainFetcher),
		cancelFuncs:      make(map[string]context.CancelFunc),
		chainParser:      config.ChainParser,
		chainID:          config.ChainID,
		apiInterface:     config.ApiInterface,
		pmetrics:         config.Pmetrics,
		averageBlockTime: avgBlockTime,
		blocksToSave:     blocksToSave,
		onFork:           config.OnFork,
		onNewBlock:       config.OnNewBlock,
		onConsistency:    config.OnConsistency,
		ctx:              ctxWithCancel,
		cancel:           cancel,
	}

	return manager
}

// GetOrCreateTracker returns an existing ChainTracker for the endpoint or creates a new one.
// Thread-safe - uses lazy initialization to avoid creating trackers for unused endpoints.
func (m *EndpointChainTrackerManager) GetOrCreateTracker(
	endpoint *lavasession.Endpoint,
	directConnection lavasession.DirectRPCConnection,
) (chaintracker.IChainTracker, error) {
	endpointURL := endpoint.NetworkAddress

	// Fast path: check if already exists
	m.mu.RLock()
	if tracker, exists := m.trackers[endpointURL]; exists {
		m.mu.RUnlock()
		return tracker, nil
	}
	m.mu.RUnlock()

	// Slow path: create new tracker
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if tracker, exists := m.trackers[endpointURL]; exists {
		return tracker, nil
	}

	// Create the chain fetcher
	fetcher := NewEndpointChainFetcher(
		endpoint,
		directConnection,
		m.chainParser,
		m.chainID,
		m.apiInterface,
	)

	// Configure the ChainTracker
	config := chaintracker.ChainTrackerConfig{
		BlocksToSave:             m.blocksToSave,
		AverageBlockTime:         m.averageBlockTime,
		ServerBlockMemory:        chaintracker.DefaultAssumedBlockMemory,
		BlocksCheckpointDistance: chaintracker.DefaultBlockCheckpointDistance,
		Pmetrics:                 m.pmetrics,
		ChainId:                  m.chainID,
		ParseDirectiveEnabled:    true, // Always enabled for direct RPC
	}

	// Set up callbacks with endpoint context
	if m.onFork != nil {
		config.ForkCallback = func(blockNum int64) {
			m.onFork(endpointURL, blockNum)
		}
	}

	if m.onNewBlock != nil {
		config.NewLatestCallback = func(fromBlock, toBlock int64, hash string) {
			m.onNewBlock(endpointURL, fromBlock, toBlock)
			// Also update the endpoint's LatestBlock atomically
			endpoint.LatestBlock.Store(toBlock)
			endpoint.LastBlockUpdate = time.Now()
		}
	} else {
		// Default: just update the endpoint's block data
		config.NewLatestCallback = func(fromBlock, toBlock int64, hash string) {
			endpoint.LatestBlock.Store(toBlock)
			endpoint.LastBlockUpdate = time.Now()
		}
	}

	if m.onConsistency != nil {
		config.ConsistencyCallback = func(oldBlock, newBlock int64) {
			m.onConsistency(endpointURL, oldBlock, newBlock)
		}
	}

	// Create a child context for this specific tracker
	// This enables stopping individual trackers without affecting others
	trackerCtx, trackerCancel := context.WithCancel(m.ctx)

	// Create the ChainTracker with its own context
	tracker, err := chaintracker.NewChainTracker(trackerCtx, fetcher, config)
	if err != nil {
		trackerCancel() // Clean up on failure
		return nil, utils.LavaFormatError("failed to create ChainTracker for endpoint", err,
			utils.LogAttr("endpoint", endpointURL),
			utils.LogAttr("chainID", m.chainID),
		)
	}

	// Start the tracker in a goroutine with its own context
	go func() {
		if err := tracker.StartAndServe(trackerCtx); err != nil {
			utils.LavaFormatWarning("ChainTracker stopped with error", err,
				utils.LogAttr("endpoint", endpointURL),
				utils.LogAttr("chainID", m.chainID),
			)
		}
	}()

	// Store tracker, fetcher, and cancel function
	m.trackers[endpointURL] = tracker
	m.fetchers[endpointURL] = fetcher
	m.cancelFuncs[endpointURL] = trackerCancel

	utils.LavaFormatInfo("created ChainTracker for endpoint",
		utils.LogAttr("endpoint", endpointURL),
		utils.LogAttr("chainID", m.chainID),
		utils.LogAttr("avgBlockTime", m.averageBlockTime),
	)

	return tracker, nil
}

// GetTracker returns the ChainTracker for an endpoint if it exists.
func (m *EndpointChainTrackerManager) GetTracker(endpointURL string) (chaintracker.IChainTracker, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tracker, exists := m.trackers[endpointURL]
	return tracker, exists
}

// GetLatestBlockNum returns the latest block number for an endpoint.
// Returns 0 if no tracker exists for the endpoint.
func (m *EndpointChainTrackerManager) GetLatestBlockNum(endpointURL string) int64 {
	m.mu.RLock()
	tracker, exists := m.trackers[endpointURL]
	m.mu.RUnlock()

	if !exists {
		return 0
	}

	return tracker.GetAtomicLatestBlockNum()
}

// GetLatestBlockData returns detailed block data for an endpoint.
// Returns latest block number, change time, and whether data exists.
func (m *EndpointChainTrackerManager) GetLatestBlockData(endpointURL string) (latestBlock int64, changeTime time.Time, exists bool) {
	m.mu.RLock()
	tracker, trackerExists := m.trackers[endpointURL]
	m.mu.RUnlock()

	if !trackerExists {
		return 0, time.Time{}, false
	}

	latestBlock, changeTime = tracker.GetLatestBlockNum()
	return latestBlock, changeTime, true
}

// RemoveTracker removes and stops a ChainTracker for an endpoint.
// It cancels the tracker's context first, which signals the goroutine to exit cleanly.
func (m *EndpointChainTrackerManager) RemoveTracker(endpointURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Cancel the tracker's context first - this signals the goroutine to exit
	if cancel, exists := m.cancelFuncs[endpointURL]; exists {
		cancel()
		delete(m.cancelFuncs, endpointURL)
	}

	// Remove from maps
	delete(m.trackers, endpointURL)
	delete(m.fetchers, endpointURL)

	utils.LavaFormatInfo("stopped and removed ChainTracker for endpoint",
		utils.LogAttr("endpoint", endpointURL),
		utils.LogAttr("chainID", m.chainID),
	)
}

// GetAllEndpoints returns all endpoint URLs with active ChainTrackers.
func (m *EndpointChainTrackerManager) GetAllEndpoints() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	endpoints := make([]string, 0, len(m.trackers))
	for url := range m.trackers {
		endpoints = append(endpoints, url)
	}
	return endpoints
}

// GetEndpointCount returns the number of active ChainTrackers.
func (m *EndpointChainTrackerManager) GetEndpointCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.trackers)
}

// Stop stops all ChainTrackers and cleans up resources.
// It cancels all individual tracker contexts first, then the parent context.
func (m *EndpointChainTrackerManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	trackerCount := len(m.trackers)

	// Cancel all individual tracker contexts first
	for url, cancel := range m.cancelFuncs {
		cancel()
		delete(m.cancelFuncs, url)
	}

	// Then cancel parent context (redundant but ensures cleanup)
	m.cancel()

	// Clear maps
	m.trackers = make(map[string]chaintracker.IChainTracker)
	m.fetchers = make(map[string]*EndpointChainFetcher)
	m.cancelFuncs = make(map[string]context.CancelFunc)

	utils.LavaFormatInfo("stopped EndpointChainTrackerManager",
		utils.LogAttr("chainID", m.chainID),
		utils.LogAttr("trackersStopped", trackerCount),
	)
}

// ValidateEndpointSync checks if an endpoint is synced within the given threshold.
// Returns true if the endpoint's latest block is within threshold of the reference block.
func (m *EndpointChainTrackerManager) ValidateEndpointSync(endpointURL string, referenceBlock int64, threshold int64) bool {
	latestBlock := m.GetLatestBlockNum(endpointURL)
	if latestBlock == 0 {
		// No data yet - don't filter
		return true
	}

	gap := referenceBlock - latestBlock
	return gap <= threshold
}

// GetSyncGap returns the sync gap between an endpoint and a reference block.
// Returns 0 if endpoint is ahead or no data exists.
func (m *EndpointChainTrackerManager) GetSyncGap(endpointURL string, referenceBlock int64) int64 {
	latestBlock := m.GetLatestBlockNum(endpointURL)
	if latestBlock == 0 || latestBlock >= referenceBlock {
		return 0
	}
	return referenceBlock - latestBlock
}

// IsDummy returns false - this is a real manager.
func (m *EndpointChainTrackerManager) IsDummy() bool {
	return false
}
