package protocol

import (
	"context"
	"time"

	grpc1 "github.com/cosmos/gogoproto/grpc"
	"google.golang.org/grpc"
)

const (
	// TARGET_VERSION is the current recommended protocol version.
	TARGET_VERSION = "6.1.0"
	// MIN_VERSION is the minimum supported protocol version.
	MIN_VERSION = "5.5.1"
)

// Version holds the provider and consumer target / minimum version strings for
// the Lava protocol. It mirrors the on-chain Version protobuf message.
type Version struct {
	ProviderTarget string `json:"provider_target"`
	ProviderMin    string `json:"provider_min"`
	ConsumerTarget string `json:"consumer_target"`
	ConsumerMin    string `json:"consumer_min"`
}

// DefaultVersion is the version embedded in binary releases.
var DefaultVersion = Version{
	ProviderTarget: TARGET_VERSION,
	ProviderMin:    MIN_VERSION,
	ConsumerTarget: TARGET_VERSION,
	ConsumerMin:    MIN_VERSION,
}

// ---------------------------------------------------------------------------
// Params holds protocol module parameters.
// ---------------------------------------------------------------------------

// Params is the protocol module parameter set.
type Params struct {
	// Version carries the current on-chain version requirement.
	Version Version `json:"version"`
}

// ---------------------------------------------------------------------------
// gRPC query request / response types
// ---------------------------------------------------------------------------

// QueryParamsRequest is the request type for the protocol Params RPC.
type QueryParamsRequest struct{}

// QueryParamsResponse is the response type for the protocol Params RPC.
type QueryParamsResponse struct {
	Params Params `json:"params"`
}

// ---------------------------------------------------------------------------
// QueryClient interface and constructor
// ---------------------------------------------------------------------------

// QueryClient is the gRPC query client interface for the protocol module.
type QueryClient interface {
	// Params queries the protocol module parameters (including current version).
	Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error)
}

type queryClient struct {
	cc grpc1.ClientConn
}

// NewQueryClient returns a QueryClient that dispatches over cc.
func NewQueryClient(cc grpc1.ClientConn) QueryClient {
	return &queryClient{cc}
}

const protocolParamsRPC = "/lavanet.lava.protocol.Query/Params"

func (c *queryClient) Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	out := new(QueryParamsResponse)
	if err := c.cc.Invoke(ctx, protocolParamsRPC, in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}
