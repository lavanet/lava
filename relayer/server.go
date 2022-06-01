package relayer

import (
	gobytes "bytes"
	context "context"
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
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	tenderbytes "github.com/tendermint/tendermint/libs/bytes"
	grpc "google.golang.org/grpc"
)

var (
	g_privKey         *btcSecp256k1.PrivateKey
	g_sessions        map[string]*UserSessions
	g_sessions_mutex  sync.Mutex
	g_sentry          *sentry.Sentry
	g_serverChainID   string
	g_txFactory       tx.Factory
	g_chainProxy      chainproxy.ChainProxy
	g_chainSentry     *chainsentry.ChainSentry
	g_latestBlockHash string
)

type UserSessions struct {
	UsedComputeUnits uint64
	MaxComputeUnits  uint64
	Sessions         map[uint64]*RelaySession
}
type RelaySession struct {
	userSessionsParent *UserSessions
	CuSum              uint64
	UniqueIdentifier   uint64
	Lock               sync.Mutex
	Proof              *pairingtypes.RelayRequest // saves last relay request of a session as proof
}

type relayServer struct {
	pairingtypes.UnimplementedRelayerServer
}

func askForRewards() {
	g_sessions_mutex.Lock()
	defer g_sessions_mutex.Unlock()

	if len(g_sessions) > 0 {
		log.Printf("active sessions: ")
		for _, userSessions := range g_sessions {
			log.Printf("%+v ", *userSessions)
		}
		log.Printf("\n")
	}

	relays := []*pairingtypes.RelayRequest{}
	for user, userSessions := range g_sessions {
		validuser, _ := g_sentry.IsAuthorizedUser(context.Background(), user)
		if validuser {
			// session still valid, skip this user
			continue
		}

		//
		// TODO: we can come up with a better locking mechanism
		for k, sess := range userSessions.Sessions {
			sess.Lock.Lock()
			relay := sess.Proof
			relays = append(relays, relay)
			delete(userSessions.Sessions, k)
			sess.Lock.Unlock()
			userAccAddr, err := sdk.AccAddressFromBech32(user)
			if err != nil {
				log.Println(fmt.Sprintf("invalid user address: %s\n", user))
			}
			g_sentry.AddExpectedPayment(sentry.PaymentRequest{CU: relay.CuSum, BlockHeightDeadline: relay.BlockHeight, Amount: sdk.Coin{}, Client: userAccAddr})
			g_sentry.UpdateCUServiced(relay.CuSum)
		}
		if len(userSessions.Sessions) == 0 {
			delete(g_sessions, user)
		}
	}
	if len(relays) == 0 {
		// no rewards to ask for
		return
	}
	log.Println("asking for rewards", g_sentry.Acc)
	msg := pairingtypes.NewMsgRelayPayment(g_sentry.Acc, relays)
	myWriter := gobytes.Buffer{}
	g_sentry.ClientCtx.Output = &myWriter
	err := tx.GenerateOrBroadcastTxWithFactory(g_sentry.ClientCtx, g_txFactory, msg)

	// #o set doubleSendTest to true to test sending payment twice
	doubleSendTest := false
	if doubleSendTest { // wait between 0.5-1.5 seconds and resend tx for testing purposes
		n := rand.Float32() // n will be between 0 and 10
		fmt.Printf("Sleeping %d seconds...\n", n)
		time.Sleep(time.Duration(n*1+0.5) * time.Second)
		fmt.Println("Done")
		err = tx.GenerateOrBroadcastTxWithFactory(g_sentry.ClientCtx, g_txFactory, msg)
	}
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

func isAuthorizedUser(ctx context.Context, userAddr string) (bool, error) {
	return g_sentry.IsAuthorizedUser(ctx, userAddr)
}

func isSupportedSpec(in *pairingtypes.RelayRequest) bool {
	return in.ChainID == g_serverChainID
}

func getOrCreateSession(ctx context.Context, userAddr string, req *pairingtypes.RelayRequest) (*RelaySession, error) {
	g_sessions_mutex.Lock()
	defer g_sessions_mutex.Unlock()

	if _, ok := g_sessions[userAddr]; !ok {
		maxcuRes, err := g_sentry.GetMaxCUForUser(ctx, userAddr, req.ChainID)
		if err != nil {
			return nil, errors.New("failed to get the Max allowed compute units for the user")
		}

		g_sessions[userAddr] = &UserSessions{UsedComputeUnits: 0, MaxComputeUnits: maxcuRes, Sessions: map[uint64]*RelaySession{}}
		log.Println("new user sessions " + strconv.FormatUint(maxcuRes, 10))
	}

	userSessions := g_sessions[userAddr]
	if _, ok := userSessions.Sessions[req.SessionId]; !ok {
		userSessions.Sessions[req.SessionId] = &RelaySession{userSessionsParent: g_sessions[userAddr]}
	}

	return userSessions.Sessions[req.SessionId], nil
}

func updateSessionCu(sess *RelaySession, serviceApi *spectypes.ServiceApi, in *pairingtypes.RelayRequest) error {
	sess.Lock.Lock()
	defer sess.Lock.Unlock()

	log.Println("updateSessionCu", serviceApi.Name, in.SessionId, serviceApi.ComputeUnits, sess.CuSum, in.CuSum)

	//
	// TODO: do we worry about overflow here?
	if sess.CuSum >= in.CuSum {
		return errors.New("bad cu sum")
	}
	if sess.CuSum+serviceApi.ComputeUnits != in.CuSum {
		return errors.New("bad cu sum")
	}
	if sess.userSessionsParent.UsedComputeUnits+serviceApi.ComputeUnits > sess.userSessionsParent.MaxComputeUnits {
		return errors.New("client cu overflow")
	}

	sess.userSessionsParent.UsedComputeUnits = sess.userSessionsParent.UsedComputeUnits + serviceApi.ComputeUnits
	sess.CuSum = in.CuSum

	// TODO:
	// save relay request here for reward submission at end of session
	return nil
}

func (s *relayServer) Relay(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
	log.Println("server got Relay")

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
	validUser, err := isAuthorizedUser(ctx, userAddr.String())
	if !validUser {
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

	// Update session
	relaySession, err := getOrCreateSession(ctx, userAddr.String(), request)
	if err != nil {
		return nil, err
	}

	if request.DataReliability != nil {
		return nil, fmt.Errorf("not implemented data reliability handling")
	}

	err = updateSessionCu(relaySession, nodeMsg.GetServiceApi(), request)
	if err != nil {
		return nil, err
	}

	relaySession.Proof = request

	// Send
	reply, err := nodeMsg.Send(ctx)
	if err != nil {
		return nil, err
	}

	//reply.LatestBlockNum = g_chainSentry.GetLatestBlockNum()
	//reply.LatestBlockHashes = g_chainSentry.GetLatestBlockNum()

	//update relay request requestedBlock to the provided one in case it was arbitrary
	sentry.UpdateRequestedBlock(request, reply)

	// Update signature, return reply to user
	sig, err := sigs.SignRelayResponse(g_privKey, reply, request)
	if err != nil {
		return nil, err
	}
	reply.Sig = sig

	sigBlocks, err := sigs.SignResponseFinalizationData(g_privKey, reply, request)
	if err != nil {
		return nil, err
	}
	reply.SigBlocks = sigBlocks

	return reply, nil
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

	//
	// Start sentry
	g_sentry := sentry.NewSentry(clientCtx, ChainID, false, askForRewards, apiInterface, nil)
	err := g_sentry.Init(ctx)
	if err != nil {
		log.Fatalln("error sentry.Init", err)
	}
	go g_sentry.Start(ctx)
	for g_sentry.GetBlockHeight() == 0 {
		time.Sleep(1 * time.Second)
	}
	g_sessions = map[string]*UserSessions{}
	g_serverChainID = ChainID
	//allow more gas
	g_txFactory = txFactory.WithGas(1000000)

	//
	// Info
	log.Println("Server starting", listenAddr, "node", nodeUrl, "spec", g_sentry.GetSpecName(), "chainID", g_sentry.GetChainID(), "api Interface", apiInterface)

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
	chainProxy, err := chainproxy.GetChainProxy(nodeUrl, 1, g_sentry)
	if err != nil {
		log.Fatalln("error: GetChainProxy", err)
	}
	chainProxy.Start(ctx)
	g_chainProxy = chainProxy

	// Start chain sentry
	chainSentry := chainsentry.NewChainSentry(clientCtx, chainProxy, ChainID)
	// err := chainSentry.Init(ctx)
	// if err != nil {
	// log.Fatalln("error chainSentry.Init", err)
	// }
	go chainSentry.Start(ctx)
	// for chainSentry.GetBlockHeight() == 0 {
	// 	time.Sleep(1 * time.Second)
	// }
	g_chainSentry = chainSentry

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

	askForRewards()
}
