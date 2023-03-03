package mock

import (
	"context"
	"errors"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

const (
	InvalidChainID        = "Invalid"
	EmptyPairingListBlock = 1000001
)

type MockPairingStateQuery struct{}

func (m MockPairingStateQuery) GetPairing(ctx context.Context, chainID string, latestBlock int64) ([]epochstoragetypes.StakeEntry, uint64, uint64, error) {
	if latestBlock == EmptyPairingListBlock {
		return []epochstoragetypes.StakeEntry{}, 50, uint64(latestBlock), nil
	}

	if chainID == InvalidChainID || latestBlock == InvalidBlockNumber {
		return []epochstoragetypes.StakeEntry{}, 50, uint64(latestBlock), errors.New("Error")
	}
	return []epochstoragetypes.StakeEntry{
		{
			Address: "0x1234",
			Chain:   "eth",
			Endpoints: []epochstoragetypes.Endpoint{
				{
					UseType:     "rest",
					IPPORT:      "127.0.0.1:8545",
					Geolocation: 1,
				},
			},
		},
	}, 50, uint64(latestBlock), nil
}

func (m MockPairingStateQuery) GetMaxCUForUser(ctx context.Context, chainID string, epoch uint64) (maxCu uint64, err error) {
	if epoch == uint64(InvalidBlockNumber) {
		return 10, errors.New("Error")
	}
	return epoch, nil
}
