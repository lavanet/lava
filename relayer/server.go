package relayer

import (
	"bytes"
	gobytes "bytes"
	context "context"
	"encoding/json"
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
		utils.LavaFormatError("Some epochs were not rewarded, catching up and asking for rewards...", nil, &map[string]string{
			"epoch":                strconv.FormatInt(staleEpochHeight, 10),
			"provider block":       strconv.FormatInt(g_sentry.GetBlockHeight(), 10),
			"rewards to claim len": strconv.FormatInt(int64(len(g_rewardsSessions)), 10),
		})

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
					utils.LavaFormatError("Missing proof, cannot get rewards for this session", nil, &map[string]string{
						"UniqueIdentifier": strconv.FormatUint(session.UniqueIdentifier, 10),
					})
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
				utils.LavaFormatError("get rewards invalid user address", err, &map[string]string{
					"address": userSessions.user,
				})
			}

			userSessionsEpochData, ok := userSessions.dataByEpoch[uint64(staleEpoch)]
			if !ok {
				utils.LavaFormatError("get rewards Missing epoch data for this user", err, &map[string]string{
					"address": userSessions.user,
					"epoch":   strconv.FormatUint(staleEpoch, 10),
				})
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

	utils.LavaFormatInfo("asking for rewards", nil, &map[string]string{
		"account":     g_sentry.Acc,
		"reliability": fmt.Sprintf("%t", reliability),
	})

	msg := pairingtypes.NewMsgRelayPayment(g_sentry.Acc, relays, strconv.FormatUint(g_serverID, 10))
	myWriter := gobytes.Buffer{}
	g_sentry.ClientCtx.Output = &myWriter
	err := tx.GenerateOrBroadcastTxWithFactory(g_sentry.ClientCtx, g_txFactory, msg)
	if err != nil {
		utils.LavaFormatError("Sending GenerateOrBroadcastTxWithFactory failed", err, &map[string]string{
			"msg": fmt.Sprintf("%+v", msg),
		})
	}

	//EWW, but unmarshalingJson doesn't work because keys aren't in quotes
	transactionResult := strings.ReplaceAll(myWriter.String(), ": ", ":")
	transactionResults := strings.Split(transactionResult, "\n")
	returnCode, err := strconv.ParseUint(strings.Split(transactionResults[0], ":")[1], 10, 32)
	if err != nil {
		utils.LavaFormatError("Failed to parse transaction result", err, &map[string]string{
			"parsing data": strings.Join(transactionResults, ","),
		})
		return
	}
	if returnCode != 0 {
		// TODO:: get rid of this code
		// This code retries asking for rewards when getting an 'incorrect sequence' error.
		// The transaction should work after a new lava block.

		utils.LavaFormatError(fmt.Sprintf("----------ERROR-------------\ntransaction results: %s\n-------------ERROR-------------\n", myWriter.String()),
			nil, &map[string]string{
				"returnCode": strconv.FormatUint(returnCode, 10),
			})
		if strings.Contains(transactionResult, "incorrect account sequence") {
			utils.LavaFormatError("incorrect account sequence detected. retrying transaction...", nil, nil)
			idx := 1
			success := false
			for idx < RETRY_INCORRECT_SEQUENCE && !success {
				time.Sleep(1 * time.Second)
				myWriter.Reset()
				err := tx.GenerateOrBroadcastTxWithFactory(g_sentry.ClientCtx, g_txFactory, msg)
				if err != nil {
					utils.LavaFormatError("Sending GenerateOrBroadcastTxWithFactory failed", err, &map[string]string{
						"msg": fmt.Sprintf("%+v", msg),
					})
					break
				}
				idx++

				transactionResult = myWriter.String()
				transactionResult := strings.ReplaceAll(transactionResult, ": ", ":")
				transactionResults := strings.Split(transactionResult, "\n")
				returnCode, err := strconv.ParseUint(strings.Split(transactionResults[0], ":")[1], 10, 32)
				if err != nil {
					utils.LavaFormatError("Failed to parse transaction result", err, &map[string]string{
						"parsing data": strings.Join(transactionResults, ","),
					})
					returnCode = 1 // just not zero
				}

				if returnCode == 0 { // if we get some other error which isnt then keep retrying
					success = true
				} else {
					if !strings.Contains(transactionResult, "incorrect account sequence") {
						utils.LavaFormatError("Unexpected Failure during retry", nil, &map[string]string{
							"response": transactionResult,
						})
					}
				}
			}

			if !success {
				utils.LavaFormatError(fmt.Sprintf("----------ERROR-------------\ntransaction results: %s\n-------------ERROR-------------\nincorrect account sequence and no success after retries", myWriter.String()),
					nil, &map[string]string{
						"retries": strconv.FormatUint(RETRY_INCORRECT_SEQUENCE, 10),
					})
				return
			} else {
				utils.LavaFormatInfo("Success after incorrect account sequence detected", nil, &map[string]string{
					"retries": strconv.FormatInt(int64(idx), 10),
				})
			}
		} else {
			return
		}
	}
	utils.LavaFormatInfo(fmt.Sprintf("----------SUCCESS-----------\ntransaction results: %s\n-----------SUCCESS-------------\n", myWriter.String()), nil, nil)
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
		return nil, nil, utils.LavaFormatError("User blocklisted!", nil, &map[string]string{
			"userAddr": userAddr,
		})
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
			return nil, nil, utils.LavaFormatError("failed to get the Max allowed compute units for the user!", err, &map[string]string{
				"userAddr": userAddr,
			})
		}
		vrf_pk = tmp_vrf_pk

		sessionEpoch = g_sentry.GetEpochFromBlockHeight(req.BlockHeight, isOverlap)

		userSessions.Lock.Lock()
		session = &RelaySession{userSessionsParent: userSessions, RelayNum: 0, UniqueIdentifier: req.SessionId, PairingEpoch: sessionEpoch}
		userSessions.Sessions[req.SessionId] = session
		if _, ok := userSessions.dataByEpoch[sessionEpoch]; !ok {
			userSessions.dataByEpoch[sessionEpoch] = &UserSessionsEpochData{UsedComputeUnits: 0, MaxComputeUnits: maxcuRes, VrfPk: *vrf_pk}
			utils.LavaFormatInfo("new user sessions in epoch", err, &map[string]string{
				"userAddr": userAddr,
				"maxcuRes": strconv.FormatUint(maxcuRes, 10),
				"epoch":    strconv.FormatUint(sessionEpoch, 10),
			})
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
		return utils.LavaFormatError("consumer requested incorrect relaynum.", nil, &map[string]string{
			"expected": strconv.FormatUint(relayNum+1, 10),
			"received": strconv.FormatUint(request.RelayNum, 10),
		})
	}

	sess.Lock.Lock()
	sess.RelayNum = sess.RelayNum + 1
	sess.Lock.Unlock()

	utils.LavaFormatInfo("updateSessionCu", nil, &map[string]string{
		"serviceApi.Name":   serviceApi.Name,
		"request.SessionId": strconv.FormatUint(request.SessionId, 10),
	})
	//
	// TODO: do we worry about overflow here?
	if cuSum >= request.CuSum {
		return utils.LavaFormatError("bad CU sum", nil, &map[string]string{
			"cuSum":         strconv.FormatUint(cuSum, 10),
			"request.CuSum": strconv.FormatUint(request.CuSum, 10),
		})
	}
	if cuSum+serviceApi.ComputeUnits != request.CuSum {
		return utils.LavaFormatError("bad CU sum", nil, &map[string]string{
			"cuSum":                   strconv.FormatUint(cuSum, 10),
			"request.CuSum":           strconv.FormatUint(request.CuSum, 10),
			"serviceApi.ComputeUnits": strconv.FormatUint(serviceApi.ComputeUnits, 10),
		})
	}

	userSessions.Lock.Lock()
	epochData := userSessions.dataByEpoch[pairingEpoch]

	if epochData.UsedComputeUnits+serviceApi.ComputeUnits > epochData.MaxComputeUnits {
		userSessions.Lock.Unlock()
		return utils.LavaFormatError("client cu overflow", nil, &map[string]string{
			"epochData.MaxComputeUnits":  strconv.FormatUint(epochData.MaxComputeUnits, 10),
			"epochData.UsedComputeUnits": strconv.FormatUint(epochData.UsedComputeUnits, 10),
			"serviceApi.ComputeUnits":    strconv.FormatUint(request.CuSum, 10),
		})
	}

	epochData.UsedComputeUnits = epochData.UsedComputeUnits + serviceApi.ComputeUnits
	userSessions.Lock.Unlock()

	sess.Lock.Lock()
	sess.CuSum = request.CuSum
	sess.Lock.Unlock()

	return nil
}

func (s *relayServer) Relay(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
	utils.LavaFormatInfo("Provider got relay request", nil, &map[string]string{
		"request.SessionId": strconv.FormatUint(request.SessionId, 10),
	})

	prevEpochStart := g_sentry.GetCurrentEpochHeight() - int64(g_sentry.EpochSize)

	if prevEpochStart < 0 {
		prevEpochStart = 0
	}

	// client blockheight can only be at at prev epoch but not ealier
	if request.BlockHeight < int64(prevEpochStart) {
		return nil, utils.LavaFormatError("user reported very old lava block height", nil, &map[string]string{
			"current lava block":   strconv.FormatInt(g_sentry.GetBlockHeight(), 10),
			"requested lava block": strconv.FormatInt(request.BlockHeight, 10),
		})
	}

	//
	// Checks
	user, err := getRelayUser(request)
	if err != nil {
		return nil, utils.LavaFormatError("get relay user", err, &map[string]string{})
	}
	userAddr, err := sdk.AccAddressFromHex(user.String())
	if err != nil {
		return nil, utils.LavaFormatError("get relay acc address", err, &map[string]string{})
	}
	//TODO: cache this client, no need to run the query every time
	authorisedUserResponse, err := isAuthorizedUser(ctx, userAddr.String())
	if err != nil {
		return nil, utils.LavaFormatError("user not authorized or error occured", err, &map[string]string{"userAddr": userAddr.String()})
	}

	if !isSupportedSpec(request) {
		return nil, utils.LavaFormatError("spec not supported by server", err, &map[string]string{"request.chainID": request.ChainID, "chainID": g_serverChainID})
	}

	//
	// Parse message, check valid api, etc
	nodeMsg, err := g_chainProxy.ParseMsg(request.ApiUrl, request.Data, request.ConnectionType)
	if err != nil {
		return nil, utils.LavaFormatError("failed parsing request message", err, &map[string]string{"apiInterface": g_sentry.ApiInterface, "request URL": request.ApiUrl, "request data": string(request.Data), "userAddr": userAddr.String()})
	}

	relaySession, vrf_pk, err := getOrCreateSession(ctx, userAddr.String(), request, authorisedUserResponse.GetOverlap())
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
			return nil, utils.LavaFormatError("dataReliability can only be used once per client per epoch", nil,
				&map[string]string{"pairingEpoch": strconv.FormatUint(pairingEpoch, 10), "userAddr": userAddr.String(), "dataReliability": fmt.Sprintf("%v", dataReliability)})
		}

		// data reliability is not session dependant, its always sent with sessionID 0 and if not we don't care
		if vrf_pk == nil {
			return nil, utils.LavaFormatError("dataReliability Triggered with vrf_pk == nil", nil,
				&map[string]string{"pairingEpoch": strconv.FormatUint(pairingEpoch, 10), "userAddr": userAddr.String()})
		}
		// verify the providerSig is ineed a signature by a valid provider on this query
		valid, err := s.VerifyReliabilityAddressSigning(ctx, userAddr, request)
		if err != nil {
			return nil, utils.LavaFormatError("VerifyReliabilityAddressSigning invalid", err,
				&map[string]string{"pairingEpoch": strconv.FormatUint(pairingEpoch, 10), "userAddr": userAddr.String(), "dataReliability": fmt.Sprintf("%v", dataReliability)})
		}
		if !valid {
			return nil, utils.LavaFormatError("invalid DataReliability Provider signing", nil,
				&map[string]string{"pairingEpoch": strconv.FormatUint(pairingEpoch, 10), "userAddr": userAddr.String(), "dataReliability": fmt.Sprintf("%v", dataReliability)})
		}
		//verify data reliability fields correspond to the right vrf
		relayEpochStart := g_sentry.GetEpochFromBlockHeight(request.BlockHeight, false)
		valid = utils.VerifyVrfProof(request, *vrf_pk, relayEpochStart)
		if !valid {
			return nil, utils.LavaFormatError("invalid DataReliability fields, VRF wasn't verified with provided proof", nil,
				&map[string]string{"pairingEpoch": strconv.FormatUint(pairingEpoch, 10), "userAddr": userAddr.String(), "dataReliability": fmt.Sprintf("%v", dataReliability), "relayEpochStart": strconv.FormatUint(relayEpochStart, 10)})
		}

		vrfIndex := utils.GetIndexForVrf(request.DataReliability.VrfValue, uint32(g_sentry.GetProvidersCount()), g_sentry.GetReliabilityThreshold())
		if authorisedUserResponse.Index != vrfIndex {
			return nil, utils.LavaFormatError("Provider identified invalid vrfIndex in data reliability request, the given index and self index are different", nil,
				&map[string]string{"pairingEpoch": strconv.FormatUint(pairingEpoch, 10), "userAddr": userAddr.String(),
					"dataReliability": fmt.Sprintf("%v", dataReliability), "relayEpochStart": strconv.FormatUint(relayEpochStart, 10),
					"vrfIndex":   strconv.FormatInt(vrfIndex, 10),
					"self Index": strconv.FormatInt(authorisedUserResponse.Index, 10)})
		}
		utils.LavaFormatInfo("server got valid DataReliability request", nil, nil)

		userSessions.Lock.Lock()

		//will get some rewards for this
		userSessions.dataByEpoch[pairingEpoch].DataReliability = request.DataReliability
		userSessions.Lock.Unlock()

	} else {
		// Validate
		if request.SessionId == 0 {
			return nil, utils.LavaFormatError("SessionID cannot be 0 for non-data reliability requests", nil,
				&map[string]string{"pairingEpoch": strconv.FormatUint(pairingEpoch, 10), "userAddr": userAddr.String(),
					"relay request": fmt.Sprintf("%v", request)})
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
		return nil, utils.LavaFormatError("Sending nodeMsg failed", err,
			nil)
	}

	latestBlock := int64(0)
	finalizedBlockHashes := map[int64]interface{}{}

	if g_sentry.GetSpecComparesHashes() {
		// Add latest block and finalized
		latestBlock, finalizedBlockHashes = g_chainSentry.GetLatestBlockData()
	}

	jsonStr, err := json.Marshal(finalizedBlockHashes)
	if err != nil {
		return nil, utils.LavaFormatError("failed unmarshaling finalizedBlockHashes", err,
			&map[string]string{"finalizedBlockHashes": fmt.Sprintf("%v", finalizedBlockHashes)})
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
			return utils.LavaFormatError("failed signing relay response", err,
				&map[string]string{"request": fmt.Sprintf("%v", request), "reply": fmt.Sprintf("%v", reply)})
		}
		reply.Sig = sig

		if g_sentry.GetSpecComparesHashes() {
			//update sig blocks signature
			sigBlocks, err := sigs.SignResponseFinalizationData(g_privKey, reply, &request, userAddr)
			if err != nil {
				return utils.LavaFormatError("failed signing finalization data", err,
					&map[string]string{"request": fmt.Sprintf("%v", request), "reply": fmt.Sprintf("%v", reply), "userAddr": userAddr.String()})
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
		return false, utils.LavaFormatError("query hash mismatch on data reliability message", nil,
			&map[string]string{"queryHash": string(queryHash), "request QueryHash": string(request.DataReliability.QueryHash)})
	}

	//validate consumer signing on VRF data
	valid, err = sigs.ValidateSignerOnVRFData(consumer, *request.DataReliability)
	if err != nil {
		return false, utils.LavaFormatError("failed to Validate Signer On VRF Data", err,
			&map[string]string{"consumer": consumer.String(), "request.DataReliability": fmt.Sprintf("%v", request.DataReliability)})
	}

	//validate provider signing on query data
	pubKey, err := sigs.RecoverProviderPubKeyFromVrfDataAndQuery(request)
	if err != nil {
		return false, utils.LavaFormatError("failed to Recover Provider PubKey From Vrf Data And Query", err,
			&map[string]string{"consumer": consumer.String(), "request": fmt.Sprintf("%v", request)})
	}
	providerAccAddress, err := sdk.AccAddressFromHex(pubKey.Address().String()) //consumer signer
	if err != nil {
		return false, utils.LavaFormatError("failed converting signer to address", err,
			&map[string]string{"consumer": consumer.String(), "PubKey": pubKey.Address().String()})
	}
	return g_sentry.IsAuthorizedPairing(ctx, consumer.String(), providerAccAddress.String(), uint64(request.BlockHeight)) //return if this pairing is authorised
}

func SendVoteCommitment(voteID string, vote *voteData) {
	msg := conflicttypes.NewMsgConflictVoteCommit(g_sentry.Acc, voteID, vote.CommitHash)
	myWriter := gobytes.Buffer{}
	g_sentry.ClientCtx.Output = &myWriter
	err := tx.GenerateOrBroadcastTxWithFactory(g_sentry.ClientCtx, g_txFactory, msg)
	if err != nil {
		utils.LavaFormatError("failed to send vote commitment", err, nil)
	}
}

func SendVoteReveal(voteID string, vote *voteData) {
	msg := conflicttypes.NewMsgConflictVoteReveal(g_sentry.Acc, voteID, vote.Nonce, vote.RelayDataHash)
	myWriter := gobytes.Buffer{}
	g_sentry.ClientCtx.Output = &myWriter
	err := tx.GenerateOrBroadcastTxWithFactory(g_sentry.ClientCtx, g_txFactory, msg)
	if err != nil {
		utils.LavaFormatError("failed to send vote Reveal", err, nil)
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
			utils.LavaFormatError("Vote Event received but it's too late to vote", nil,
				&map[string]string{"deadline": strconv.FormatUint(voteDeadline, 10), "nodeHeight": strconv.FormatUint(nodeHeight, 10)})
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
				utils.LavaFormatInfo("Received Vote termination event for vote, cleared entry", nil,
					&map[string]string{"voteID": voteID})
				delete(g_votes, voteID)
				return
			}
			//expected to start a new vote but found an existing one
			utils.LavaFormatError("new vote Request for vote had existing entry", nil,
				&map[string]string{"voteParams": fmt.Sprintf("%+v", voteParams), "voteID": voteID, "voteData": fmt.Sprintf("%+v", vote)})
			return
		}
		utils.LavaFormatInfo(" Received Vote Reveal for vote, sending Reveal for result", nil,
			&map[string]string{"voteID": voteID, "voteData": fmt.Sprintf("%+v", vote)})
		SendVoteReveal(voteID, vote)
		return
	} else {
		// new vote
		if voteParams == nil {
			utils.LavaFormatError("vote reveal Request didn't have a vote entry", nil,
				&map[string]string{"voteID": voteID})
			return
		}
		if voteParams.GetCloseVote() {
			utils.LavaFormatError("vote closing received but didn't have a vote entry", nil,
				&map[string]string{"voteID": voteID})
			return
		}
		//try to find this provider in the jury
		found := slices.Contains(voteParams.Voters, g_sentry.Acc)
		if !found {
			utils.LavaFormatInfo("new vote initiated but not for this provider to vote", nil, nil)
			// this is a new vote but not for us
			return
		}
		// we need to send a commit, first we need to use the chainProxy and get the response
		//TODO: implement code that verified the requested block is finalized and if its not waits and tries again
		nodeMsg, err := g_chainProxy.ParseMsg(voteParams.ApiURL, voteParams.RequestData, voteParams.ConnectionType)
		if err != nil {
			utils.LavaFormatError("vote Request did not pass the api check on chain proxy", err,
				&map[string]string{"voteID": voteID, "chainID": voteParams.ChainID})
			return
		}
		reply, err := nodeMsg.Send(ctx)
		if err != nil {
			utils.LavaFormatError("vote relay send has failed", err,
				&map[string]string{"ApiURL": voteParams.ApiURL, "RequestData": string(voteParams.RequestData)})
			return
		}
		nonce := rand.Int63()
		replyDataHash := sigs.HashMsg(reply.Data)
		commitHash := conflicttypes.CommitVoteData(nonce, replyDataHash)

		vote = &voteData{RelayDataHash: replyDataHash, Nonce: nonce, CommitHash: commitHash}
		g_votes[voteID] = vote
		utils.LavaFormatInfo("Received Vote start, sending commitment for result", nil, &map[string]string{"voteID": voteID, "voteData": fmt.Sprintf("%+v", vote)})
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
		utils.LavaFormatError("sentry init failure to initialize", err, &map[string]string{"apiInterface": apiInterface, "ChainID": ChainID})
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
	utils.LavaFormatInfo("Server starting", nil, &map[string]string{"listenAddr": listenAddr, "ChainID": newSentry.GetChainID(), "node": nodeUrl, "spec": newSentry.GetSpecName(), "api Interface": apiInterface})

	//
	// Keys
	keyName, err := sigs.GetKeyName(clientCtx)
	if err != nil {
		utils.LavaFormatFatal("provider failure to getKeyName", err, &map[string]string{"apiInterface": apiInterface, "ChainID": ChainID})
	}

	privKey, err := sigs.GetPrivKey(clientCtx, keyName)
	if err != nil {
		utils.LavaFormatFatal("provider failure to getPrivKey", err, &map[string]string{"apiInterface": apiInterface, "ChainID": ChainID})
	}
	g_privKey = privKey
	serverKey, _ := clientCtx.Keyring.Key(keyName)
	log.Println("Server pubkey", serverKey.GetPubKey().Address())
	utils.LavaFormatInfo("Server loaded keys", nil, &map[string]string{"PublicKey": serverKey.GetPubKey().Address().String()})
	//
	// Node
	chainProxy, err := chainproxy.GetChainProxy(nodeUrl, 1, newSentry)
	if err != nil {
		utils.LavaFormatFatal("provider failure to GetChainProxy", err, &map[string]string{"apiInterface": apiInterface, "ChainID": ChainID})
	}
	chainProxy.Start(ctx)
	g_chainProxy = chainProxy

	if g_sentry.GetSpecComparesHashes() {
		// Start chain sentry
		chainSentry := chainsentry.NewChainSentry(clientCtx, chainProxy, ChainID)
		err = chainSentry.Init(ctx)
		if err != nil {
			utils.LavaFormatFatal("provider failure initializing chainSentry", err, &map[string]string{"apiInterface": apiInterface, "ChainID": ChainID, "nodeUrl": nodeUrl})
		}
		chainSentry.Start(ctx)
		g_chainSentry = chainSentry
	}

	//
	// GRPC
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		utils.LavaFormatFatal("provider failure setting up listener", err, &map[string]string{"listenAddr": listenAddr, "ChainID": ChainID})
	}
	s := grpc.NewServer()
	go func() {
		select {
		case <-ctx.Done():
			utils.LavaFormatInfo("Provider Server ctx.Done", nil, nil)
		case <-signalChan:
			utils.LavaFormatInfo("Provider Server signalChan", nil, nil)
		}
		cancel()
		s.Stop()
	}()

	Server := &relayServer{}
	pairingtypes.RegisterRelayerServer(s, Server)

	utils.LavaFormatInfo("Server listening", nil, &map[string]string{"Address": lis.Addr().String()})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
		utils.LavaFormatFatal("provider failed to serve", err, &map[string]string{"Address": lis.Addr().String(), "ChainID": ChainID})
	}
	askForRewards(g_sentry.GetCurrentEpochHeight())
}
