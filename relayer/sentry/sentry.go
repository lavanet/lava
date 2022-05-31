package sentry

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coniks-sys/coniks-go/crypto/vrf"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tenderminttypes "github.com/tendermint/tendermint/types"
	grpc "google.golang.org/grpc"
)

type ClientSession struct {
	CuSum     uint64
	SessionId int64
	Client    *RelayerClientWrapper
	Lock      sync.Mutex
	RelayNum  uint64
}

type RelayerClientWrapper struct {
	Client *pairingtypes.RelayerClient
	Acc    string
	Addr   string

	SessionsLock     sync.Mutex
	Sessions         map[int64]*ClientSession
	MaxComputeUnits  uint64
	UsedComputeUnits uint64
}

type PaymentRequest struct {
	CU                  uint64
	BlockHeightDeadline int64
	Amount              sdk.Coin
	Client              sdk.AccAddress
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
	newBlockCb              func()
	ApiInterface            string
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
	blockHeight int64

	//
	// Spec storage (rw mutex)
	specMu     sync.RWMutex
	specHash   []byte
	serverSpec spectypes.Spec
	serverApis map[string]spectypes.ServiceApi

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
			Acc:             servicer.Address,
			Addr:            relevantEndpoints[0].IPPORT,
			Sessions:        map[int64]*ClientSession{},
			MaxComputeUnits: maxcu,
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

	return nil
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
			}
		}
	}
	s.specMu.Lock()
	defer s.specMu.Unlock()
	s.serverSpec = spec.Spec
	s.serverApis = serverApis

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
						s.UpdatePaidCU(paidCU)
						s.AppendToReceivedPayments(PaymentRequest{CU: paidCU, BlockHeightDeadline: data.Height, Amount: coin, Client: clientAddr})
						found := s.RemoveExpectedPayment(paidCU, clientAddr, data.Height)
						if !found {
							fmt.Printf("ERROR: payment received, did not find matching expectancy from correct client Need to add suppot for partial payment\n %s", s.PrintExpectedPAyments())
						} else {
							fmt.Printf("SUCCESS: payment received as expected\n")
						}

					}
				}
			}

		}
	}
}

func (s *Sentry) RemoveExpectedPayment(paidCUToFInd uint64, expectedClient sdk.AccAddress, blockHeight int64) bool {
	s.PaymentsMu.Lock()
	defer s.PaymentsMu.Unlock()
	for idx, expectedPayment := range s.expectedPayments {
		//TODO: make sure the payment is not too far from expected block, expectedPayment.BlockHeightDeadline == blockHeight
		if expectedPayment.CU == paidCUToFInd && expectedPayment.Client.Equals(expectedClient) {
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

			if s.newBlockCb != nil {
				go s.newBlockCb()
			}

			if _, ok := e.Events["lava_new_epoch.height"]; ok {
				fmt.Printf("New session: Height: %d \n", data.Block.Height)

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

	/*connectCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()*/
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	/*defer conn.Close()*/

	c := pairingtypes.NewRelayerClient(conn)
	return &c, nil
}

func (s *Sentry) specificPairing(ctx context.Context, address string) (*RelayerClientWrapper, int, error) {

	s.pairingMu.RLock()
	defer s.pairingMu.RUnlock()
	if len(s.pairing) == 0 {
		return nil, -1, errors.New("no pairings available")
	}
	//
	for index, wrap := range s.pairing {
		if wrap.Addr != address {
			continue
		}
		if wrap.Client == nil {
			wrap.SessionsLock.Lock()
			defer wrap.SessionsLock.Unlock()
			//
			// TODO: we should retry with another addr
			conn, err := s.connectRawClient(ctx, wrap.Addr)
			if err != nil {
				return nil, -1, err
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
	if len(s.pairing) == 0 {
		return nil, -1, errors.New("no pairings available")
	}

	//
	index := rand.Intn(len(s.pairing))
	wrap := s.pairing[index]

	if wrap.Client == nil {
		wrap.SessionsLock.Lock()
		defer wrap.SessionsLock.Unlock()
		//
		// TODO: we should retry with another addr
		conn, err := s.connectRawClient(ctx, wrap.Addr)
		if err != nil {

			return nil, -1, err
		}
		wrap.Client = conn
	}
	return wrap, index, nil
}

func (s *Sentry) DataReliabilityAddress(vrf0 []byte, vrf1 []byte) (address0 string, address1 string) {
	// check for the VRF thresholds and if holds true send a relay to the provider
	s.specMu.RLock()
	reliabilityThreshold := s.serverSpec.ReliabilityThreshold
	s.specMu.RUnlock()
	getAddressForVrf := func(vrf []byte) (address string) {
		vrf_num := binary.LittleEndian.Uint32(vrf)
		if vrf_num <= reliabilityThreshold {
			// need to send relay with VRF
			s.pairingMu.RLock()
			modulo := uint32(len(s.pairingAddresses))
			index := vrf_num % modulo
			address = s.pairingAddresses[index]
			s.pairingMu.RUnlock()
		}
		return
	}
	address0 = getAddressForVrf(vrf0)
	address1 = getAddressForVrf(vrf1)
	if address0 == address1 {
		//can't have both with the same provider
		address1 = ""
	}
	return
}

func (s *Sentry) SendRelay(
	ctx context.Context,
	cb func(clientSession *ClientSession) (*pairingtypes.RelayReply, *pairingtypes.RelayRequest, error),
	cb_reliability func(clientSession *ClientSession, dataReliability *pairingtypes.VRFData) (*pairingtypes.RelayReply, *pairingtypes.RelayRequest, error),
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

		clientSession := &ClientSession{
			SessionId: rand.Int63(),
			Client:    wrap,
		}
		clientSession.Lock.Lock()
		wrap.Sessions[clientSession.SessionId] = clientSession

		return clientSession
	}
	// Get or create session and lock it
	clientSession := getClientSessionFromWrap(wrap)

	//
	// call user
	reply, request, err := cb(clientSession)
	//error using this provider
	if err != nil {
		s.movePairingEntryToPurge(wrap, index)
		return reply, err
	}

	providerAcc := clientSession.Client.Acc
	clientSession.Lock.Unlock() //function call returns a locked session, we need to unlock it

	if s.IsFinalizedBlock(request.RequestBlock, reply.LatestBlock) {
		// handle data reliability
		s.VrfSkMu.Lock()
		vrfRes0, vrfRes1 := utils.CalculateVrfOnRelay(request, reply, s.VrfSk)
		s.VrfSkMu.Unlock()
		address0, address1 := s.DataReliabilityAddress(vrfRes0, vrfRes1)
		sendReliabilityRelay := func(address string, differentiator bool, vrfProof []byte) error {
			if address != "" && address != providerAcc {
				wrap, index, err := s.specificPairing(ctx, address)
				if err != nil {
					// failed to get clientWrapper for this address, skip reliability
					log.Println("Reliability error: Could not get client specific pairing wrap for address: ", address, err)
				} else {
					dataReliability := &pairingtypes.VRFData{Differentiator: differentiator,
						VrfProof:    vrfProof,
						ProviderSig: reply.Sig,
						AllDataHash: sigs.AllDataHash(reply, request),
						QueryHash:   nil, //calculated from query body anyway, this field is for the consensus payment
						Sig:         nil, //calculated in the callback
					}
					clientSession = getClientSessionFromWrap(wrap)
					_, _, err = cb_reliability(clientSession, dataReliability)
					if err != nil {
						log.Println("error: Could not send reliability relay to provider: ", address, err)
						s.movePairingEntryToPurge(wrap, index)
					}
				}
				return err
			}
			return nil
		}

		go sendReliabilityRelay(address0, false, vrfRes0)
		go sendReliabilityRelay(address1, true, vrfRes1)
	}
	return reply, nil
}

func (s *Sentry) IsFinalizedBlock(requestedBlock int64, latestBlock int64) bool {
	//TODO: implement this for the chain, make a method for spec to verify this on chain?
	switch requestedBlock {
	case parser.NOT_APPLICABLE:
		return false
	default:
		//TODO: load this from spec
		//TODO: regard earliest block from spec
		finalization_criteria := int64(7)
		if requestedBlock <= latestBlock-finalization_criteria {
			return true
		}
	}
	return false
}

func (s *Sentry) movePairingEntryToPurge(wrap *RelayerClientWrapper, index int) {
	s.pairingMu.Lock()
	s.pairingPurgeLock.Lock()
	defer s.pairingMu.Unlock()
	defer s.pairingPurgeLock.Unlock()
	//move to purge list
	s.pairingPurge = append(s.pairingPurge, wrap)
	s.pairing[index] = s.pairing[len(s.pairing)-1]
	s.pairing = s.pairing[:len(s.pairing)-1]
}

func (s *Sentry) IsAuthorizedUser(ctx context.Context, user string) (bool, error) {
	//
	// TODO: cache results!

	res, err := s.pairingQueryClient.VerifyPairing(context.Background(), &pairingtypes.QueryVerifyPairingRequest{
		ChainID:  s.ChainID,
		Client:   user,
		Provider: s.Acc,
		Block:    uint64(s.GetBlockHeight()),
	})
	if err != nil {
		return false, err
	}
	if res.Valid {
		return true, nil
	}
	return false, fmt.Errorf("invalid pairing with user CurrentBlock: %d", s.GetBlockHeight())
}

func (s *Sentry) GetSpecName() string {
	return s.serverSpec.Name
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

func (s *Sentry) GetBlockHeight() int64 {
	return atomic.LoadInt64(&s.blockHeight)
}

func (s *Sentry) SetBlockHeight(blockHeight int64) {
	atomic.StoreInt64(&s.blockHeight, blockHeight)
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

func (s *Sentry) GetMaxCUForUser(ctx context.Context, address string, chainID string) (uint64, error) {
	maxcuRes, err := s.pairingQueryClient.UserMaxCu(ctx, &pairingtypes.QueryUserMaxCuRequest{ChainID: chainID, Address: address})
	if err != nil {
		return 0, err
	}

	return maxcuRes.GetMaxCu(), err
}

func NewSentry(
	clientCtx client.Context,
	chainID string,
	isUser bool,
	newBlockCb func(),
	apiInterface string,
	vrf_sk vrf.PrivateKey,
) *Sentry {
	rpcClient := clientCtx.Client
	specQueryClient := spectypes.NewQueryClient(clientCtx)
	pairingQueryClient := pairingtypes.NewQueryClient(clientCtx)
	epochStorageQueryClient := epochstoragetypes.NewQueryClient(clientCtx)
	acc := clientCtx.GetFromAddress().String()

	return &Sentry{
		ClientCtx:               clientCtx,
		rpcClient:               rpcClient,
		specQueryClient:         specQueryClient,
		pairingQueryClient:      pairingQueryClient,
		epochStorageQueryClient: epochStorageQueryClient,
		ChainID:                 chainID,
		isUser:                  isUser,
		Acc:                     acc,
		newBlockCb:              newBlockCb,
		ApiInterface:            apiInterface,
		VrfSk:                   vrf_sk,
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
