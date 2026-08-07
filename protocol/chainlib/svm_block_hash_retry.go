package chainlib

import (
	"context"
	"time"

	"github.com/goccy/go-json"

	"github.com/lavanet/lava/v5/protocol/common"
)

const (
	// MaxSameSlotRetries is the number of times to retry the same slot with a short
	// delay before falling back to previous slots. Handles propagation delays where
	// the slot data hasn't reached the node yet.
	MaxSameSlotRetries = 2
	// MaxPreviousSlotRetries is the number of previous slots to try after same-slot
	// retries are exhausted. Handles skipped slots where Solana produces no block.
	MaxPreviousSlotRetries = 3
	// SameSlotRetryDelay is the wait between same-slot retries.
	SameSlotRetryDelay = 150 * time.Millisecond
)

// blockHashFetchFn is the signature for a single block hash fetch attempt.
// Returns the hash, raw response bytes (for error classification), and any error.
type blockHashFetchFn func(ctx context.Context, blockNum int64) (string, []byte, error)

// fetchBlockHashWithSolanaRetry implements the two-phase -32004 retry strategy:
//  1. Retry the same slot up to MaxSameSlotRetries times with retryDelay (propagation delays).
//  2. Try up to MaxPreviousSlotRetries previous slots without delay (skipped slots).
//
// Returns the hash, the block number that succeeded, and any error.
// Pass retryDelay=SameSlotRetryDelay in production; use a shorter value in tests.
func FetchBlockHashWithSolanaRetry(ctx context.Context, blockNum int64, retryDelay time.Duration, fetch blockHashFetchFn) (hash string, fetchedBlock int64, err error) {
	var lastErr error

	// Phase 1: retry same slot with delay (handles propagation delays).
	for attempt := 0; attempt <= MaxSameSlotRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", blockNum, ctx.Err()
			case <-time.After(retryDelay):
			}
		}

		h, responseData, e := fetch(ctx, blockNum)
		if e == nil {
			return h, blockNum, nil
		}
		if !IsBlockNotAvailableError(responseData) {
			return "", blockNum, e
		}
		lastErr = e
	}

	// Phase 2: try previous slots (handles skipped slots).
	// ctx cancellation is propagated implicitly: fetch(ctx, …) returns a non-nil
	// error, and IsBlockNotAvailableError(nil) == false, so the loop exits immediately.
	for prevOffset := int64(1); prevOffset <= MaxPreviousSlotRetries; prevOffset++ {
		currentBlock := blockNum - prevOffset
		if currentBlock < 0 {
			break
		}

		h, responseData, e := fetch(ctx, currentBlock)
		if e == nil {
			return h, currentBlock, nil
		}
		if !IsBlockNotAvailableError(responseData) {
			return "", currentBlock, e
		}
		lastErr = e
	}

	return "", blockNum, lastErr
}

// IsBlockNotAvailableError checks if a JSON-RPC response contains error code -32004
// ("Block not available for slot X"), indicating a skipped slot or propagation delay.
// The caller is responsible for gating this check to the appropriate chain family.
func IsBlockNotAvailableError(responseData []byte) bool {
	if len(responseData) == 0 {
		return false
	}
	var resp struct {
		Error *struct {
			Code int `json:"code"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(responseData, &resp); err != nil {
		return false
	}
	return resp.Error != nil && resp.Error.Code == common.SolanaBlockNotAvailableCode
}
