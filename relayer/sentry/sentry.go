package sentry

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/lavanet/lava/relayer/common"
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
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/spf13/pflag"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tenderminttypes "github.com/tendermint/tendermint/types"
	"golang.org/x/exp/slices"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	maxRetries             = 10
	providerWasntFound     = -1
	findPairingFailedIndex = -1
	supportedNumberOfVRFs  = 2
	GeolocationFlag        = "geolocation"
)

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
		// default returns false
		return false
	}
	return vp.CloseVote
}

// Constants

var AvailabilityPercentage sdk.Dec = sdk.NewDecWithPrec(5, 2) // TODO move to params pairing
const (
	MaxConsecutiveConnectionAttempts = 3
	PercentileToCalculateLatency     = 0.9
	MinProvidersForSync              = 0.6
	LatencyThresholdStatic           = 1 * time.Second
	LatencyThresholdSlope            = 1 * time.Millisecond
	StaleEpochDistance               = 3 // relays done 3 epochs back are ready to be rewarded
)

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
	// TODO:: keep relay request for conflict reporting
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
	authorizationCache      map[uint64]map[string]*pairingtypes.QueryVerifyPairingResponse
	authorizationCacheMutex sync.RWMutex
	txFactory               tx.Factory
	geolocation             uint64
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
	prevEpoch          uint64
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

	VrfSkMu utils.LavaMutex
	VrfSk   vrf.PrivateKey

	// every entry in providerHashesConsensus is conflicted with the other entries
	providerHashesConsensus          []ProviderHashesConsensus
	prevEpochProviderHashesConsensus []ProviderHashesConsensus
	providerDataContainersMu         utils.LavaMutex

	consumerSessionManager *lavasession.ConsumerSessionManager

	// Whitelists
	epochErrorWhitelist *common.EpochErrorWhitelist // whitelist for all errors which shouldn't be logged in the current epoch
}

func (s *Sentry) SetupConsumerSessionManager(ctx context.Context, consumerSessionManager *lavasession.ConsumerSessionManager, epochErrorWhitelist *common.EpochErrorWhitelist) error {
	utils.LavaFormatInfo("Setting up ConsumerSessionManager", nil)
	s.consumerSessionManager = consumerSessionManager

	// Add epoch error whitelist to the consumer Session manager
	s.consumerSessionManager.EpochErrorWhitelist = epochErrorWhitelist

	// Get pairing for the first time, for clients
	pairingList, err := s.getPairing(ctx)
	if err != nil {
		utils.LavaFormatFatal("Failed getting pairing for consumer in initialization", err, &map[string]string{"Address": s.Acc})
	}
	err = s.consumerSessionManager.UpdateAllProviders(ctx, s.GetCurrentEpochHeight(), pairingList)
	if err != nil {
		utils.LavaFormatFatal("Failed UpdateAllProviders", err, &map[string]string{"Address": s.Acc})
	}
	return nil
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

func (s *Sentry) getPairing(ctx context.Context) ([]*lavasession.ConsumerSessionsWithProvider, error) {
	//
	// sentry for server module does not need a pairing
	if !s.isUser {
		return nil, nil
	}

	//
	// Get
	res, err := s.pairingQueryClient.GetPairing(ctx, &pairingtypes.QueryGetPairingRequest{
		ChainID: s.GetChainID(),
		Client:  s.Acc,
	})
	if err != nil {
		return nil, utils.LavaFormatError("Failed in get pairing query", err, &map[string]string{})
	}

	providers := res.GetProviders()
	if len(providers) == 0 {
		return nil, utils.LavaFormatError("no providers found in pairing, returned empty list", nil, &map[string]string{})
	}

	//
	// Set
	pairing := []*lavasession.ConsumerSessionsWithProvider{}
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
			// only take into account endpoints that use the same api interface and the same geolocation
			if endpoint.UseType == s.ApiInterface && endpoint.Geolocation == s.geolocation {
				relevantEndpoints = append(relevantEndpoints, endpoint)
			}
		}
		if len(relevantEndpoints) == 0 {
			utils.LavaFormatError("skipping provider, No relevant endpoints for apiInterface", nil, &map[string]string{"Address": provider.Address, "ChainID": provider.Chain, "apiInterface": s.ApiInterface, "Endpoints": fmt.Sprintf("%v", providerEndpoints)})
			continue
		}

		maxcu, err := s.GetMaxCUForUser(ctx, s.Acc, provider.Chain)
		if err != nil {
			return nil, utils.LavaFormatError("Failed getting max CU for user", err, &map[string]string{"Address": s.Acc, "ChainID": provider.Chain})
		}
		//
		pairingEndpoints := make([]*lavasession.Endpoint, len(relevantEndpoints))
		for idx, relevantEndpoint := range relevantEndpoints {
			endp := &lavasession.Endpoint{Addr: relevantEndpoint.IPPORT, Enabled: true, Client: nil, ConnectionRefusals: 0}
			pairingEndpoints[idx] = endp
		}

		pairing = append(pairing, &lavasession.ConsumerSessionsWithProvider{
			Acc:             provider.Address,
			Endpoints:       pairingEndpoints,
			Sessions:        map[int64]*lavasession.SingleConsumerSession{},
			MaxComputeUnits: maxcu,
			ReliabilitySent: false,
			PairingEpoch:    s.GetCurrentEpochHeight(),
		})
	}
	if len(pairing) == 0 {
		utils.LavaFormatError("Failed getting pairing for consumer, pairing is empty", err, &map[string]string{"Address": s.Acc, "ChainID": s.GetChainID(), "geolocation": strconv.FormatUint(s.geolocation, 10)})
	}
	// replace previous pairing with new providers
	return pairing, nil
}

func (s *Sentry) GetSpecHash() []byte {
	s.specMu.Lock()
	defer s.specMu.Unlock()
	return s.specHash
}

func (s *Sentry) GetAllSpecNames(ctx context.Context) (map[string][]spectypes.ApiInterface, error) {
	spec, err := s.specQueryClient.Spec(ctx, &spectypes.QueryGetSpecRequest{
		ChainID: s.ChainID,
	})
	if err != nil {
		return nil, utils.LavaFormatError("Failed Querying spec for chain", err, &map[string]string{"ChainID": s.ChainID})
	}
	serverApis, _ := s.getServiceApis(spec)
	allSpecNames := make(map[string][]spectypes.ApiInterface)
	for _, api := range serverApis {
		allSpecNames[api.Name] = api.ApiInterfaces
	}
	return allSpecNames, nil
}

func (s *Sentry) getServiceApis(spec *spectypes.QueryGetSpecResponse) (retServerApis map[string]spectypes.ServiceApi, retTaggedApis map[string]spectypes.ServiceApi) {
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
					// spec will contain many api interfaces, we only need those that belong to the apiInterface of this sentry
					continue
				}
				if apiInterface.Interface == spectypes.APIInterfaceRest {
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
	spec, err := s.specQueryClient.Spec(ctx, &spectypes.QueryGetSpecRequest{
		ChainID: s.ChainID,
	})
	if err != nil {
		return utils.LavaFormatError("Failed Querying spec for chain", err, &map[string]string{"ChainID": s.ChainID})
	}

	//
	// Check if updated
	hash := tendermintcrypto.Sha256([]byte(spec.String())) // TODO: we use cheaper algo for speed
	if bytes.Equal(s.specHash, hash) {
		// spec for chain didnt change
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

	s.SetPrevEpochHeight(0)
	err = s.FetchChainParams(ctx)
	if err != nil {
		return err
	}

	geolocation, err := s.cmdFlags.GetUint64(GeolocationFlag)
	if err != nil {
		utils.LavaFormatFatal("failed to read geolocation flag, required flag", err, nil)
	}
	if geolocation > 0 && (geolocation&(geolocation-1)) == 0 {
		// geolocation is a power of 2
		s.geolocation = geolocation
	} else {
		// geolocation is not a power of 2
		utils.LavaFormatFatal("geolocation flag needs to set only one geolocation, 1<<X where X is the geolocation i.e 1,2,4,8 etc..", err, &map[string]string{"Geolocation": strconv.FormatUint(geolocation, 10)})
	}

	// Sanity
	if !s.isUser {
		providers, err := s.pairingQueryClient.Providers(ctx, &pairingtypes.QueryProvidersRequest{
			ChainID: s.GetChainID(),
		})
		if err != nil {
			return utils.LavaFormatError("failed querying providers for spec", err, &map[string]string{"spec name": s.GetSpecName(), "ChainID": s.GetChainID()})
		}
		found := false
	endpointsLoop:
		for _, provider := range providers.GetStakeEntry() {
			if provider.Address == s.Acc {
				for _, endpoint := range provider.Endpoints {
					if endpoint.Geolocation == s.geolocation && endpoint.UseType == s.ApiInterface {
						found = true
						break endpointsLoop
					}
				}
				// if we reached here we didnt find a geolocation appropriate endpoint
				utils.LavaFormatFatal("provider endpoint mismatch", err, &map[string]string{"spec name": s.GetSpecName(), "ChainID": s.GetChainID(), "Geolocation": strconv.FormatUint(geolocation, 10), "StakeEntry": fmt.Sprintf("%+v", provider)})
			}
		}
		if !found {
			return utils.LavaFormatError("provider stake verification mismatch", err, &map[string]string{"spec name": s.GetSpecName(), "ChainID": s.GetChainID(), "Geolocation": strconv.FormatUint(geolocation, 10)})
		}
	}

	return nil
}

func (s *Sentry) ListenForTXEvents(ctx context.Context) {
	for e := range s.NewTransactionEvents {
		switch data := e.Data.(type) {
		case tenderminttypes.EventDataTx:
			// got new TX event
			if providerAddrList, ok := e.Events["lava_relay_payment.provider"]; ok {
				for idx, providerAddr := range providerAddrList {
					if s.Acc == providerAddr && s.ChainID == e.Events["lava_relay_payment.chainID"][idx] {
						utils.LavaFormatInfo("Received relay payment",
							&map[string]string{
								"Amount": e.Events["lava_relay_payment.Mint"][idx],
								"CU":     e.Events["lava_relay_payment.CU"][idx],
							})
						CU := e.Events["lava_relay_payment.CU"][idx]
						paidCU, err := strconv.ParseUint(CU, 10, 64)
						if err != nil {
							utils.LavaFormatError("failed to parse payment event CU", err, &map[string]string{"event": e.Events["lava_relay_payment.CU"][idx]})
							continue
						}
						clientAddr, err := sdk.AccAddressFromBech32(e.Events["lava_relay_payment.client"][idx])
						if err != nil {
							utils.LavaFormatError("failed to parse payment event client", err, &map[string]string{"event": e.Events["lava_relay_payment.client"][idx]})
							continue
						}
						coin, err := sdk.ParseCoinNormalized(e.Events["lava_relay_payment.Mint"][idx])
						if err != nil {
							utils.LavaFormatError("failed to parse payment event mint", err, &map[string]string{"event": e.Events["lava_relay_payment.Mint"][idx]})
							continue
						}
						uniqueID, err := strconv.ParseUint(e.Events["lava_relay_payment.uniqueIdentifier"][idx], 10, 64)
						if err != nil {
							utils.LavaFormatError("failed to parse payment event uniqueIdentifier", err, &map[string]string{"event": e.Events["lava_relay_payment.uniqueIdentifier"][idx]})
							continue
						}
						serverID, err := strconv.ParseUint(e.Events["lava_relay_payment.descriptionString"][idx], 10, 64)
						if err != nil {
							utils.LavaFormatError("failed to parse payment event serverID", err, &map[string]string{"event": e.Events["lava_relay_payment.descriptionString"][idx]})
							continue
						}

						if serverID == s.serverID {
							s.UpdatePaidCU(paidCU)
							receivedPayment := PaymentRequest{CU: paidCU, BlockHeightDeadline: data.Height, Amount: coin, Client: clientAddr, UniqueIdentifier: uniqueID}
							s.AppendToReceivedPayments(receivedPayment)
							found := s.RemoveExpectedPayment(paidCU, clientAddr, data.Height, uniqueID)
							if !found {
								utils.LavaFormatError("payment received, did not find matching expectancy from correct client", nil, &map[string]string{"expected payments": fmt.Sprintf("%v", s.PrintExpectedPayments()), "received payment": fmt.Sprintf("%v", receivedPayment)})
							} else {
								utils.LavaFormatInfo("success: payment received as expected", nil)
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
						utils.LavaFormatError("vote requested block could not be parsed", err, &map[string]string{"requested block": num_str, "voteID": voteID})
						continue
					}
					num_str = e.Events[eventToListen+".voteDeadline"][idx]
					voteDeadline, err := strconv.ParseUint(num_str, 10, 64)
					if err != nil {
						utils.LavaFormatError("vote deadline could not be parsed", err, &map[string]string{"deadline": num_str, "voteID": voteID})
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
		default:
			{
			}
		}
	}
}

func (s *Sentry) RemoveExpectedPayment(paidCUToFInd uint64, expectedClient sdk.AccAddress, blockHeight int64, uniqueID uint64) bool {
	s.PaymentsMu.Lock()
	defer s.PaymentsMu.Unlock()
	for idx, expectedPayment := range s.expectedPayments {
		// TODO: make sure the payment is not too far from expected block, expectedPayment.BlockHeightDeadline == blockHeight
		if expectedPayment.CU == paidCUToFInd && expectedPayment.Client.Equals(expectedClient) && uniqueID == expectedPayment.UniqueIdentifier {
			// found payment for expected payment
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
	// we lock because we dont want the value changing after we read it before we store
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

func (s *Sentry) PrintExpectedPayments() string {
	s.PaymentsMu.Lock()
	defer s.PaymentsMu.Unlock()
	return fmt.Sprintf("last Received: %v\n Expected: %v\n", s.receivedPayments[len(s.receivedPayments)-1], s.expectedPayments)
}

func (s *Sentry) Start(ctx context.Context) {
	if !s.isUser {
		// listen for transactions for proof of relay payment
		go s.ListenForTXEvents(ctx)
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
				utils.LavaFormatInfo("New Epoch Event", nil)
				utils.LavaFormatInfo("New Epoch Info:", &map[string]string{"Height": strconv.FormatInt(data.Block.Height, 10)})

				// Reset epochErrorWhitelist
				s.epochErrorWhitelist.Reset()

				// New epoch height will be set in FetchChainParams
				s.SetPrevEpochHeight(s.GetCurrentEpochHeight())
				err := s.FetchChainParams(ctx)
				if err != nil {
					utils.LavaFormatError("failed in FetchChainParams", err, nil)
				}

				if s.newEpochCb != nil {
					go s.newEpochCb(data.Block.Height - StaleEpochDistance*int64(s.GetEpochSize())) // Currently this is only askForRewards
				}

				//
				// Update specs
				err = s.getSpec(ctx)
				if err != nil {
					utils.LavaFormatError("failed to get spec", err, nil)
				}

				// update expected payments deadline, and log missing payments
				// TODO: make this from the event lava_earliest_epoch instead
				if !s.isUser {
					s.IdentifyMissingPayments()
				}
				//
				// Update pairing
				if s.isUser {
					pairingList, err := s.getPairing(ctx)
					if err != nil {
						utils.LavaFormatFatal("Failed getting pairing for consumer in initialization", err, &map[string]string{"Address": s.Acc})
					}
					err = s.consumerSessionManager.UpdateAllProviders(ctx, s.GetCurrentEpochHeight(), pairingList)
					if err != nil {
						utils.LavaFormatFatal("Failed UpdateAllProviders", err, &map[string]string{"Address": s.Acc})
					}
				}

				s.clearAuthResponseCache(data.Block.Height) // TODO: Remove this after provider session manager is fully functional
			}

			if !s.isUser {
				// listen for vote reveal event from new block handler on conflict/module.go
				eventToListen := utils.EventPrefix + conflicttypes.ConflictVoteRevealEventName
				if votesList, ok := e.Events[eventToListen+".voteID"]; ok {
					for idx, voteID := range votesList {
						num_str := e.Events[eventToListen+".voteDeadline"][idx]
						voteDeadline, err := strconv.ParseUint(num_str, 10, 64)
						if err != nil {
							utils.LavaFormatError("parsing vote deadline", err, &map[string]string{"VoteDeadline": num_str})
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
		default:
			{
			}
		}
	}
}

func (s *Sentry) FetchChainParams(ctx context.Context) error {
	err := s.FetchEpochSize(ctx)
	if err != nil {
		return err
	}

	err = s.FetchOverlapSize(ctx)
	if err != nil {
		return err
	}

	err = s.FetchEpochParams(ctx)
	if err != nil {
		return err
	}

	err = s.FetchProvidersCount(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *Sentry) IdentifyMissingPayments() {
	lastBlockInMemory := atomic.LoadUint64(&s.earliestSavedBlock)
	s.PaymentsMu.Lock()

	var updatedExpectedPayments []PaymentRequest

	for idx, expectedPay := range s.expectedPayments {
		// Exclude and log missing payments
		if uint64(expectedPay.BlockHeightDeadline) < lastBlockInMemory {
			utils.LavaFormatError("Identified Missing Payment", nil,
				&map[string]string{
					"expectedPay.CU":                  strconv.FormatUint(expectedPay.CU, 10),
					"expectedPay.BlockHeightDeadline": strconv.FormatInt(expectedPay.BlockHeightDeadline, 10),
					"lastBlockInMemory":               strconv.FormatUint(lastBlockInMemory, 10),
				})

			continue
		}

		// Include others
		updatedExpectedPayments = append(updatedExpectedPayments, s.expectedPayments[idx])
	}

	// Update expectedPayment
	s.expectedPayments = updatedExpectedPayments

	s.PaymentsMu.Unlock()
	// can be modified in this race window, so we double-check

	utils.LavaFormatInfo("Service report", &map[string]string{
		"total CU serviced":      strconv.FormatUint(s.GetCUServiced(), 10),
		"total CU that got paid": strconv.FormatUint(s.GetPaidCU(), 10),
	})
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
	conn, err := grpc.DialContext(connectCtx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	/*defer conn.Close()*/

	c := pairingtypes.NewRelayerClient(conn)
	return &c, nil
}

func (s *Sentry) CompareRelaysAndReportConflict(reply0 *pairingtypes.RelayReply, request0 *pairingtypes.RelayRequest, reply1 *pairingtypes.RelayReply, request1 *pairingtypes.RelayRequest) (ok bool) {
	compare_result := bytes.Compare(reply0.Data, reply1.Data)
	if compare_result == 0 {
		// they have equal data
		return true
	}
	// they have different data! report!
	utils.LavaFormatWarning("Simulation: DataReliability detected mismatching results, Reporting...", nil, &map[string]string{"Data0": string(reply0.Data), "Data1": string(reply1.Data)})
	responseConflict := conflicttypes.ResponseConflict{
		ConflictRelayData0: &conflicttypes.ConflictRelayData{Reply: reply0, Request: request0},
		ConflictRelayData1: &conflicttypes.ConflictRelayData{Reply: reply1, Request: request1},
	}
	msg := conflicttypes.NewMsgDetection(s.Acc, nil, &responseConflict, nil)
	s.ClientCtx.SkipConfirm = true
	// txFactory := tx.NewFactoryCLI(s.ClientCtx, s.cmdFlags).WithChainID("lava")
	err := SimulateAndBroadCastTx(s.ClientCtx, s.txFactory, msg)
	if err != nil {
		utils.LavaFormatError("CompareRelaysAndReportConflict - SimulateAndBroadCastTx Failed", err, nil)
	}
	// report the conflict
	return false
}

func (s *Sentry) DataReliabilityThresholdToSession(vrfs [][]byte, uniqueIdentifiers []bool) (indexes map[int64]bool) {
	// check for the VRF thresholds and if holds true send a relay to the provider
	// TODO: improve with blacklisted address, and the module-1
	s.specMu.RLock()
	reliabilityThreshold := s.serverSpec.ReliabilityThreshold
	s.specMu.RUnlock()

	providersCount := uint32(s.consumerSessionManager.GetAtomicPairingAddressesLength())
	indexes = make(map[int64]bool, len(vrfs))
	for vrfIndex, vrf := range vrfs {
		index, err := utils.GetIndexForVrf(vrf, providersCount, reliabilityThreshold)
		if index == -1 || err != nil {
			continue // no reliability this time.
		}
		if _, ok := indexes[index]; !ok {
			indexes[index] = uniqueIdentifiers[vrfIndex]
		}
	}
	return
}

func (s *Sentry) discrepancyChecker(finalizedBlocksA map[int64]string, consensus ProviderHashesConsensus) (discrepancy bool, errRet error) {
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
				// txFactory := tx.NewFactoryCLI(s.ClientCtx, s.cmdFlags).WithChainID("lava")
				err := SimulateAndBroadCastTx(s.ClientCtx, s.txFactory, msg)
				if err != nil {
					return false, utils.LavaFormatError("discrepancyChecker - SimulateAndBroadCastTx Failed", err, nil)
				}
				// TODO:: should break here? is one enough or search for more?
				return true, utils.LavaFormatError("Simulation: reliability discrepancy, different hashes detected for block", nil, &map[string]string{"blockNum": strconv.FormatInt(blockNum, 10), "Hashes": fmt.Sprintf("%s vs %s", blockHash, otherHash), "toIterate": fmt.Sprintf("%v", toIterate), "otherBlocks": fmt.Sprintf("%v", otherBlocks)})
			}
		}
	}

	return false, nil
}

func (s *Sentry) validateProviderReply(finalizedBlocks map[int64]string, latestBlock int64, providerAcc string, session *lavasession.SingleConsumerSession) error {
	sorted := make([]int64, len(finalizedBlocks))
	idx := 0
	maxBlockNum := int64(0)
	for blockNum := range finalizedBlocks {
		if !s.IsFinalizedBlock(blockNum, latestBlock) {
			return utils.LavaFormatError("Simulation: provider returned non finalized block reply for reliability", nil, &map[string]string{"blockNum": strconv.FormatInt(blockNum, 10), "latestBlock": strconv.FormatInt(latestBlock, 10), "ChainID": s.ChainID, "Provider": providerAcc, "finalizedBlocks": fmt.Sprintf("%+v", finalizedBlocks)})
		}

		sorted[idx] = blockNum

		if blockNum > maxBlockNum {
			maxBlockNum = blockNum
		}
		idx++
		// TODO: check blockhash length and format
	}

	// check for consecutive blocks
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	for index := range sorted {
		if index != 0 && sorted[index]-1 != sorted[index-1] {
			// log.Println("provider returned non consecutive finalized blocks reply.\n Provider: %s", providerAcc)
			return utils.LavaFormatError("Simulation: provider returned non consecutive finalized blocks reply", nil, &map[string]string{"curr block": strconv.FormatInt(sorted[index], 10), "prev block": strconv.FormatInt(sorted[index-1], 10), "ChainID": s.ChainID, "Provider": providerAcc, "finalizedBlocks": fmt.Sprintf("%+v", finalizedBlocks)})
		}
	}

	// check that latest finalized block address + 1 points to a non finalized block
	if s.IsFinalizedBlock(maxBlockNum+1, latestBlock) {
		return utils.LavaFormatError("Simulation: provider returned finalized hashes for an older latest block", nil, &map[string]string{
			"maxBlockNum": strconv.FormatInt(maxBlockNum, 10),
			"latestBlock": strconv.FormatInt(latestBlock, 10), "ChainID": s.ChainID, "Provider": providerAcc, "finalizedBlocks": fmt.Sprintf("%+v", finalizedBlocks),
		})
	}

	// New reply should have blocknum >= from block same provider
	if session.LatestBlock > latestBlock {
		//
		// Report same provider discrepancy
		// TODO:: Fill msg with incriminating data
		msg := conflicttypes.NewMsgDetection(s.Acc, nil, nil, nil)
		s.ClientCtx.SkipConfirm = true
		// txFactory := tx.NewFactoryCLI(s.ClientCtx, s.cmdFlags).WithChainID("lava")
		err := SimulateAndBroadCastTx(s.ClientCtx, s.txFactory, msg)
		if err != nil {
			return utils.LavaFormatError("validateProviderReply - SimulateAndBroadCastTx Failed", err, nil)
		}

		return utils.LavaFormatError("Simulation: Provider supplied an older latest block than it has previously", nil, &map[string]string{
			"session.LatestBlock": strconv.FormatInt(session.LatestBlock, 10),
			"latestBlock":         strconv.FormatInt(latestBlock, 10), "ChainID": s.ChainID, "Provider": providerAcc,
		})
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

type DataReliabilitySession struct {
	singleConsumerSession *lavasession.SingleConsumerSession
	epoch                 uint64
	providerPublicAddress string
	uniqueIdentifier      bool
}

type DataReliabilityResult struct {
	reply                 *pairingtypes.RelayReply
	relayRequest          *pairingtypes.RelayRequest
	providerPublicAddress string
}

func (s *Sentry) SendRelay(
	ctx context.Context,
	consumerSession *lavasession.SingleConsumerSession,
	sessionEpoch uint64,
	providerPubAddress string,
	cb_send_relay func(consumerSession *lavasession.SingleConsumerSession) (*pairingtypes.RelayReply, *pairingtypes.Relayer_RelaySubscribeClient, *pairingtypes.RelayRequest, time.Duration, bool, error),
	cb_send_reliability func(consumerSession *lavasession.SingleConsumerSession, dataReliability *pairingtypes.VRFData, providerAddress string) (*pairingtypes.RelayReply, *pairingtypes.RelayRequest, time.Duration, error),
	specCategory *spectypes.SpecCategory,
) (*pairingtypes.RelayReply, *pairingtypes.Relayer_RelaySubscribeClient, time.Duration, bool, error) {
	// callback user
	reply, replyServer, request, latency, fromCache, err := cb_send_relay(consumerSession)
	// error using this provider
	if err != nil {
		return nil, nil, 0, fromCache, utils.LavaFormatError("failed sending relay", lavasession.SendRelayError, &map[string]string{"ErrMsg": err.Error()})
	}

	if s.GetSpecDataReliabilityEnabled() && reply != nil && !fromCache {
		finalizedBlocks := map[int64]string{} // TODO:: define struct in relay response
		err = json.Unmarshal(reply.FinalizedBlocksHashes, &finalizedBlocks)
		if err != nil {
			return nil, nil, latency, fromCache, utils.LavaFormatError("failed in unmarshalling finalized blocks data", lavasession.SendRelayError, &map[string]string{"ErrMsg": err.Error()})
		}
		latestBlock := reply.LatestBlock

		// validate that finalizedBlocks makes sense
		err = s.validateProviderReply(finalizedBlocks, latestBlock, providerPubAddress, consumerSession)
		if err != nil {
			return nil, nil, latency, fromCache, utils.LavaFormatError("failed provider reply validation", lavasession.SendRelayError, &map[string]string{"ErrMsg": err.Error()})
		}
		//
		// Compare finalized block hashes with previous providers
		// Looks for discrepancy with current epoch providers
		// if no conflicts, insert into consensus and break
		// create new consensus group if no consensus matched
		// check for discrepancy with old epoch
		_, err := checkFinalizedHashes(s, providerPubAddress, latestBlock, finalizedBlocks, request, reply)
		if err != nil {
			return nil, nil, latency, fromCache, utils.LavaFormatError("failed to check finalized hashes", lavasession.SendRelayError, &map[string]string{"ErrMsg": err.Error()})
		}

		if specCategory.Deterministic && s.IsFinalizedBlock(request.RequestBlock, reply.LatestBlock) {
			var dataReliabilitySessions []*DataReliabilitySession

			// handle data reliability
			s.VrfSkMu.Lock()
			vrfRes0, vrfRes1 := utils.CalculateVrfOnRelay(request, reply, s.VrfSk, sessionEpoch)
			s.VrfSkMu.Unlock()
			// get two indexesMap for data reliability.
			indexesMap := s.DataReliabilityThresholdToSession([][]byte{vrfRes0, vrfRes1}, []bool{false, true})
			utils.LavaFormatDebug("DataReliability Randomized Values", &map[string]string{"vrf0": strconv.FormatUint(uint64(binary.LittleEndian.Uint32(vrfRes0)), 10), "vrf1": strconv.FormatUint(uint64(binary.LittleEndian.Uint32(vrfRes1)), 10), "decisionMap": fmt.Sprintf("%+v", indexesMap)})
			for idxExtract, uniqueIdentifier := range indexesMap { // go over each unique index and get a session.
				// the key in the indexesMap are unique indexes to fetch from consumerSessionManager
				dataReliabilityConsumerSession, providerPublicAddress, epoch, err := s.consumerSessionManager.GetDataReliabilitySession(ctx, providerPubAddress, idxExtract, sessionEpoch)
				if err != nil {
					if lavasession.DataReliabilityIndexRequestedIsOriginalProviderError.Is(err) {
						// index belongs to original provider, nothing is wrong here, print info and continue
						utils.LavaFormatInfo("DataReliability: Trying to get the same provider index as original request", &map[string]string{"provider": providerPubAddress, "Index": strconv.FormatInt(idxExtract, 10)})
					} else if lavasession.DataReliabilityAlreadySentThisEpochError.Is(err) {
						utils.LavaFormatInfo("DataReliability: Already Sent Data Reliability This Epoch To This Provider.", &map[string]string{"Provider": providerPubAddress, "Epoch": strconv.FormatUint(epoch, 10)})
					} else if lavasession.DataReliabilityEpochMismatchError.Is(err) {
						utils.LavaFormatInfo("DataReliability: Epoch changed cannot send data reliability", &map[string]string{"original_epoch": strconv.FormatUint(sessionEpoch, 10), "data_reliability_epoch": strconv.FormatUint(epoch, 10)})
						// if epoch changed, we can stop trying to get data reliability sessions
						break
					} else {
						utils.LavaFormatError("GetDataReliabilitySession", err, nil)
					}
					continue // if got an error continue to next index.
				}
				dataReliabilitySessions = append(dataReliabilitySessions, &DataReliabilitySession{
					singleConsumerSession: dataReliabilityConsumerSession,
					epoch:                 epoch,
					providerPublicAddress: providerPublicAddress,
					uniqueIdentifier:      uniqueIdentifier,
				})
			}

			sendReliabilityRelay := func(singleConsumerSession *lavasession.SingleConsumerSession, providerAddress string, differentiator bool) (relay_rep *pairingtypes.RelayReply, relay_req *pairingtypes.RelayRequest, err error) {
				var dataReliabilityLatency time.Duration
				s.VrfSkMu.Lock()
				vrf_res, vrf_proof := utils.ProveVrfOnRelay(request, reply, s.VrfSk, differentiator, sessionEpoch)
				s.VrfSkMu.Unlock()
				dataReliability := &pairingtypes.VRFData{
					Differentiator: differentiator,
					VrfValue:       vrf_res,
					VrfProof:       vrf_proof,
					ProviderSig:    reply.Sig,
					AllDataHash:    sigs.AllDataHash(reply, request),
					QueryHash:      utils.CalculateQueryHash(*request), // calculated from query body anyway, but we will use this on payment
					Sig:            nil,                                // calculated in cb_send_reliability
				}
				relay_rep, relay_req, dataReliabilityLatency, err = cb_send_reliability(singleConsumerSession, dataReliability, providerAddress)
				if err != nil {
					errRet := s.consumerSessionManager.OnDataReliabilitySessionFailure(singleConsumerSession, err)
					if errRet != nil {
						return nil, nil, utils.LavaFormatError("OnDataReliabilitySessionFailure Error", errRet, &map[string]string{"sendReliabilityError": err.Error()})
					}
					return nil, nil, utils.LavaFormatError("sendReliabilityRelay Could not get reply to reliability relay from provider", err, &map[string]string{"Address": providerAddress})
				}

				expectedBH, numOfProviders := s.ExpectedBlockHeight()
				err = s.consumerSessionManager.OnDataReliabilitySessionDone(singleConsumerSession, relay_rep.LatestBlock, singleConsumerSession.LatestRelayCu, dataReliabilityLatency, expectedBH, numOfProviders, s.GetProvidersCount())
				return relay_rep, relay_req, err
			}

			checkReliability := func() {
				numberOfReliabilitySessions := len(dataReliabilitySessions)
				if numberOfReliabilitySessions > supportedNumberOfVRFs {
					utils.LavaFormatError("Trying to use DataReliability with more than two vrf sessions, currently not supported", nil, &map[string]string{"number_of_DataReliabilitySessions": strconv.Itoa(numberOfReliabilitySessions)})
					return
				} else if numberOfReliabilitySessions == 0 {
					return
				}
				// apply first request and reply to dataReliabilityVerifications
				originalDataReliabilityResult := &DataReliabilityResult{reply: reply, relayRequest: request, providerPublicAddress: providerPubAddress}
				dataReliabilityVerifications := make([]*DataReliabilityResult, 0)

				for _, dataReliabilitySession := range dataReliabilitySessions {
					reliabilityReply, reliabilityRequest, err := sendReliabilityRelay(dataReliabilitySession.singleConsumerSession, dataReliabilitySession.providerPublicAddress, dataReliabilitySession.uniqueIdentifier)
					if err == nil && reliabilityReply != nil {
						dataReliabilityVerifications = append(dataReliabilityVerifications,
							&DataReliabilityResult{
								reply:                 reliabilityReply,
								relayRequest:          reliabilityRequest,
								providerPublicAddress: dataReliabilitySession.providerPublicAddress,
							})
					}
				}
				if len(dataReliabilityVerifications) > 0 {
					s.verifyReliabilityResults(originalDataReliabilityResult, dataReliabilityVerifications, numberOfReliabilitySessions)
				}
			}
			go checkReliability()
		}
	}

	return reply, replyServer, latency, fromCache, nil
}

// Verify all dataReliabilityVerifications with one another
// The original reply and request should be in dataReliabilityVerifications as well.
func (s *Sentry) verifyReliabilityResults(originalResult *DataReliabilityResult, dataReliabilityResults []*DataReliabilityResult, totalNumberOfSessions int) {
	verificationsLength := len(dataReliabilityResults)
	var conflict bool // if conflict is true at the end of the function, reliability failed.
	participatingProviders := make(map[string]string, verificationsLength+1)
	participatingProviders["originalAddress"] = originalResult.providerPublicAddress
	for idx, drr := range dataReliabilityResults {
		add := drr.providerPublicAddress
		participatingProviders["address"+strconv.Itoa(idx)] = add
		if !s.CompareRelaysAndReportConflict(originalResult.reply, originalResult.relayRequest, drr.reply, drr.relayRequest) {
			// if we failed to compare relays with original reply and result we need to stop and compare them to one another.
			conflict = true
		}
	}

	if conflict {
		// CompareRelaysAndReportConflict to each one of the data reliability relays to confirm that the first relay was'nt ok
		for idx1 := 0; idx1 < verificationsLength; idx1++ {
			for idx2 := (idx1 + 1); idx2 < verificationsLength; idx2++ {
				s.CompareRelaysAndReportConflict(
					dataReliabilityResults[idx1].reply,        // reply 1
					dataReliabilityResults[idx1].relayRequest, // request 1
					dataReliabilityResults[idx2].reply,        // reply 2
					dataReliabilityResults[idx2].relayRequest) // request 2
			}
		}
	}

	if !conflict && totalNumberOfSessions == verificationsLength { // if no conflict was detected data reliability was successful
		// all reliability sessions succeeded
		utils.LavaFormatInfo("Reliability verified successfully!", &participatingProviders)
	} else {
		utils.LavaFormatInfo("Reliability failed to verify!", &participatingProviders)
	}
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
				return false, utils.LavaFormatError("Simulation: Conflict found in discrepancyChecker", err, nil)
			}

			// if no conflicts, insert into consensus and break
			if !discrepancyResult {
				matchWithExistingConsensus = true
			} else {
				utils.LavaFormatError("Simulation: Conflict found between consensus and provider", err, &map[string]string{"Consensus idx": strconv.Itoa(idx), "provider": providerAcc})
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
				return false, utils.LavaFormatError("Simulation: prev epoch Conflict found in discrepancyChecker", err, nil)
			}

			if discrepancyResult {
				utils.LavaFormatError("Simulation: prev epoch Conflict found between consensus and provider", err, &map[string]string{"Consensus idx": strconv.Itoa(idx), "provider": providerAcc})
			}
		}
	}

	return false, nil
}

func (s *Sentry) IsFinalizedBlock(requestedBlock int64, latestBlock int64) bool {
	return spectypes.IsFinalizedBlock(requestedBlock, latestBlock, s.GetSpecBlockDistanceForFinalizedData())
}

func (s *Sentry) GetLatestFinalizedBlock(latestBlock int64) int64 {
	finalization_criteria := int64(s.GetSpecBlockDistanceForFinalizedData())
	return latestBlock - finalization_criteria
}

func (s *Sentry) clearAuthResponseCache(blockHeight int64) {
	// Clear cache
	s.authorizationCacheMutex.Lock()
	defer s.authorizationCacheMutex.Unlock()
	for key := range s.authorizationCache {
		if key < s.GetPrevEpochHeight() {
			delete(s.authorizationCache, key)
		}
	}
}

func (s *Sentry) getAuthResponseFromCache(consumer string, blockHeight uint64) *pairingtypes.QueryVerifyPairingResponse {
	// Check cache
	s.authorizationCacheMutex.RLock()
	defer s.authorizationCacheMutex.RUnlock()
	if entry, hasEntryForBlockHeight := s.authorizationCache[blockHeight]; hasEntryForBlockHeight {
		if cachedResponse, ok := entry[consumer]; ok {
			return cachedResponse
		}
	}

	return nil
}

func (s *Sentry) IsAuthorizedConsumer(ctx context.Context, consumer string, blockHeight uint64) (*pairingtypes.QueryVerifyPairingResponse, error) {
	res := s.getAuthResponseFromCache(consumer, blockHeight)
	if res != nil {
		// User was authorized before, response returned from cache.
		return res, nil
	}

	res, err := s.pairingQueryClient.VerifyPairing(context.Background(), &pairingtypes.QueryVerifyPairingRequest{
		ChainID:  s.ChainID,
		Client:   consumer,
		Provider: s.Acc,
		Block:    blockHeight,
	})
	if err != nil {
		return nil, err
	}
	if res.GetValid() {
		s.authorizationCacheMutex.Lock()
		if _, ok := s.authorizationCache[blockHeight]; !ok {
			s.authorizationCache[blockHeight] = map[string]*pairingtypes.QueryVerifyPairingResponse{} // init
		}
		s.authorizationCache[blockHeight][consumer] = res
		s.authorizationCacheMutex.Unlock()
		return res, nil
	}

	return nil, utils.LavaFormatError("invalid self pairing with consumer", nil, &map[string]string{"consumer address": consumer, "CurrentBlock": strconv.FormatInt(s.GetBlockHeight(), 10)})
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
	return false, utils.LavaFormatError("invalid pairing with consumer", nil, &map[string]string{"consumer address": consumer, "CurrentBlock": strconv.FormatInt(s.GetBlockHeight(), 10), "requested block": strconv.FormatUint(block, 10)})
}

func (s *Sentry) GetReliabilityThreshold() uint32 {
	return s.serverSpec.ReliabilityThreshold
}

func (s *Sentry) GetSpecName() string {
	return s.serverSpec.Name
}

func (s *Sentry) GetSpecDataReliabilityEnabled() bool {
	return s.serverSpec.DataReliabilityEnabled
}

func (s *Sentry) GetSpecBlockDistanceForFinalizedData() uint32 {
	return s.serverSpec.BlockDistanceForFinalizedData
}

func (s *Sentry) GetSpecBlocksInFinalizationProof() uint32 {
	return s.serverSpec.BlocksInFinalizationProof
}

func (s *Sentry) GetChainID() string {
	return s.serverSpec.Index
}

func (s *Sentry) GetAverageBlockTime() int64 {
	return s.serverSpec.AverageBlockTime
}

func (s *Sentry) MatchSpecApiByName(name string) (spectypes.ServiceApi, bool) {
	s.specMu.RLock()
	defer s.specMu.RUnlock()
	// TODO: make it faster and better by not doing a regex instead using a better algorithm
	for apiName, api := range s.serverApis {
		re, err := regexp.Compile(apiName)
		if err != nil {
			utils.LavaFormatError("regex Compile api", err, &map[string]string{"apiName": apiName})
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

func (s *Sentry) GetPrevEpochHeight() uint64 {
	return atomic.LoadUint64(&s.prevEpoch)
}

func (s *Sentry) SetPrevEpochHeight(blockHeight uint64) {
	atomic.StoreUint64(&s.prevEpoch, blockHeight)
}

func (s *Sentry) GetOverlapSize() uint64 {
	return atomic.LoadUint64(&s.EpochBlocksOverlap)
}

func (s *Sentry) GetCUServiced() uint64 {
	return atomic.LoadUint64(&s.totalCUServiced)
}

func (s *Sentry) SetCUServiced(cu uint64) {
	atomic.StoreUint64(&s.totalCUServiced, cu)
}

func (s *Sentry) UpdateCUServiced(cu uint64) {
	// we lock because we dont want the value changing after we read it before we store
	s.PaymentsMu.Lock()
	defer s.PaymentsMu.Unlock()
	currentCU := atomic.LoadUint64(&s.totalCUServiced)
	atomic.StoreUint64(&s.totalCUServiced, currentCU+cu)
}

func (s *Sentry) GetMaxCUForUser(ctx context.Context, address string, chainID string) (maxCu uint64, err error) {
	UserEntryRes, err := s.pairingQueryClient.UserEntry(ctx, &pairingtypes.QueryUserEntryRequest{ChainID: chainID, Address: address, Block: uint64(s.GetBlockHeight())})
	if err != nil {
		return 0, utils.LavaFormatError("failed querying StakeEntry for consumer", err, &map[string]string{"chainID": chainID, "address": address, "block": strconv.FormatInt(s.GetBlockHeight(), 10)})
	}
	return UserEntryRes.GetMaxCU(), nil
}

func (s *Sentry) GetVrfPkAndMaxCuForUser(ctx context.Context, address string, chainID string, requestBlock int64) (vrfPk *utils.VrfPubKey, maxCu uint64, err error) {
	UserEntryRes, err := s.pairingQueryClient.UserEntry(ctx, &pairingtypes.QueryUserEntryRequest{ChainID: chainID, Address: address, Block: uint64(requestBlock)})
	if err != nil {
		return nil, 0, utils.LavaFormatError("StakeEntry querying for consumer failed", err, &map[string]string{"chainID": chainID, "address": address, "block": strconv.FormatInt(requestBlock, 10)})
	}
	vrfPk = &utils.VrfPubKey{}
	vrfPk, err = vrfPk.DecodeFromBech32(UserEntryRes.GetConsumer().Vrfpk)
	if err != nil {
		err = utils.LavaFormatError("decoding vrfpk from bech32", err, &map[string]string{"chainID": chainID, "address": address, "block": strconv.FormatInt(requestBlock, 10), "UserEntryRes": fmt.Sprintf("%v", UserEntryRes)})
	}
	return vrfPk, UserEntryRes.GetMaxCU(), err
}

func (s *Sentry) ExpectedBlockHeight() (int64, int) {
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
	highestBlockNumber = FindHighestBlockNumber(s.prevEpochProviderHashesConsensus) // update the highest in place
	highestBlockNumber = FindHighestBlockNumber(s.providerHashesConsensus)

	now := time.Now()
	calcExpectedBlocks := func(listProviderHashesConsensus []ProviderHashesConsensus) []int64 {
		listExpectedBH := []int64{}
		for _, providerHashesConsensus := range listProviderHashesConsensus {
			for _, providerDataContainer := range providerHashesConsensus.agreeingProviders {
				expected := providerDataContainer.LatestFinalizedBlock + (now.Sub(providerDataContainer.LatestBlockTime).Milliseconds() / averageBlockTime_ms) // interpolation
				// limit the interpolation to the highest seen block height
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
		data_len := len(data)
		if data_len == 0 {
			return 0
		} else if data_len%2 == 0 {
			median = (data[data_len/2-1] + data[data_len/2]/2.0)
		} else {
			median = data[data_len/2]
		}
		return median
	}

	return median(listExpectedBlockHeights) - s.serverSpec.AllowedBlockLagForQosSync, len(listExpectedBlockHeights)
}

func NewSentry(
	clientCtx client.Context,
	txFactory tx.Factory,
	chainID string,
	isUser bool,
	voteInitiationCb func(ctx context.Context, voteID string, voteDeadline uint64, voteParams *VoteParams),
	newEpochCb func(epochHeight int64),
	apiInterface string,
	vrf_sk vrf.PrivateKey,
	flagSet *pflag.FlagSet,
	serverID uint64,
	epochErrorWhitelist *common.EpochErrorWhitelist,
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
		txFactory:               txFactory,
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
		authorizationCache:      map[uint64]map[string]*pairingtypes.QueryVerifyPairingResponse{},
		epochErrorWhitelist:     epochErrorWhitelist,
	}
}

func UpdateRequestedBlock(request *pairingtypes.RelayRequest, response *pairingtypes.RelayReply) {
	// since sometimes the user is sending requested block that is a magic like latest, or earliest we need to specify to the reliability what it is
	request.RequestBlock = ReplaceRequestedBlock(request.RequestBlock, response.LatestBlock)
}

func ReplaceRequestedBlock(requestedBlock int64, latestBlock int64) int64 {
	switch requestedBlock {
	case spectypes.LATEST_BLOCK:
		return latestBlock
	case spectypes.SAFE_BLOCK:
		return latestBlock
	case spectypes.FINALIZED_BLOCK:
		return latestBlock
	case spectypes.EARLIEST_BLOCK:
		return spectypes.NOT_APPLICABLE // TODO: add support for earliest block reliability
	}
	return requestedBlock
}
