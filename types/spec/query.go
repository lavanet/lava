package spec

import (
	"context"

	"google.golang.org/grpc"
)

// QueryGetSpecRequest is the request message for the Spec gRPC query.
type QueryGetSpecRequest struct {
	ChainID string `json:"ChainID"`
}

// QueryGetSpecResponse is the response message for the Spec gRPC query.
type QueryGetSpecResponse struct {
	Spec Spec `json:"Spec"`
}

// GetChainID returns the ChainID field (nil-safe getter matching the original
// protobuf-generated API).
func (m *QueryGetSpecRequest) GetChainID() string {
	if m != nil {
		return m.ChainID
	}
	return ""
}

// GetSpec returns the Spec field (nil-safe getter matching the original
// protobuf-generated API).
func (m *QueryGetSpecResponse) GetSpec() Spec {
	if m != nil {
		return m.Spec
	}
	return Spec{}
}

// QueryClient is a subset of the full spec module query client interface.
// Only the Spec method is retained here; callers that need the full set of
// queries (SpecAll, ShowAllChains, etc.) should extend this interface or use
// the generated gRPC client directly.
type QueryClient interface {
	Spec(ctx context.Context, in *QueryGetSpecRequest, opts ...grpc.CallOption) (*QueryGetSpecResponse, error)
}

// NewQueryClient returns a QueryClient that sends requests over the provided
// gRPC connection.  The cc parameter is intentionally typed as
// grpc.ClientConnInterface so that both standard *grpc.ClientConn values and
// Cosmos gogoproto grpc.ClientConn implementations (which satisfy the same
// Invoke / NewStream contract) can be passed without a cast.
func NewQueryClient(cc grpc.ClientConnInterface) QueryClient {
	return &queryClient{cc: cc}
}

type queryClient struct {
	cc grpc.ClientConnInterface
}

func (c *queryClient) Spec(ctx context.Context, in *QueryGetSpecRequest, opts ...grpc.CallOption) (*QueryGetSpecResponse, error) {
	out := new(QueryGetSpecResponse)
	if err := c.cc.Invoke(ctx, "/lavanet.lava.spec.Query/Spec", in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}
