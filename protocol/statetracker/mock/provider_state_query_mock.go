package mock

import (
	"context"
	"errors"

	"github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/protocol/rpcprovider/rewardserver"
)

type MockProviderStateQuery struct {
}

func (m MockProviderStateQuery) CurrentEpochStart(ctx context.Context) (uint64, error) {
	return 10, nil
}

func (m MockProviderStateQuery) PaymentEvents(ctx context.Context, latestBlock int64) (payments []*rewardserver.PaymentRequest, err error) {
	return []*rewardserver.PaymentRequest{
		{Description: "paymentDescription"},
	}, nil
}

func (m MockProviderStateQuery) VoteEvents(ctx context.Context, latestBlock int64) (votes []*reliabilitymanager.VoteParams, err error) {
	return []*reliabilitymanager.VoteParams{
		{ChainID: "LAV1", ApiInterface: "rest"},
	}, nil
}

type MockErrorProviderStateQuery struct {
}

func (m MockErrorProviderStateQuery) CurrentEpochStart(ctx context.Context) (uint64, error) {
	return 10, errors.New("ERROR")
}

func (m MockErrorProviderStateQuery) PaymentEvents(ctx context.Context, latestBlock int64) (payments []*rewardserver.PaymentRequest, err error) {
	return []*rewardserver.PaymentRequest{}, errors.New("ERROR")
}

func (m MockErrorProviderStateQuery) VoteEvents(ctx context.Context, latestBlock int64) (votes []*reliabilitymanager.VoteParams, err error) {
	return []*reliabilitymanager.VoteParams{}, errors.New("ERROR")
}
