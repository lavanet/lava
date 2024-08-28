package updaters

import (
	"context"
	"fmt"
	"strconv"
	"time"

	downtimev1 "github.com/lavanet/lava/v2/x/downtime/v1"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/dgraph-io/ristretto"
	reliabilitymanager "github.com/lavanet/lava/v2/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/v2/utils"
	conflicttypes "github.com/lavanet/lava/v2/x/conflict/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	plantypes "github.com/lavanet/lava/v2/x/plans/types"
	protocoltypes "github.com/lavanet/lava/v2/x/protocol/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	CacheMaxCost                = 10 * 1024 // 10K cost
	CacheNumCounters            = 10 * 1025 // expect 10K items
	DefaultTimeToLiveExpiration = 30 * time.Minute
	PairingRespKey              = "pairing-resp"
	VerifyPairingRespKey        = "verify-pairing-resp"
	MaxCuResponseKey            = "max-cu-resp"
	EffectivePolicyRespKey      = "effective-policy-resp"
)

type ProtocolVersionResponse struct {
	Version     *protocoltypes.Version
	BlockNumber string
}

type StateQuery struct {
	SpecQueryClient         spectypes.QueryClient
	PairingQueryClient      pairingtypes.QueryClient
	EpochStorageQueryClient epochstoragetypes.QueryClient
	ProtocolClient          protocoltypes.QueryClient
	DowntimeClient          downtimev1.QueryClient
	ResponsesCache          *ristretto.Cache
}

func NewStateQuery(ctx context.Context, clientCtx client.Context) *StateQuery {
	sq := &StateQuery{}
	sq.SpecQueryClient = spectypes.NewQueryClient(clientCtx)
	sq.PairingQueryClient = pairingtypes.NewQueryClient(clientCtx)
	sq.EpochStorageQueryClient = epochstoragetypes.NewQueryClient(clientCtx)
	sq.ProtocolClient = protocoltypes.NewQueryClient(clientCtx)
	sq.DowntimeClient = downtimev1.NewQueryClient(clientCtx)
	cache, err := ristretto.NewCache(&ristretto.Config{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64})
	if err != nil {
		utils.LavaFormatFatal("failed setting up cache for queries", err)
	}
	sq.ResponsesCache = cache
	return sq
}

func (csq *StateQuery) GetProtocolVersion(ctx context.Context) (*ProtocolVersionResponse, error) {
	header := metadata.MD{}
	param, err := csq.ProtocolClient.Params(ctx, &protocoltypes.QueryParamsRequest{}, grpc.Header(&header))
	if err != nil {
		return nil, err
	}
	blockHeight := "unInitialized"
	blockHeights := header.Get("x-cosmos-block-height")
	if len(blockHeights) > 0 {
		blockHeight = blockHeights[0]
	}
	return &ProtocolVersionResponse{BlockNumber: blockHeight, Version: &param.Params.Version}, nil
}

func (csq *StateQuery) GetSpec(ctx context.Context, chainID string) (*spectypes.Spec, error) {
	spec, err := csq.SpecQueryClient.Spec(ctx, &spectypes.QueryGetSpecRequest{
		ChainID: chainID,
	})
	if err != nil {
		return nil, utils.LavaFormatError("Failed Querying spec for chain", err, utils.Attribute{Key: "ChainID", Value: chainID})
	}
	return &spec.Spec, nil
}

func (csq *StateQuery) GetDowntimeParams(ctx context.Context) (*downtimev1.Params, error) {
	res, err := csq.DowntimeClient.QueryParams(ctx, &downtimev1.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}
	return res.Params, nil
}

type ConsumerStateQuery struct {
	StateQuery
	clientCtx   client.Context
	lastChainID string
}

func NewConsumerStateQuery(ctx context.Context, clientCtx client.Context) *ConsumerStateQuery {
	csq := &ConsumerStateQuery{StateQuery: *NewStateQuery(ctx, clientCtx), clientCtx: clientCtx, lastChainID: ""}
	return csq
}

func (csq *ConsumerStateQuery) GetEffectivePolicy(ctx context.Context, consumerAddress, specID string) (*plantypes.Policy, error) {
	cachedInterface, found := csq.ResponsesCache.Get(EffectivePolicyRespKey + specID)
	if found && cachedInterface != nil {
		if cachedResp, ok := cachedInterface.(*pairingtypes.QueryEffectivePolicyResponse); ok {
			return cachedResp.GetPolicy(), nil
		} else {
			utils.LavaFormatError("invalid cache entry - failed casting response", nil, utils.Attribute{Key: "castingType", Value: "*pairingtypes.QueryEffectivePolicyResponse"}, utils.Attribute{Key: "type", Value: cachedInterface})
		}
	}

	resp, err := csq.PairingQueryClient.EffectivePolicy(ctx, &pairingtypes.QueryEffectivePolicyRequest{
		Consumer: consumerAddress,
		SpecID:   specID,
	})
	if err != nil || resp.GetPolicy() == nil {
		return nil, err
	}
	csq.ResponsesCache.SetWithTTL(EffectivePolicyRespKey+specID, resp, 1, DefaultTimeToLiveExpiration)
	return resp.GetPolicy(), nil
}

func (csq *ConsumerStateQuery) GetPairing(ctx context.Context, chainID string, latestBlock int64) (pairingList []epochstoragetypes.StakeEntry, epoch, nextBlockForUpdate uint64, errRet error) {
	if chainID == "" {
		return nil, 0, 0, utils.LavaFormatError("chain id is empty in GetPairing while not allowed", nil, utils.Attribute{Key: "chainID", Value: chainID})
	}

	cachedInterface, found := csq.ResponsesCache.Get(PairingRespKey + chainID)
	if found && cachedInterface != nil {
		if cachedResp, ok := cachedInterface.(*pairingtypes.QueryGetPairingResponse); ok {
			if cachedResp.BlockOfNextPairing > uint64(latestBlock) {
				return cachedResp.Providers, cachedResp.CurrentEpoch, cachedResp.BlockOfNextPairing, nil
			}
		} else {
			utils.LavaFormatError("invalid cache entry - failed casting response", nil, utils.Attribute{Key: "castingType", Value: "*pairingtypes.QueryGetPairingResponse"}, utils.Attribute{Key: "type", Value: cachedInterface})
		}
	}

	pairingResp, err := csq.PairingQueryClient.GetPairing(ctx, &pairingtypes.QueryGetPairingRequest{
		ChainID: chainID,
		Client:  csq.clientCtx.FromAddress.String(),
	})
	if err != nil {
		return nil, 0, 0, utils.LavaFormatError("Failed in get pairing query", err, utils.Attribute{})
	}
	csq.lastChainID = chainID
	csq.ResponsesCache.SetWithTTL(PairingRespKey+chainID, pairingResp, 1, DefaultTimeToLiveExpiration)
	if len(pairingResp.Providers) == 0 {
		utils.LavaFormatWarning("Chain returned empty provider list, check node connection and consumer subscription status, or no providers provide this chain", nil,
			utils.LogAttr("chainId", chainID),
			utils.LogAttr("epoch", pairingResp.CurrentEpoch),
			utils.LogAttr("consumer_address", csq.clientCtx.FromAddress.String()),
		)
	}
	return pairingResp.Providers, pairingResp.CurrentEpoch, pairingResp.BlockOfNextPairing, nil
}

func (csq *ConsumerStateQuery) GetMaxCUForUser(ctx context.Context, chainID string, epoch uint64) (maxCu uint64, err error) {
	var userEntryRes *pairingtypes.QueryUserEntryResponse = nil

	key := csq.entryKey(chainID, epoch)
	cachedInterface, found := csq.ResponsesCache.Get(key)

	if found && cachedInterface != nil {
		if cachedResp, ok := cachedInterface.(*pairingtypes.QueryUserEntryResponse); ok {
			userEntryRes = cachedResp
		} else {
			utils.LavaFormatError("invalid cache entry - failed casting response", nil, utils.Attribute{Key: "castingType", Value: "*pairingtypes.QueryUserEntryResponse"}, utils.Attribute{Key: "type", Value: fmt.Sprintf("%T", cachedInterface)})
		}
	}

	if userEntryRes == nil {
		address := csq.clientCtx.FromAddress.String()
		userEntryRes, err = csq.PairingQueryClient.UserEntry(ctx, &pairingtypes.QueryUserEntryRequest{ChainID: chainID, Address: address, Block: epoch})
		if err != nil {
			return 0, utils.LavaFormatError("failed querying StakeEntry for consumer", err, utils.Attribute{Key: "chainID", Value: chainID}, utils.Attribute{Key: "address", Value: address}, utils.Attribute{Key: "block", Value: epoch})
		}

		csq.ResponsesCache.SetWithTTL(key, userEntryRes, 1, DefaultTimeToLiveExpiration)
	}

	return userEntryRes.GetMaxCU(), nil
}

func (csq *ConsumerStateQuery) entryKey(chainID string, epoch uint64) string {
	return MaxCuResponseKey + chainID + strconv.FormatUint(epoch, 10)
}

type EpochStateQuery struct {
	StateQuery
}

func (esq *EpochStateQuery) CurrentEpochStart(ctx context.Context) (uint64, error) {
	epochDetails, err := esq.EpochStorageQueryClient.EpochDetails(ctx, &epochstoragetypes.QueryGetEpochDetailsRequest{})
	if err != nil {
		return 0, utils.LavaFormatError("Failed Querying EpochDetails", err)
	}
	details := epochDetails.GetEpochDetails()
	return details.StartBlock, nil
}

func NewEpochStateQuery(stateQuery *StateQuery) *EpochStateQuery {
	return &EpochStateQuery{StateQuery: *stateQuery}
}

type ProviderStateQuery struct {
	StateQuery
	EpochStateQuery
	clientCtx client.Context
}

func NewProviderStateQuery(ctx context.Context, clientCtx client.Context) *ProviderStateQuery {
	sq := NewStateQuery(ctx, clientCtx)
	esq := NewEpochStateQuery(sq)
	csq := &ProviderStateQuery{StateQuery: *sq, EpochStateQuery: *esq, clientCtx: clientCtx}
	return csq
}

func (psq *ProviderStateQuery) GetMaxCuForUser(ctx context.Context, consumerAddress, chainID string, epoch uint64) (maxCu uint64, err error) {
	key := psq.entryKey(consumerAddress, chainID, epoch, "")
	cachedInterface, found := psq.ResponsesCache.Get(MaxCuResponseKey + key)
	var userEntryRes *pairingtypes.QueryUserEntryResponse = nil
	if found && cachedInterface != nil {
		if cachedResp, ok := cachedInterface.(*pairingtypes.QueryUserEntryResponse); ok {
			userEntryRes = cachedResp
		} else {
			utils.LavaFormatError("invalid cache entry - failed casting response", nil, utils.Attribute{Key: "castingType", Value: "*pairingtypes.QueryUserEntryResponse"}, utils.Attribute{Key: "type", Value: fmt.Sprintf("%T", cachedInterface)})
		}
	}
	if userEntryRes == nil {
		userEntryRes, err = psq.PairingQueryClient.UserEntry(ctx, &pairingtypes.QueryUserEntryRequest{ChainID: chainID, Address: consumerAddress, Block: epoch})
		if err != nil {
			return 0, utils.LavaFormatError("StakeEntry querying for consumer failed", err, utils.Attribute{Key: "chainID", Value: chainID}, utils.Attribute{Key: "address", Value: consumerAddress}, utils.Attribute{Key: "block", Value: epoch})
		}
		psq.ResponsesCache.SetWithTTL(MaxCuResponseKey+key, userEntryRes, 1, DefaultTimeToLiveExpiration)
	}

	return userEntryRes.GetMaxCU(), err
}

func (psq *ProviderStateQuery) entryKey(consumerAddress, chainID string, epoch uint64, providerAddress string) string {
	return consumerAddress + chainID + strconv.FormatUint(epoch, 10) + providerAddress
}

func (psq *ProviderStateQuery) VoteEvents(ctx context.Context, latestBlock int64) (votes []*reliabilitymanager.VoteParams, err error) {
	brp, err := TryIntoTendermintRPC(psq.clientCtx.Client)
	if err != nil {
		return nil, utils.LavaFormatError("failed to get block result provider", err)
	}
	blockResults, err := brp.BlockResults(ctx, &latestBlock)
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
					return nil, utils.LavaFormatError("failed conflict_vote_detection_event parsing", err, utils.Attribute{Key: "event", Value: event})
				}
				utils.LavaFormatDebug("conflict_vote_detection_event", utils.Attribute{Key: "voteID", Value: vote.VoteID})
				votes = append(votes, vote)
			}
		}
	}

	beginBlockEvents := blockResults.BeginBlockEvents
	for _, event := range beginBlockEvents {
		if event.Type == utils.EventPrefix+conflicttypes.ConflictVoteRevealEventName {
			voteID, voteDeadline, err := reliabilitymanager.BuildBaseVoteDataFromEvent(event)
			if err != nil {
				return nil, utils.LavaFormatError("failed conflict_vote_reveal_event parsing", err, utils.Attribute{Key: "event", Value: event})
			}
			vote_reveal := &reliabilitymanager.VoteParams{VoteID: voteID, VoteDeadline: voteDeadline, ParamsType: reliabilitymanager.RevealVoteType}
			utils.LavaFormatDebug("conflict_vote_reveal_event", utils.Attribute{Key: "voteID", Value: voteID})
			votes = append(votes, vote_reveal)
		}
		if event.Type == utils.EventPrefix+conflicttypes.ConflictVoteResolvedEventName {
			voteID, _, err := reliabilitymanager.BuildBaseVoteDataFromEvent(event)
			if err != nil {
				if !reliabilitymanager.NoVoteDeadline.Is(err) {
					return nil, utils.LavaFormatError("failed conflict_vote_resolved_event parsing", err, utils.Attribute{Key: "event", Value: event})
				}
			}
			vote_resolved := &reliabilitymanager.VoteParams{VoteID: voteID, VoteDeadline: 0, ParamsType: reliabilitymanager.CloseVoteType, CloseVote: true}
			votes = append(votes, vote_resolved)
			utils.LavaFormatDebug("conflict_vote_resolved_event", utils.Attribute{Key: "voteID", Value: voteID})
		}
	}
	return votes, err
}

func (psq *ProviderStateQuery) VerifyPairing(ctx context.Context, consumerAddress, providerAddress string, epoch uint64, chainID string) (valid bool, total int64, projectId string, err error) {
	key := psq.entryKey(consumerAddress, chainID, epoch, providerAddress)
	extractedResultFromCache := false
	cachedInterface, found := psq.ResponsesCache.Get(VerifyPairingRespKey + key)
	var verifyResponse *pairingtypes.QueryVerifyPairingResponse = nil
	if found && cachedInterface != nil {
		if cachedResp, ok := cachedInterface.(*pairingtypes.QueryVerifyPairingResponse); ok {
			verifyResponse = cachedResp
			extractedResultFromCache = true
		} else {
			utils.LavaFormatError("invalid cache entry - failed casting response", nil, utils.Attribute{Key: "castingType", Value: "*pairingtypes.QueryVerifyPairingResponse"}, utils.Attribute{Key: "type", Value: fmt.Sprintf("%T", cachedInterface)})
		}
	}
	if verifyResponse == nil {
		verifyResponse, err = psq.PairingQueryClient.VerifyPairing(context.Background(), &pairingtypes.QueryVerifyPairingRequest{
			ChainID:  chainID,
			Client:   consumerAddress,
			Provider: providerAddress,
			Block:    epoch,
		})
		if err != nil {
			return false, 0, "", err
		}
		psq.ResponsesCache.SetWithTTL(VerifyPairingRespKey+key, verifyResponse, 1, DefaultTimeToLiveExpiration)
	}
	if !verifyResponse.Valid {
		return false, 0, "", utils.LavaFormatError("invalid self pairing with consumer", nil,
			utils.LogAttr("provider", providerAddress),
			utils.LogAttr("consumer_address", consumerAddress),
			utils.LogAttr("epoch", epoch),
			utils.LogAttr("from_cache", extractedResultFromCache),
		)
	}
	return verifyResponse.Valid, int64(verifyResponse.GetPairedProviders()), verifyResponse.ProjectId, nil
}

func (psq *ProviderStateQuery) GetEpochSize(ctx context.Context) (uint64, error) {
	res, err := psq.EpochStorageQueryClient.Params(ctx, &epochstoragetypes.QueryParamsRequest{})
	if err != nil {
		return 0, err
	}
	return res.Params.EpochBlocks, nil
}

func (psq *ProviderStateQuery) EarliestBlockInMemory(ctx context.Context) (uint64, error) {
	res, err := psq.EpochStorageQueryClient.EpochDetails(ctx, &epochstoragetypes.QueryGetEpochDetailsRequest{})
	if err != nil {
		return 0, err
	}
	return res.EpochDetails.EarliestStart, nil
}

func (psq *ProviderStateQuery) GetRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error) {
	res, err := psq.PairingQueryClient.Params(ctx, &pairingtypes.QueryParamsRequest{})
	if err != nil {
		return 0, err
	}
	return res.GetParams().RecommendedEpochNumToCollectPayment, nil
}

func (psq *ProviderStateQuery) GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error) {
	epochSize, err := psq.GetEpochSize(ctx)
	if err != nil {
		return 0, err
	}
	recommendedEpochNumToCollectPayment, err := psq.GetRecommendedEpochNumToCollectPayment(ctx)
	if err != nil {
		return 0, err
	}
	return epochSize * recommendedEpochNumToCollectPayment, nil
}
