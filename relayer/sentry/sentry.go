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
	"github.com/lavanet/lava/relayer/parser"
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

//Constants

var AvailabilityPrecentage sdk.Dec = sdk.NewDecWithPrec(5, 2) //TODO move to params pairing
const (
	MaxConsecutiveConnectionAttemts = 3
	PercentileToCalculateLatency    = 0.9
	MinProvidersForSync             = 0.6
	LatencyThresholdStatic          = 1 * time.Second
	LatencyThresholdSlope           = 1 * time.Millisecond
	StaleEpochDistance              = 3 // relays done 3 epochs back are ready to be rewarded
)

func (cs *ClientSession) CalculateQoS(cu uint64, latency time.Duration, blockHeightDiff int64, numOfProviders int, servicersToCount int64) {

	if cs.QoSInfo.LastQoSReport == nil {
		cs.QoSInfo.LastQoSReport = &pairingtypes.QualityOfServiceReport{}
	}

	downtimePrecentage := sdk.NewDecWithPrec(int64(cs.QoSInfo.TotalRelays-cs.QoSInfo.AnsweredRelays), 0).Quo(sdk.NewDecWithPrec(int64(cs.QoSInfo.TotalRelays), 0))
	cs.QoSInfo.LastQoSReport.Availability = sdk.MaxDec(sdk.ZeroDec(), AvailabilityPrecentage.Sub(downtimePrecentage).Quo(AvailabilityPrecentage))
	if sdk.OneDec().GT(cs.QoSInfo.LastQoSReport.Availability) {
		fmt.Printf("QoS Availibility: %s, downtime precent : %s \n", cs.QoSInfo.LastQoSReport.Availability.String(), downtimePrecentage.String())
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
		fmt.Printf("QoS Sync: %s, block diff: %d , sync score: %d / %d \n", cs.QoSInfo.LastQoSReport.Sync.String(), blockHeightDiff, cs.QoSInfo.SyncScoreSum, cs.QoSInfo.TotalSyncScore)
	}
}

type RelayerClientWrapper struct {
	Client *pairingtypes.RelayerClient
	Acc    string
	Addr   string

	ConnectionRefusals uint64
	SessionsLock       sync.Mutex
	Sessions           map[int64]*ClientSession
	MaxComputeUnits    uint64
	UsedComputeUnits   uint64
	ReliabilitySent    bool
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
	blockHeight  int64
	currentEpoch int64
	EpochSize    uint64

	//
	// Spec storage (rw mutex)
	specMu     sync.RWMutex
	specHash   []byte
	serverSpec spectypes.Spec
	serverApis map[string]spectypes.ServiceApi
	taggedApis map[string]spectypes.ServiceApi

	// (client only)
	// Pairing storage (rw mutex)
	pairingMu        sync.RWMutex
	pairingHash      []byte
	pairing          []*RelayerClientWrapper
	pairingAddresses []string
	pairingPurgeLock sync.Mutex
	pairingPurge     []*RelayerClientWrapper
	VrfSkMu          sync.Mutex
	VrfSk            vrf.PrivateKey

	// every entry in providerHashesConsensus is conflicted with the other entries
	providerHashesConsensus          []ProviderHashesConsensus
	prevEpochProviderHashesConsensus []ProviderHashesConsensus
	providerDataContainersMu         sync.Mutex
}

func (s *Sentry) GetEpochSize(ctx context.Context) error {
	res, err := s.epochStorageQueryClient.Params(ctx, &epochstoragetypes.QueryParamsRequest{})
	if err != nil {
		return err
	}
	atomic.StoreUint64(&s.EpochSize, res.GetParams().EpochBlocks)
	return nil
}

func (s *Sentry) getEarliestSession(ctx context.Context) error {
	res, err := s.epochStorageQueryClient.EpochDetails(ctx, &epochstoragetypes.QueryGetEpochDetailsRequest{})
	if err != nil {
		return err
	}
	earliestBlock := res.GetEpochDetails().EarliestStart
	atomic.StoreUint64(&s.earliestSavedBlock, earliestBlock)
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
		return err
	}
	servicers := res.GetProviders()
	if servicers == nil || len(servicers) == 0 {
		return errors.New("no servicers found")
	}

	//
	// Check if updated
	hash := tendermintcrypto.Sha256([]byte(res.String())) // TODO: we use cheaper algo for speed
	if bytes.Equal(s.pairingHash, hash) {
		return nil
	}
	s.pairingHash = hash

	//
	// Set
	pairing := []*RelayerClientWrapper{}
	pairingAddresses := []string{} //this object will not be mutated for vrf calculations
	for _, servicer := range servicers {
		//
		// Sanity
		servicerEndpoints := servicer.GetEndpoints()
		if servicerEndpoints == nil || len(servicerEndpoints) == 0 {
			log.Println("servicerEndpoints == nil || len(servicerEndpoints) == 0")
			continue
		}

		relevantEndpoints := []epochstoragetypes.Endpoint{}
		for _, endpoint := range servicerEndpoints {
			//only take into account endpoints that use the same api interface
			if endpoint.UseType == s.ApiInterface {
				relevantEndpoints = append(relevantEndpoints, endpoint)
			}
		}
		if len(relevantEndpoints) == 0 {
			log.Println(fmt.Sprintf("No relevant endpoints for apiInterface %s: %v", s.ApiInterface, servicerEndpoints))
			continue
		}

		maxcu, err := s.GetMaxCUForUser(ctx, s.Acc, servicer.Chain)
		if err != nil {
			return err
		}
		//
		// TODO: decide how to use multiple addresses from the same operator
		pairing = append(pairing, &RelayerClientWrapper{
			Acc:                servicer.Address,
			Addr:               relevantEndpoints[0].IPPORT,
			Sessions:           map[int64]*ClientSession{},
			MaxComputeUnits:    maxcu,
			ReliabilitySent:    false,
			ConnectionRefusals: 0,
		})
		pairingAddresses = append(pairingAddresses, servicer.Address)
	}
	s.pairingMu.Lock()
	defer s.pairingMu.Unlock()
	s.pairingPurgeLock.Lock()
	defer s.pairingPurgeLock.Unlock()
	s.pairingPurge = append(s.pairingPurge, s.pairing...) // append old connections to purge list
	s.pairing = pairing                                   // replace with new connections
	s.pairingAddresses = pairingAddresses
	log.Println("update pairing list!", pairing)

	// TODO:: new epoch, reset field containing previous finalizedBlocks of providers of epoch - 2
	// TODO:: keep latest provider finalized blocks and prev finalied from epoch - 1
	s.prevEpochProviderHashesConsensus = s.providerHashesConsensus

	return nil
}

func (s *Sentry) GetSpecHash() []byte {
	s.specMu.Lock()
	defer s.specMu.Unlock()
	return s.specHash
}

func (s *Sentry) GetServicersToPairCount() int64 {
	s.pairingMu.Lock()
	defer s.pairingMu.Unlock()
	return int64(len(s.pairingAddresses))
}

func (s *Sentry) getSpec(ctx context.Context) error {
	//
	// TODO: decide if it's fatal to not have spec (probably!)
	spec, err := s.specQueryClient.Chain(ctx, &spectypes.QueryChainRequest{
		ChainID: s.ChainID,
	})
	if err != nil {
		return err
	}

	//
	// Check if updated
	hash := tendermintcrypto.Sha256([]byte(spec.String())) // TODO: we use cheaper algo for speed
	if bytes.Equal(s.specHash, hash) {
		return nil
	}
	s.specHash = hash

	//
	// Update

	log.Println(fmt.Sprintf("Sentry updated spec for chainID: %s Spec name:%s", spec.Spec.Index, spec.Spec.Name))
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
		fmt.Printf("BAD: %s", err)
		return err
	}
	s.NewBlockEvents = txs

	query = "tm.event = 'Tx'"
	txs, err = s.rpcClient.Subscribe(ctx, "test-client", query)
	if err != nil {
		fmt.Printf("BAD: %s", err)
		return err
	}
	s.NewTransactionEvents = txs
	//
	// Get spec for the first time
	err = s.getSpec(ctx)
	if err != nil {
		return err
	}

	//
	// Get pairing for the first time, for clients
	err = s.getPairing(ctx)
	if err != nil {
		return err
	}

	//
	// Sanity
	if !s.isUser {
		servicers, err := s.pairingQueryClient.Providers(ctx, &pairingtypes.QueryProvidersRequest{
			ChainID: s.GetChainID(),
		})
		if err != nil {
			return err
		}
		found := false
		for _, servicer := range servicers.GetStakeEntry() {
			if servicer.Address == s.Acc {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("servicer not staked for spec: %s %s", s.GetSpecName(), s.GetChainID())
		}
	}

	s.GetEpochSize(ctx) // ARITODO:: Tell omer tihs has to be here since we use epoch size early on

	return nil
}

func removeFromSlice(s []int, i int) []int {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func (s *Sentry) ListenForTXEvents(ctx context.Context) {
	for e := range s.NewTransactionEvents {

		switch data := e.Data.(type) {
		case tenderminttypes.EventDataTx:
			//got new TX event
			if servicerAddrList, ok := e.Events["lava_relay_payment.provider"]; ok {
				for idx, servicerAddr := range servicerAddrList {
					if s.Acc == servicerAddr && s.ChainID == e.Events["lava_relay_payment.chainID"][idx] {
						fmt.Printf("\nReceived relay payment of %s for CU: %s\n", e.Events["lava_relay_payment.Mint"][idx], e.Events["lava_relay_payment.CU"][idx])

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

				s.SetCurrentEpochHeight(data.Block.Height)
				s.GetEpochSize(ctx)

				if s.newEpochCb != nil {
					go s.newEpochCb(data.Block.Height - StaleEpochDistance*int64(s.EpochSize)) // Currently this is only askForRewards
				}

				//
				// Update specs
				err := s.getSpec(ctx)
				if err != nil {
					log.Println("error: getSpec", err)
				}

				//update expected payments deadline, and log missing payments
				//TODO: make this from the event lava_earliest_epoch instead
				s.getEarliestSession(ctx)
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

		}
	}
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

//expecting caller to lock
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
			//
			// TODO: we should retry with another addr
			conn, err := s.connectRawClient(ctx, wrap.Addr)
			if err != nil {
				return nil, -1, fmt.Errorf("Error getting pairing from: %s, error: %w", wrap.Addr, err)
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
					s.movePairingEntryToPurge(wrap, index, false)
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

func (s *Sentry) CompareRelaysAndReportConflict(reply0 *pairingtypes.RelayReply, reply1 *pairingtypes.RelayReply) (ok bool) {
	compare_result := bytes.Compare(reply0.Data, reply1.Data)
	if compare_result == 0 {
		//they have equal data
		return true
	}
	//they have different data! report!
	log.Println(fmt.Sprintf("[-] DataReliability detected mismatching results! \n1>>%s \n2>>%s\nReporting...", reply0.Data, reply1.Data))
	msg := conflicttypes.NewMsgDetection(s.Acc, nil, nil)
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

//
//
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
				msg := conflicttypes.NewMsgDetection(s.Acc, nil, nil)
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
		msg := conflicttypes.NewMsgDetection(s.Acc, nil, nil)
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
	cb_send_reliability func(clientSession *ClientSession, dataReliability *pairingtypes.VRFData) (*pairingtypes.RelayReply, error),
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
	clientSession := getClientSessionFromWrap(wrap)

	// call user
	reply, request, err := cb_send_relay(clientSession)
	//error using this provider
	if err != nil {
		if clientSession.QoSInfo.ConsecutiveTimeOut >= MaxConsecutiveConnectionAttemts && clientSession.QoSInfo.LastQoSReport.Availability.IsZero() {
			s.movePairingEntryToPurge(wrap, index, true)
		}
		return reply, err
	}

	providerAcc := clientSession.Client.Acc
	clientSession.Lock.Unlock() //function call returns a locked session, we need to unlock it

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
			vrfRes0, vrfRes1 := utils.CalculateVrfOnRelay(request, reply, s.VrfSk)
			s.VrfSkMu.Unlock()
			address0, address1 := s.DataReliabilityThresholdToAddress(vrfRes0, vrfRes1)

			//Printing VRF Data
			// st1, _ := bech32.ConvertAndEncode("", vrfRes0)
			// st2, _ := bech32.ConvertAndEncode("", vrfRes1)
			// log.Printf("Finalized Block reply from %s received res %s, %s, addresses: %s, %s\n", providerAcc, st1, st2, address0, address1)

			sendReliabilityRelay := func(address string, differentiator bool) (relay_rep *pairingtypes.RelayReply, err error) {
				if address != "" && address != providerAcc {
					wrap, index, err := s.specificPairing(ctx, address)
					if err != nil {
						// failed to get clientWrapper for this address, skip reliability
						log.Println("Reliability error: Could not get client specific pairing wrap for address: ", address, err)
						return nil, err
					} else {
						canSendReliability := s.CheckAndMarkReliabilityForThisPairing(wrap) //TODO: this will still not perform well for multiple clients, we need to get the reliability proof in the error and not burn the provider
						if canSendReliability {
							s.VrfSkMu.Lock()
							vrf_res, vrf_proof := utils.ProveVrfOnRelay(request, reply, s.VrfSk, differentiator)
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
							relay_rep, err = cb_send_reliability(clientSession, dataReliability)
							if err != nil {
								log.Println("Reliability ERROR: Could not get reply to reliability relay from provider: ", address, err)
								if clientSession.QoSInfo.ConsecutiveTimeOut >= 3 && clientSession.QoSInfo.LastQoSReport.Availability.IsZero() {
									s.movePairingEntryToPurge(wrap, index, true)
								}
								return nil, err
							}
							clientSession.Lock.Unlock() //function call returns a locked session, we need to unlock it
							return relay_rep, nil
						} else {
							log.Println("Reliability already Sent in this epoch to this provider")
							return nil, nil
						}
					}
				} else {
					if isSecure {
						//send reliability on the client's expense
						log.Println("secure flag Not Implemented, TODO:")
					}
					return nil, fmt.Errorf("Reliability ERROR: is not a valid reliability VRF address result")
				}
			}

			checkReliability := func() {
				reply0, err0 := sendReliabilityRelay(address0, false)
				reply1, err1 := sendReliabilityRelay(address1, true)
				ok := true
				check0 := err0 == nil && reply0 != nil
				check1 := err1 == nil && reply1 != nil
				if check0 {
					ok = ok && s.CompareRelaysAndReportConflict(reply, reply0)
				}
				if check1 {
					ok = ok && s.CompareRelaysAndReportConflict(reply, reply1)
				}
				if !ok && check0 && check1 {
					s.CompareRelaysAndReportConflict(reply0, reply1)
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
	//TODO: implement this for the chain, make a method for spec to verify this on chain?
	switch requestedBlock {
	case parser.NOT_APPLICABLE:
		return false
	default:
		//TODO: load this from spec
		//TODO: regard earliest block from spec
		finalization_criteria := int64(s.GetSpecFinalizationCriteria())
		if requestedBlock <= latestBlock-finalization_criteria {
			// log.Println("requestedBlock <= latestBlock-finalization_criteria returns true: ", requestedBlock, latestBlock)
			return true
			// return false
		}
	}
	return false
}

func (s *Sentry) GetLatestFinalizedBlock(latestBlock int64) int64 {
	finalization_criteria := int64(s.GetSpecFinalizationCriteria())
	return latestBlock - finalization_criteria
}

func (s *Sentry) movePairingEntryToPurge(wrap *RelayerClientWrapper, index int, lockpairing bool) {
	log.Printf("Warning! Jailing provider %s for this epoch\n", wrap.Acc)
	if lockpairing {
		s.pairingMu.Lock()
		defer s.pairingMu.Unlock()
	}

	s.pairingPurgeLock.Lock()
	defer s.pairingPurgeLock.Unlock()
	//move to purge list
	s.pairingPurge = append(s.pairingPurge, wrap)
	s.pairing[index] = s.pairing[len(s.pairing)-1]
	s.pairing = s.pairing[:len(s.pairing)-1]
}

func (s *Sentry) IsAuthorizedUser(ctx context.Context, user string) (*pairingtypes.QueryVerifyPairingResponse, error) {
	//
	// TODO: cache results!
	res, err := s.pairingQueryClient.VerifyPairing(context.Background(), &pairingtypes.QueryVerifyPairingRequest{
		ChainID:  s.ChainID,
		Client:   user,
		Provider: s.Acc,
		Block:    uint64(s.GetBlockHeight()),
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
	return false, fmt.Errorf("invalid pairing with consumer %s, provider %s block: %d", consumer, provider, block)
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

func (s *Sentry) GetCurrentEpochHeight() int64 {
	return atomic.LoadInt64(&s.currentEpoch)
}

func (s *Sentry) SetCurrentEpochHeight(blockHeight int64) {
	atomic.StoreInt64(&s.currentEpoch, blockHeight)
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

func NewSentry(
	clientCtx client.Context,
	chainID string,
	isUser bool,
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
		log.Fatal("Invalid block height, error: ", err)
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
		serverID:                serverID,
	}
}

func UpdateRequestedBlock(request *pairingtypes.RelayRequest, response *pairingtypes.RelayReply) {
	//since sometimes the user is sending requested block that is a magic like latest, or earliest we need to specify to the reliability what it is
	switch request.RequestBlock {
	case parser.LATEST_BLOCK:
		request.RequestBlock = response.LatestBlock
	case parser.EARLIEST_BLOCK:
		request.RequestBlock = parser.NOT_APPLICABLE // TODO: add support for earliest block reliability
	}
}
