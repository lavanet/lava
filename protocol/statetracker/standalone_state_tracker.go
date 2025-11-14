package statetracker

import (
	"context"
	"time"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/v5/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	protocoltypes "github.com/lavanet/lava/v5/x/protocol/types"
)

const (
	// DefaultAverageBlockTime is the default block time for standalone mode
	// This represents the Lava blockchain (LAV1) average block time, NOT the serviced chains
	// Based on LAV1 blockchain average block time (15 seconds)
	// See: specs/mainnet-1/specs/lava.json - "average_block_time": 15000 (milliseconds)
	// Note: This is used for approximate calculations (e.g., epoch size in blocks)
	DefaultAverageBlockTime = 15 * time.Second
)

// StandaloneStateTracker is a minimal implementation of ProviderStateTrackerInf
// for standalone mode (--static-providers). It doesn't connect to the Lava blockchain
// but provides all required interfaces using defaults and time-based epochs.
type StandaloneStateTracker struct {
	epochTimer       *common.EpochTimer
	epochUpdatables  []updaters.EpochUpdatable
	version          *updaters.ProtocolVersionResponse
	averageBlockTime time.Duration
}

// NewStandaloneStateTracker creates a new standalone state tracker for static providers
// averageBlockTime: Block time for the Lava blockchain (LAV1), not serviced chains.
//
//	Pass 0 to use DefaultAverageBlockTime (15 seconds)
func NewStandaloneStateTracker(epochTimer *common.EpochTimer, averageBlockTime time.Duration) *StandaloneStateTracker {
	// Use default LAV1 block time if not specified
	if averageBlockTime == 0 {
		averageBlockTime = DefaultAverageBlockTime
	}

	sst := &StandaloneStateTracker{
		epochTimer:       epochTimer,
		epochUpdatables:  []updaters.EpochUpdatable{},
		averageBlockTime: averageBlockTime,
		version: &updaters.ProtocolVersionResponse{
			Version:     &protocoltypes.DefaultVersion, // Use official version constants
			BlockNumber: "1",
		},
	}

	// Register epoch callback to notify all updatables
	if epochTimer != nil {
		epochTimer.RegisterCallback(func(epoch uint64) {
			utils.LavaFormatDebug("Standalone state tracker epoch update", utils.LogAttr("epoch", epoch))
			for _, updatable := range sst.epochUpdatables {
				updatable.UpdateEpoch(epoch)
			}
		})
	}

	return sst
}

// RegisterForVersionUpdates registers for protocol version updates (no-op in standalone)
func (sst *StandaloneStateTracker) RegisterForVersionUpdates(ctx context.Context, version *protocoltypes.Version, versionValidator updaters.VersionValidationInf) {
	utils.LavaFormatDebug("Standalone mode: skipping version updates registration")
}

// RegisterForSpecUpdates registers for spec updates (no-op in standalone, using static specs)
func (sst *StandaloneStateTracker) RegisterForSpecUpdates(ctx context.Context, specUpdatable updaters.SpecUpdatable, endpoint lavasession.RPCEndpoint) error {
	utils.LavaFormatDebug("Standalone mode: skipping spec updates registration", utils.LogAttr("chainID", endpoint.ChainID))
	return nil
}

// RegisterForSpecVerifications registers for spec verifications (no-op in standalone)
func (sst *StandaloneStateTracker) RegisterForSpecVerifications(ctx context.Context, specVerifier updaters.SpecVerifier, chainId string) error {
	utils.LavaFormatDebug("Standalone mode: skipping spec verifications registration", utils.LogAttr("chainID", chainId))
	return nil
}

// RegisterReliabilityManagerForVoteUpdates registers reliability manager (no-op in standalone, no voting)
func (sst *StandaloneStateTracker) RegisterReliabilityManagerForVoteUpdates(ctx context.Context, voteUpdatable updaters.VoteUpdatable, endpointP *lavasession.RPCProviderEndpoint) {
	utils.LavaFormatDebug("Standalone mode: skipping reliability manager vote updates registration")
}

// RegisterForEpochUpdates registers for epoch updates using the time-based EpochTimer
func (sst *StandaloneStateTracker) RegisterForEpochUpdates(ctx context.Context, epochUpdatable updaters.EpochUpdatable) {
	sst.epochUpdatables = append(sst.epochUpdatables, epochUpdatable)

	// Trigger initial epoch update
	if sst.epochTimer != nil {
		currentEpoch := sst.epochTimer.GetCurrentEpoch()
		epochUpdatable.UpdateEpoch(currentEpoch)
		utils.LavaFormatDebug("Standalone mode: registered for epoch updates", utils.LogAttr("currentEpoch", currentEpoch))
	}
}

// RegisterForDowntimeParamsUpdates registers for downtime params updates (no-op in standalone)
func (sst *StandaloneStateTracker) RegisterForDowntimeParamsUpdates(ctx context.Context, downtimeParamsUpdatable updaters.DowntimeParamsUpdatable) error {
	utils.LavaFormatDebug("Standalone mode: skipping downtime params updates registration")
	return nil
}

// TxRelayPayment submits relay payment transaction (not supported in standalone mode)
func (sst *StandaloneStateTracker) TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelaySession, description string, latestBlocks []*pairingtypes.LatestBlockReport) error {
	return utils.LavaFormatWarning("TxRelayPayment not supported in standalone mode", nil)
}

// SendVoteReveal sends vote reveal (not supported in standalone mode)
func (sst *StandaloneStateTracker) SendVoteReveal(voteID string, vote *reliabilitymanager.VoteData, specID string) error {
	return utils.LavaFormatWarning("SendVoteReveal not supported in standalone mode", nil)
}

// SendVoteCommitment sends vote commitment (not supported in standalone mode)
func (sst *StandaloneStateTracker) SendVoteCommitment(voteID string, vote *reliabilitymanager.VoteData, specID string) error {
	return utils.LavaFormatWarning("SendVoteCommitment not supported in standalone mode", nil)
}

// LatestBlock returns the latest block number (returns 0 in standalone mode)
func (sst *StandaloneStateTracker) LatestBlock() int64 {
	return 0
}

// GetMaxCuForUser returns max CU for a user (returns unlimited in standalone mode)
func (sst *StandaloneStateTracker) GetMaxCuForUser(ctx context.Context, consumerAddress, chainID string, epocu uint64) (maxCu uint64, err error) {
	return ^uint64(0), nil // Return max uint64 (unlimited)
}

// VerifyPairing verifies consumer-provider pairing (always valid in standalone mode)
func (sst *StandaloneStateTracker) VerifyPairing(ctx context.Context, consumerAddress, providerAddress string, epoch uint64, chainID string) (valid bool, total int64, projectId string, err error) {
	return true, 1, "standalone-project", nil
}

// GetEpochSize returns the epoch size in blocks
func (sst *StandaloneStateTracker) GetEpochSize(ctx context.Context) (uint64, error) {
	if sst.epochTimer != nil {
		// Convert epoch duration to approximate number of blocks
		epochDuration := sst.epochTimer.GetEpochDuration()
		blocksInEpoch := uint64(epochDuration / sst.averageBlockTime)
		return blocksInEpoch, nil
	}
	return 100, nil // Default: 100 blocks per epoch
}

// EarliestBlockInMemory returns the earliest block kept in memory
func (sst *StandaloneStateTracker) EarliestBlockInMemory(ctx context.Context) (uint64, error) {
	return 0, nil
}

// RegisterPaymentUpdatableForPayments registers for payment updates (no-op in standalone)
func (sst *StandaloneStateTracker) RegisterPaymentUpdatableForPayments(ctx context.Context, paymentUpdatable updaters.PaymentUpdatable) {
	utils.LavaFormatDebug("Standalone mode: skipping payment updates registration")
}

// GetRecommendedEpochNumToCollectPayment returns recommended epochs to collect payment
func (sst *StandaloneStateTracker) GetRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error) {
	return 1, nil // Keep data for 1 epoch in standalone mode
}

// GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment returns memory size for session management
func (sst *StandaloneStateTracker) GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error) {
	// Return a reasonable default for block memory size
	// This is used to determine how many blocks to keep in ProviderSessionManager
	return 1000, nil // Keep 1000 blocks in memory
}

// GetProtocolVersion returns the protocol version
func (sst *StandaloneStateTracker) GetProtocolVersion(ctx context.Context) (*updaters.ProtocolVersionResponse, error) {
	return sst.version, nil
}

// GetVirtualEpoch returns the virtual epoch (same as real epoch in standalone)
func (sst *StandaloneStateTracker) GetVirtualEpoch(epoch uint64) uint64 {
	return epoch
}

// GetAverageBlockTime returns the Lava blockchain's average block time
// Note: This is NOT the block time of serviced chains (NEAR, ETH, etc.)
func (sst *StandaloneStateTracker) GetAverageBlockTime() time.Duration {
	return sst.averageBlockTime
}

// RegisterForUpdates registers for general updates (no-op in standalone)
func (sst *StandaloneStateTracker) RegisterForUpdates(ctx context.Context, updater Updater) Updater {
	utils.LavaFormatDebug("Standalone mode: skipping general updates registration")
	return updater
}
