package mock

import (
	"context"
	"errors"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

// In the mock this number is considered invalid block number
const InvalidBlockNumber = int64(1000000)

type MockConsumerStateQuery struct{}

func (m MockConsumerStateQuery) GetPairing(ctx context.Context, chainID string, latestBlock int64) ([]epochstoragetypes.StakeEntry, uint64, uint64, error) {
	if latestBlock == InvalidBlockNumber {
		return []epochstoragetypes.StakeEntry{}, 50, uint64(latestBlock), errors.New("Error")
	}
	return []epochstoragetypes.StakeEntry{}, 50, uint64(latestBlock), nil
}
