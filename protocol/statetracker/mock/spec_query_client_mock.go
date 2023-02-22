package mock

import (
	"context"
	"errors"

	spectypes "github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc"
)

type MockSpecQueryClient struct {
	spectypes.QueryClient
}

func (msqc MockSpecQueryClient) Spec(ctx context.Context, in *spectypes.QueryGetSpecRequest, opts ...grpc.CallOption) (*spectypes.QueryGetSpecResponse, error) {
	if in.ChainID == "ERR" {
		return &spectypes.QueryGetSpecResponse{}, errors.New("Err")
	}

	return &spectypes.QueryGetSpecResponse{
		spectypes.Spec{Name: "test"},
	}, nil
}
