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
	"sync/atomic"
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

const RETRY_INCORRECT_SEQUENCE = 3

var (
	g_privKey               *btcSecp256k1.PrivateKey
	g_sessions              map[string]*UserSessions
	g_sessions_mutex        sync.Mutex
	g_votes                 map[string]*voteData
	g_votes_mutex           sync.Mutex
	g_sentry                *sentry.Sentry
	g_serverChainID         string
	g_txFactory             tx.Factory
	g_chainProxy            chainproxy.ChainProxy
	g_chainSentry           *chainsentry.ChainSentry
	g_rewardsSessions       map[uint64][]*RelaySession // map[epochHeight][]*rewardableSessions
	g_rewardsSessions_mutex sync.Mutex
	g_serverID              uint64
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

func (r *RelaySession) GetPairingEpoch() uint64 {
	return atomic.LoadUint64(&r.PairingEpoch)
}

func (r *RelaySession) SetPairingEpoch(epoch uint64) {
	atomic.StoreUint64(&r.PairingEpoch, epoch)
}

type voteData struct {
	RelayDataHash []byte
	Nonce         int64
	CommitHash    []byte
}

type relayServer struct {
	pairingtypes.UnimplementedRelayerServer
}

func askForRewards(staleEpochHeight int64) {
	staleEpochs := []uint64{uint64(staleEpochHeight)}
	if len(g_rewardsSessions) > sentry.StaleEpochDistance+1 {
		fmt.Printf("Error: Some epochs were not rewarded, catching up and asking for rewards...")

		// go over all epochs and look for stale unhandled epochs
		for epoch := range g_rewardsSessions {
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
		g_rewardsSessions_mutex.Unlock()
		if !ok {
			continue
		}

		for _, session := range staleEpochSessions {
			session.Lock.Lock() // TODO:: is it ok to lock session without g_sessions_mutex?
			if session.Proof == nil {
				//this can happen if the data reliability created a session, we dont save a proof on data reliability message
				session.Lock.Unlock()
				if session.UniqueIdentifier != 0 {
					fmt.Printf("Error: Missing proof, cannot get rewards for this session: %d\n", session.UniqueIdentifier)
				}
				continue
			}
			relay := session.Proof
			session.Proof = &pairingtypes.RelayRequest{} // Just in case askForRewards is running more than once at the same time and it might ask to be rewarded for this relay twice
			relays = append(relays, relay)
			sessionsToDelete = append(sessionsToDelete, session)

			userSessions := session.userSessionsParent
			session.Lock.Unlock()
			userSessions.Lock.Lock()
			userAccAddr, err := sdk.AccAddressFromBech32(userSessions.user)
			if err != nil {
				log.Println(fmt.Sprintf("invalid user address: %s\n We can continue without the addr but the it shouldnt be invalid.", userSessions.user))
			}

			userSessionsEpochData, ok := userSessions.dataByEpoch[uint64(staleEpoch)]
			if !ok {
				log.Printf("Error: Missing epoch data for this user: %s, Epoch: %d\n", userSessions.user, staleEpoch)
				userSessions.Lock.Unlock()
				continue
			}

			if userSessionsEpochData.DataReliability != nil {
				relay.DataReliability = userSessionsEpochData.DataReliability
				userSessionsEpochData.DataReliability = nil
				reliability = true
			}
			userSessions.Lock.Unlock()

			g_sentry.AddExpectedPayment(sentry.PaymentRequest{CU: relay.CuSum, BlockHeightDeadline: relay.BlockHeight, Amount: sdk.Coin{}, Client: userAccAddr, UniqueIdentifier: relay.SessionId})
			g_sentry.UpdateCUServiced(relay.CuSum)
		}

		g_rewardsSessions_mutex.Lock()
		delete(g_rewardsSessions, uint64(staleEpoch)) // All rewards handles for that epoch
		g_rewardsSessions_mutex.Unlock()
	}

	userSessionObjsToDelete := make([]string, 0)
	for _, session := range sessionsToDelete {
		session.Lock.Lock()
		userSessions := session.userSessionsParent
		sessionID := session.UniqueIdentifier
		session.Lock.Unlock()
		userSessions.Lock.Lock()
		delete(userSessions.Sessions, sessionID)
		if len(userSessions.Sessions) == 0 {
			userSessionObjsToDelete = append(userSessionObjsToDelete, userSessions.user)
		}
		userSessions.Lock.Unlock()
	}

	g_sessions_mutex.Lock()
	for _, user := range userSessionObjsToDelete {
		delete(g_sessions, user)
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
	msg := pairingtypes.NewMsgRelayPayment(g_sentry.Acc, relays, strconv.FormatUint(g_serverID, 10))
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
	if returnCode != 0 {
		// TODO:: get rid of this code
		// This code retries asking for rewards when getting an 'incorrect sequence' error.
		// The transaction should work after a new lava block.
		fmt.Printf("----------ERROR-------------\ntransaction results: %s\n-------------ERROR-------------\n", myWriter.String())
		if strings.Contains(transactionResult, "incorrect account sequence") {
			fmt.Printf("incorrect account sequence detected. retrying transaction... \n")
			idx := 1
			success := false
			for idx < RETRY_INCORRECT_SEQUENCE && !success {
				time.Sleep(1 * time.Second)
				myWriter.Reset()
				err := tx.GenerateOrBroadcastTxWithFactory(g_sentry.ClientCtx, g_txFactory, msg)
				if err != nil {
					log.Println("GenerateOrBroadcastTxWithFactory", err)
					break
				}
				idx++

				transactionResult = myWriter.String()
				transactionResult := strings.ReplaceAll(transactionResult, ": ", ":")
				transactionResults := strings.Split(transactionResult, "\n")
				returnCode, err := strconv.ParseUint(strings.Split(transactionResults[0], ":")[1], 10, 32)
				if err != nil {
					fmt.Printf("ERR: %s", err)
					returnCode = 1 // just not zero
				}

				if returnCode == 0 { // if we get some other error which isnt then keep retrying
					success = true
				} else {
					if !strings.Contains(transactionResult, "incorrect account sequence") {
						fmt.Printf("Expected an incorrect account sequence error when retrying rewards transaction but received another error: %s\n", transactionResult)
						fmt.Printf("Retrying anyway\n")
					}
				}
			}

			if !success {
				fmt.Printf("----------ERROR-------------\ntransaction results: %s\n-------------ERROR-------------\n", myWriter.String())
				fmt.Printf("incorrect account sequence detected but and no success after %d retries \n", RETRY_INCORRECT_SEQUENCE)
				return
			} else {
				fmt.Printf("success in %d retries after incorrect account sequence detected \n", idx)
			}
		} else {
			return
		}
	}
	fmt.Printf("----------SUCCESS-----------\ntransaction results: %s\n-----------SUCCESS-------------\n", myWriter.String())
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

func getOrCreateSession(ctx context.Context, userAddr string, req *pairingtypes.RelayRequest, isOverlap bool) (*RelaySession, *utils.VrfPubKey, error) {
	g_sessions_mutex.Lock()
	userSessions, ok := g_sessions[userAddr]
	if !ok {
		userSessions = &UserSessions{dataByEpoch: map[uint64]*UserSessionsEpochData{}, Sessions: map[uint64]*RelaySession{}, user: userAddr}
		g_sessions[userAddr] = userSessions
	}
	g_sessions_mutex.Unlock()

	userSessions.Lock.Lock()
	if userSessions.IsBlockListed {
		userSessions.Lock.Unlock()
		return nil, nil, fmt.Errorf("User blocklisted! userAddr: %s", userAddr)
	}

	var sessionEpoch uint64
	var vrf_pk *utils.VrfPubKey
	session, ok := userSessions.Sessions[req.SessionId]
	userSessions.Lock.Unlock()

	if ok {
		sessionEpoch = session.GetPairingEpoch()

		userSessions.Lock.Lock()
		vrf_pk = &userSessions.dataByEpoch[sessionEpoch].VrfPk
		userSessions.Lock.Unlock()
	} else {
		tmp_vrf_pk, maxcuRes, err := g_sentry.GetVrfPkAndMaxCuForUser(ctx, userAddr, req.ChainID, req.BlockHeight)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get the Max allowed compute units for the user! %s", err)
		}
		vrf_pk = tmp_vrf_pk

		sessionEpoch = g_sentry.GetEpochFromBlockHeight(req.BlockHeight, isOverlap)

		userSessions.Lock.Lock()
		session = &RelaySession{userSessionsParent: userSessions, RelayNum: 0, UniqueIdentifier: req.SessionId, PairingEpoch: sessionEpoch}
		userSessions.Sessions[req.SessionId] = session
		if _, ok := userSessions.dataByEpoch[sessionEpoch]; !ok {
			userSessions.dataByEpoch[sessionEpoch] = &UserSessionsEpochData{UsedComputeUnits: 0, MaxComputeUnits: maxcuRes, VrfPk: *vrf_pk}
			log.Println("new user sessions in epoch " + strconv.FormatUint(maxcuRes, 10))
		}
		userSessions.Lock.Unlock()

		g_rewardsSessions_mutex.Lock()
		if _, ok := g_rewardsSessions[sessionEpoch]; !ok {
			g_rewardsSessions[sessionEpoch] = make([]*RelaySession, 0)
		}
		g_rewardsSessions[sessionEpoch] = append(g_rewardsSessions[sessionEpoch], session)
		g_rewardsSessions_mutex.Unlock()
	}

	return session, vrf_pk, nil
}

func updateSessionCu(sess *RelaySession, userSessions *UserSessions, serviceApi *spectypes.ServiceApi, request *pairingtypes.RelayRequest, pairingEpoch uint64) error {
	sess.Lock.Lock()
	relayNum := sess.RelayNum
	cuSum := sess.CuSum
	sess.Lock.Unlock()
	// Check that relaynum gets incremented by user
	if relayNum+1 != request.RelayNum {
		userSessions.Lock.Lock()
		userSessions.IsBlockListed = true
		userSessions.Lock.Unlock()
		return fmt.Errorf("consumer requested incorrect relaynum. expected: %d, received: %d", relayNum+1, request.RelayNum)
	}

	sess.Lock.Lock()
	sess.RelayNum = sess.RelayNum + 1
	sess.Lock.Unlock()

	log.Println("updateSessionCu", serviceApi.Name, request.SessionId, serviceApi.ComputeUnits, cuSum, request.CuSum)

	//
	// TODO: do we worry about overflow here?
	if cuSum >= request.CuSum {
		return errors.New("bad cu sum")
	}
	if cuSum+serviceApi.ComputeUnits != request.CuSum {
		return errors.New("bad cu sum")
	}

	userSessions.Lock.Lock()
	epochData := userSessions.dataByEpoch[pairingEpoch]

	if epochData.UsedComputeUnits+serviceApi.ComputeUnits > epochData.MaxComputeUnits {
		userSessions.Lock.Unlock()
		return errors.New("client cu overflow")
	}

	epochData.UsedComputeUnits = epochData.UsedComputeUnits + serviceApi.ComputeUnits
	userSessions.Lock.Unlock()

	sess.Lock.Lock()
	sess.CuSum = request.CuSum
	sess.Lock.Unlock()

	return nil
}

func (s *relayServer) Relay(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
	log.Println("server got Relay")

	prevEpochStart := g_sentry.GetCurrentEpochHeight() - int64(g_sentry.EpochSize)

	if prevEpochStart < 0 {
		prevEpochStart = 0
	}

	// client blockheight can only be at at prev epoch but not ealier
	if request.BlockHeight < int64(prevEpochStart) {
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

	if !isSupportedSpec(request) {
		return nil, errors.New("spec not supported by server")
	}

	//
	// Parse message, check valid api, etc
	nodeMsg, err := g_chainProxy.ParseMsg(request.ApiUrl, request.Data, "")
	if err != nil {
		return nil, err
	}

	relaySession, vrf_pk, err := getOrCreateSession(ctx, userAddr.String(), request, res.GetOverlap())
	if err != nil {
		return nil, err
	}

	relaySession.Lock.Lock()
	pairingEpoch := relaySession.GetPairingEpoch()
	userSessions := relaySession.userSessionsParent
	relaySession.Lock.Unlock()

	if request.DataReliability != nil {
		userSessions.Lock.Lock()
		dataReliability := userSessions.dataByEpoch[pairingEpoch].DataReliability
		userSessions.Lock.Unlock()

		//data reliability message
		if dataReliability != nil {
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
		relayEpochStart := g_sentry.GetEpochFromBlockHeight(request.BlockHeight, false)
		valid = utils.VerifyVrfProof(request, *vrf_pk, relayEpochStart)
		if !valid {
			return nil, fmt.Errorf("invalid DataReliability fields, VRF wasn't verified with provided proof")
		}
		//TODO: verify reliability result corresponds to this provider

		log.Println("server got valid DataReliability request")

		userSessions.Lock.Lock()

		//will get some rewards for this
		userSessions.dataByEpoch[pairingEpoch].DataReliability = request.DataReliability
		userSessions.Lock.Unlock()

	} else {
		// Validate
		if request.SessionId == 0 {
			return nil, fmt.Errorf("SessionID cannot be 0 for non-data reliability requests")
		}

		// Update session
		err = updateSessionCu(relaySession, userSessions, nodeMsg.GetServiceApi(), request, pairingEpoch)
		if err != nil {
			return nil, err
		}

		relaySession.Lock.Lock()
		relaySession.Proof = request
		relaySession.Lock.Unlock()
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

func SendVoteCommitment(voteID string, vote *voteData) {
	msg := conflicttypes.NewMsgConflictVoteCommit(g_sentry.Acc, voteID, vote.CommitHash)
	myWriter := gobytes.Buffer{}
	g_sentry.ClientCtx.Output = &myWriter
	err := tx.GenerateOrBroadcastTxWithFactory(g_sentry.ClientCtx, g_txFactory, msg)
	if err != nil {
		log.Printf("Error: failed to send vote commitment! error: %s\n", err)
	}
}

func SendVoteReveal(voteID string, vote *voteData) {
	msg := conflicttypes.NewMsgConflictVoteReveal(g_sentry.Acc, voteID, vote.Nonce, vote.RelayDataHash)
	myWriter := gobytes.Buffer{}
	g_sentry.ClientCtx.Output = &myWriter
	err := tx.GenerateOrBroadcastTxWithFactory(g_sentry.ClientCtx, g_txFactory, msg)
	if err != nil {
		log.Printf("Error: failed to send vote Reveal! error: %s\n", err)
	}
}

func voteEventHandler(ctx context.Context, voteID string, voteDeadline uint64, voteParams *sentry.VoteParams) {
	//got a vote event, handle the cases here

	if !voteParams.GetCloseVote() {
		//meaning we dont close a vote, so we should check stuff
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
	}
	g_votes_mutex.Lock()
	defer g_votes_mutex.Unlock()
	vote, ok := g_votes[voteID]
	if ok {
		//we have an existing vote with this ID
		if voteParams != nil {
			if voteParams.GetCloseVote() {
				//we are closing the vote, so its okay we ahve this voteID
				log.Printf("[+] Received Vote termination event for voteID: %s, Cleared entry\n", voteID)
				delete(g_votes, voteID)
				return
			}
			//expected to start a new vote but found an existing one
			log.Printf("Error: new vote Request for vote %+v and voteID: %s had existing entry %v\n", voteParams, voteID, vote)
			return
		}
		log.Printf("[+] Received Vote Reveal for voteID: %s, sending Reveal for result: %v \n", voteID, vote)
		SendVoteReveal(voteID, vote)
		return
	} else {
		// new vote
		if voteParams == nil {
			log.Printf("Error: vote reveal Request voteID: %s didn't have a vote entry\n", voteID)
			return
		}
		if voteParams.GetCloseVote() {
			log.Printf("Error: vote closing received for voteID: %s but didn't have a vote entry\n", voteID)
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
		nodeMsg, err := g_chainProxy.ParseMsg(voteParams.ApiURL, voteParams.RequestData, "")
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
		log.Printf("[+] Received Vote start for voteID: %s, sending commit for result: %v \n", voteID, vote)
		SendVoteCommitment(voteID, vote)
		return
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

	// Init random seed
	rand.Seed(time.Now().UnixNano())
	g_serverID = uint64(rand.Int63())

	//
	// Start newSentry
	newSentry := sentry.NewSentry(clientCtx, ChainID, false, voteEventHandler, askForRewards, apiInterface, nil, nil, g_serverID)
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
	g_votes = map[string]*voteData{}
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
