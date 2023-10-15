package chaintracker

import (
	"context"

	empty "github.com/golang/protobuf/ptypes/empty"
)

type ChainTrackerService struct {
	UnimplementedChainTrackerServiceServer
	ChainTracker *ChainTracker
}

func (cts *ChainTrackerService) GetLatestBlockNum(context.Context, *empty.Empty) (*GetLatestBlockNumResponse, error) {
	latestBlockNum, changeTime := cts.ChainTracker.GetLatestBlockNum()
	if latestBlockNum <= 0 {
		return nil, InvalidLatestBlockNumValue
	}
	return &GetLatestBlockNumResponse{Block: uint64(latestBlockNum), Timestamp: changeTime.Unix()}, nil
}

func (cts *ChainTrackerService) GetLatestBlockData(ctx context.Context, latestBlockData *LatestBlockData) (*LatestBlockDataResponse, error) {
	latestBlockNum, requestedHashes, _, err := cts.ChainTracker.GetLatestBlockData(latestBlockData.FromBlock, latestBlockData.ToBlock, latestBlockData.SpecificBlock)
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
