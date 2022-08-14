package sentry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coniks-sys/coniks-go/crypto/vrf"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/lavanet/lava/x/spec/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/spf13/pflag"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tenderminttypes "github.com/tendermint/tendermint/types"
	"golang.org/x/exp/slices"
	grpc "google.golang.org/grpc"
)

type ClientSession struct {
	CuSum                 uint64
	QoSInfo               QoSInfo
	SessionId             int64
	Client                *RelayerClientWrapper
	Lock                  sync.Mutex
	RelayNum              uint64
	LatestBlock           int64
	FinalizedBlocksHashes map[int64]string
}

type QoSInfo struct {
	LastQoSReport      *pairingtypes.QualityOfServiceReport
	LatencyScoreList   []sdk.Dec
	SyncScoreSum       int64
	TotalSyncScore     int64
	TotalRelays        uint64
	AnsweredRelays     uint64
	ConsecutiveTimeOut uint64
}

type VoteParams struct {
	CloseVote      bool
	ChainID        string
	ApiURL         string
	RequestData    []byte
	RequestBlock   uint64
	Voters         []string
	ConnectionType string
}

func (vp *VoteParams) GetCloseVote() bool {
	if vp == nil {
		//default returns false
		return false
	}
	return vp.CloseVote
}

//Constants

var AvailabilityPercentage sdk.Dec = sdk.NewDecWithPrec(5, 2) //TODO move to params pairing
const (
	MaxConsecutiveConnectionAttemts = 3
	PercentileToCalculateLatency    = 0.9
	MinProvidersForSync             = 0.6
	LatencyThresholdStatic          = 1 * time.Second
	LatencyThresholdSlope           = 1 * time.Millisecond
	StaleEpochDistance              = 3 // relays done 3 epochs back are ready to be rewarded
)

type RelayerClientWrapper struct {
	Client *pairingtypes.RelayerClient
	Acc    string //public lava address
	Addr   string //ip:port

	ConnectionRefusals uint64
	SessionsLock       sync.Mutex
	Sessions           map[int64]*ClientSession
	MaxComputeUnits    uint64
	UsedComputeUnits   uint64
	ReliabilitySent    bool
	PairingEpoch       uint64
}

type PaymentRequest struct {
	CU                  uint64
	BlockHeightDeadline int64
	Amount              sdk.Coin
	Client              sdk.AccAddress
	UniqueIdentifier    uint64
}

type providerDataContainer struct {
	// keep all data used to sign sigblocks
	LatestFinalizedBlock  int64
	LatestBlockTime       time.Time
	FinalizedBlocksHashes map[int64]string
	SigBlocks             []byte
	SessionId             uint64
	BlockHeight           int64
	RelayNum              uint64
	LatestBlock           int64
	//TODO:: keep relay request for conflict reporting
}

type ProviderHashesConsensus struct {
	FinalizedBlocksHashes map[int64]string
	agreeingProviders     map[string]providerDataContainer
}

type Sentry struct {
	ClientCtx               client.Context
	rpcClient               rpcclient.Client
	specQueryClient         spectypes.QueryClient
	pairingQueryClient      pairingtypes.QueryClient
	epochStorageQueryClient epochstoragetypes.QueryClient
	ChainID                 string
	NewTransactionEvents    <-chan ctypes.ResultEvent
	NewBlockEvents          <-chan ctypes.ResultEvent
	isUser                  bool
	Acc                     string // account address (bech32)
	voteInitiationCb        func(ctx context.Context, voteID string, voteDeadline uint64, voteParams *VoteParams)
	newEpochCb              func(epochHeight int64)
	ApiInterface            string
	cmdFlags                *pflag.FlagSet
	serverID                uint64
	//
	// expected payments storage
	PaymentsMu       sync.RWMutex
	expectedPayments []PaymentRequest
	receivedPayments []PaymentRequest
	totalCUServiced  uint64
	totalCUPaid      uint64

	// server Blocks To Save (atomic)
	earliestSavedBlock uint64
	// Block storage (atomic)
	blockHeight        int64
	currentEpoch       uint64
	EpochSize          uint64
	EpochBlocksOverlap uint64
	providersCount     uint64
	//
	// Spec storage (rw mutex)
	specMu     sync.RWMutex
	specHash   []byte
	serverSpec spectypes.Spec
	serverApis map[string]spectypes.ServiceApi
	taggedApis map[string]spectypes.ServiceApi

	// (client only)
	// Pairing storage (rw mutex)
	pairingMu            sync.RWMutex
	pairingNextMu        sync.RWMutex
	pairing              []*RelayerClientWrapper
	PairingBlockStart    int64
	pairingAddresses     []string
	pairingPurgeLock     sync.Mutex
	pairingPurge         []*RelayerClientWrapper
	pairingNext          []*RelayerClientWrapper
	pairingNextAddresses []string
	VrfSkMu              sync.Mutex
	VrfSk                vrf.PrivateKey

	// every entry in providerHashesConsensus is conflicted with the other entries
	providerHashesConsensus          []ProviderHashesConsensus
	prevEpochProviderHashesConsensus []ProviderHashesConsensus
	providerDataContainersMu         sync.Mutex
}

func (cs *ClientSession) CalculateQoS(cu uint64, latency time.Duration, blockHeightDiff int64, numOfProviders int, servicersToCount int64) {

	if cs.QoSInfo.LastQoSReport == nil {
		cs.QoSInfo.LastQoSReport = &pairingtypes.QualityOfServiceReport{}
	}

	downtimePrecentage := sdk.NewDecWithPrec(int64(cs.QoSInfo.TotalRelays-cs.QoSInfo.AnsweredRelays), 0).Quo(sdk.NewDecWithPrec(int64(cs.QoSInfo.TotalRelays), 0))
	cs.QoSInfo.LastQoSReport.Availability = sdk.MaxDec(sdk.ZeroDec(), AvailabilityPercentage.Sub(downtimePrecentage).Quo(AvailabilityPercentage))
	if sdk.OneDec().GT(cs.QoSInfo.LastQoSReport.Availability) {
		utils.LavaFormatInfo("QoS Availability report", &map[string]string{"Availibility": cs.QoSInfo.LastQoSReport.Availability.String(), "down percent": downtimePrecentage.String()})
	}

	var latencyThreshold time.Duration = LatencyThresholdStatic + time.Duration(cu)*LatencyThresholdSlope
	latencyScore := sdk.MinDec(sdk.OneDec(), sdk.NewDecFromInt(sdk.NewInt(int64(latencyThreshold))).Quo(sdk.NewDecFromInt(sdk.NewInt(int64(latency)))))

	insertSorted := func(list []sdk.Dec, value sdk.Dec) []sdk.Dec {
		index := sort.Search(len(list), func(i int) bool {
			return list[i].GTE(value)
		})
		if len(list) == index { // nil or empty slice or after last element
			return append(list, value)
		}
		list = append(list[:index+1], list[index:]...) // index < len(a)
		list[index] = value
		return list
	}
	cs.QoSInfo.LatencyScoreList = insertSorted(cs.QoSInfo.LatencyScoreList, latencyScore)
	cs.QoSInfo.LastQoSReport.Latency = cs.QoSInfo.LatencyScoreList[int(float64(len(cs.QoSInfo.LatencyScoreList))*PercentileToCalculateLatency)]

	if int64(numOfProviders) > int64(math.Ceil(float64(servicersToCount)*MinProvidersForSync)) { //
		if blockHeightDiff <= 0 { //if the diff is bigger than 0 than the block is too old (blockHeightDiff = expected - allowedLag - blockheight) and we dont give him the score
			cs.QoSInfo.SyncScoreSum++
		}
	} else {
		cs.QoSInfo.SyncScoreSum++
	}
	cs.QoSInfo.TotalSyncScore++

	cs.QoSInfo.LastQoSReport.Sync = sdk.NewDec(cs.QoSInfo.SyncScoreSum).QuoInt64(cs.QoSInfo.TotalSyncScore)

	if sdk.OneDec().GT(cs.QoSInfo.LastQoSReport.Sync) {
		utils.LavaFormatInfo("QoS Sync report",
			&map[string]string{"Sync": cs.QoSInfo.LastQoSReport.Sync.String(),
				"block diff": strconv.FormatInt(blockHeightDiff, 10),
				"sync score": strconv.FormatInt(cs.QoSInfo.SyncScoreSum, 10) + "/" + strconv.FormatInt(cs.QoSInfo.TotalSyncScore, 10)})
	}
}

func (r *RelayerClientWrapper) GetPairingEpoch() uint64 {
	return atomic.LoadUint64(&r.PairingEpoch)
}

func (s *Sentry) FetchProvidersCount(ctx context.Context) error {
	res, err := s.pairingQueryClient.Params(ctx, &pairingtypes.QueryParamsRequest{})
	if err != nil {
		return err
	}
	atomic.StoreUint64(&s.providersCount, res.GetParams().ServicersToPairCount)
	return nil
}

func (s *Sentry) GetProvidersCount() uint64 {
	return atomic.LoadUint64(&s.providersCount)
}

func (s *Sentry) GetEpochSize() uint64 {
	return atomic.LoadUint64(&s.EpochSize)
}

func (s *Sentry) FetchEpochSize(ctx context.Context) error {
	res, err := s.epochStorageQueryClient.Params(ctx, &epochstoragetypes.QueryParamsRequest{})
	if err != nil {
		return err
	}
	atomic.StoreUint64(&s.EpochSize, res.GetParams().EpochBlocks)

	return nil
}

func (s *Sentry) FetchOverlapSize(ctx context.Context) error {
	res, err := s.pairingQueryClient.Params(ctx, &pairingtypes.QueryParamsRequest{})
	if err != nil {
		return err
	}
	atomic.StoreUint64(&s.EpochBlocksOverlap, res.GetParams().EpochBlocksOverlap)
	return nil
}

func (s *Sentry) FetchEpochParams(ctx context.Context) error {
	res, err := s.epochStorageQueryClient.EpochDetails(ctx, &epochstoragetypes.QueryGetEpochDetailsRequest{})
	if err != nil {
		return err
	}
	earliestBlock := res.GetEpochDetails().EarliestStart
	currentEpoch := res.GetEpochDetails().StartBlock
	atomic.StoreUint64(&s.earliestSavedBlock, earliestBlock)
	atomic.StoreUint64(&s.currentEpoch, currentEpoch)
	return nil
}

func (s *Sentry) handlePairingChange(ctx context.Context, blockHeight int64, init bool) error {
	if !s.isUser {
		return nil
	}

	// switch pairing every epochSize blocks
	if uint64(blockHeight) < s.GetCurrentEpochHeight()+s.GetOverlapSize() && !init {
		return nil
	}

	s.pairingNextMu.Lock()
	defer s.pairingNextMu.Unlock()

	// If we entered this handler more than once then the pairing was already changed
	if len(s.pairingNext) == 0 {
		return nil
	}

	s.pairingMu.Lock()
	defer s.pairingMu.Unlock()
	s.pairingPurgeLock.Lock()
	defer s.pairingPurgeLock.Unlock()

	s.pairingPurge = append(s.pairingPurge, s.pairing...) // append old connections to purge list
	s.PairingBlockStart = blockHeight
	s.pairing = s.pairingNext
	s.pairingAddresses = s.pairingNextAddresses
	s.pairingNext = []*RelayerClientWrapper{}

	// Time to reset the consensuses for this pairing epoch
	s.providerDataContainersMu.Lock()
	s.prevEpochProviderHashesConsensus = s.providerHashesConsensus
	s.providerHashesConsensus = make([]ProviderHashesConsensus, 0)
	s.providerDataContainersMu.Unlock()
	return nil
}

func (s *Sentry) getPairing(ctx context.Context) error {
	//
	// sentry for server module does not need a pairing
	if !s.isUser {
		return nil
	}

	//
	// Get
	res, err := s.pairingQueryClient.GetPairing(ctx, &pairingtypes.QueryGetPairingRequest{
		ChainID: s.GetChainID(),
		Client:  s.Acc,
	})
	if err != nil {
		return utils.LavaFormatError("Failed in get pairing query", err, &map[string]string{})
	}

	providers := res.GetProviders()
	if len(providers) == 0 {
		return utils.LavaFormatError("no providers found in pairing, returned empty list", nil, &map[string]string{})
	}

	//
	// Set
	pairing := []*RelayerClientWrapper{}
	pairingAddresses := []string{} //this object will not be mutated for vrf calculations
	for _, provider := range providers {
		//
		// Sanity
		providerEndpoints := provider.GetEndpoints()
		if len(providerEndpoints) == 0 {
			utils.LavaFormatError("skipping provider with no endoints", nil, &map[string]string{"Address": provider.Address, "ChainID": provider.Chain})
			continue
		}

		relevantEndpoints := []epochstoragetypes.Endpoint{}
		for _, endpoint := range providerEndpoints {
			//only take into account endpoints that use the same api interface
			if endpoint.UseType == s.ApiInterface {
				relevantEndpoints = append(relevantEndpoints, endpoint)
			}
		}
		if len(relevantEndpoints) == 0 {
			utils.LavaFormatError("skipping provider, No relevant endpoints for apiInterface", nil, &map[string]string{"Address": provider.Address, "ChainID": provider.Chain, "apiInterface": s.ApiInterface, "Endpoints": fmt.Sprintf("%v", providerEndpoints)})
			continue
		}

		maxcu, err := s.GetMaxCUForUser(ctx, s.Acc, provider.Chain)
		if err != nil {
			return utils.LavaFormatError("Failed getting max CU for user", err, &map[string]string{"Address": s.Acc, "ChainID": provider.Chain})
		}
		//
		// TODO: decide how to use multiple addresses from the same operator
		pairing = append(pairing, &RelayerClientWrapper{
			Acc:                provider.Address,
			Addr:               relevantEndpoints[0].IPPORT,
			Sessions:           map[int64]*ClientSession{},
			MaxComputeUnits:    maxcu,
			ReliabilitySent:    false,
			ConnectionRefusals: 0,
			PairingEpoch:       s.GetCurrentEpochHeight(),
		})
		pairingAddresses = append(pairingAddresses, provider.Address)
	}

	// replace previous pairing with new providers
	s.pairingNextMu.Lock()
	s.pairingNext = pairing
	s.pairingNextAddresses = pairingAddresses
	s.pairingNextMu.Unlock()
	return nil
}

func (s *Sentry) GetSpecHash() []byte {
	s.specMu.Lock()
	defer s.specMu.Unlock()
	return s.specHash
}

func (s *Sentry) GetAllSpecNames(ctx context.Context) (map[string][]types.ApiInterface, error) {
	spec, err := s.specQueryClient.Chain(ctx, &spectypes.QueryChainRequest{
		ChainID: s.ChainID,
	})
	if err != nil {
		return nil, utils.LavaFormatError("Failed Querying spec for chain", err, &map[string]string{"ChainID": s.ChainID})
	}
	serverApis, _ := s.getServiceApis(spec)
	allSpecNames := make(map[string][]types.ApiInterface)
	for _, api := range serverApis {
		allSpecNames[api.Name] = api.ApiInterfaces
	}
	return allSpecNames, nil
}

func (s *Sentry) getServiceApis(spec *spectypes.QueryChainResponse) (retServerApis map[string]spectypes.ServiceApi, retTaggedApis map[string]spectypes.ServiceApi) {
	serverApis := map[string]spectypes.ServiceApi{}
	taggedApis := map[string]spectypes.ServiceApi{}
	if spec.Spec.Enabled {
		for _, api := range spec.Spec.Apis {
			if !api.Enabled {
				continue
			}
			//
			// TODO: find a better spot for this (more optimized, precompile regex, etc)
			for _, apiInterface := range api.ApiInterfaces {
				if apiInterface.Interface != s.ApiInterface {
					//spec will contain many api interfaces, we only need those that belong to the apiInterface of this sentry
					continue
				}
				if apiInterface.Interface == "rest" {
					re := regexp.MustCompile(`{[^}]+}`)
					processedName := string(re.ReplaceAll([]byte(api.Name), []byte("replace-me-with-regex")))
					processedName = regexp.QuoteMeta(processedName)
					processedName = strings.ReplaceAll(processedName, "replace-me-with-regex", `[^\/\s]+`)
					serverApis[processedName] = api
				} else {
					serverApis[api.Name] = api
				}

				if api.Parsing.GetFunctionTag() != "" {
					taggedApis[api.Parsing.GetFunctionTag()] = api
				}
			}
		}
	}
	return serverApis, taggedApis
}

func (s *Sentry) getSpec(ctx context.Context) error {
	//
	// TODO: decide if it's fatal to not have spec (probably!)
	spec, err := s.specQueryClient.Chain(ctx, &spectypes.QueryChainRequest{
		ChainID: s.ChainID,
	})
	if err != nil {
		return utils.LavaFormatError("Failed Querying spec for chain", err, &map[string]string{"ChainID": s.ChainID})
	}

	//
	// Check if updated
	hash := tendermintcrypto.Sha256([]byte(spec.String())) // TODO: we use cheaper algo for speed
	if bytes.Equal(s.specHash, hash) {
		//spec for chain didnt change
		return nil
	}
	s.specHash = hash

	//
	// Update
	utils.LavaFormatInfo("Sentry updated spec", &map[string]string{"ChainID": spec.Spec.Index, "spec name": spec.Spec.Name})
	serverApis, taggedApis := s.getServiceApis(spec)

	s.specMu.Lock()
	defer s.specMu.Unlock()
	s.serverSpec = spec.Spec
	s.serverApis = serverApis
	s.taggedApis = taggedApis

	return nil
}

func (s *Sentry) Init(ctx context.Context) error {
	//
	// New client
	err := s.rpcClient.Start()
	if err != nil {
		return err
	}

	//
	// Listen to new blocks
	query := "tm.event = 'NewBlock'"
	//
	txs, err := s.rpcClient.Subscribe(ctx, "test-client", query)
	if err != nil {
		return utils.LavaFormatError("Failed subscribing to new blocks", err, &map[string]string{})
	}
	s.NewBlockEvents = txs

	query = "tm.event = 'Tx'"
	txs, err = s.rpcClient.Subscribe(ctx, "test-client", query)
	if err != nil {
		return utils.LavaFormatError("Failed subscribing to transactions", err, &map[string]string{})
	}
	s.NewTransactionEvents = txs
	//
	// Get spec for the first time
	err = s.getSpec(ctx)
	if err != nil {
		return utils.LavaFormatError("Failed getting spec in initialization", err, &map[string]string{})
	}

	s.FetchChainParams(ctx)

	//
	// Get pairing for the first time, for clients
	err = s.getPairing(ctx)
	if err != nil {
		return utils.LavaFormatError("Failed getting pairing for consumer in initialization", err, &map[string]string{"Address": s.Acc})
	}

	s.handlePairingChange(ctx, 0, true)

	//
	// Sanity
	if !s.isUser {
		providers, err := s.pairingQueryClient.Providers(ctx, &pairingtypes.QueryProvidersRequest{
			ChainID: s.GetChainID(),
		})
		if err != nil {
			return utils.LavaFormatError("failed querying providers for spec", err, &map[string]string{"spec name": s.GetSpecName(), "ChainID": s.GetChainID()})
		}
		found := false
		for _, provider := range providers.GetStakeEntry() {
			if provider.Address == s.Acc {
				found = true
				break
			}
		}
		if !found {
			return utils.LavaFormatError("provider stake verification mismatch", err, &map[string]string{"spec name": s.GetSpecName(), "ChainID": s.GetChainID()})
		}
	}

	return nil
}

func (s *Sentry) ListenForTXEvents(ctx context.Context) {
	for e := range s.NewTransactionEvents {

		switch data := e.Data.(type) {
		case tenderminttypes.EventDataTx:
			//got new TX event
			if providerAddrList, ok := e.Events["lava_relay_payment.provider"]; ok {
				for idx, providerAddr := range providerAddrList {
					if s.Acc == providerAddr && s.ChainID == e.Events["lava_relay_payment.chainID"][idx] {
						utils.LavaFormatInfo("Received relay payment",
							&map[string]string{"Amount": e.Events["lava_relay_payment.Mint"][idx],
								"CU": e.Events["lava_relay_payment.CU"][idx],
							})
						CU := e.Events["lava_relay_payment.CU"][idx]
						paidCU, err := strconv.ParseUint(CU, 10, 64)
						if err != nil {
							fmt.Printf("failed to parse event: %s\n", e.Events["lava_relay_payment.CU"])
							continue
						}
						clientAddr, err := sdk.AccAddressFromBech32(e.Events["lava_relay_payment.client"][idx])
						if err != nil {
							fmt.Printf("failed to parse event: %s\n", e.Events["lava_relay_payment.client"])
							continue
						}
						coin, err := sdk.ParseCoinNormalized(e.Events["lava_relay_payment.Mint"][idx])
						if err != nil {
							fmt.Printf("failed to parse event: %s\n", e.Events["lava_relay_payment.Mint"])
							continue
						}
						uniqueID, err := strconv.ParseUint(e.Events["lava_relay_payment.uniqueIdentifier"][idx], 10, 64)
						if err != nil {
							fmt.Printf("failed to parse event: %s\n", e.Events["lava_relay_payment.uniqueIdentifier"])
							continue
						}
						serverID, err := strconv.ParseUint(e.Events["lava_relay_payment.descriptionString"][idx], 10, 64)
						if err != nil {
							fmt.Printf("failed to parse event: %s\n", e.Events["lava_relay_payment.descriptionString"])
							continue
						}

						if serverID == s.serverID {
							s.UpdatePaidCU(paidCU)
							s.AppendToReceivedPayments(PaymentRequest{CU: paidCU, BlockHeightDeadline: data.Height, Amount: coin, Client: clientAddr, UniqueIdentifier: uniqueID})
							found := s.RemoveExpectedPayment(paidCU, clientAddr, data.Height, uniqueID)
							if !found {
								fmt.Printf("ERROR: payment received, did not find matching expectancy from correct client Need to add support for partial payment\n %s", s.PrintExpectedPAyments())
							} else {
								fmt.Printf("SUCCESS: payment received as expected\n")
							}
						}
					}
				}
			}

			eventToListen := utils.EventPrefix + conflicttypes.ConflictVoteDetectionEventName
			// listen for vote commit event from tx handler on conflict/detection
			if newVotesList, ok := e.Events[eventToListen+".voteID"]; ok {
				for idx, voteID := range newVotesList {
					chainID := e.Events[eventToListen+".chainID"][idx]
					apiURL := e.Events[eventToListen+".apiURL"][idx]
					requestData := []byte(e.Events[eventToListen+".requestData"][idx])
					connectionType := e.Events[eventToListen+".connectionType"][idx]
					num_str := e.Events[eventToListen+".requestBlock"][idx]
					requestBlock, err := strconv.ParseUint(num_str, 10, 64)
					if err != nil {
						log.Printf("Error: requested block could not be parsed as uint64 %s\n", num_str)
						continue
					}
					num_str = e.Events[eventToListen+".voteDeadline"][idx]
					voteDeadline, err := strconv.ParseUint(num_str, 10, 64)
					if err != nil {
						log.Printf("Error: parsing vote deadline %s, err:%s\n", num_str, err)
						continue
					}
					voters_st := e.Events[eventToListen+".voters"][idx]
					voters := strings.Split(voters_st, ",")
					voteParams := &VoteParams{
						ChainID:        chainID,
						ApiURL:         apiURL,
						RequestData:    requestData,
						RequestBlock:   requestBlock,
						Voters:         voters,
						CloseVote:      false,
						ConnectionType: connectionType,
					}
					go s.voteInitiationCb(ctx, voteID, voteDeadline, voteParams)
				}
			}

		}
	}
}

func (s *Sentry) RemoveExpectedPayment(paidCUToFInd uint64, expectedClient sdk.AccAddress, blockHeight int64, uniqueID uint64) bool {
	s.PaymentsMu.Lock()
	defer s.PaymentsMu.Unlock()
	for idx, expectedPayment := range s.expectedPayments {
		//TODO: make sure the payment is not too far from expected block, expectedPayment.BlockHeightDeadline == blockHeight
		if expectedPayment.CU == paidCUToFInd && expectedPayment.Client.Equals(expectedClient) && uniqueID == expectedPayment.UniqueIdentifier {
			//found payment for expected payment
			s.expectedPayments[idx] = s.expectedPayments[len(s.expectedPayments)-1] // replace the element at delete index with the last one
			s.expectedPayments = s.expectedPayments[:len(s.expectedPayments)-1]     // remove last element
			return true
		}
	}
	return false
}

func (s *Sentry) GetPaidCU() uint64 {
	return atomic.LoadUint64(&s.totalCUPaid)
}

func (s *Sentry) UpdatePaidCU(extraPaidCU uint64) {
	//we lock because we dont want the value changing after we read it before we store
	s.PaymentsMu.Lock()
	defer s.PaymentsMu.Unlock()
	currentCU := atomic.LoadUint64(&s.totalCUPaid)
	atomic.StoreUint64(&s.totalCUPaid, currentCU+extraPaidCU)
}

func (s *Sentry) AppendToReceivedPayments(paymentReq PaymentRequest) {
	s.PaymentsMu.Lock()
	defer s.PaymentsMu.Unlock()
	s.receivedPayments = append(s.receivedPayments, paymentReq)
}
func (s *Sentry) PrintExpectedPAyments() string {
	s.PaymentsMu.Lock()
	defer s.PaymentsMu.Unlock()
	return fmt.Sprintf("last Received: %v\n Expected: %v\n", s.receivedPayments[len(s.receivedPayments)-1], s.expectedPayments)
}

func (s *Sentry) Start(ctx context.Context) {

	if !s.isUser {
		//listen for transactions for proof of relay payment
		go s.ListenForTXEvents(ctx)
	}
	//
	// Purge finished sessions
	if s.isUser {
		ticker := time.NewTicker(5 * time.Second)
		quit := make(chan struct{})
		go func() {
			for {
				select {
				case <-ticker.C:
					func() {
						s.pairingPurgeLock.Lock()
						defer s.pairingPurgeLock.Unlock()

						for i := len(s.pairingPurge) - 1; i >= 0; i-- {
							client := s.pairingPurge[i]
							client.SessionsLock.Lock()

							//
							// remove done sessions
							removeList := []int64{}
							for k, sess := range client.Sessions {
								if sess.Lock.TryLock() {
									removeList = append(removeList, k)
								}
							}
							for _, i := range removeList {
								sess := client.Sessions[i]
								delete(client.Sessions, i)
								sess.Lock.Unlock()
							}

							//
							// remove empty client (TODO: efficiently delete)
							if len(client.Sessions) == 0 {
								s.pairingPurge = append(s.pairingPurge[:i], s.pairingPurge[i+1:]...)
							}
							client.SessionsLock.Unlock()
						}
					}()

				case <-quit:
					ticker.Stop()
					return
				}
			}
		}()
	}
	//
	// Listen for blockchain events
	for e := range s.NewBlockEvents {
		switch data := e.Data.(type) {
		case tenderminttypes.EventDataNewBlock:
			//
			// Update block
			s.SetBlockHeight(data.Block.Height)

			if _, ok := e.Events["lava_new_epoch.height"]; ok {
				fmt.Printf("New epoch: Height: %d \n", data.Block.Height)

				s.FetchChainParams(ctx)

				if s.newEpochCb != nil {
					go s.newEpochCb(data.Block.Height - StaleEpochDistance*int64(s.GetEpochSize())) // Currently this is only askForRewards
				}

				//
				// Update specs
				err := s.getSpec(ctx)
				if err != nil {
					log.Println("error: getSpec", err)
				}

				//update expected payments deadline, and log missing payments
				//TODO: make this from the event lava_earliest_epoch instead
				if !s.isUser {
					s.IdentifyMissingPayments(ctx)
				}
				//
				// Update pairing
				err = s.getPairing(ctx)
				if err != nil {
					log.Println("error: getPairing", err)
				}
			}

			s.handlePairingChange(ctx, data.Block.Height, false)

			if !s.isUser {
				// listen for vote reveal event from new block handler on conflict/module.go
				eventToListen := utils.EventPrefix + conflicttypes.ConflictVoteRevealEventName
				if votesList, ok := e.Events[eventToListen+".voteID"]; ok {
					for idx, voteID := range votesList {
						num_str := e.Events[eventToListen+".voteDeadline"][idx]
						voteDeadline, err := strconv.ParseUint(num_str, 10, 64)
						if err != nil {
							fmt.Printf("ERROR: parsing vote deadline %s, err:%s\n", num_str, err)
							continue
						}
						go s.voteInitiationCb(ctx, voteID, voteDeadline, nil)
					}
				}

				eventToListen = utils.EventPrefix + conflicttypes.ConflictVoteResolvedEventName
				if votesList, ok := e.Events[eventToListen+".voteID"]; ok {
					for _, voteID := range votesList {
						voteParams := &VoteParams{CloseVote: true}
						go s.voteInitiationCb(ctx, voteID, 0, voteParams)
					}
				}
			}

		}
	}
}

func (s *Sentry) FetchChainParams(ctx context.Context) {
	s.FetchEpochSize(ctx)
	s.FetchOverlapSize(ctx)
	s.FetchEpochParams(ctx)
	s.FetchProvidersCount(ctx)
}

func (s *Sentry) IdentifyMissingPayments(ctx context.Context) {
	lastBlockInMemory := atomic.LoadUint64(&s.earliestSavedBlock)
	s.PaymentsMu.RLock()
	defer s.PaymentsMu.RUnlock()
	for _, expectedPay := range s.expectedPayments {
		if uint64(expectedPay.BlockHeightDeadline) < lastBlockInMemory {
			fmt.Printf("ERROR: Identified Missing Payment for CU %d on Block %d current earliestBlockInMemory: %d\n", expectedPay.CU, expectedPay.BlockHeightDeadline, lastBlockInMemory)
		}
	}
	fmt.Printf("total CU serviced: %d, total CU paid: %d\n", s.GetCUServiced(), s.GetPaidCU())
}

// expecting caller to lock
func (s *Sentry) AddExpectedPayment(expectedPay PaymentRequest) {
	s.PaymentsMu.Lock()
	defer s.PaymentsMu.Unlock()
	s.expectedPayments = append(s.expectedPayments, expectedPay)
}

func (s *Sentry) connectRawClient(ctx context.Context, addr string) (*pairingtypes.RelayerClient, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(connectCtx, addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	/*defer conn.Close()*/

	c := pairingtypes.NewRelayerClient(conn)
	return &c, nil
}

func (s *Sentry) CheckAndMarkReliabilityForThisPairing(wrap *RelayerClientWrapper) (valid bool) {
	wrap.SessionsLock.Lock()
	defer wrap.SessionsLock.Unlock()
	if wrap.ReliabilitySent {
		return false
	}
	wrap.ReliabilitySent = true
	return true
}

func (s *Sentry) specificPairing(ctx context.Context, address string) (*RelayerClientWrapper, int, error) {
	s.pairingMu.RLock()
	defer s.pairingMu.RUnlock()
	if len(s.pairing) == 0 {
		return nil, -1, errors.New("no pairings available")
	}
	//
	for index, wrap := range s.pairing {
		if wrap.Acc != address {
			continue
		}
		if wrap.Client == nil {
			wrap.SessionsLock.Lock()
			defer wrap.SessionsLock.Unlock()

			conn, err := s.connectRawClient(ctx, wrap.Addr)
			if err != nil {
				return nil, -1, fmt.Errorf("error making initial connection to provider: %s, error: %w", wrap.Addr, err)
			}
			wrap.Client = conn
		}
		return wrap, index, nil
	}
	return nil, -1, fmt.Errorf("did not find requested address")
}

func (s *Sentry) _findPairing(ctx context.Context) (*RelayerClientWrapper, int, error) {

	s.pairingMu.RLock()

	defer s.pairingMu.RUnlock()
	if len(s.pairing) <= 0 {
		return nil, -1, errors.New("no pairings available")
	}

	//
	maxAttempts := len(s.pairing) * MaxConsecutiveConnectionAttemts
	for attempts := 0; attempts <= maxAttempts; attempts++ {
		if len(s.pairing) == 0 {
			return nil, -1, fmt.Errorf("pairing list is empty")
		}

		index := rand.Intn(len(s.pairing))
		wrap := s.pairing[index]

		if wrap.Client == nil {
			wrap.SessionsLock.Lock()

			conn, err := s.connectRawClient(ctx, wrap.Addr)
			if err != nil {
				wrap.ConnectionRefusals++
				fmt.Printf("Error getting pairing from: %s, error: %s \n", wrap.Addr, err.Error())
				if wrap.ConnectionRefusals >= MaxConsecutiveConnectionAttemts {
					s.pairingMu.RUnlock() // we release read lock here, we assume pairing can change in movePairingEntryToPurge and it needs rw lock
					s.movePairingEntryToPurge(wrap, index)
					s.pairingMu.RLock() // we resume read lock here, so we can continue
					fmt.Printf("moving %s to purge list after max consecutive tries\n", wrap.Addr)
				}

				wrap.SessionsLock.Unlock()
				continue
			}
			wrap.ConnectionRefusals = 0
			wrap.Client = conn
			wrap.SessionsLock.Unlock()
		}
		return wrap, index, nil
	}
	return nil, -1, fmt.Errorf("error getting pairing from all providers in pairing")
}

func (s *Sentry) CompareRelaysAndReportConflict(reply0 *pairingtypes.RelayReply, request0 *pairingtypes.RelayRequest, reply1 *pairingtypes.RelayReply, request1 *pairingtypes.RelayRequest) (ok bool) {
	compare_result := bytes.Compare(reply0.Data, reply1.Data)
	if compare_result == 0 {
		//they have equal data
		return true
	}
	//they have different data! report!
	log.Println(fmt.Sprintf("[-] DataReliability detected mismatching results! \n1>>%s \n2>>%s\nReporting...", reply0.Data, reply1.Data))
	responseConflict := conflicttypes.ResponseConflict{ConflictRelayData0: &conflicttypes.ConflictRelayData{Reply: reply0, Request: request0},
		ConflictRelayData1: &conflicttypes.ConflictRelayData{Reply: reply1, Request: request1}}
	msg := conflicttypes.NewMsgDetection(s.Acc, nil, &responseConflict, nil)
	s.ClientCtx.SkipConfirm = true
	txFactory := tx.NewFactoryCLI(s.ClientCtx, s.cmdFlags).WithChainID("lava")
	tx.GenerateOrBroadcastTxWithFactory(s.ClientCtx, txFactory, msg)
	//report the conflict
	return false
}

func (s *Sentry) DataReliabilityThresholdToAddress(vrf0 []byte, vrf1 []byte) (address0 string, address1 string) {
	// check for the VRF thresholds and if holds true send a relay to the provider
	//TODO: improve with blacklisted address, and the module-1
	s.specMu.RLock()
	reliabilityThreshold := s.serverSpec.ReliabilityThreshold
	s.specMu.RUnlock()
	s.pairingMu.RLock()

	providersCount := uint32(len(s.pairingAddresses))
	index0 := utils.GetIndexForVrf(vrf0, providersCount, reliabilityThreshold)
	index1 := utils.GetIndexForVrf(vrf1, providersCount, reliabilityThreshold)
	parseIndex := func(idx int64) (address string) {
		if idx == -1 {
			address = ""
		} else {
			address = s.pairingAddresses[idx]
		}
		return
	}
	address0 = parseIndex(index0)
	address1 = parseIndex(index1)
	s.pairingMu.RUnlock()
	if address0 == address1 {
		//can't have both with the same provider
		address1 = ""
	}
	return
}

func (s *Sentry) discrepancyChecker(finalizedBlocksA map[int64]string, consensus ProviderHashesConsensus) (bool, error) {
	var toIterate map[int64]string   // the smaller map between the two to compare
	var otherBlocks map[int64]string // the other map

	if len(finalizedBlocksA) < len(consensus.FinalizedBlocksHashes) {
		toIterate = finalizedBlocksA
		otherBlocks = consensus.FinalizedBlocksHashes
	} else {
		toIterate = consensus.FinalizedBlocksHashes
		otherBlocks = finalizedBlocksA
	}

	// Iterate over smaller array, looks for mismatching hashes between the inputs
	for blockNum, blockHash := range toIterate {
		if otherHash, ok := otherBlocks[blockNum]; ok {
			if blockHash != otherHash {
				//
				// TODO:: Fill msg with incriminating data
				msg := conflicttypes.NewMsgDetection(s.Acc, nil, nil, nil)
				s.ClientCtx.SkipConfirm = true
				txFactory := tx.NewFactoryCLI(s.ClientCtx, s.cmdFlags).WithChainID("lava")
				tx.GenerateOrBroadcastTxWithFactory(s.ClientCtx, txFactory, msg)

				// TODO:: should break here? is one enough or search for more?
				log.Printf("reliability discrepancy - block %d has different hashes:[ %s, %s ]\n", blockNum, blockHash, otherHash)
				return true, fmt.Errorf("is not a valid reliability VRF address result")
			}
		}
	}

	return false, nil
}

func (s *Sentry) validateProviderReply(finalizedBlocks map[int64]string, latestBlock int64, providerAcc string, session *ClientSession) error {
	sorted := make([]int64, len(finalizedBlocks))
	idx := 0
	maxBlockNum := int64(0)
	for blockNum := range finalizedBlocks {
		if !s.IsFinalizedBlock(blockNum, latestBlock) {
			// log.Println("provider returned non finalized block reply.\n Provider: %s, blockNum: %s", providerAcc, blockNum)
			return errors.New("Reliability ERROR: provider returned non finalized block reply")
		}

		sorted[idx] = blockNum

		if blockNum > maxBlockNum {
			maxBlockNum = blockNum
		}
		idx++
		// check blockhash length and format?
	}

	// check for consecutive blocks
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	for index := range sorted {
		if index != 0 && sorted[index]-1 != sorted[index-1] {
			// log.Println("provider returned non consecutive finalized blocks reply.\n Provider: %s", providerAcc)
			return errors.New("Reliability ERROR: provider returned non consecutive finalized blocks reply")
		}
	}

	// check that latest finalized block address + 1 points to a non finalized block
	if s.IsFinalizedBlock(maxBlockNum+1, latestBlock) {
		return errors.New("Reliability ERROR: provider returned finalized hashes for an older latest block")
	}

	// New reply should have blocknum >= from block same provider
	if session.LatestBlock > latestBlock {
		//
		// Report same provider discrepancy
		// TODO:: Fill msg with incriminating data
		msg := conflicttypes.NewMsgDetection(s.Acc, nil, nil, nil)
		s.ClientCtx.SkipConfirm = true
		txFactory := tx.NewFactoryCLI(s.ClientCtx, s.cmdFlags).WithChainID("lava")
		tx.GenerateOrBroadcastTxWithFactory(s.ClientCtx, txFactory, msg)

		return fmt.Errorf("Reliability ERROR: Provider supplied an older latest block than it has previously")
	}

	return nil
}

func (s *Sentry) getConsensusByProvider(providerId string) *ProviderHashesConsensus {
	for _, consensus := range s.providerHashesConsensus {
		if _, ok := consensus.agreeingProviders[providerId]; ok {
			return &consensus
		}
	}

	for _, consensus := range s.prevEpochProviderHashesConsensus {
		if _, ok := consensus.agreeingProviders[providerId]; ok {
			return &consensus
		}
	}
	return nil
}

func (s *Sentry) initProviderHashesConsensus(providerAcc string, latestBlock int64, finalizedBlocks map[int64]string, reply *pairingtypes.RelayReply, req *pairingtypes.RelayRequest) ProviderHashesConsensus {
	newProviderDataContainer := providerDataContainer{
		LatestFinalizedBlock:  s.GetLatestFinalizedBlock(latestBlock),
		LatestBlockTime:       time.Now(),
		FinalizedBlocksHashes: finalizedBlocks,
		SigBlocks:             reply.SigBlocks,
		SessionId:             req.SessionId,
		RelayNum:              req.RelayNum,
		BlockHeight:           req.BlockHeight,
		LatestBlock:           latestBlock,
	}
	providerDataContainers := map[string]providerDataContainer{}
	providerDataContainers[providerAcc] = newProviderDataContainer
	return ProviderHashesConsensus{
		FinalizedBlocksHashes: finalizedBlocks,
		agreeingProviders:     providerDataContainers,
	}
}

func (s *Sentry) insertProviderToConsensus(consensus *ProviderHashesConsensus, finalizedBlocks map[int64]string, latestBlock int64, reply *pairingtypes.RelayReply, req *pairingtypes.RelayRequest, providerAcc string) {
	newProviderDataContainer := providerDataContainer{
		LatestFinalizedBlock:  s.GetLatestFinalizedBlock(latestBlock),
		LatestBlockTime:       time.Now(),
		FinalizedBlocksHashes: finalizedBlocks,
		SigBlocks:             reply.SigBlocks,
		SessionId:             req.SessionId,
		RelayNum:              req.RelayNum,
		BlockHeight:           req.BlockHeight,
		LatestBlock:           latestBlock,
	}
	consensus.agreeingProviders[providerAcc] = newProviderDataContainer

	for blockNum, blockHash := range finalizedBlocks {
		consensus.FinalizedBlocksHashes[blockNum] = blockHash
	}
}

func (s *Sentry) SendRelay(
	ctx context.Context,
	cb_send_relay func(clientSession *ClientSession) (*pairingtypes.RelayReply, *pairingtypes.RelayRequest, error),
	cb_send_reliability func(clientSession *ClientSession, dataReliability *pairingtypes.VRFData) (*pairingtypes.RelayReply, *pairingtypes.RelayRequest, error),
	specCategory *spectypes.SpecCategory, // TODO::
) (*pairingtypes.RelayReply, error) {
	//
	// Get pairing
	wrap, index, err := s._findPairing(ctx)
	if err != nil {
		return nil, err
	}

	//
	getClientSessionFromWrap := func(wrap *RelayerClientWrapper) *ClientSession {
		wrap.SessionsLock.Lock()
		defer wrap.SessionsLock.Unlock()

		//try to lock an existing session, if can't create a new one
		for _, session := range wrap.Sessions {
			if session.Lock.TryLock() {
				return session
			}
		}

		randomSessId := int64(0)
		for randomSessId == 0 { //we don't allow 0
			randomSessId = rand.Int63()
		}

		clientSession := &ClientSession{
			SessionId: randomSessId,
			Client:    wrap,
		}
		clientSession.Lock.Lock()
		wrap.Sessions[clientSession.SessionId] = clientSession
		return clientSession
	}
	// Get or create session and lock it
	clientSession := getClientSessionFromWrap(wrap) // clientSession is LOCKED!

	// call user
	reply, request, err := cb_send_relay(clientSession)
	//error using this provider
	if err != nil {
		if clientSession.QoSInfo.ConsecutiveTimeOut >= MaxConsecutiveConnectionAttemts && clientSession.QoSInfo.LastQoSReport.Availability.IsZero() {
			s.movePairingEntryToPurge(wrap, index)
		}
		return reply, err
	}

	providerAcc := clientSession.Client.Acc // TODO:: should lock client before access?
	clientSession.Lock.Unlock()             //function call returns a locked session, we need to unlock it

	if s.GetSpecComparesHashes() {
		finalizedBlocks := map[int64]string{}                               // TODO:: define struct in relay response
		err = json.Unmarshal(reply.FinalizedBlocksHashes, &finalizedBlocks) // TODO:: check that this works
		if err != nil {
			log.Println("Reliability ERROR: Finalized Block reply err", err)
			return nil, err
		}
		latestBlock := reply.LatestBlock

		// validate that finalizedBlocks makes sense
		err = s.validateProviderReply(finalizedBlocks, latestBlock, providerAcc, clientSession)
		if err != nil {
			log.Println("Provider reply error, ", err)
			return nil, err
		}
		// Save in current session and compare in the next
		clientSession.FinalizedBlocksHashes = finalizedBlocks
		clientSession.LatestBlock = latestBlock
		//
		// Compare finalized block hashes with previous providers
		// Looks for discrepancy with current epoch providers
		// if no conflicts, insert into consensus and break
		// create new consensus group if no consensus matched
		// check for discrepancy with old epoch
		_, err := checkFinalizedHashes(s, providerAcc, latestBlock, finalizedBlocks, request, reply)
		if err != nil {
			return nil, err
		}

		if specCategory.Deterministic && s.IsFinalizedBlock(request.RequestBlock, reply.LatestBlock) {
			// handle data reliability

			isSecure, err := s.cmdFlags.GetBool("secure")
			if err != nil {
				log.Println("Error: Could not get flag --secure")
			}

			s.VrfSkMu.Lock()

			currentEpoch := clientSession.Client.GetPairingEpoch()
			vrfRes0, vrfRes1 := utils.CalculateVrfOnRelay(request, reply, s.VrfSk, currentEpoch)
			s.VrfSkMu.Unlock()
			address0, address1 := s.DataReliabilityThresholdToAddress(vrfRes0, vrfRes1)

			//Printing VRF Data
			// st1, _ := bech32.ConvertAndEncode("", vrfRes0)
			// st2, _ := bech32.ConvertAndEncode("", vrfRes1)
			// log.Printf("Finalized Block reply from %s received res %s, %s, addresses: %s, %s\n", providerAcc, st1, st2, address0, address1)

			sendReliabilityRelay := func(address string, differentiator bool) (relay_rep *pairingtypes.RelayReply, relay_req *pairingtypes.RelayRequest, err error) {
				if address != "" && address != providerAcc {
					wrap, index, err := s.specificPairing(ctx, address)
					if err != nil {
						// failed to get clientWrapper for this address, skip reliability
						log.Println("Reliability error: Could not get client specific pairing wrap for address: ", address, err)
						return nil, nil, err
					} else {
						canSendReliability := s.CheckAndMarkReliabilityForThisPairing(wrap) //TODO: this will still not perform well for multiple clients, we need to get the reliability proof in the error and not burn the provider
						if canSendReliability {
							s.VrfSkMu.Lock()
							vrf_res, vrf_proof := utils.ProveVrfOnRelay(request, reply, s.VrfSk, differentiator, currentEpoch)
							s.VrfSkMu.Unlock()
							dataReliability := &pairingtypes.VRFData{Differentiator: differentiator,
								VrfValue:    vrf_res,
								VrfProof:    vrf_proof,
								ProviderSig: reply.Sig,
								AllDataHash: sigs.AllDataHash(reply, request),
								QueryHash:   utils.CalculateQueryHash(*request), //calculated from query body anyway, but we will use this on payment
								Sig:         nil,                                //calculated in cb_send_reliability
							}
							clientSession = getClientSessionFromWrap(wrap)
							relay_rep, relay_req, err := cb_send_reliability(clientSession, dataReliability)
							if err != nil {
								log.Println("Reliability ERROR: Could not get reply to reliability relay from provider: ", address, err)
								if clientSession.QoSInfo.ConsecutiveTimeOut >= 3 && clientSession.QoSInfo.LastQoSReport.Availability.IsZero() {
									s.movePairingEntryToPurge(wrap, index)
								}
								return nil, nil, err
							}
							clientSession.Lock.Unlock() //function call returns a locked session, we need to unlock it
							return relay_rep, relay_req, nil
						} else {
							log.Println("Reliability already Sent in this epoch to this provider")
							return nil, nil, nil
						}
					}
				} else {
					if isSecure {
						//send reliability on the client's expense
						log.Println("secure flag Not Implemented, TODO:")
					}
					return nil, nil, fmt.Errorf("reliability ERROR: is not a valid reliability VRF address result")
				}
			}

			checkReliability := func() {
				reply0, request0, err0 := sendReliabilityRelay(address0, false)
				reply1, request1, err1 := sendReliabilityRelay(address1, true)
				ok := true
				check0 := err0 == nil && reply0 != nil
				check1 := err1 == nil && reply1 != nil
				if check0 {
					ok = ok && s.CompareRelaysAndReportConflict(reply, request, reply0, request0)
				}
				if check1 {
					ok = ok && s.CompareRelaysAndReportConflict(reply, request, reply1, request1)
				}
				if !ok && check0 && check1 {
					s.CompareRelaysAndReportConflict(reply0, request0, reply1, request1)
				}
				if (ok && check0) || (ok && check1) {
					log.Printf("[+] Reliability verified and Okay! ----\n\n")
				}
			}
			go checkReliability()
		}
	}
	return reply, nil
}

func checkFinalizedHashes(s *Sentry, providerAcc string, latestBlock int64, finalizedBlocks map[int64]string, req *pairingtypes.RelayRequest, reply *pairingtypes.RelayReply) (bool, error) {
	s.providerDataContainersMu.Lock()
	defer s.providerDataContainersMu.Unlock()

	if len(s.providerHashesConsensus) == 0 && len(s.prevEpochProviderHashesConsensus) == 0 {
		newHashConsensus := s.initProviderHashesConsensus(providerAcc, latestBlock, finalizedBlocks, reply, req)
		s.providerHashesConsensus = append(make([]ProviderHashesConsensus, 0), newHashConsensus)
	} else {
		matchWithExistingConsensus := false

		// Looks for discrepancy wit current epoch providers
		for idx, consensus := range s.providerHashesConsensus {
			discrepancyResult, err := s.discrepancyChecker(finalizedBlocks, consensus)
			if err != nil {
				log.Println("Reliability ERROR: Discrepancy Checker err", err)
				return false, err
			}

			// if no conflicts, insert into consensus and break
			if !discrepancyResult {
				matchWithExistingConsensus = true
			} else {
				log.Printf("Reliability ERROR: Conflict found between consensus %d and provider %s\n", idx, providerAcc)
			}

			// if no discrepency with this group -> insert into consensus and break
			if matchWithExistingConsensus {
				// TODO:: Add more increminiating data to consensus
				s.insertProviderToConsensus(&consensus, finalizedBlocks, latestBlock, reply, req, providerAcc)
				break
			}
		}

		// create new consensus group if no consensus matched
		if !matchWithExistingConsensus {
			newHashConsensus := s.initProviderHashesConsensus(providerAcc, latestBlock, finalizedBlocks, reply, req)
			s.providerHashesConsensus = append(make([]ProviderHashesConsensus, 0), newHashConsensus)
		}

		// check for discrepancy with old epoch
		for idx, consensus := range s.prevEpochProviderHashesConsensus {
			discrepancyResult, err := s.discrepancyChecker(finalizedBlocks, consensus)
			if err != nil {
				log.Println("Reliability ERROR: Discrepancy Checker err", err)
				return false, err
			}

			if discrepancyResult {
				log.Printf("Reliability ERROR: Conflict found between consensus %d and provider %s\n", idx, providerAcc)
			}
		}
	}

	return false, nil
}

func (s *Sentry) IsFinalizedBlock(requestedBlock int64, latestBlock int64) bool {
	return spectypes.IsFinalizedBlock(requestedBlock, latestBlock, s.GetSpecFinalizationCriteria())
}

func (s *Sentry) GetLatestFinalizedBlock(latestBlock int64) int64 {
	finalization_criteria := int64(s.GetSpecFinalizationCriteria())
	return latestBlock - finalization_criteria
}

func (s *Sentry) movePairingEntryToPurge(wrap *RelayerClientWrapper, index int) {
	log.Printf("Warning! Jailing provider %s for this epoch\n", wrap.Acc)
	s.pairingMu.Lock()
	defer s.pairingMu.Unlock()

	if len(s.pairing) == 0 {
		return
	}

	s.pairingPurgeLock.Lock()
	defer s.pairingPurgeLock.Unlock()
	//move to purge list
	findPairingIndex := func() bool {
		for idx, entry := range s.pairing {
			if entry.Acc == wrap.Acc {
				index = idx
				return true
			}
		}
		return false
	}
	if index >= len(s.pairing) || index < 0 {
		log.Printf("Info! Trying to move pairing entry to purge but index is bigger than pairing length! provider: endpoint: %s address: %s index: %d, length: %d\n", wrap.Acc, wrap.Addr, index, len(s.pairing))
		if !findPairingIndex() {
			return
		}
	}
	if s.pairing[index].Acc != wrap.Acc {
		log.Printf("Info! Trying to move pairing entry to purge but expected address is different! provider: endpoint: %s address: %s index: %d, length: %d, current index address:%s \n", wrap.Addr, wrap.Acc, index, len(s.pairing), s.pairing[index].Addr)
		if !findPairingIndex() {
			return
		}
	}
	s.pairingPurge = append(s.pairingPurge, wrap)
	s.pairing[index] = s.pairing[len(s.pairing)-1]
	s.pairing = s.pairing[:len(s.pairing)-1]
}

func (s *Sentry) IsAuthorizedUser(ctx context.Context, user string, blockheight uint64) (*pairingtypes.QueryVerifyPairingResponse, error) {
	//
	// TODO: cache results!
	res, err := s.pairingQueryClient.VerifyPairing(context.Background(), &pairingtypes.QueryVerifyPairingRequest{
		ChainID:  s.ChainID,
		Client:   user,
		Provider: s.Acc,
		Block:    blockheight,
	})
	if err != nil {
		return nil, err
	}
	if res.GetValid() {
		return res, nil
	}

	return nil, fmt.Errorf("invalid pairing with user. CurrentBlock: %d", s.GetBlockHeight())
}

func (s *Sentry) IsAuthorizedPairing(ctx context.Context, consumer string, provider string, block uint64) (bool, error) {
	//
	// TODO: cache results!

	res, err := s.pairingQueryClient.VerifyPairing(context.Background(), &pairingtypes.QueryVerifyPairingRequest{
		ChainID:  s.ChainID,
		Client:   consumer,
		Provider: provider,
		Block:    block,
	})
	if err != nil {
		return false, err
	}
	if res.GetValid() {
		return true, nil
	}
	return false, fmt.Errorf("invalid pairing with consumer %s, provider %s block: %d, provider block height: %d", consumer, provider, block, s.GetBlockHeight())
}

func (s *Sentry) GetReliabilityThreshold() uint32 {
	return s.serverSpec.ReliabilityThreshold
}

func (s *Sentry) GetSpecName() string {
	return s.serverSpec.Name
}

func (s *Sentry) GetSpecComparesHashes() bool {
	return s.serverSpec.ComparesHashes
}

func (s *Sentry) GetSpecFinalizationCriteria() uint32 {
	return s.serverSpec.FinalizationCriteria
}

func (s *Sentry) GetSpecSavedBlocks() uint32 {
	return s.serverSpec.SavedBlocks
}

func (s *Sentry) GetChainID() string {
	return s.serverSpec.Index
}

func (s *Sentry) MatchSpecApiByName(name string) (spectypes.ServiceApi, bool) {
	s.specMu.RLock()
	defer s.specMu.RUnlock()
	//TODO: make it faster and better by not doing a regex instead using a better algorithm
	for apiName, api := range s.serverApis {
		re, err := regexp.Compile(apiName)
		if err != nil {
			log.Println("error: Compile", apiName, err)
			continue
		}
		if re.Match([]byte(name)) {
			return api, true
		}
	}
	return spectypes.ServiceApi{}, false
}

func (s *Sentry) GetSpecApiByName(name string) (spectypes.ServiceApi, bool) {
	s.specMu.RLock()
	defer s.specMu.RUnlock()

	val, ok := s.serverApis[name]
	return val, ok
}

func (s *Sentry) GetSpecApiByTag(tag string) (spectypes.ServiceApi, bool) {
	s.specMu.RLock()
	defer s.specMu.RUnlock()

	val, ok := s.taggedApis[tag]
	return val, ok
}

func (s *Sentry) GetBlockHeight() int64 {
	return atomic.LoadInt64(&s.blockHeight)
}

func (s *Sentry) SetBlockHeight(blockHeight int64) {
	atomic.StoreInt64(&s.blockHeight, blockHeight)
}

func (s *Sentry) GetCurrentEpochHeight() uint64 {
	return atomic.LoadUint64(&s.currentEpoch)
}

func (s *Sentry) SetCurrentEpochHeight(blockHeight int64) {
	atomic.StoreUint64(&s.currentEpoch, uint64(blockHeight))
}

func (s *Sentry) GetOverlapSize() uint64 {
	return atomic.LoadUint64(&s.EpochBlocksOverlap)
}

func (s *Sentry) GetCUServiced() uint64 {
	return atomic.LoadUint64(&s.totalCUServiced)
}

func (s *Sentry) SetCUServiced(CU uint64) {
	atomic.StoreUint64(&s.totalCUServiced, CU)
}

func (s *Sentry) UpdateCUServiced(CU uint64) {
	//we lock because we dont want the value changing after we read it before we store
	s.PaymentsMu.Lock()
	defer s.PaymentsMu.Unlock()
	currentCU := atomic.LoadUint64(&s.totalCUServiced)
	atomic.StoreUint64(&s.totalCUServiced, currentCU+CU)
}

func (s *Sentry) GetMaxCUForUser(ctx context.Context, address string, chainID string) (maxCu uint64, err error) {
	UserEntryRes, err := s.pairingQueryClient.UserEntry(ctx, &pairingtypes.QueryUserEntryRequest{ChainID: chainID, Address: address, Block: uint64(s.GetBlockHeight())})
	if err != nil {
		return 0, err
	}
	return UserEntryRes.GetMaxCU(), err
}

func (s *Sentry) GetVrfPkAndMaxCuForUser(ctx context.Context, address string, chainID string, requestBlock int64) (vrfPk *utils.VrfPubKey, maxCu uint64, err error) {
	UserEntryRes, err := s.pairingQueryClient.UserEntry(ctx, &pairingtypes.QueryUserEntryRequest{ChainID: chainID, Address: address, Block: uint64(requestBlock)})
	if err != nil {
		return nil, 0, err
	}
	vrfPk = &utils.VrfPubKey{}
	vrfPk, err = vrfPk.DecodeFromBech32(UserEntryRes.GetConsumer().Vrfpk)
	return vrfPk, UserEntryRes.GetMaxCU(), err
}

func (s *Sentry) ExpecedBlockHeight() (int64, int) {

	averageBlockTime_ms := s.serverSpec.AverageBlockTime
	listExpectedBlockHeights := []int64{}

	var highestBlockNumber int64 = 0
	FindHighestBlockNumber := func(listProviderHashesConsensus []ProviderHashesConsensus) int64 {
		for _, providerHashesConsensus := range listProviderHashesConsensus {
			for _, providerDataContainer := range providerHashesConsensus.agreeingProviders {
				if highestBlockNumber < providerDataContainer.LatestFinalizedBlock {
					highestBlockNumber = providerDataContainer.LatestFinalizedBlock
				}

			}
		}
		return highestBlockNumber
	}
	highestBlockNumber = FindHighestBlockNumber(s.prevEpochProviderHashesConsensus) //update the highest in place
	highestBlockNumber = FindHighestBlockNumber(s.providerHashesConsensus)

	now := time.Now()
	calcExpectedBlocks := func(listProviderHashesConsensus []ProviderHashesConsensus) []int64 {
		listExpectedBH := []int64{}
		for _, providerHashesConsensus := range listProviderHashesConsensus {
			for _, providerDataContainer := range providerHashesConsensus.agreeingProviders {
				expected := providerDataContainer.LatestFinalizedBlock + (now.Sub(providerDataContainer.LatestBlockTime).Milliseconds() / averageBlockTime_ms) //interpolation
				//limit the interpolation to the highest seen block height
				if expected > highestBlockNumber {
					expected = highestBlockNumber
				}
				listExpectedBH = append(listExpectedBH, expected)
			}
		}
		return listExpectedBH
	}
	listExpectedBlockHeights = append(listExpectedBlockHeights, calcExpectedBlocks(s.prevEpochProviderHashesConsensus)...)
	listExpectedBlockHeights = append(listExpectedBlockHeights, calcExpectedBlocks(s.providerHashesConsensus)...)

	median := func(data []int64) int64 {
		slices.Sort(data)

		var median int64
		l := len(data)
		if l == 0 {
			return 0
		} else if l%2 == 0 {
			median = int64((data[l/2-1] + data[l/2]) / 2.0)
		} else {
			median = int64(data[l/2])
		}
		return median
	}

	return median(listExpectedBlockHeights) - s.serverSpec.AllowedBlockLagForQosSync, len(listExpectedBlockHeights)
}

// TODO:: Dont calc. get this info from blockchain - if LAVA params change, this calc is obsolete
func (s *Sentry) GetEpochFromBlockHeight(blockHeight int64) uint64 {
	epochSize := s.GetEpochSize()
	epoch := uint64(blockHeight - blockHeight%int64(epochSize))
	return epoch
}

func NewSentry(
	clientCtx client.Context,
	chainID string,
	isUser bool,
	voteInitiationCb func(ctx context.Context, voteID string, voteDeadline uint64, voteParams *VoteParams),
	newEpochCb func(epochHeight int64),
	apiInterface string,
	vrf_sk vrf.PrivateKey,
	flagSet *pflag.FlagSet,
	serverID uint64,
) *Sentry {
	rpcClient := clientCtx.Client
	specQueryClient := spectypes.NewQueryClient(clientCtx)
	pairingQueryClient := pairingtypes.NewQueryClient(clientCtx)
	epochStorageQueryClient := epochstoragetypes.NewQueryClient(clientCtx)
	acc := clientCtx.GetFromAddress().String()
	currentBlock, err := rpc.GetChainHeight(clientCtx)
	if err != nil {
		utils.LavaFormatError("Sentry failed to get chain height", err, &map[string]string{"account": acc, "ChainID": chainID, "apiInterface": apiInterface})
		currentBlock = 0
	}
	return &Sentry{
		ClientCtx:               clientCtx,
		rpcClient:               rpcClient,
		specQueryClient:         specQueryClient,
		pairingQueryClient:      pairingQueryClient,
		epochStorageQueryClient: epochStorageQueryClient,
		ChainID:                 chainID,
		isUser:                  isUser,
		Acc:                     acc,
		newEpochCb:              newEpochCb,
		ApiInterface:            apiInterface,
		VrfSk:                   vrf_sk,
		blockHeight:             currentBlock,
		specHash:                nil,
		cmdFlags:                flagSet,
		voteInitiationCb:        voteInitiationCb,
		serverID:                serverID,
	}
}

func UpdateRequestedBlock(request *pairingtypes.RelayRequest, response *pairingtypes.RelayReply) {
	//since sometimes the user is sending requested block that is a magic like latest, or earliest we need to specify to the reliability what it is
	switch request.RequestBlock {
	case spectypes.LATEST_BLOCK:
		request.RequestBlock = response.LatestBlock
	case spectypes.EARLIEST_BLOCK:
		request.RequestBlock = spectypes.NOT_APPLICABLE // TODO: add support for earliest block reliability
	}
}
