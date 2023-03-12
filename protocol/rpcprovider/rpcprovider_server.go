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
	SendNewProof(ctx context.Context, proof *pairingtypes.RelayRequest, epoch uint64, consumerAddr string) (existingCU uint64, updatedWithProof bool)
	SendNewDataReliabilityProof(ctx context.Context, dataReliability *pairingtypes.VRFData, epoch uint64, consumerAddr string) (updatedWithProof bool)
	SubscribeStarted(consumer string, epoch uint64, subscribeID string)
	SubscribeEnded(consumer string, epoch uint64, subscribeID string)
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

// function used to handle relay requests from a consumer, it is called by a provider_listener by calling RegisterReceiver
func (rpcps *RPCProviderServer) Relay(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
	utils.LavaFormatDebug("Provider got relay request", &map[string]string{
		"request.SessionId":   strconv.FormatUint(request.SessionId, 10),
		"request.relayNumber": strconv.FormatUint(request.RelayNum, 10),
		"request.cu":          strconv.FormatUint(request.CuSum, 10),
	})
	relaySession, consumerAddress, chainMessage, err := rpcps.initRelay(ctx, request)
	if err != nil {
		return nil, rpcps.handleRelayErrorStatus(err)
	}
	reply, err := rpcps.TryRelay(ctx, request, consumerAddress, chainMessage)
	if err != nil {
		// failed to send relay. we need to adjust session state. cuSum and relayNumber.
		relayFailureError := rpcps.providerSessionManager.OnSessionFailure(relaySession)
		if relayFailureError != nil {
			err = sdkerrors.Wrapf(relayFailureError, "On relay failure: "+err.Error())
		}
		err = utils.LavaFormatError("TryRelay Failed", err, &map[string]string{
			"request.SessionId": strconv.FormatUint(request.SessionId, 10),
			"request.userAddr":  consumerAddress.String(),
		})
	} else {
		relayError := rpcps.providerSessionManager.OnSessionDone(relaySession)
		if relayError != nil {
			err = sdkerrors.Wrapf(relayError, "OnSession Done failure: "+err.Error())
		} else {
			if request.DataReliability == nil {
				err = rpcps.SendProof(ctx, relaySession, request, consumerAddress)
				if err != nil {
					return nil, err
				}
				utils.LavaFormatDebug("Provider Finished Relay Successfully", &map[string]string{
					"request.SessionId":   strconv.FormatUint(request.SessionId, 10),
					"request.relayNumber": strconv.FormatUint(request.RelayNum, 10),
				})
			} else {
				updated := rpcps.rewardServer.SendNewDataReliabilityProof(ctx, request.DataReliability, relaySession.PairingEpoch, consumerAddress.String())
				if !updated {
					return nil, utils.LavaFormatError("existing data reliability proof", lavasession.DataReliabilityAlreadySentThisEpochError, nil)
				}
				utils.LavaFormatDebug("Provider Finished DataReliability Relay Successfully", &map[string]string{
					"request.SessionId":   strconv.FormatUint(request.SessionId, 10),
					"request.relayNumber": strconv.FormatUint(request.RelayNum, 10),
				})
			}
		}
	}
	return reply, rpcps.handleRelayErrorStatus(err)
}

func (rpcps *RPCProviderServer) initRelay(ctx context.Context, request *pairingtypes.RelayRequest) (relaySession *lavasession.SingleProviderSession, consumerAddress sdk.AccAddress, chainMessage chainlib.ChainMessage, err error) {
	relaySession, consumerAddress, err = rpcps.verifyRelaySession(ctx, request)
	if err != nil {
		return nil, nil, nil, err
	}
	// parse the message to extract the cu and chainMessage for sending it
	chainMessage, err = rpcps.chainParser.ParseMsg(request.ApiUrl, request.Data, request.ConnectionType)
	if err != nil {
		return nil, nil, nil, err
	}
	relayCU := chainMessage.GetServiceApi().ComputeUnits
	err = relaySession.PrepareSessionForUsage(relayCU, request.CuSum, request.RelayNum)
	if err != nil {
		// If PrepareSessionForUsage, session lose sync.
		// We then wrap the error with the SessionOutOfSyncError that has a unique error code.
		// The consumer knows the session lost sync using the code and will create a new session.
		return nil, nil, nil, utils.LavaFormatError("Session Out of sync", lavasession.SessionOutOfSyncError, &map[string]string{"PrepareSessionForUsage_Error": err.Error()})
	}
	return relaySession, consumerAddress, chainMessage, nil
}

func (rpcps *RPCProviderServer) RelaySubscribe(request *pairingtypes.RelayRequest, srv pairingtypes.Relayer_RelaySubscribeServer) error {
	if request.DataReliability != nil {
		return utils.LavaFormatError("subscribe data reliability not supported", nil, nil)
	}
	utils.LavaFormatDebug("Provider got relay subscribe request", &map[string]string{
		"request.SessionId":   strconv.FormatUint(request.SessionId, 10),
		"request.relayNumber": strconv.FormatUint(request.RelayNum, 10),
		"request.cu":          strconv.FormatUint(request.CuSum, 10),
	})
	ctx := context.Background()
	relaySession, consumerAddress, chainMessage, err := rpcps.initRelay(ctx, request)
	if err != nil {
		return rpcps.handleRelayErrorStatus(err)
	}
	subscribed, err := rpcps.TryRelaySubscribe(ctx, uint64(request.BlockHeight), srv, chainMessage, consumerAddress, relaySession) // this function does not return until subscription ends
	if subscribed {
		// meaning we created a subscription and used it for at least a message
		relayError := rpcps.providerSessionManager.OnSessionDone(relaySession) // TODO: when we pay as u go on subscription this will need to change
		if relayError != nil {
			err = sdkerrors.Wrapf(relayError, "OnSession Done failure: "+err.Error())
		} else {
			err = rpcps.SendProof(ctx, relaySession, request, consumerAddress)
			if err != nil {
				return err
			}
			utils.LavaFormatDebug("Provider finished subscribing", &map[string]string{
				"request.SessionId":   strconv.FormatUint(request.SessionId, 10),
				"request.relayNumber": strconv.FormatUint(request.RelayNum, 10),
				"request.cu":          strconv.FormatUint(request.CuSum, 10),
				"termination":         err.Error(),
			})
			err = nil // we don't want to return an error here
		}
	} else {
		// we didn't even manage to subscribe
		relayFailureError := rpcps.providerSessionManager.OnSessionFailure(relaySession)
		if relayFailureError != nil {
			err = utils.LavaFormatError("failed subscribing", lavasession.SubscriptionInitiationError, &map[string]string{"onSessionFailureError": relayFailureError.Error()})
		} else {
			err = utils.LavaFormatError("failed subscribing", lavasession.SubscriptionInitiationError, nil)
		}
	}
	return rpcps.handleRelayErrorStatus(err)
}

func (rpcps *RPCProviderServer) SendProof(ctx context.Context, relaySession *lavasession.SingleProviderSession, request *pairingtypes.RelayRequest, consumerAddress sdk.AccAddress) error {
	epoch := relaySession.PairingEpoch
	storedCU, updatedWithProof := rpcps.rewardServer.SendNewProof(ctx, request.ShallowCopy(), epoch, consumerAddress.String())
	if !updatedWithProof && storedCU > request.CuSum {
		rpcps.providerSessionManager.UpdateSessionCU(consumerAddress.String(), epoch, request.SessionId, storedCU)
		err := utils.LavaFormatError("Cu in relay smaller than existing proof", lavasession.ProviderConsumerCuMisMatch, &map[string]string{"existing_proof_cu": strconv.FormatUint(storedCU, 10)})
		return rpcps.handleRelayErrorStatus(err)
	}
	return nil
}

func (rpcps *RPCProviderServer) TryRelaySubscribe(ctx context.Context, requestBlockHeight uint64, srv pairingtypes.Relayer_RelaySubscribeServer, chainMessage chainlib.ChainMessage, consumerAddress sdk.AccAddress, relaySession *lavasession.SingleProviderSession) (subscribed bool, errRet error) {
	var reply *pairingtypes.RelayReply
	var clientSub *rpcclient.ClientSubscription
	var subscriptionID string
	subscribeRepliesChan := make(chan interface{})
	reply, subscriptionID, clientSub, err := rpcps.chainProxy.SendNodeMsg(ctx, subscribeRepliesChan, chainMessage)
	if err != nil {
		return false, utils.LavaFormatError("Subscription failed", err, nil)
	}
	subscription := &lavasession.RPCSubscription{
		Id:                   subscriptionID,
		Sub:                  clientSub,
		SubscribeRepliesChan: subscribeRepliesChan,
	}
	err = rpcps.providerSessionManager.ReleaseSessionAndCreateSubscription(relaySession, subscription, consumerAddress.String(), requestBlockHeight)
	if err != nil {
		return false, err
	}
	rpcps.rewardServer.SubscribeStarted(consumerAddress.String(), requestBlockHeight, subscriptionID)
	processSubscribeMessages := func() (subscribed bool, errRet error) {
		err = srv.Send(reply) // this reply contains the RPC ID
		if err != nil {
			utils.LavaFormatError("Error getting RPC ID", err, nil)
		} else {
			subscribed = true
		}

		for {
			select {
			case <-clientSub.Err():
				utils.LavaFormatError("client sub", err, nil)
				// delete this connection from the subs map

				return subscribed, err
			case subscribeReply := <-subscribeRepliesChan:
				data, err := json.Marshal(subscribeReply)
				if err != nil {
					return subscribed, utils.LavaFormatError("client sub unmarshal", err, nil)
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
					return subscribed, err
				} else {
					subscribed = true
				}

				utils.LavaFormatDebug("Sending data", &map[string]string{"data": string(data)})
			}
		}
	}
	subscribed, errRet = processSubscribeMessages()
	rpcps.providerSessionManager.SubscriptionEnded(consumerAddress.String(), requestBlockHeight, subscriptionID)
	rpcps.rewardServer.SubscribeEnded(consumerAddress.String(), requestBlockHeight, subscriptionID)
	return subscribed, errRet
}

// verifies basic relay fields, and gets a provider session
func (rpcps *RPCProviderServer) verifyRelaySession(ctx context.Context, request *pairingtypes.RelayRequest) (singleProviderSession *lavasession.SingleProviderSession, extractedConsumerAddress sdk.AccAddress, err error) {
	valid := rpcps.providerSessionManager.IsValidEpoch(uint64(request.BlockHeight))
	if !valid {
		latestBlock := rpcps.stateTracker.LatestBlock()
		errorMessage := "user reported invalid lava block height"
		if request.BlockHeight > latestBlock {
			errorMessage = "provider is behind user's block height"
		}
		return nil, nil, utils.LavaFormatError(errorMessage, nil, &map[string]string{
			"current lava block":   strconv.FormatInt(latestBlock, 10),
			"requested lava block": strconv.FormatInt(request.BlockHeight, 10),
			"threshold":            strconv.FormatUint(rpcps.providerSessionManager.GetBlockedEpochHeight(), 10),
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
		singleProviderSession, err = rpcps.getSingleProviderSession(ctx, request, extractedConsumerAddress.String())
		return singleProviderSession, extractedConsumerAddress, err
	}

	// data reliability session verifications
	err = rpcps.verifyDataReliabilityRelayRequest(ctx, request, extractedConsumerAddress)
	if err != nil {
		return nil, nil, utils.LavaFormatError("failed data reliability validation", err, nil)
	}
	dataReliabilitySingleProviderSession, err := rpcps.providerSessionManager.GetDataReliabilitySession(extractedConsumerAddress.String(), uint64(request.BlockHeight), request.SessionId, request.RelayNum)
	if err != nil {
		if lavasession.DataReliabilityAlreadySentThisEpochError.Is(err) {
			return nil, nil, err
		}
		return nil, nil, utils.LavaFormatError("failed to get a provider data reliability session", err, &map[string]string{"sessionID": strconv.FormatUint(request.SessionId, 10), "consumer": extractedConsumerAddress.String(), "epoch": strconv.FormatInt(request.BlockHeight, 10)})
	}
	return dataReliabilitySingleProviderSession, extractedConsumerAddress, nil
}

func (rpcps *RPCProviderServer) getSingleProviderSession(ctx context.Context, request *pairingtypes.RelayRequest, consumerAddressString string) (*lavasession.SingleProviderSession, error) {
	// regular session, verifies pairing epoch and relay number
	singleProviderSession, err := rpcps.providerSessionManager.GetSession(consumerAddressString, uint64(request.BlockHeight), request.SessionId, request.RelayNum)
	if err != nil {
		if lavasession.ConsumerNotRegisteredYet.Is(err) {

			valid, _, verifyPairingError := rpcps.stateTracker.VerifyPairing(ctx, consumerAddressString, rpcps.providerAddress.String(), uint64(request.BlockHeight), request.ChainID)
			if verifyPairingError != nil {
				return nil, utils.LavaFormatError("Failed to VerifyPairing after ConsumerNotRegisteredYet", verifyPairingError, &map[string]string{"sessionID": strconv.FormatUint(request.SessionId, 10), "consumer": consumerAddressString, "provider": rpcps.providerAddress.String(), "relayNum": strconv.FormatUint(request.RelayNum, 10)})
			}
			if !valid {
				return nil, utils.LavaFormatError("VerifyPairing, this consumer address is not valid with this provider", nil, &map[string]string{"epoch": strconv.FormatInt(request.BlockHeight, 10), "sessionID": strconv.FormatUint(request.SessionId, 10), "consumer": consumerAddressString, "provider": rpcps.providerAddress.String(), "relayNum": strconv.FormatUint(request.RelayNum, 10)})
			}
			_, maxCuForConsumer, getVrfAndMaxCuError := rpcps.stateTracker.GetVrfPkAndMaxCuForUser(ctx, consumerAddressString, request.ChainID, uint64(request.BlockHeight))
			if getVrfAndMaxCuError != nil {
				return nil, utils.LavaFormatError("ConsumerNotRegisteredYet: GetVrfPkAndMaxCuForUser failed", getVrfAndMaxCuError, &map[string]string{"epoch": strconv.FormatInt(request.BlockHeight, 10), "sessionID": strconv.FormatUint(request.SessionId, 10), "consumer": consumerAddressString, "provider": rpcps.providerAddress.String(), "relayNum": strconv.FormatUint(request.RelayNum, 10)})
			}
			// After validating the consumer we can register it with provider session manager.
			singleProviderSession, err = rpcps.providerSessionManager.RegisterProviderSessionWithConsumer(consumerAddressString, uint64(request.BlockHeight), request.SessionId, request.RelayNum, maxCuForConsumer)
			if err != nil {
				return nil, utils.LavaFormatError("Failed to RegisterProviderSessionWithConsumer", err, &map[string]string{"sessionID": strconv.FormatUint(request.SessionId, 10), "consumer": consumerAddressString, "relayNum": strconv.FormatUint(request.RelayNum, 10)})
			}
		} else {
			return nil, utils.LavaFormatError("Failed to get a provider session", err, &map[string]string{"sessionID": strconv.FormatUint(request.SessionId, 10), "consumer": consumerAddressString, "relayNum": strconv.FormatUint(request.RelayNum, 10)})
		}
	}
	return singleProviderSession, nil
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
		toBlock := spectypes.LATEST_BLOCK - int64(blockDistanceToFinalization)
		fromBlock := toBlock - int64(blocksInFinalizationData) + 1
		var requestedHashes []*chaintracker.BlockStore
		latestBlock, requestedHashes, err = rpcps.reliabilityManager.GetLatestBlockData(fromBlock, toBlock, request.RequestBlock)
		if err != nil {
			if chaintracker.InvalidRequestedSpecificBlock.Is(err) {
				// specific block is invalid, try again without specific block
				latestBlock, requestedHashes, err = rpcps.reliabilityManager.GetLatestBlockData(fromBlock, toBlock, spectypes.NOT_APPLICABLE)
				if err != nil {
					return nil, utils.LavaFormatError("error getting range even without specific block", err, &map[string]string{"fromBlock": strconv.FormatInt(fromBlock, 10), "latestBlock": strconv.FormatInt(latestBlock, 10), "toBlock": strconv.FormatInt(toBlock, 10)})
				}
			} else {
				return nil, utils.LavaFormatError("Could not guarantee data reliability", err, &map[string]string{"requestedBlock": strconv.FormatInt(request.RequestBlock, 10), "latestBlock": strconv.FormatInt(latestBlock, 10), "fromBlock": strconv.FormatInt(fromBlock, 10), "toBlock": strconv.FormatInt(toBlock, 10)})
			}
		}
		request.RequestBlock = lavaprotocol.ReplaceRequestedBlock(request.RequestBlock, latestBlock)
		for _, block := range requestedHashes {
			if block.Block == request.RequestBlock {
				requestedBlockHash = []byte(block.Hash)
				if int64(len(requestedHashes)) == (toBlock - fromBlock + 1) {
					finalizedBlockHashes[block.Block] = block.Hash
				}
			} else {
				finalizedBlockHashes[block.Block] = block.Hash
			}
		}
		if requestedBlockHash == nil && request.RequestBlock != spectypes.NOT_APPLICABLE {
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
			if err != nil && !performance.NotInitialisedError.Is(err) && request.BlockHeight != spectypes.NOT_APPLICABLE {
				utils.LavaFormatWarning("error updating cache with new entry", err, nil)
			}
		}
	}

	apiName := chainMsg.GetServiceApi().Name
	if reqMsg != nil && strings.Contains(apiName, "unsubscribe") {
		err := rpcps.processUnsubscribe(apiName, consumerAddr, reqParams, uint64(request.RequestBlock))
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

func (rpcps *RPCProviderServer) processUnsubscribe(apiName string, consumerAddr sdk.AccAddress, reqParams interface{}, epoch uint64) error {
	var subscriptionID string
	switch p := reqParams.(type) {
	case []interface{}:
		var ok bool
		subscriptionID, ok = p[0].(string)
		if !ok {
			return utils.LavaFormatError("processUnsubscribe - p[0].(string) - type assertion failed", nil, &map[string]string{"type": fmt.Sprintf("%s", p[0])})
		}
	case map[string]interface{}:
		if apiName == "unsubscribe" {
			var ok bool
			subscriptionID, ok = p["query"].(string)
			if !ok {
				return utils.LavaFormatError("processUnsubscribe - p['query'].(string) - type assertion failed", nil, &map[string]string{"type": fmt.Sprintf("%s", p["query"])})
			}
		}
	}
	return rpcps.providerSessionManager.ProcessUnsubscribe(apiName, subscriptionID, consumerAddr.String(), epoch)
}
