package updaters

import (
	"context"
	"fmt"

	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	protocoltypes "github.com/lavanet/lava/v5/x/protocol/types"
)

const (
	BlockResultRetry = 20
)

// ProtocolVersionResponse holds the on-chain protocol version together with
// the block number at which it was queried.
type ProtocolVersionResponse struct {
	Version     *protocoltypes.Version
	BlockNumber string
}

// ConsumerStateQuery is a minimal stub that always returns an error for spec
// queries. The smart router uses static spec loading exclusively; this stub
// exists only to satisfy the testing command's interface without importing
// cosmos-sdk. For live blockchain queries, use a fully-featured client.
type ConsumerStateQuery struct{}

// NewConsumerStateQuery constructs a ConsumerStateQuery stub. The context
// parameter is accepted for API compatibility but is not used.
func NewConsumerStateQuery(_ context.Context) *ConsumerStateQuery {
	return &ConsumerStateQuery{}
}

// GetSpec always returns an error because the smart router does not query the
// blockchain. Callers should use static spec loading instead.
func (csq *ConsumerStateQuery) GetSpec(_ context.Context, chainID string) (*spectypes.Spec, error) {
	return nil, fmt.Errorf("ConsumerStateQuery stub: blockchain spec queries are not supported in smart-router mode; use --static-spec-paths to load specs from files (chainID: %s)", chainID)
}
