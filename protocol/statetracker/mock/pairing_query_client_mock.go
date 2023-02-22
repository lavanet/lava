package mock

import (
	"context"
	"errors"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc"
)

type MockPairingQueryClient struct {
	pairingtypes.QueryClient
}

func (mpqc MockPairingQueryClient) GetPairing(ctx context.Context, in *pairingtypes.QueryGetPairingRequest, opts ...grpc.CallOption) (*pairingtypes.QueryGetPairingResponse, error) {
	if in.ChainID == "ERR" {
		return &pairingtypes.QueryGetPairingResponse{}, errors.New("Err")
	}

	return &pairingtypes.QueryGetPairingResponse{
		Providers:          []epochstoragetypes.StakeEntry{},
		CurrentEpoch:       1,
		BlockOfNextPairing: 100,
	}, nil
}

func (mpqc MockPairingQueryClient) UserEntry(ctx context.Context, in *pairingtypes.QueryUserEntryRequest, opts ...grpc.CallOption) (*pairingtypes.QueryUserEntryResponse, error) {
	if in.ChainID == "ERR" {
		return &pairingtypes.QueryUserEntryResponse{}, errors.New("Err")
	}

	return &pairingtypes.QueryUserEntryResponse{
		MaxCU: in.Block,
	}, nil
}

func (mpqc MockPairingQueryClient) Params(ctx context.Context, in *pairingtypes.QueryParamsRequest, opts ...grpc.CallOption) (*pairingtypes.QueryParamsResponse, error) {
	return &pairingtypes.QueryParamsResponse{Params: pairingtypes.Params{ServicersToPairCount: uint64(10), RecommendedEpochNumToCollectPayment: uint64(10)}}, nil
}
