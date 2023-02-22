package statetracker

import (
	"context"
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/protocol/statetracker/mock"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/assert"
)

func TestNewStateQuery(t *testing.T) {
	ctx := context.Background()
	clientCtx := client.Context{}

	stateQuery := NewStateQuery(ctx, clientCtx)

	if stateQuery.SpecQueryClient == nil {
		t.Error("expected SpecQueryClient to not be nil")
	}
	if stateQuery.PairingQueryClient == nil {
		t.Error("expected PairingQueryClient to not be nil")
	}
	if stateQuery.EpochStorageQueryClient == nil {
		t.Error("expected EpochStorageQueryClient to not be nil")
	}
}

func TestNewConsumerStateQuery(t *testing.T) {
	ctx := context.Background()
	clientCtx := client.Context{}

	consumerStateQuery := NewConsumerStateQuery(ctx, clientCtx)

	if consumerStateQuery.StateQuery.SpecQueryClient == nil {
		t.Error("expected SpecQueryClient to not be nil")
	}
	if consumerStateQuery.StateQuery.PairingQueryClient == nil {
		t.Error("expected PairingQueryClient to not be nil")
	}
	if consumerStateQuery.StateQuery.EpochStorageQueryClient == nil {
		t.Error("expected EpochStorageQueryClient to not be nil")
	}
}

func TestNewProviderStateQuery(t *testing.T) {
	ctx := context.Background()
	clientCtx := client.Context{}

	providerStateQuery := NewProviderStateQuery(ctx, clientCtx)

	if providerStateQuery.StateQuery.SpecQueryClient == nil {
		t.Error("expected SpecQueryClient to not be nil")
	}
	if providerStateQuery.StateQuery.PairingQueryClient == nil {
		t.Error("expected PairingQueryClient to not be nil")
	}
	if providerStateQuery.StateQuery.EpochStorageQueryClient == nil {
		t.Error("expected EpochStorageQueryClient to not be nil")
	}
}

func getConsumerStateQuery() *ConsumerStateQuery {
	// create a mock client context
	clientCtx := client.Context{}

	// create a mock pairing query client
	pairingQueryClient := &mock.MockPairingQueryClient{}

	// create a mock spec query client
	specQueryClient := &mock.MockSpecQueryClient{}

	// create a mock state query with the mock pairing query client
	stateQuery := &StateQuery{
		PairingQueryClient: pairingQueryClient,
		SpecQueryClient:    specQueryClient,
	}

	// create a consumer state query with the mock state query and client context
	consumerStateQuery := &ConsumerStateQuery{
		StateQuery: *stateQuery,
		clientCtx:  clientCtx,
	}

	return consumerStateQuery
}

func TestConsumerStateQuery_GetPairing(t *testing.T) {
	consumerStateQuery := getConsumerStateQuery()

	// call GetPairing with an empty chain ID to get the first cached pairing
	pairingList, epoch, nextBlockForUpdate, err := consumerStateQuery.GetPairing(context.Background(), "", 10)

	// create a mock pairing response
	pairingResp := &pairingtypes.QueryGetPairingResponse{
		Providers:          []epochstoragetypes.StakeEntry{},
		CurrentEpoch:       1,
		BlockOfNextPairing: 100,
	}

	assert.Nil(t, err)
	assert.Equal(t, pairingResp.Providers, pairingList)
	assert.Equal(t, pairingResp.CurrentEpoch, epoch)
	assert.Equal(t, pairingResp.BlockOfNextPairing, nextBlockForUpdate)

	pairingList, epoch, nextBlockForUpdate, err = consumerStateQuery.GetPairing(context.Background(), "", 11)
	assert.Nil(t, err)
	assert.Equal(t, pairingResp.Providers, pairingList)
	assert.Equal(t, pairingResp.CurrentEpoch, epoch)
	assert.Equal(t, pairingResp.BlockOfNextPairing, nextBlockForUpdate)

	_, _, _, err = consumerStateQuery.GetPairing(context.Background(), "ERR", 12)
	assert.NotNil(t, err)

	pairingList, epoch, nextBlockForUpdate, err = consumerStateQuery.GetPairing(context.Background(), "LAV1", 0)
	assert.Nil(t, err)
	assert.Equal(t, pairingResp.Providers, pairingList)
	assert.Equal(t, pairingResp.CurrentEpoch, epoch)
	assert.Equal(t, pairingResp.BlockOfNextPairing, nextBlockForUpdate)
}

func TestConsumerStateQuery_GetMaxCUForUser(t *testing.T) {
	consumerStateQuery := getConsumerStateQuery()

	// call GetPairing with an empty chain ID to get the first cached pairing
	maxCu, err := consumerStateQuery.GetMaxCUForUser(context.Background(), "", 10)
	assert.Nil(t, err)
	assert.Equal(t, maxCu, uint64(10))

	// call GetPairing with an empty chain ID to get the first cached pairing
	_, err = consumerStateQuery.GetMaxCUForUser(context.Background(), "ERR", 10)
	assert.Error(t, err)
}

func TestConsumerStateQuery_GetSpec(t *testing.T) {
	consumerStateQuery := getConsumerStateQuery()

	spec, err := consumerStateQuery.GetSpec(context.Background(), "")
	assert.Nil(t, err)
	assert.Equal(t, spec.Name, "test")

	// call GetPairing with an empty chain ID to get the first cached pairing
	_, err = consumerStateQuery.GetSpec(context.Background(), "ERR")
	assert.Error(t, err)
}

func TestEntryKey(t *testing.T) {
	// Create a mock ProviderStateQuery to use in the test
	clientCtx := client.Context{}
	psq := &ProviderStateQuery{clientCtx: clientCtx}

	// Test with some example inputs
	consumerAddress := "0x1234567890123456789012345678901234567890"
	chainID := "testnet-1"
	epoch := uint64(12345)
	providerAddress := "0x0987654321098765432109876543210987654321"

	expectedKey := consumerAddress + chainID + strconv.FormatUint(epoch, 10) + providerAddress
	actualKey := psq.entryKey(consumerAddress, chainID, epoch, providerAddress)

	assert.Equal(t, expectedKey, actualKey)
}

func getProviderStateQuery() *ProviderStateQuery {
	// create a mock client context
	clientCtx := client.Context{}

	// create a mock pairing query client
	epochStorageQuery := &mock.MockEpochStorageQueryClient{}

	// create a mock pairing query client
	pairingQueryClient := &mock.MockPairingQueryClient{}

	// create a mock state query with the mock pairing query client
	stateQuery := &StateQuery{
		EpochStorageQueryClient: epochStorageQuery,
		PairingQueryClient:      pairingQueryClient,
	}

	// create a consumer state query with the mock state query and client context
	providerStateQuery := &ProviderStateQuery{
		StateQuery: *stateQuery,
		clientCtx:  clientCtx,
	}
	return providerStateQuery
}

func TestProviderStateQuery_CurrentEpochStart(t *testing.T) {
	providerStateQuery := getProviderStateQuery()
	res, err := providerStateQuery.CurrentEpochStart(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, res, uint64(10))
}

func TestProviderStateQuery_GetEpochSize(t *testing.T) {
	providerStateQuery := getProviderStateQuery()
	res, err := providerStateQuery.GetEpochSize(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, res, uint64(10))
}

func TestProviderStateQuery_GetProvidersCountForConsumer(t *testing.T) {
	providerStateQuery := getProviderStateQuery()
	res, err := providerStateQuery.GetProvidersCountForConsumer(context.Background(), "", 0, "")
	assert.Nil(t, err)
	assert.Equal(t, res, uint32(10))
}

func TestProviderStateQuery_EarliestBlockInMemory(t *testing.T) {
	providerStateQuery := getProviderStateQuery()
	res, err := providerStateQuery.EarliestBlockInMemory(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, res, uint64(10))
}

func TestProviderStateQuery_GetRecommendedEpochNumToCollectPayment(t *testing.T) {
	providerStateQuery := getProviderStateQuery()
	res, err := providerStateQuery.GetRecommendedEpochNumToCollectPayment(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, res, uint64(10))
}

func TestProviderStateQuery_GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(t *testing.T) {
	providerStateQuery := getProviderStateQuery()
	res, err := providerStateQuery.GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, res, uint64(10*10))
}
