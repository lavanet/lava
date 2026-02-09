package relaycore

import (
	"github.com/lavanet/lava/v5/protocol/lavaprotocol/protocolerrors"
	"github.com/lavanet/lava/v5/utils"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
)

// ShouldSkipConsistencyValidation returns true when consistency validation
// should not be applied to the request. Validation is skipped for:
// - NOT_APPLICABLE requests (no block parsing)
// - Historical block requests (specific block number > 0)
// - Special block tags (EARLIEST, FINALIZED, SAFE, PENDING)
//
// Validation is only applied for LATEST_BLOCK requests where we expect
// the most recent state.
func ShouldSkipConsistencyValidation(requestedBlock int64) bool {
	switch requestedBlock {
	case spectypes.NOT_APPLICABLE:
		// Block parsing failed or API doesn't use blocks
		return true
	case spectypes.EARLIEST_BLOCK:
		// Historical request for genesis/earliest block
		return true
	case spectypes.FINALIZED_BLOCK:
		// Tag-based finalized block request
		return true
	case spectypes.SAFE_BLOCK:
		// Tag-based safe block request
		return true
	case spectypes.PENDING_BLOCK:
		// Pending block request (future state)
		return true
	case spectypes.LATEST_BLOCK:
		// LATEST_BLOCK should be validated
		return false
	default:
		// requestedBlock >= 0 means specific historical block (including genesis block 0)
		// requestedBlock < -6 would be unknown/invalid
		if requestedBlock >= 0 {
			return true // Historical block request (including genesis)
		}
		// Unknown negative value, validate to be safe
		return false
	}
}

// ValidateEndpointCapability checks if an endpoint can serve a request
// based on its current latest block compared to the user's seen block.
// This is the pre-request validation used for endpoint selection/prioritization.
//
// Parameters:
//   - endpointLatestBlock: The endpoint's tracked latest block
//   - seenBlock: The previously seen block for this user
//   - requestedBlock: The block requested in the original request
//   - config: Validation configuration with thresholds
//
// Returns:
//   - nil if the endpoint is capable or validation should be skipped
//   - ConsistencyError if the endpoint is too far behind
func ValidateEndpointCapability(
	endpointLatestBlock int64,
	seenBlock int64,
	requestedBlock int64,
	config *ConsistencyValidationConfig,
) error {
	// Skip if no config (validation disabled)
	if config == nil {
		return nil
	}

	// Skip if no prior seen block (no requirement)
	if seenBlock <= 0 {
		return nil
	}

	// Skip if endpoint's latest block is unknown
	if endpointLatestBlock <= 0 {
		// Unknown state - allow the request to proceed
		// Post-response validation will catch any issues
		return nil
	}

	// Skip for requests that shouldn't be validated
	if ShouldSkipConsistencyValidation(requestedBlock) {
		return nil
	}

	// If endpoint is at or ahead of seen block, it's capable
	if endpointLatestBlock >= seenBlock {
		return nil
	}

	// Calculate lag: how many blocks behind is the endpoint?
	lag := seenBlock - endpointLatestBlock

	// Check if lag exceeds threshold
	if config.IsEndpointTooFarBehind(lag) {
		utils.LavaFormatDebug("endpoint failed capability validation: too far behind",
			utils.LogAttr("endpointLatestBlock", endpointLatestBlock),
			utils.LogAttr("seenBlock", seenBlock),
			utils.LogAttr("lag", lag),
			utils.LogAttr("threshold", config.EndpointLagThreshold),
			utils.LogAttr("requestedBlock", requestedBlock),
		)
		return protocolerrors.ConsistencyError.Wrapf(
			"endpoint block %d is too far behind (seen block: %d, lag: %d blocks, threshold: %d)",
			endpointLatestBlock, seenBlock, lag, config.EndpointLagThreshold,
		)
	}

	// Lag is within acceptable threshold
	utils.LavaFormatDebug("endpoint within lag threshold",
		utils.LogAttr("endpointLatestBlock", endpointLatestBlock),
		utils.LogAttr("seenBlock", seenBlock),
		utils.LogAttr("lag", lag),
		utils.LogAttr("threshold", config.EndpointLagThreshold),
	)
	return nil
}
