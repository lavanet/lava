package chainlib

import (
	"github.com/goccy/go-json"

	"github.com/lavanet/lava/v5/protocol/common"
)

const (
	// MaxBlockNotAvailableRetries is the maximum number of previous slots to try
	// when a Solana endpoint returns -32004 ("Block not available for slot X").
	// This handles both propagation delays and skipped slots.
	MaxBlockNotAvailableRetries = 5
)

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
