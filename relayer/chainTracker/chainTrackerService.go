package chaintracker

import (
	"context"

	empty "github.com/golang/protobuf/ptypes/empty"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
)

type ChainTrackerService struct {
	UnimplementedChainTrackerServiceServer
	ChainTracker *ChainTracker
}

func (cts *ChainTrackerService) GetLatestBlockNum(context.Context, *empty.Empty) (*wrappers.UInt64Value, error) {
	latestBlockNum := cts.ChainTracker.GetLatestBlockNum()
	if latestBlockNum <= 0 {
		return nil, InvalidLatestBlockNumValue
	}
	return &wrappers.UInt64Value{Value: uint64(latestBlockNum)}, nil
}

func (cts *ChainTrackerService) GetLatestBlockData(ctx context.Context, latestBlockData *LatestBlockData) (*LatestBlockDataResponse, error) {
	latestBlockNum, requestedHashes, err := cts.ChainTracker.GetLatestBlockData(latestBlockData.FromBlock, latestBlockData.ToBlock)
	if err != nil {
		return nil, err
	}
	if latestBlockNum <= 0 {
		return nil, InvalidLatestBlockNumValue
	}
	if len(requestedHashes) == 0 {
		return nil, InvalidReturnedHashes
	}
	return &LatestBlockDataResponse{LatestBlock: latestBlockNum, RequestedHashes: requestedHashes}, nil
}
