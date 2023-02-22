package mock

import (
	"context"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"google.golang.org/grpc"
)

type MockEpochStorageQueryClient struct {
	epochstoragetypes.QueryClient
}

func (m *MockEpochStorageQueryClient) EpochDetails(ctx context.Context, in *epochstoragetypes.QueryGetEpochDetailsRequest, opts ...grpc.CallOption) (*epochstoragetypes.QueryGetEpochDetailsResponse, error) {
	return &epochstoragetypes.QueryGetEpochDetailsResponse{EpochDetails: epochstoragetypes.EpochDetails{EarliestStart: uint64(10), StartBlock: uint64(10)}}, nil
}

func (m *MockEpochStorageQueryClient) Params(ctx context.Context, in *epochstoragetypes.QueryParamsRequest, opts ...grpc.CallOption) (*epochstoragetypes.QueryParamsResponse, error) {
	return &epochstoragetypes.QueryParamsResponse{Params: epochstoragetypes.Params{EpochBlocks: 10}}, nil
}
