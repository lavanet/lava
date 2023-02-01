package statetracker

import (
	"context"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type StateQuery struct {
	SpecQueryClient         spectypes.QueryClient
	PairingQueryClient      pairingtypes.QueryClient
	EpochStorageQueryClient epochstoragetypes.QueryClient
}

func NewStateQuery(ctx context.Context, clientCtx client.Context) *StateQuery {
	sq := &StateQuery{}
	sq.SpecQueryClient = spectypes.NewQueryClient(clientCtx)
	sq.PairingQueryClient = pairingtypes.NewQueryClient(clientCtx)
	sq.EpochStorageQueryClient = epochstoragetypes.NewQueryClient(clientCtx)
	return sq
}

type ConsumerStateQuery struct {
	StateQuery
	clientCtx      client.Context
	cachedPairings map[string]*pairingtypes.QueryGetPairingResponse
}

func NewConsumerStateQuery(ctx context.Context, clientCtx client.Context) *ConsumerStateQuery {
	csq := &ConsumerStateQuery{StateQuery: *NewStateQuery(ctx, clientCtx), clientCtx: clientCtx, cachedPairings: map[string]*pairingtypes.QueryGetPairingResponse{}}
	return csq
}
func (csq *ConsumerStateQuery) GetPairing(ctx context.Context, chainID string, latestBlock int64) (pairingList []epochstoragetypes.StakeEntry, epoch uint64, nextBlockForUpdate uint64, errRet error) {
	if chainID == "" {
		// the caller doesn;t care which so just return the first
		for key := range csq.cachedPairings {
			chainID = key
		}
		if chainID == "" {
			chainID = "LAV1"
			utils.LavaFormatWarning("failed to run get pairing as there is no cached entry for empty chainID call, using default chainID", nil, &map[string]string{"chainID": chainID})
		}
	}

	if cachedResp, ok := csq.cachedPairings[chainID]; ok {
		if cachedResp.BlockOfNextPairing > uint64(latestBlock) {
			return cachedResp.Providers, cachedResp.CurrentEpoch, cachedResp.BlockOfNextPairing, nil
		}
	}

	pairingResp, err := csq.PairingQueryClient.GetPairing(ctx, &pairingtypes.QueryGetPairingRequest{
		ChainID: chainID,
		Client:  csq.clientCtx.FromAddress.String(),
	})
	if err != nil {
		return nil, 0, 0, utils.LavaFormatError("Failed in get pairing query", err, &map[string]string{})
	}
	csq.cachedPairings[chainID] = pairingResp
	return pairingResp.Providers, pairingResp.CurrentEpoch, pairingResp.BlockOfNextPairing, nil
}

func (csq *ConsumerStateQuery) GetMaxCUForUser(ctx context.Context, chainID string, epoch uint64) (maxCu uint64, err error) {
	address := csq.clientCtx.FromAddress.String()
	UserEntryRes, err := csq.PairingQueryClient.UserEntry(ctx, &pairingtypes.QueryUserEntryRequest{ChainID: chainID, Address: address, Block: epoch})
	if err != nil {
		return 0, utils.LavaFormatError("failed querying StakeEntry for consumer", err, &map[string]string{"chainID": chainID, "address": address, "block": strconv.FormatUint(epoch, 10)})
	}
	return UserEntryRes.GetMaxCU(), nil
}

func (csq *ConsumerStateQuery) GetSpec(ctx context.Context, chainID string) (*spectypes.Spec, error) {
	spec, err := csq.SpecQueryClient.Spec(ctx, &spectypes.QueryGetSpecRequest{
		ChainID: chainID,
	})
	if err != nil {
		return nil, utils.LavaFormatError("Failed Querying spec for chain", err, &map[string]string{"ChainID": chainID})
	}
	return &spec.Spec, nil
}
