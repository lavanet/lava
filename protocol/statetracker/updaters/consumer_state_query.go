package updaters

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
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

// ConsumerStateQuery is a minimal stub for querying chain state from a consumer
// context. In the stripped-down smart-router build it wraps a cosmos-sdk client
// context and delegates spec queries directly over gRPC so the testing command
// can still fetch specs from a running node.
type ConsumerStateQuery struct {
	clientCtx   client.Context
	fromAddress string
}

// NewConsumerStateQuery constructs a ConsumerStateQuery backed by the provided
// cosmos client context.  It is intentionally minimal: only GetSpec is
// implemented, which is all the smart-router testing command requires.
func NewConsumerStateQuery(_ context.Context, clientCtx client.Context) *ConsumerStateQuery {
	return &ConsumerStateQuery{
		clientCtx:   clientCtx,
		fromAddress: clientCtx.FromAddress.String(),
	}
}

// GetSpec queries the spec for the given chainID from the connected node.
func (csq *ConsumerStateQuery) GetSpec(ctx context.Context, chainID string) (*spectypes.Spec, error) {
	specClient := spectypes.NewQueryClient(csq.clientCtx)
	resp, err := specClient.Spec(ctx, &spectypes.QueryGetSpecRequest{ChainID: chainID})
	if err != nil {
		return nil, fmt.Errorf("failed querying spec for chain %s: %w", chainID, err)
	}
	return &resp.Spec, nil
}
