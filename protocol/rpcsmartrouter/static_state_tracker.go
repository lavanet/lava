package rpcsmartrouter

import (
	"context"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol/finalizationconsensus"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/statetracker"
	"github.com/lavanet/lava/v5/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v5/utils"
	conflicttypes "github.com/lavanet/lava/v5/x/conflict/types"
	plantypes "github.com/lavanet/lava/v5/x/plans/types"
	protocoltypes "github.com/lavanet/lava/v5/x/protocol/types"
)

// StaticStateTracker is a minimal implementation of ConsumerStateTrackerInf
// that doesn't connect to the Lava blockchain. It's designed for enterprise
// smart router deployments that use only static provider configurations.
type StaticStateTracker struct {
	// ConsumerEmergencyTrackerInf is nil - emergency tracking not used with static providers
	ConsumerEmergencyTrackerInf interface{}
}

// NewStaticStateTracker creates a new static state tracker that doesn't
// require blockchain connectivity
func NewStaticStateTracker() *StaticStateTracker {
	return &StaticStateTracker{}
}

// RegisterForVersionUpdates is a no-op - smart router uses fixed protocol version
func (s *StaticStateTracker) RegisterForVersionUpdates(ctx context.Context, version *protocoltypes.Version, versionValidator updaters.VersionValidationInf) {
	// No-op: Smart router doesn't need protocol version updates from blockchain
}

// RegisterConsumerSessionManagerForPairingUpdates initializes static providers for smart router
// Unlike the blockchain version, this directly creates provider sessions without blockchain queries
func (s *StaticStateTracker) RegisterConsumerSessionManagerForPairingUpdates(ctx context.Context, consumerSessionManager *lavasession.ConsumerSessionManager, staticProvidersList []*lavasession.RPCStaticProviderEndpoint, backupProvidersList []*lavasession.RPCStaticProviderEndpoint) {
	// Smart router uses epoch 0 since we don't have blockchain epochs
	epoch := uint64(0)
	rpcEndpoint := consumerSessionManager.RPCEndpoint()

	// Create provider sessions from static providers with StaticProvider=true flag
	pairingList := s.createProviderSessionsFromConfig(staticProvidersList, rpcEndpoint, epoch, nil, "StaticProvider_")

	// Create backup provider sessions if provided
	backupProviders := map[uint64]*lavasession.ConsumerSessionsWithProvider{}
	if len(backupProvidersList) > 0 {
		backupProviders = s.createProviderSessionsFromConfig(backupProvidersList, rpcEndpoint, epoch, pairingList, "BackupProvider_")
	}

	// Update the consumer session manager with the static providers
	err := consumerSessionManager.UpdateAllProviders(epoch, pairingList, backupProviders)
	if err != nil {
		utils.LavaFormatError("Failed to update static providers in smart router", err,
			utils.LogAttr("chainID", rpcEndpoint.ChainID),
			utils.LogAttr("apiInterface", rpcEndpoint.ApiInterface))
	}
}

// RegisterForSpecUpdates is a no-op - specs loaded from static files or GitHub
func (s *StaticStateTracker) RegisterForSpecUpdates(ctx context.Context, specUpdatable updaters.SpecUpdatable, endpoint lavasession.RPCEndpoint) error {
	// No-op: Specs are loaded statically, not from blockchain
	return nil
}

// RegisterFinalizationConsensusForUpdates is a no-op - no consensus tracking needed
func (s *StaticStateTracker) RegisterFinalizationConsensusForUpdates(ctx context.Context, finalizationConsensus *finalizationconsensus.FinalizationConsensus, staticProvidersActive bool) {
	// No-op: Smart router doesn't track finalization consensus from blockchain
}

// RegisterForDowntimeParamsUpdates is a no-op - no downtime tracking needed
func (s *StaticStateTracker) RegisterForDowntimeParamsUpdates(ctx context.Context, downtimeParamsUpdatable updaters.DowntimeParamsUpdatable) error {
	// No-op: Smart router doesn't track downtime params from blockchain
	return nil
}

// TxConflictDetection is a no-op - smart router doesn't send conflict transactions
func (s *StaticStateTracker) TxConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, conflictHandler common.ConflictHandlerInterface) error {
	// No-op: Smart router doesn't send conflict detection transactions to blockchain
	return nil
}

// GetConsumerPolicy returns nil - smart router doesn't use blockchain policies
func (s *StaticStateTracker) GetConsumerPolicy(ctx context.Context, consumerAddress, chainID string) (*plantypes.Policy, error) {
	// Return nil policy - smart router doesn't query policies from blockchain
	// Policy configuration should be handled via config files instead
	return nil, nil
}

// GetProtocolVersion returns a fixed protocol version
func (s *StaticStateTracker) GetProtocolVersion(ctx context.Context) (*updaters.ProtocolVersionResponse, error) {
	// Return a fixed protocol version - smart router doesn't query from blockchain
	return &updaters.ProtocolVersionResponse{
		Version: &protocoltypes.Version{
			ProviderTarget: "1.0.0",
			ConsumerTarget: "1.0.0",
			ProviderMin:    "1.0.0",
			ConsumerMin:    "1.0.0",
		},
		BlockNumber: "0",
	}, nil
}

// GetLatestVirtualEpoch returns 0 - smart router doesn't use blockchain epochs
func (s *StaticStateTracker) GetLatestVirtualEpoch() uint64 {
	// Return 0 - smart router with static providers doesn't use blockchain epochs
	return 0
}

// RegisterForUpdates is a no-op - smart router doesn't listen for blockchain state updates
func (s *StaticStateTracker) RegisterForUpdates(ctx context.Context, updater statetracker.Updater) {
	// No-op: Smart router doesn't register for blockchain state updates
}

// RegisterForPairingUpdates is a no-op - smart router uses static provider configuration
func (s *StaticStateTracker) RegisterForPairingUpdates(ctx context.Context, pairingUpdater statetracker.Updater, chainId string) {
	// No-op: Smart router doesn't register for pairing updates from blockchain
}

// createProviderSessionsFromConfig transforms static provider configurations into
// ConsumerSessionsWithProvider objects with StaticProvider=true flag set.
// This is adapted from the pairing_updater.go implementation but simplified for smart router.
func (s *StaticStateTracker) createProviderSessionsFromConfig(
	providers []*lavasession.RPCStaticProviderEndpoint,
	rpcEndpoint lavasession.RPCEndpoint,
	epoch uint64,
	existingProviders map[uint64]*lavasession.ConsumerSessionsWithProvider,
	providerNamePrefix string,
) map[uint64]*lavasession.ConsumerSessionsWithProvider {
	providerSessions := map[uint64]*lavasession.ConsumerSessionsWithProvider{}

	// Calculate start index to avoid conflicts with existing provider indices
	startIdx := uint64(0)
	for key := range existingProviders {
		if key >= startIdx {
			startIdx = key + 1
		}
	}

	for idx, provider := range providers {
		// Only take the provider entries relevant for this apiInterface
		if provider.ApiInterface != rpcEndpoint.ApiInterface || provider.ChainID != rpcEndpoint.ChainID {
			continue
		}

		endpoints := []*lavasession.Endpoint{}
		for _, url := range provider.NodeUrls {
			extensions := map[string]struct{}{}
			for _, extension := range url.Addons {
				extensions[extension] = struct{}{}
			}

			endpoint := &lavasession.Endpoint{
				NetworkAddress: url.Url,
				Enabled:        true,
				Addons:         extensions,
				Extensions:     extensions,
				Connections:    []*lavasession.EndpointConnection{},
			}
			endpoints = append(endpoints, endpoint)
		}

		// Create provider entry with high compute units and stake for availability
		providerEntry := lavasession.NewConsumerSessionWithProvider(
			provider.Name,
			endpoints,
			math.MaxUint64/2, // High compute units for availability
			epoch,
			sdk.NewInt64Coin("ulava", 1000000000000000), // 1b LAVA stake
		)
		// CRITICAL: Mark as static provider for proper handling in lavasession
		providerEntry.StaticProvider = true
		providerSessions[startIdx+uint64(idx)] = providerEntry
	}

	return providerSessions
}
