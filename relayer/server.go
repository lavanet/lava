package relayer

import (
	"bytes"
	gobytes "bytes"
	context "context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/relayer/chainsentry"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	tenderbytes "github.com/tendermint/tendermint/libs/bytes"
	grpc "google.golang.org/grpc"
)

var (
	g_privKey               *btcSecp256k1.PrivateKey
	g_sessions              map[string]*UserSessions
	g_sessions_mutex        sync.Mutex
	g_sentry                *sentry.Sentry
	g_serverChainID         string
	g_txFactory             tx.Factory
	g_chainProxy            chainproxy.ChainProxy
	g_chainSentry           *chainsentry.ChainSentry
	g_rewardsSessions       map[uint64][]*RelaySession
	g_rewardsSessions_mutex sync.Mutex
)

type UserSessionsEpochData struct {
	UsedComputeUnits uint64
	MaxComputeUnits  uint64
	DataReliability  *pairingtypes.VRFData
	VrfPk            utils.VrfPubKey
}

type UserSessions struct {
	Sessions      map[uint64]*RelaySession
	IsBlockListed bool
	user          string
	dataByEpoch   map[uint64]*UserSessionsEpochData
	Lock          sync.Mutex
}
type RelaySession struct {
	userSessionsParent *UserSessions
	CuSum              uint64
	UniqueIdentifier   uint64
	Lock               sync.Mutex
	Proof              *pairingtypes.RelayRequest // saves last relay request of a session as proof
	RelayNum           uint64
	PairingEpoch       uint64
}

type relayServer struct {
	pairingtypes.UnimplementedRelayerServer
}

func askForRewards(staleEpochHeight int64) {
	staleEpochs := []uint64{uint64(staleEpochHeight)}
	if len(g_rewardsSessions) > sentry.StaleEpochDistance {
		// go over all epochs and look for stale unhandled epochs
		for epoch, _ := range g_rewardsSessions {
			if epoch < uint64(staleEpochHeight) {
				staleEpochs = append(staleEpochs, epoch)
			}
		}
	}

	relays := []*pairingtypes.RelayRequest{}
	reliability := false
	sessionsToDelete := make([]*RelaySession, 0)

	for _, staleEpoch := range staleEpochs {
		g_rewardsSessions_mutex.Lock()
		staleEpochSessions, ok := g_rewardsSessions[uint64(staleEpoch)]
		if !ok {
			g_rewardsSessions_mutex.Unlock()
			continue
		}
		g_rewardsSessions_mutex.Unlock()

		for _, session := range staleEpochSessions {
			session.Lock.Lock() // TODO:: is it ok to lock session without g_sessions_mutex?
			if session.Proof == nil {
				//this can happen if the data reliability created a session, we dont save a proof on data reliability message
				session.Lock.Unlock()
				if session.UniqueIdentifier == 0 {
					fmt.Printf("Error: Missing proof, cannot get rewards for this session: %d", session.UniqueIdentifier)
				}
				continue
			}
			relay := session.Proof
			relays = append(relays, relay)
			sessionsToDelete = append(sessionsToDelete, session)

			session.Lock.Unlock()

			session.userSessionsParent.Lock.Lock()
			if userSessionsEpochData, ok := session.userSessionsParent.dataByEpoch[uint64(staleEpoch)]; ok && userSessionsEpochData.DataReliability != nil {
				relay.DataReliability = userSessionsEpochData.DataReliability
				userSessionsEpochData.DataReliability = nil
				reliability = true
			}
			session.userSessionsParent.Lock.Unlock()

			userAccAddr, err := sdk.AccAddressFromBech32(session.userSessionsParent.user)
			if err != nil {
				log.Println(fmt.Sprintf("invalid user address: %s\n", session.userSessionsParent.user))
			}
			g_sentry.AddExpectedPayment(sentry.PaymentRequest{CU: relay.CuSum, BlockHeightDeadline: relay.BlockHeight, Amount: sdk.Coin{}, Client: userAccAddr})
			g_sentry.UpdateCUServiced(relay.CuSum)
		}

		g_rewardsSessions_mutex.Lock()
		delete(g_rewardsSessions, uint64(staleEpoch)) // All rewards handles for that epoch
		g_rewardsSessions_mutex.Unlock()
	}

	g_sessions_mutex.Lock()
	for _, session := range sessionsToDelete {
		session.userSessionsParent.Lock.Lock()
		session.Lock.Lock()
		delete(session.userSessionsParent.Sessions, session.UniqueIdentifier)

		if len(session.userSessionsParent.Sessions) == 0 {
			delete(g_sessions, session.userSessionsParent.user)
		}

		session.Lock.Unlock()
		session.userSessionsParent.Lock.Unlock()
	}
	g_sessions_mutex.Unlock()

	if len(relays) == 0 {
		// no rewards to ask for
		return
	}
	if reliability {
		log.Println("asking for rewards", g_sentry.Acc, "with reliability")
	} else {
		log.Println("asking for rewards", g_sentry.Acc)
	}
	msg := pairingtypes.NewMsgRelayPayment(g_sentry.Acc, relays)
	myWriter := gobytes.Buffer{}
	g_sentry.ClientCtx.Output = &myWriter
	err := tx.GenerateOrBroadcastTxWithFactory(g_sentry.ClientCtx, g_txFactory, msg)
	if err != nil {
		log.Println("GenerateOrBroadcastTxWithFactory", err)
	}

	//EWW, but unmarshalingJson doesn't work because keys aren't in quotes
	transactionResult := strings.ReplaceAll(myWriter.String(), ": ", ":")
	transactionResults := strings.Split(transactionResult, "\n")
	returnCode, err := strconv.ParseUint(strings.Split(transactionResults[0], ":")[1], 10, 32)
	if err != nil {
		fmt.Printf("ERR: %s", err)
	}
	if returnCode != 0 {
		fmt.Printf("----------ERROR-------------\ntransaction results: %s\n-------------ERROR-------------\n", myWriter.String())
	} else {
		fmt.Printf("----------SUCCESS-----------\ntransaction results: %s\n-----------SUCCESS-------------\n", myWriter.String())
	}
}

func getRelayUser(in *pairingtypes.RelayRequest) (tenderbytes.HexBytes, error) {
	pubKey, err := sigs.RecoverPubKeyFromRelay(*in)
	if err != nil {
		return nil, err
	}

	return pubKey.Address(), nil
}

func isAuthorizedUser(ctx context.Context, userAddr string) (*pairingtypes.QueryVerifyPairingResponse, error) {
	return g_sentry.IsAuthorizedUser(ctx, userAddr)
}

func isSupportedSpec(in *pairingtypes.RelayRequest) bool {
	return in.ChainID == g_serverChainID
}

// TODO:: Dont calc. ge tthis info from blockchain
func getEpochFromBlockHeight(blockHeight int64) uint64 {
	return uint64(blockHeight - blockHeight%int64(g_sentry.EpochSize))
}

func getOrCreateSession(ctx context.Context, userAddr string, req *pairingtypes.RelayRequest, isOverlap bool) (*RelaySession, *utils.VrfPubKey, error) {
	g_sessions_mutex.Lock()
	defer g_sessions_mutex.Unlock()
	userReportedEpoch := getEpochFromBlockHeight(req.BlockHeight)

	if _, ok := g_sessions[userAddr]; !ok {
		newUserSessions := &UserSessions{dataByEpoch: map[uint64]*UserSessionsEpochData{}, Sessions: map[uint64]*RelaySession{}, user: userAddr}
		g_sessions[userAddr] = newUserSessions
	}

	userSessions := g_sessions[userAddr]

	userSessions.Lock.Lock()
	defer userSessions.Lock.Unlock()
	if userSessions.IsBlockListed {
		return nil, nil, fmt.Errorf("User blocklisted! userAddr: %s", userAddr)
	}

	if _, ok := userSessions.Sessions[req.SessionId]; !ok {
		userSessions.Sessions[req.SessionId] = &RelaySession{userSessionsParent: g_sessions[userAddr], RelayNum: 0, UniqueIdentifier: req.SessionId}
		sessionEpoch := userReportedEpoch
		if isOverlap {
			sessionEpoch = sessionEpoch - g_sentry.EpochSize
		}
		userSessions.Sessions[req.SessionId].PairingEpoch = sessionEpoch

		g_rewardsSessions_mutex.Lock()
		if _, ok := g_rewardsSessions[sessionEpoch]; !ok {
			g_rewardsSessions[sessionEpoch] = make([]*RelaySession, 0)
		}
		g_rewardsSessions[sessionEpoch] = append(g_rewardsSessions[sessionEpoch], userSessions.Sessions[req.SessionId])
		g_rewardsSessions_mutex.Unlock()

		vrf_pk, maxcuRes, err := g_sentry.GetVrfPkAndMaxCuForUser(ctx, userAddr, req.ChainID, req.BlockHeight)
		if err != nil {
			//ARITODO:: is anything still locked before this return?
			return nil, nil, fmt.Errorf("failed to get the Max allowed compute units for the user! %s", err)
		}
		if _, ok := userSessions.dataByEpoch[sessionEpoch]; !ok {
			userSessions.dataByEpoch[userReportedEpoch] = &UserSessionsEpochData{UsedComputeUnits: 0, MaxComputeUnits: maxcuRes, VrfPk: *vrf_pk}
			log.Println("new user sessions in epoch " + strconv.FormatUint(maxcuRes, 10))
		}
	}

	session := userSessions.Sessions[req.SessionId]
	return session, &userSessions.dataByEpoch[session.PairingEpoch].VrfPk, nil
}

func updateSessionCu(sess *RelaySession, serviceApi *spectypes.ServiceApi, request *pairingtypes.RelayRequest) error {
	sess.userSessionsParent.Lock.Lock()
	defer sess.userSessionsParent.Lock.Unlock()
	sess.Lock.Lock()
	defer sess.Lock.Unlock()

	// Check that relaynum gets incremented by user
	if sess.RelayNum+1 != request.RelayNum {
		sess.userSessionsParent.IsBlockListed = true
		return fmt.Errorf("consumer requested incorrect relaynum. expected: %d, received: %d", sess.RelayNum+1, request.RelayNum)
	}

	sess.RelayNum = sess.RelayNum + 1

	log.Println("updateSessionCu", serviceApi.Name, request.SessionId, serviceApi.ComputeUnits, sess.CuSum, request.CuSum)

	//
	// TODO: do we worry about overflow here?
	if sess.CuSum >= request.CuSum {
		return errors.New("bad cu sum")
	}
	if sess.CuSum+serviceApi.ComputeUnits != request.CuSum {
		return errors.New("bad cu sum")
	}
	if sess.userSessionsParent.dataByEpoch[sess.PairingEpoch].UsedComputeUnits+serviceApi.ComputeUnits > sess.userSessionsParent.dataByEpoch[sess.PairingEpoch].MaxComputeUnits {
		return errors.New("client cu overflow")
	}

	sess.userSessionsParent.dataByEpoch[sess.PairingEpoch].UsedComputeUnits = sess.userSessionsParent.dataByEpoch[sess.PairingEpoch].UsedComputeUnits + serviceApi.ComputeUnits
	sess.CuSum = request.CuSum

	return nil
}

func (s *relayServer) Relay(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
	log.Println("server got Relay")

	blockHeightDiff := g_sentry.GetBlockHeight() - request.BlockHeight

	// client can only be at a previous blockheight that is within epoch+overlap
	if blockHeightDiff > int64(g_sentry.EpochSize+pairingtypes.DefaultEpochBlocksOverlap) { // DefaultEpochBlocksOverlap is the correct overlap value?
		return nil, fmt.Errorf("user reported very old lava block height: %d vs %d", g_sentry.GetBlockHeight(), request.BlockHeight)
	}

	//
	// Checks
	user, err := getRelayUser(request)
	if err != nil {
		// log.Println("Error: %s", err)
		return nil, err
	}
	userAddr, err := sdk.AccAddressFromHex(user.String())
	if err != nil {
		// log.Println("Error: %s", err)
		return nil, err
	}
	//TODO: cache this client, no need to run the query every time
	res, err := isAuthorizedUser(ctx, userAddr.String())
	if err != nil {
		return nil, fmt.Errorf("user not authorized or error occured, err: %s", err)
	}

	if !res.Valid {
		return nil, fmt.Errorf("user not authorized or bad signature, err: %s", err)
	}
	if !isSupportedSpec(request) {
		return nil, errors.New("spec not supported by server")
	}

	//
	// Parse message, check valid api, etc
	nodeMsg, err := g_chainProxy.ParseMsg(request.ApiUrl, request.Data)
	if err != nil {
		return nil, err
	}

	relaySession, vrf_pk, err := getOrCreateSession(ctx, userAddr.String(), request, res.GetOverlap())
	if err != nil {
		return nil, err
	}

	if request.DataReliability != nil {
		relaySession.userSessionsParent.Lock.Lock()
		//data reliability message
		if relaySession.userSessionsParent.dataByEpoch[relaySession.PairingEpoch].DataReliability != nil {
			relaySession.userSessionsParent.Lock.Unlock()
			return nil, fmt.Errorf("dataReliability can only be used once per client per epoch")
		}

		relaySession.userSessionsParent.Lock.Unlock()

		// data reliability is not session dependant, its always sent with sessionID 0 and if not we don't care
		if vrf_pk == nil {
			return nil, fmt.Errorf("dataReliability Triggered with vrf_pk == nil")
		}
		// verify the providerSig is ineed a signature by a valid provider on this query
		valid, err := s.VerifyReliabilityAddressSigning(ctx, userAddr, request)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, fmt.Errorf("invalid DataReliability Provider signing")
		}
		//verify data reliability fields correspond to the right vrf
		valid = utils.VerifyVrfProof(request, *vrf_pk)
		if !valid {
			return nil, fmt.Errorf("invalid DataReliability fields, VRF wasn't verified with provided proof")
		}
		//TODO: verify reliability result corresponds to this provider

		log.Println("server got valid DataReliability request")

		relaySession.userSessionsParent.Lock.Lock()

		//will get some rewards for this
		relaySession.userSessionsParent.dataByEpoch[relaySession.PairingEpoch].DataReliability = request.DataReliability
		relaySession.userSessionsParent.Lock.Unlock()

	} else {
		// Validate
		if request.SessionId == 0 {
			return nil, fmt.Errorf("SessionID cannot be 0 for non-data reliability requests")
		}

		// Update session
		err = updateSessionCu(relaySession, nodeMsg.GetServiceApi(), request)
		if err != nil {
			return nil, err
		}

		relaySession.Proof = request
	}
	// Send
	reply, err := nodeMsg.Send(ctx)
	if err != nil {
		return nil, err
	}

	latestBlock := int64(0)
	finalizedBlockHashes := map[int64]interface{}{}

	if g_sentry.GetSpecComparesHashes() {
		// Add latest block and finalized
		latestBlock, finalizedBlockHashes, err = g_chainSentry.GetLatestBlockData()
		if err != nil {
			return nil, err
		}
	}

	jsonStr, err := json.Marshal(finalizedBlockHashes)
	if err != nil {
		return nil, err
	}

	reply.FinalizedBlocksHashes = []byte(jsonStr)
	reply.LatestBlock = latestBlock

	getSignaturesFromRequest := func(request pairingtypes.RelayRequest) error {
		// request is a copy of the original request, but won't modify it
		// update relay request requestedBlock to the provided one in case it was arbitrary
		sentry.UpdateRequestedBlock(&request, reply)
		// Update signature,
		sig, err := sigs.SignRelayResponse(g_privKey, reply, &request)
		if err != nil {
			return err
		}
		reply.Sig = sig

		if g_sentry.GetSpecComparesHashes() {
			//update sig blocks signature
			sigBlocks, err := sigs.SignResponseFinalizationData(g_privKey, reply, &request, userAddr)
			if err != nil {
				return err
			}
			reply.SigBlocks = sigBlocks
		}
		return nil
	}
	err = getSignaturesFromRequest(*request)
	if err != nil {
		return nil, err
	}

	// return reply to user
	return reply, nil
}

func (relayServ *relayServer) VerifyReliabilityAddressSigning(ctx context.Context, consumer sdk.AccAddress, request *pairingtypes.RelayRequest) (valid bool, err error) {

	queryHash := utils.CalculateQueryHash(*request)
	if !bytes.Equal(queryHash, request.DataReliability.QueryHash) {
		return false, fmt.Errorf("query hash mismatch on data reliability message: %s VS %s", queryHash, request.DataReliability.QueryHash)
	}

	//validate consumer signing on VRF data
	valid, err = sigs.ValidateSignerOnVRFData(consumer, *request.DataReliability)
	if err != nil {
		return
	}

	//validate provider signing on query data
	pubKey, err := sigs.RecoverProviderPubKeyFromVrfDataAndQuery(request)
	if err != nil {
		return false, fmt.Errorf("RecoverProviderPubKeyFromVrfDataAndQuery: %w", err)
	}
	providerAccAddress, err := sdk.AccAddressFromHex(pubKey.Address().String()) //consumer signer
	if err != nil {
		return false, fmt.Errorf("AccAddressFromHex provider: %w", err)
	}
	return g_sentry.IsAuthorizedPairing(ctx, consumer.String(), providerAccAddress.String(), uint64(request.BlockHeight)) //return if this pairing is authorised
}

func Server(
	ctx context.Context,
	clientCtx client.Context,
	txFactory tx.Factory,
	listenAddr string,
	nodeUrl string,
	ChainID string,
	apiInterface string,
) {
	//
	// ctrl+c
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	// Init random seed
	rand.Seed(time.Now().UnixNano())

	//
	// Start newSentry
	newSentry := sentry.NewSentry(clientCtx, ChainID, false, askForRewards, apiInterface, nil, nil)
	err := newSentry.Init(ctx)
	if err != nil {
		log.Fatalln("error sentry.Init", err)
		return
	}
	go newSentry.Start(ctx)
	for newSentry.GetSpecHash() == nil {
		time.Sleep(1 * time.Second)
	}
	g_sentry = newSentry
	g_sessions = map[string]*UserSessions{}
	g_rewardsSessions = map[uint64][]*RelaySession{}
	g_serverChainID = ChainID
	//allow more gas
	g_txFactory = txFactory.WithGas(1000000)

	//
	// Info
	log.Println("Server starting", listenAddr, "node", nodeUrl, "spec", newSentry.GetSpecName(), "chainID", newSentry.GetChainID(), "api Interface", apiInterface)

	//
	// Keys
	keyName, err := sigs.GetKeyName(clientCtx)
	if err != nil {
		log.Fatalln("error: getKeyName", err)
	}

	privKey, err := sigs.GetPrivKey(clientCtx, keyName)
	if err != nil {
		log.Fatalln("error: getPrivKey", err)
	}
	g_privKey = privKey
	serverKey, _ := clientCtx.Keyring.Key(keyName)
	log.Println("Server pubkey", serverKey.GetPubKey().Address())

	//
	// Node
	chainProxy, err := chainproxy.GetChainProxy(nodeUrl, 1, newSentry)
	if err != nil {
		log.Fatalln("error: GetChainProxy", err)
	}
	chainProxy.Start(ctx)
	g_chainProxy = chainProxy

	if g_sentry.GetSpecComparesHashes() {
		// Start chain sentry
		chainSentry := chainsentry.NewChainSentry(clientCtx, chainProxy, ChainID)
		err = chainSentry.Init(ctx)
		if err != nil {
			log.Fatalln("error sentry.Init", err)
		}
		chainSentry.Start(ctx)
		g_chainSentry = chainSentry
	}

	//
	// GRPC
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	go func() {
		select {
		case <-ctx.Done():
			log.Println("server ctx.Done")
		case <-signalChan:
			log.Println("signalChan")
		}

		cancel()
		s.Stop()
	}()

	Server := &relayServer{}
	pairingtypes.RegisterRelayerServer(s, Server)

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	askForRewards(g_sentry.GetCurrentEpochHeight())
}
