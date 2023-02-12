package statetracker

import (
	"context"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	reliabilitymanager "github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
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

func (csq *StateQuery) GetSpec(ctx context.Context, chainID string) (*spectypes.Spec, error) {
	spec, err := csq.SpecQueryClient.Spec(ctx, &spectypes.QueryGetSpecRequest{
		ChainID: chainID,
	})
	if err != nil {
		return nil, utils.LavaFormatError("Failed Querying spec for chain", err, &map[string]string{"ChainID": chainID})
	}
	return &spec.Spec, nil
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

type ProviderStateQuery struct {
	StateQuery
	clientCtx client.Context
}

func NewProviderStateQuery(ctx context.Context, clientCtx client.Context) *ProviderStateQuery {
	csq := &ProviderStateQuery{StateQuery: *NewStateQuery(ctx, clientCtx), clientCtx: clientCtx}
	return csq
}

func (psq *ProviderStateQuery) CurrentEpochStart(ctx context.Context) (uint64, error) {
	epochDetails, err := psq.EpochStorageQueryClient.EpochDetails(ctx, &epochstoragetypes.QueryGetEpochDetailsRequest{})
	if err != nil {
		return 0, utils.LavaFormatError("Failed Querying EpochDetails", err, nil)
	}
	details := epochDetails.GetEpochDetails()
	return details.StartBlock, nil

}

func (psq *ProviderStateQuery) VoteEvents(ctx context.Context, latestBlock int64) (votes []*reliabilitymanager.VoteParams, err error) {
	blockResults, err := psq.clientCtx.Client.BlockResults(ctx, &latestBlock)
	if err != nil {
		return nil, err
	}
	transactionResults := blockResults.TxsResults
	for _, tx := range transactionResults {
		events := tx.Events
		for _, event := range events {
			if event.Type == utils.EventPrefix+conflicttypes.ConflictVoteDetectionEventName {
				vote, err := reliabilitymanager.BuildVoteParamsFromDetectionEvent(event)
				if err != nil {
					return nil, err
				}
				votes = append(votes, vote)
			}
		}
	}

	beginBlockEvents := blockResults.BeginBlockEvents
	for _, event := range beginBlockEvents {
		if event.Type == utils.EventPrefix+conflicttypes.ConflictVoteRevealEventName {
			// eventToListen := utils.EventPrefix + conflicttypes.ConflictVoteRevealEventName
			// 	if votesList, ok := e.Events[eventToListen+".voteID"]; ok {
			// 		for idx, voteID := range votesList {
			// 			num_str := e.Events[eventToListen+".voteDeadline"][idx]
			// 			voteDeadline, err := strconv.ParseUint(num_str, 10, 64)
			// 			if err != nil {
			// 				utils.LavaFormatError("parsing vote deadline", err, &map[string]string{"VoteDeadline": num_str})
			// 				continue
			// 			}
			// 			go s.voteInitiationCb(ctx, voteID, voteDeadline, nil)
			// 		}
			// 	}

			// 	eventToListen = utils.EventPrefix + conflicttypes.ConflictVoteResolvedEventName
			// 	if votesList, ok := e.Events[eventToListen+".voteID"]; ok {
			// 		for _, voteID := range votesList {
			// 			voteParams := &VoteParams{CloseVote: true}
			// 			go s.voteInitiationCb(ctx, voteID, 0, voteParams)
			// 		}
			// 	}
		}
	}
	return
}
