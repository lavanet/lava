package rpcprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/gogo/status"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/relayer/performance"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc/codes"
)

type RPCProviderServer struct {
	cache                  *performance.Cache
	chainProxy             chainlib.ChainProxy
	privKey                *btcec.PrivateKey
	reliabilityManager     ReliabilityManagerInf
	providerSessionManager *lavasession.ProviderSessionManager
	rewardServer           RewardServerInf
	chainParser            chainlib.ChainParser
	rpcProviderEndpoint    *lavasession.RPCProviderEndpoint
	stateTracker           StateTrackerInf
	providerAddress        sdk.AccAddress
}

type ReliabilityManagerInf interface {
	GetLatestBlockData(fromBlock int64, toBlock int64, specificBlock int64) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, err error)
	GetLatestBlockNum() int64
}

type RewardServerInf interface {
	SendNewProof(ctx context.Context, proof *pairingtypes.RelayRequest, epoch uint64, consumerAddr string)
	SendNewDataReliabilityProof(ctx context.Context, dataReliability *pairingtypes.VRFData, epoch uint64, consumerAddr string)
}

type StateTrackerInf interface {
	LatestBlock() int64
	GetVrfPkAndMaxCuForUser(ctx context.Context, consumerAddress string, chainID string, epocu uint64) (vrfPk *utils.VrfPubKey, maxCu uint64, err error)
	VerifyPairing(ctx context.Context, consumerAddress string, providerAddress string, epoch uint64, chainID string) (valid bool, index int64, err error)
	GetProvidersCountForConsumer(ctx context.Context, consumerAddress string, epoch uint64, chainID string) (uint32, error)
}

func (rpcps *RPCProviderServer) ServeRPCRequests(
	ctx context.Context, rpcProviderEndpoint *lavasession.RPCProviderEndpoint,
	chainParser chainlib.ChainParser,
	rewardServer RewardServerInf,
	providerSessionManager *lavasession.ProviderSessionManager,
	reliabilityManager ReliabilityManagerInf,
	privKey *btcec.PrivateKey,
	cache *performance.Cache, chainProxy chainlib.ChainProxy,
	stateTracker StateTrackerInf,
	providerAddress sdk.AccAddress,
) {
	rpcps.cache = cache
	rpcps.chainProxy = chainProxy
	rpcps.privKey = privKey
	rpcps.providerSessionManager = providerSessionManager
	rpcps.reliabilityManager = reliabilityManager
	rpcps.rewardServer = rewardServer
	rpcps.chainParser = chainParser
	rpcps.rpcProviderEndpoint = rpcProviderEndpoint
	rpcps.stateTracker = stateTracker
	rpcps.providerAddress = providerAddress
}
func (rpcps *RPCProviderServer) Relay(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
	// verify the relay metadata is valid (epoch, signature)
	// verify the consumer is authorised
	// create/bring a session
	// verify the relay data is valid (cu, chainParser, requested block)
	// check cache hit
	// send the relay to the node using chainProxy
	// set cache entry (async)
	// attach data reliability finalization data
	// sign the response
	// send the proof to reward server
	// finalize the session
	utils.LavaFormatDebug("Provider got relay request", &map[string]string{
		"request.SessionId":   strconv.FormatUint(request.SessionId, 10),
		"request.relayNumber": strconv.FormatUint(request.RelayNum, 10),
		"request.cu":          strconv.FormatUint(request.CuSum, 10),
	})
	relaySession, consumerAddress, err := rpcps.initRelay(ctx, request)
	if err != nil {
		return nil, rpcps.handleRelayErrorStatus(err)
	}
	// parse the message to extract the cu and chainMessage for sending it
	chainMessage, err := rpcps.chainParser.ParseMsg(request.ApiUrl, request.Data, request.ConnectionType)
	if err != nil {
		return nil, rpcps.handleRelayErrorStatus(err)
	}
	relayCU := chainMessage.GetServiceApi().ComputeUnits
	err = relaySession.PrepareSessionForUsage(relayCU, request.CuSum)
	if err != nil { // TODO: any error here we need to convert to session out of sync error and return that to the user
		return nil, rpcps.handleRelayErrorStatus(err)
	}
	reply, err := rpcps.TryRelay(ctx, request, consumerAddress, chainMessage)
	if err != nil {
		// failed to send relay. we need to adjust session state. cuSum and relayNumber.
		relayFailureError := rpcps.providerSessionManager.OnSessionFailure(relaySession)
		if relayFailureError != nil {
			err = sdkerrors.Wrapf(relayFailureError, "On relay failure: "+err.Error())
		}
		utils.LavaFormatError("TryRelay Failed", err, &map[string]string{
			"request.SessionId": strconv.FormatUint(request.SessionId, 10),
			"request.userAddr":  consumerAddress.String(),
		})
	} else {
		relayError := rpcps.providerSessionManager.OnSessionDone(relaySession, request)
		if relayError != nil {
			err = sdkerrors.Wrapf(relayError, "OnSession Done failure: "+err.Error())
		} else {
			if request.DataReliability == nil {
				rpcps.rewardServer.SendNewProof(ctx, request.ShallowCopy(), relaySession.PairingEpoch, consumerAddress.String())
				utils.LavaFormatDebug("Provider Finished Relay Successfully", &map[string]string{
					"request.SessionId":   strconv.FormatUint(request.SessionId, 10),
					"request.relayNumber": strconv.FormatUint(request.RelayNum, 10),
				})
			} else {
				rpcps.rewardServer.SendNewDataReliabilityProof(ctx, request.DataReliability, relaySession.PairingEpoch, consumerAddress.String())
				utils.LavaFormatDebug("Provider Finished DataReliability Relay Successfully", &map[string]string{
					"request.SessionId":   strconv.FormatUint(request.SessionId, 10),
					"request.relayNumber": strconv.FormatUint(request.RelayNum, 10),
				})
			}
		}
	}
	return reply, rpcps.handleRelayErrorStatus(err)
}
func (rpcps *RPCProviderServer) RelaySubscribe(request *pairingtypes.RelayRequest, srv pairingtypes.Relayer_RelaySubscribeServer) error {
	utils.LavaFormatDebug("Provider got relay subscribe request", &map[string]string{
		"request.SessionId":   strconv.FormatUint(request.SessionId, 10),
		"request.relayNumber": strconv.FormatUint(request.RelayNum, 10),
		"request.cu":          strconv.FormatUint(request.CuSum, 10),
	})
	ctx := context.Background()
	relaySession, consumerAddress, err := rpcps.initRelay(ctx, request)
	if err != nil {
		return rpcps.handleRelayErrorStatus(err)
	}
	// parse the message to extract the cu and chainMessage for sending it
	chainMessage, err := rpcps.chainParser.ParseMsg(request.ApiUrl, request.Data, request.ConnectionType)
	if err != nil {
		return rpcps.handleRelayErrorStatus(err)
	}
	relayCU := chainMessage.GetServiceApi().ComputeUnits
	err = relaySession.PrepareSessionForUsage(relayCU, request.CuSum)
	if err != nil {
		return rpcps.handleRelayErrorStatus(err)
	}
	err = rpcps.TryRelaySubscribe(ctx, request, srv, chainMessage, consumerAddress) // this function does not return until subscription ends
	return err
}

func (rpcps *RPCProviderServer) TryRelaySubscribe(ctx context.Context, request *pairingtypes.RelayRequest, srv pairingtypes.Relayer_RelaySubscribeServer, chainMessage chainlib.ChainMessage, consumerAddress sdk.AccAddress) error {
	var reply *pairingtypes.RelayReply
	var clientSub *rpcclient.ClientSubscription
	var subscriptionID string
	subscribeRepliesChan := make(chan interface{})
	reply, subscriptionID, clientSub, err := rpcps.chainProxy.SendNodeMsg(ctx, subscribeRepliesChan, chainMessage)
	if err != nil {
		return utils.LavaFormatError("Subscription failed", err, nil)
	}
	subscription := &lavasession.RPCSubscription{
		Id:                   subscriptionID,
		Sub:                  clientSub,
		SubscribeRepliesChan: subscribeRepliesChan,
	}
	err = rpcps.providerSessionManager.NewSubscription(consumerAddress.String(), uint64(request.BlockHeight), subscription)
	if err != nil {
		return err
	}
	err = srv.Send(reply) // this reply contains the RPC ID
	if err != nil {
		utils.LavaFormatError("Error getting RPC ID", err, nil)
	}

	for {
		select {
		case <-clientSub.Err():
			utils.LavaFormatError("client sub", err, nil)
			// delete this connection from the subs map
			rpcps.providerSessionManager.SubscriptionFailure(consumerAddress.String(), uint64(request.BlockHeight), subscriptionID)
			return err
		case subscribeReply := <-subscribeRepliesChan:
			data, err := json.Marshal(subscribeReply)
			if err != nil {
				utils.LavaFormatError("client sub unmarshal", err, nil)
				rpcps.providerSessionManager.SubscriptionFailure(consumerAddress.String(), uint64(request.BlockHeight), subscriptionID)
				return err
			}

			err = srv.Send(
				&pairingtypes.RelayReply{
					Data: data,
				},
			)
			if err != nil {
				// usually triggered when client closes connection
				if strings.Contains(err.Error(), "Canceled desc = context canceled") {
					err = utils.LavaFormatWarning("Client closed connection", err, nil)
				} else {
					err = utils.LavaFormatError("srv.Send", err, nil)
				}
				rpcps.providerSessionManager.SubscriptionFailure(consumerAddress.String(), uint64(request.BlockHeight), subscriptionID)
				return err
			}

			utils.LavaFormatDebug("Sending data", &map[string]string{"data": string(data)})
		}
	}
}

// verifies basic relay fields, and gets a provider session
func (rpcps *RPCProviderServer) initRelay(ctx context.Context, request *pairingtypes.RelayRequest) (singleProviderSession *lavasession.SingleProviderSession, extractedConsumerAddress sdk.AccAddress, err error) {
	valid, thresholdEpoch := rpcps.providerSessionManager.IsValidEpoch(uint64(request.BlockHeight))
	if !valid {
		return nil, nil, utils.LavaFormatError("user reported invalid lava block height", nil, &map[string]string{
			"current lava block":   strconv.FormatInt(rpcps.stateTracker.LatestBlock(), 10),
			"requested lava block": strconv.FormatInt(request.BlockHeight, 10),
			"threshold":            strconv.FormatUint(thresholdEpoch, 10),
		})
	}

	// Check data
	err = rpcps.verifyRelayRequestMetaData(request)
	if err != nil {
		return nil, nil, utils.LavaFormatError("did not pass relay validation", err, nil)
	}
	// check signature
	consumerBytes, err := lavaprotocol.ExtractSignerAddress(request)
	if err != nil {
		return nil, nil, utils.LavaFormatError("extract signer address from relay", err, nil)
	}
	extractedConsumerAddress, err = sdk.AccAddressFromHex(consumerBytes.String())
	if err != nil {
		return nil, nil, utils.LavaFormatError("get relay consumer address", err, nil)
	}

	// handle non data reliability relays
	if request.DataReliability == nil {
		// regular session, verifies pairing epoch and relay number
		singleProviderSession, err = rpcps.providerSessionManager.GetSession(extractedConsumerAddress.String(), uint64(request.BlockHeight), request.SessionId, request.RelayNum)
		if err != nil {
			if lavasession.ConsumerNotRegisteredYet.Is(err) {
				// TODO:: validate consumer address get max cu and vrf data and transfer register.

				singleProviderSession, err = rpcps.providerSessionManager.RegisterProviderSessionWithConsumer(extractedConsumerAddress.String(), uint64(request.BlockHeight), request.SessionId)
				if err != nil {
					return nil, nil, utils.LavaFormatError("failed to RegisterProviderSessionWithConsumer", err, &map[string]string{"sessionID": strconv.FormatUint(request.SessionId, 10), "consumer": extractedConsumerAddress.String(), "relayNum": strconv.FormatUint(request.RelayNum, 10)})
				}
			} else {
				return nil, nil, utils.LavaFormatError("failed to get a provider session", err, &map[string]string{"sessionID": strconv.FormatUint(request.SessionId, 10), "consumer": extractedConsumerAddress.String(), "relayNum": strconv.FormatUint(request.RelayNum, 10)})
			}
		}
		return singleProviderSession, extractedConsumerAddress, nil
	}

	// data reliability session verifications
	err = rpcps.verifyDataReliabilityRelayRequest(ctx, request, extractedConsumerAddress)
	if err != nil {
		return nil, nil, utils.LavaFormatError("failed data reliability validation", err, nil)
	}
	dataReliabilitySingleProviderSession, err := rpcps.providerSessionManager.GetDataReliabilitySession(extractedConsumerAddress.String(), uint64(request.BlockHeight))
	if err != nil {
		return nil, nil, utils.LavaFormatError("failed to get a provider data reliability session", err, &map[string]string{"sessionID": strconv.FormatUint(request.SessionId, 10), "consumer": extractedConsumerAddress.String(), "epoch": strconv.FormatInt(request.BlockHeight, 10)})
	}
	return dataReliabilitySingleProviderSession, extractedConsumerAddress, nil
}

func (rpcps *RPCProviderServer) verifyRelayRequestMetaData(request *pairingtypes.RelayRequest) error {
	providerAddress := rpcps.providerAddress.String()
	if request.Provider != providerAddress {
		return utils.LavaFormatError("request had the wrong provider", nil, &map[string]string{"providerAddress": providerAddress, "request_provider": request.Provider})
	}
	if request.ChainID != rpcps.rpcProviderEndpoint.ChainID {
		return utils.LavaFormatError("request had the wrong chainID", nil, &map[string]string{"request_chainID": request.ChainID, "chainID": rpcps.rpcProviderEndpoint.ChainID})
	}
	return nil
}

func (rpcps *RPCProviderServer) verifyDataReliabilityRelayRequest(ctx context.Context, request *pairingtypes.RelayRequest, consumerAddress sdk.AccAddress) error {

	if request.RelayNum > lavasession.DataReliabilitySessionId {
		return utils.LavaFormatError("request's relay num is larger than the data reliability session ID", nil, &map[string]string{"relayNum": strconv.FormatUint(request.RelayNum, 10), "DataReliabilitySessionId": strconv.Itoa(lavasession.DataReliabilitySessionId)})
	}
	if request.CuSum != lavasession.DataReliabilityCuSum {
		return utils.LavaFormatError("request's CU sum is not equal to the data reliability CU sum", nil, &map[string]string{"cuSum": strconv.FormatUint(request.CuSum, 10), "DataReliabilityCuSum": strconv.Itoa(lavasession.DataReliabilityCuSum)})
	}
	vrf_pk, _, err := rpcps.stateTracker.GetVrfPkAndMaxCuForUser(ctx, consumerAddress.String(), request.ChainID, uint64(request.BlockHeight))
	if err != nil {
		return utils.LavaFormatError("failed to get vrfpk and maxCURes for data reliability!", err, &map[string]string{
			"userAddr": consumerAddress.String(),
		})
	}

	// data reliability is not session dependant, its always sent with sessionID 0 and if not we don't care
	if vrf_pk == nil {
		return utils.LavaFormatError("dataReliability Triggered with vrf_pk == nil", nil,
			&map[string]string{"requested epoch": strconv.FormatInt(request.BlockHeight, 10), "userAddr": consumerAddress.String()})
	}
	// verify the providerSig is indeed a signature by a valid provider on this query
	valid, index, err := rpcps.VerifyReliabilityAddressSigning(ctx, consumerAddress, request)
	if err != nil {
		return utils.LavaFormatError("VerifyReliabilityAddressSigning invalid", err,
			&map[string]string{"requested epoch": strconv.FormatInt(request.BlockHeight, 10), "userAddr": consumerAddress.String(), "dataReliability": fmt.Sprintf("%v", request.DataReliability)})
	}
	if !valid {
		return utils.LavaFormatError("invalid DataReliability Provider signing", nil,
			&map[string]string{"requested epoch": strconv.FormatInt(request.BlockHeight, 10), "userAddr": consumerAddress.String(), "dataReliability": fmt.Sprintf("%v", request.DataReliability)})
	}
	// verify data reliability fields correspond to the right vrf
	valid = utils.VerifyVrfProof(request, *vrf_pk, uint64(request.BlockHeight))
	if !valid {
		return utils.LavaFormatError("invalid DataReliability fields, VRF wasn't verified with provided proof", nil,
			&map[string]string{"requested epoch": strconv.FormatInt(request.BlockHeight, 10), "userAddr": consumerAddress.String(), "dataReliability": fmt.Sprintf("%v", request.DataReliability)})
	}
	_, dataReliabilityThreshold := rpcps.chainParser.DataReliabilityParams()
	providersCount, err := rpcps.stateTracker.GetProvidersCountForConsumer(ctx, consumerAddress.String(), uint64(request.BlockHeight), request.ChainID)
	if err != nil {
		return utils.LavaFormatError("VerifyReliabilityAddressSigning failed fetching providers count for consumer", err, &map[string]string{"chainID": request.ChainID, "consumer": consumerAddress.String(), "epoch": strconv.FormatInt(request.BlockHeight, 10)})
	}
	vrfIndex, vrfErr := utils.GetIndexForVrf(request.DataReliability.VrfValue, providersCount, dataReliabilityThreshold)
	if vrfErr != nil {
		dataReliabilityMarshalled, err := json.Marshal(request.DataReliability)
		if err != nil {
			dataReliabilityMarshalled = []byte{}
		}
		return utils.LavaFormatError("Provider identified vrf value in data reliability request does not meet threshold", vrfErr,
			&map[string]string{
				"requested epoch": strconv.FormatInt(request.BlockHeight, 10), "userAddr": consumerAddress.String(),
				"dataReliability": string(dataReliabilityMarshalled), "relayEpochStart": strconv.FormatInt(request.BlockHeight, 10),
				"vrfIndex":   strconv.FormatInt(vrfIndex, 10),
				"self Index": strconv.FormatInt(index, 10),
			})
	}
	if index != vrfIndex {
		dataReliabilityMarshalled, err := json.Marshal(request.DataReliability)
		if err != nil {
			dataReliabilityMarshalled = []byte{}
		}
		return utils.LavaFormatError("Provider identified invalid vrfIndex in data reliability request, the given index and self index are different", nil,
			&map[string]string{
				"requested epoch": strconv.FormatInt(request.BlockHeight, 10), "userAddr": consumerAddress.String(),
				"dataReliability": string(dataReliabilityMarshalled), "relayEpochStart": strconv.FormatInt(request.BlockHeight, 10),
				"vrfIndex":   strconv.FormatInt(vrfIndex, 10),
				"self Index": strconv.FormatInt(index, 10),
			})
	}
	utils.LavaFormatInfo("Simulation: server got valid DataReliability request", nil)
	return nil
}

func (rpcps *RPCProviderServer) VerifyReliabilityAddressSigning(ctx context.Context, consumer sdk.AccAddress, request *pairingtypes.RelayRequest) (valid bool, index int64, err error) {
	queryHash := utils.CalculateQueryHash(*request)
	if !bytes.Equal(queryHash, request.DataReliability.QueryHash) {
		return false, 0, utils.LavaFormatError("query hash mismatch on data reliability message", nil,
			&map[string]string{"queryHash": string(queryHash), "request QueryHash": string(request.DataReliability.QueryHash)})
	}

	// validate consumer signing on VRF data
	valid, err = sigs.ValidateSignerOnVRFData(consumer, *request.DataReliability)
	if err != nil {
		return false, 0, utils.LavaFormatError("failed to Validate Signer On VRF Data", err,
			&map[string]string{"consumer": consumer.String(), "request.DataReliability": fmt.Sprintf("%v", request.DataReliability)})
	}
	if !valid {
		return false, 0, nil
	}
	// validate provider signing on query data
	pubKey, err := sigs.RecoverProviderPubKeyFromVrfDataAndQuery(request)
	if err != nil {
		return false, 0, utils.LavaFormatError("failed to Recover Provider PubKey From Vrf Data And Query", err,
			&map[string]string{"consumer": consumer.String(), "request": fmt.Sprintf("%v", request)})
	}
	providerAccAddress, err := sdk.AccAddressFromHex(pubKey.Address().String()) // consumer signer
	if err != nil {
		return false, 0, utils.LavaFormatError("failed converting signer to address", err,
			&map[string]string{"consumer": consumer.String(), "PubKey": pubKey.Address().String()})
	}
	return rpcps.stateTracker.VerifyPairing(ctx, consumer.String(), providerAccAddress.String(), uint64(request.BlockHeight), request.ChainID) // return if this pairing is authorised
}

func (rpcps *RPCProviderServer) handleRelayErrorStatus(err error) error {
	if err == nil {
		return nil
	}
	if lavasession.SessionOutOfSyncError.Is(err) {
		err = status.Error(codes.Code(lavasession.SessionOutOfSyncError.ABCICode()), err.Error())
	}
	return err
}

func (rpcps *RPCProviderServer) TryRelay(ctx context.Context, request *pairingtypes.RelayRequest, consumerAddr sdk.AccAddress, chainMsg chainlib.ChainMessage) (*pairingtypes.RelayReply, error) {
	// Send
	var reqMsg *rpcInterfaceMessages.JsonrpcMessage
	var reqParams interface{}
	switch msg := chainMsg.GetRPCMessage().(type) {
	case *rpcInterfaceMessages.JsonrpcMessage:
		reqMsg = msg
		reqParams = reqMsg.Params
	default:
		reqMsg = nil
	}
	latestBlock := int64(0)
	finalizedBlockHashes := map[int64]interface{}{}
	var requestedBlockHash []byte = nil
	finalized := false
	dataReliabilityEnabled, _ := rpcps.chainParser.DataReliabilityParams()
	if dataReliabilityEnabled {
		// Add latest block and finalization data
		var err error
		_, _, blockDistanceToFinalization, blocksInFinalizationData := rpcps.chainParser.ChainBlockStats()
		fromBlock := spectypes.LATEST_BLOCK - int64(blockDistanceToFinalization) - int64(blocksInFinalizationData)
		toBlock := spectypes.LATEST_BLOCK - int64(blockDistanceToFinalization)
		latestBlock, requestedHashes, err := rpcps.reliabilityManager.GetLatestBlockData(fromBlock, toBlock, request.RequestBlock)
		if err != nil {
			return nil, utils.LavaFormatError("Could not guarantee data reliability", err, &map[string]string{"requestedBlock": strconv.FormatInt(request.RequestBlock, 10), "latestBlock": strconv.FormatInt(latestBlock, 10)})
		}
		request.RequestBlock = lavaprotocol.ReplaceRequestedBlock(request.RequestBlock, latestBlock)
		for _, block := range requestedHashes {
			if block.Block == request.RequestBlock {
				requestedBlockHash = []byte(block.Hash)
			} else {
				finalizedBlockHashes[block.Block] = block.Hash
			}
		}
		if requestedBlockHash == nil {
			// avoid using cache, but can still service
			utils.LavaFormatWarning("no hash data for requested block", nil, &map[string]string{"requestedBlock": strconv.FormatInt(request.RequestBlock, 10), "latestBlock": strconv.FormatInt(latestBlock, 10)})
		}

		if request.RequestBlock > latestBlock {
			// consumer asked for a block that is newer than our state tracker, we cant sign this for DR
			return nil, utils.LavaFormatError("Requested a block that is too new", err, &map[string]string{"requestedBlock": strconv.FormatInt(request.RequestBlock, 10), "latestBlock": strconv.FormatInt(latestBlock, 10)})
		}

		finalized = spectypes.IsFinalizedBlock(request.RequestBlock, latestBlock, blockDistanceToFinalization)
	}
	cache := rpcps.cache
	// TODO: handle cache on fork for dataReliability = false
	var reply *pairingtypes.RelayReply = nil
	var err error = nil
	if requestedBlockHash != nil || finalized {
		reply, err = cache.GetEntry(ctx, request, rpcps.rpcProviderEndpoint.ApiInterface, requestedBlockHash, rpcps.rpcProviderEndpoint.ChainID, finalized)
	}
	if err != nil || reply == nil {
		if err != nil && performance.NotConnectedError.Is(err) {
			utils.LavaFormatWarning("cache not connected", err, nil)
		}
		// cache miss or invalid
		reply, _, _, err = rpcps.chainProxy.SendNodeMsg(ctx, nil, chainMsg)
		if err != nil {
			return nil, utils.LavaFormatError("Sending chainMsg failed", err, nil)
		}
		if requestedBlockHash != nil || finalized {
			err := cache.SetEntry(ctx, request, rpcps.rpcProviderEndpoint.ApiInterface, requestedBlockHash, rpcps.rpcProviderEndpoint.ChainID, consumerAddr.String(), reply, finalized)
			if err != nil && !performance.NotInitialisedError.Is(err) {
				utils.LavaFormatWarning("error updating cache with new entry", err, nil)
			}
		}
	}

	apiName := chainMsg.GetServiceApi().Name
	if reqMsg != nil && strings.Contains(apiName, "unsubscribe") {
		err := rpcps.processUnsubscribe(apiName, consumerAddr, reqParams)
		if err != nil {
			return nil, err
		}
	}
	// TODO: verify that the consumer still listens, if it took to much time to get the response we cant update the CU.

	jsonStr, err := json.Marshal(finalizedBlockHashes)
	if err != nil {
		return nil, utils.LavaFormatError("failed unmarshaling finalizedBlockHashes", err,
			&map[string]string{"finalizedBlockHashes": fmt.Sprintf("%v", finalizedBlockHashes)})
	}

	reply.FinalizedBlocksHashes = jsonStr
	reply.LatestBlock = latestBlock

	reply, err = lavaprotocol.SignRelayResponse(consumerAddr, *request, rpcps.privKey, reply, dataReliabilityEnabled)
	if err != nil {
		return nil, err
	}

	// return reply to user
	return reply, nil
}

func (rpcps *RPCProviderServer) processUnsubscribe(apiName string, consumerAddr sdk.AccAddress, reqParams interface{}) error {
	switch p := reqParams.(type) {
	case []interface{}:
		subscriptionID, ok := p[0].(string)
		if !ok {
			return fmt.Errorf("processUnsubscribe - p[0].(string) - type assertion failed, type:" + fmt.Sprintf("%s", p[0]))
		}
		return rpcps.providerSessionManager.ProcessUnsubscribeEthereum(subscriptionID, consumerAddr)
	case map[string]interface{}:
		subscriptionID := ""
		if apiName == "unsubscribe" {
			var ok bool
			subscriptionID, ok = p["query"].(string)
			if !ok {
				return fmt.Errorf("processUnsubscribe - p['query'].(string) - type assertion failed, type:" + fmt.Sprintf("%s", p["query"]))
			}
		}
		return rpcps.providerSessionManager.ProcessUnsubscribeTendermint(apiName, subscriptionID, consumerAddr)
	}
	return nil
}
