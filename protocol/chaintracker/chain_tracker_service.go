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
