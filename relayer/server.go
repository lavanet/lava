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

	"golang.org/x/exp/slices"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/relayer/chainsentry"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	tenderbytes "github.com/tendermint/tendermint/libs/bytes"
	grpc "google.golang.org/grpc"
)

var (
	g_privKey         *btcSecp256k1.PrivateKey
	g_sessions        map[string]*UserSessions
	g_sessions_mutex  sync.Mutex
	g_votes           map[uint64]*voteData
	g_votes_mutex     sync.Mutex
	g_sentry          *sentry.Sentry
	g_serverChainID   string
	g_txFactory       tx.Factory
	g_chainProxy      chainproxy.ChainProxy
	g_chainSentry     *chainsentry.ChainSentry
	g_latestBlockHash string
)

type UserSessions struct {
	UsedComputeUnits uint64
	Sessions         map[uint64]*RelaySession
	MaxComputeUnits  uint64
	DataReliability  *pairingtypes.VRFData
	VrfPk            utils.VrfPubKey
}
type RelaySession struct {
	userSessionsParent *UserSessions
	CuSum              uint64
	UniqueIdentifier   uint64
	Lock               sync.Mutex
	Proof              *pairingtypes.RelayRequest // saves last relay request of a session as proof
}

type voteData struct {
	RelayDataHash []byte
	Nonce         int64
	CommitHash    []byte
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
	reliability := false
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
			if sess.Proof == nil {
				continue //this can happen if the data reliability created a session, we dont save a proof on data reliability message
			}
			relay := sess.Proof
			relays = append(relays, relay)
			delete(userSessions.Sessions, k)
			if userSessions.DataReliability != nil {
				relay.DataReliability = userSessions.DataReliability
				userSessions.DataReliability = nil
				reliability = true
			}
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
	if reliability {
		log.Println("asking for rewards", g_sentry.Acc, "with reliability")
	} else {
		log.Println("asking for rewards", g_sentry.Acc)
	}
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

func getOrCreateSession(ctx context.Context, userAddr string, req *pairingtypes.RelayRequest) (*RelaySession, *utils.VrfPubKey, error) {
	g_sessions_mutex.Lock()
	defer g_sessions_mutex.Unlock()

	if _, ok := g_sessions[userAddr]; !ok {
		vrf_pk, maxcuRes, err := g_sentry.GetVrfPkAndMaxCuForUser(ctx, userAddr, req.ChainID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get the Max allowed compute units for the user! %s", err)
		}

		g_sessions[userAddr] = &UserSessions{UsedComputeUnits: 0, MaxComputeUnits: maxcuRes, Sessions: map[uint64]*RelaySession{}, VrfPk: *vrf_pk}
		log.Println("new user sessions " + strconv.FormatUint(maxcuRes, 10))
	}

	userSessions := g_sessions[userAddr]
	if _, ok := userSessions.Sessions[req.SessionId]; !ok {
		userSessions.Sessions[req.SessionId] = &RelaySession{userSessionsParent: g_sessions[userAddr]}
	}

	return userSessions.Sessions[req.SessionId], &userSessions.VrfPk, nil
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

	relaySession, vrf_pk, err := getOrCreateSession(ctx, userAddr.String(), request)
	if err != nil {
		return nil, err
	}

	if request.DataReliability != nil {
		//data reliability message
		if relaySession.userSessionsParent.DataReliability != nil {
			return nil, fmt.Errorf("dataReliability can only be used once per client per epoch")
		}
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
		//will get some rewards for this
		relaySession.userSessionsParent.DataReliability = request.DataReliability
	} else {
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

func SendVoteCommitment(voteID uint64, vote *voteData) {
	msg := conflicttypes.NewMsgConflictVoteCommit(g_sentry.Acc, voteID, vote.CommitHash)
	myWriter := gobytes.Buffer{}
	g_sentry.ClientCtx.Output = &myWriter
	err := tx.GenerateOrBroadcastTxWithFactory(g_sentry.ClientCtx, g_txFactory, msg)
	if err != nil {
		log.Printf("Error: failed to send vote commitment! error: %s\n", err)
	}
}

func SendVoteReveal(voteID uint64, vote *voteData) {
	msg := conflicttypes.NewMsgConflictVoteReveal(g_sentry.Acc, voteID, vote.Nonce, vote.RelayDataHash)
	myWriter := gobytes.Buffer{}
	g_sentry.ClientCtx.Output = &myWriter
	err := tx.GenerateOrBroadcastTxWithFactory(g_sentry.ClientCtx, g_txFactory, msg)
	if err != nil {
		log.Printf("Error: failed to send vote Reveal! error: %s\n", err)
	}
}

func voteEventHandler(ctx context.Context, voteID uint64, voteDeadline uint64, voteParams *sentry.VoteParams) {
	//got a vote event, handle the cases here
	if voteParams != nil {
		//chainID is sent only on new votes
		chainID := voteParams.ChainID
		if chainID != g_serverChainID {
			// not our chain ID
			return
		}
	}
	nodeHeight := uint64(g_sentry.GetBlockHeight())
	if voteDeadline < nodeHeight {
		// its too late to vote
		log.Printf("Error: Vote Event received for deadline %d but current block is %d\n", voteDeadline, nodeHeight)
		return
	}
	g_votes_mutex.Lock()
	defer g_votes_mutex.Unlock()
	vote, ok := g_votes[voteID]
	if ok {
		if voteParams != nil {
			//expected to start a new vote but found an existing one
			log.Printf("Error: new vote Request for vote %+v and voteID: %d had existing entry %v\n", voteParams, voteID, vote)
			return
		}
		SendVoteReveal(voteID, vote)
	} else {
		// new vote
		if voteParams == nil {
			log.Printf("Error: vote reveal Request voteID: %d didn't have a vote entry\n", voteID)
			return
		}
		//try to find this provider in the jury
		found := slices.Contains(voteParams.Voters, g_sentry.Acc)
		if !found {
			// this is a new vote but not for us
			return
		}
		// we need to send a commit, first we need to use the chainProxy and get the response
		//TODO: implement code that verified the requested block is finalized and if its not waits and tries again
		nodeMsg, err := g_chainProxy.ParseMsg(voteParams.ApiURL, voteParams.RequestData)
		if err != nil {
			log.Printf("Error: vote Request for chainID %s did not pass the api check on chain proxy error: %s\n", voteParams.ChainID, err)
			return
		}
		reply, err := nodeMsg.Send(ctx)
		if err != nil {
			log.Printf("Error: vote relay send was failed for: api URL:%s and data: %s, error: %s\n", voteParams.ApiURL, voteParams.RequestData, err)
			return
		}
		nonce := rand.Int63()
		replyDataHash := sigs.HashMsg(reply.Data)
		commitHash := conflicttypes.CommitVoteData(nonce, replyDataHash)

		vote = &voteData{RelayDataHash: replyDataHash, Nonce: nonce, CommitHash: commitHash}
		g_votes[voteID] = vote

		SendVoteCommitment(voteID, vote)
	}
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
	// Start newSentry
	newSentry := sentry.NewSentry(clientCtx, ChainID, false, askForRewards, voteEventHandler, apiInterface, nil, nil)
	err := newSentry.Init(ctx)
	if err != nil {
		log.Fatalln("error sentry.Init", err)
	}
	go newSentry.Start(ctx)
	for newSentry.GetSpecHash() == nil {
		time.Sleep(1 * time.Second)
	}
	g_sentry = newSentry
	g_sessions = map[string]*UserSessions{}
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

	askForRewards()
}
