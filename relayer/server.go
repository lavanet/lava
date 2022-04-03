package relayer

import (
	gobytes "bytes"
	context "context"
	"errors"
	"fmt"
	"log"
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
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/relayer/sigs"
	servicertypes "github.com/lavanet/lava/x/servicer/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	tenderbytes "github.com/tendermint/tendermint/libs/bytes"
	grpc "google.golang.org/grpc"
)

var (
	g_privKey        *btcSecp256k1.PrivateKey
	g_sessions       map[string]map[uint64]*RelaySession
	g_sessions_mutex sync.Mutex
	g_sentry         *sentry.Sentry
	g_serverSpecId   uint64
	g_txFactory      tx.Factory
	g_chainProxy     chainproxy.ChainProxy
)

type RelaySession struct {
	CuSum uint64
	Lock  sync.Mutex
	Proof *servicertypes.RelayRequest // saves last relay request of a session as proof
}

type relayServer struct {
	servicertypes.UnimplementedRelayerServer
}

func askForRewards() {
	g_sessions_mutex.Lock()
	defer g_sessions_mutex.Unlock()

	if len(g_sessions) > 0 {
		log.Println("active sessions", g_sessions)
	}

	relays := []*servicertypes.RelayRequest{}
	for user, userSessions := range g_sessions {

		if g_sentry.IsAuthorizedUser(context.Background(), user) {
			// session still valid, skip this user
			continue
		}

		//
		// TODO: we can come up with a better locking mechanism
		for k, sess := range userSessions {
			sess.Lock.Lock()
			relay := sess.Proof
			relays = append(relays, relay)
			delete(userSessions, k)
			sess.Lock.Unlock()
			userAccAddr, err := sdk.AccAddressFromBech32(user)
			if err != nil {
				log.Println("invalid user address: %s", user)
			}
			g_sentry.AddExpectedPayment(sentry.PaymentRequest{CU: relay.CuSum, BlockHeightDeadline: relay.BlockHeight, Amount: sdk.Coin{}, Client: userAccAddr})
			g_sentry.UpdateCUServiced(relay.CuSum)
		}
		if len(userSessions) == 0 {
			delete(g_sessions, user)
		}
	}
	if len(relays) == 0 {
		// no rewards to ask for
		return
	}
	log.Println("asking for rewards", g_sentry.Acc)
	msg := servicertypes.NewMsgProofOfWork(g_sentry.Acc, relays)
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

func getRelayUser(in *servicertypes.RelayRequest) (tenderbytes.HexBytes, error) {
	pubKey, err := sigs.RecoverPubKeyFromRelay(in)
	if err != nil {
		return nil, err
	}

	return pubKey.Address(), nil
}

func isAuthorizedUser(ctx context.Context, userAddr string) bool {
	return g_sentry.IsAuthorizedUser(ctx, userAddr)
}

func isSupportedSpec(in *servicertypes.RelayRequest) bool {
	return uint64(in.SpecId) == g_serverSpecId
}

func getOrCreateSession(userAddr string, sessionId uint64) *RelaySession {
	g_sessions_mutex.Lock()
	defer g_sessions_mutex.Unlock()

	if _, ok := g_sessions[userAddr]; !ok {
		g_sessions[userAddr] = map[uint64]*RelaySession{}
	}

	userSessions := g_sessions[userAddr]
	if _, ok := userSessions[sessionId]; !ok {
		userSessions[sessionId] = &RelaySession{}
	}

	return userSessions[sessionId]
}

func updateSessionCu(sess *RelaySession, serviceApi *spectypes.ServiceApi, in *servicertypes.RelayRequest) error {
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

	sess.CuSum = in.CuSum

	// TODO:
	// save relay request here for reward submission at end of session
	return nil
}

func (s *relayServer) Relay(ctx context.Context, in *servicertypes.RelayRequest) (*servicertypes.RelayReply, error) {
	log.Println("server got Relay")

	//
	// Checks
	user, err := getRelayUser(in)
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
	if !isAuthorizedUser(ctx, userAddr.String()) {
		return nil, errors.New("user not authorized or bad signature")
	}
	if !isSupportedSpec(in) {
		return nil, errors.New("spec not supported by server")
	}

	//
	// Parse message, check valid api, etc
	nodeMsg, err := g_chainProxy.ParseMsg(in.ApiUrl, in.Data)
	if err != nil {
		return nil, err
	}

	// Update session
	relaySession := getOrCreateSession(userAddr.String(), in.SessionId)
	updateSessionCu(relaySession, nodeMsg.GetServiceApi(), in)
	relaySession.Proof = in

	// Send
	reply, err := nodeMsg.Send(ctx)
	if err != nil {
		return nil, err
	}

	// Update signature, return reply to user
	sig, err := sigs.SignRelay(g_privKey, []byte(reply.String()))
	if err != nil {
		return nil, err
	}
	reply.Sig = sig
	return reply, nil
}

func Server(
	ctx context.Context,
	clientCtx client.Context,
	txFactory tx.Factory,
	listenAddr string,
	nodeUrl string,
	specId uint64,
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
	sentry := sentry.NewSentry(clientCtx, specId, false, askForRewards)
	err := sentry.Init(ctx)
	if err != nil {
		log.Fatalln("error sentry.Init", err)
	}
	go sentry.Start(ctx)
	for sentry.GetBlockHeight() == 0 {
		time.Sleep(1 * time.Second)
	}
	g_sentry = sentry
	g_sessions = map[string]map[uint64]*RelaySession{}
	g_serverSpecId = specId
	g_txFactory = txFactory

	//
	// Info
	log.Println("Server starting", listenAddr, "node", nodeUrl, "spec", sentry.GetSpecName())

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
	chainProxy, err := chainproxy.GetChainProxy(specId, nodeUrl, 1, sentry)
	if err != nil {
		log.Fatalln("error: GetChainProxy", err)
	}
	chainProxy.Start(ctx)
	g_chainProxy = chainProxy

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
	servicertypes.RegisterRelayerServer(s, Server)

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	askForRewards()
}
