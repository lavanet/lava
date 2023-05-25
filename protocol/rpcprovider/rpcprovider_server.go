package rpcprovider

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/gogo/status"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/protocol/performance"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc/codes"
)

type RPCProviderServer struct {
	cache                     *performance.Cache
	chainProxy                chainlib.ChainProxy
	privKey                   *btcec.PrivateKey
	reliabilityManager        ReliabilityManagerInf
	providerSessionManager    *lavasession.ProviderSessionManager
	rewardServer              RewardServerInf
	chainParser               chainlib.ChainParser
	rpcProviderEndpoint       *lavasession.RPCProviderEndpoint
	stateTracker              StateTrackerInf
	providerAddress           sdk.AccAddress
	lavaChainID               string
	allowedMissingCUThreshold float64
	metrics                   *metrics.ProviderMetrics
}

type ReliabilityManagerInf interface {
	GetLatestBlockData(fromBlock int64, toBlock int64, specificBlock int64) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, err error)
	GetLatestBlockNum() int64
}

type RewardServerInf interface {
	SendNewProof(ctx context.Context, proof *pairingtypes.RelaySession, epoch uint64, consumerAddr string, apiInterface string) (existingCU uint64, updatedWithProof bool)
	SubscribeStarted(consumer string, epoch uint64, subscribeID string)
	SubscribeEnded(consumer string, epoch uint64, subscribeID string)
}

type StateTrackerInf interface {
	LatestBlock() int64
	GetMaxCuForUser(ctx context.Context, consumerAddress string, chainID string, epocu uint64) (maxCu uint64, err error)
	VerifyPairing(ctx context.Context, consumerAddress string, providerAddress string, epoch uint64, chainID string) (valid bool, total int64, err error)
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
	lavaChainID string,
	allowedMissingCUThreshold float64,
	providerMetrics *metrics.ProviderMetrics,
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
	rpcps.lavaChainID = lavaChainID
	rpcps.allowedMissingCUThreshold = allowedMissingCUThreshold
	rpcps.metrics = providerMetrics
}

// function used to handle relay requests from a consumer, it is called by a provider_listener by calling RegisterReceiver
func (rpcps *RPCProviderServer) Relay(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
	if request.RelayData == nil || request.RelaySession == nil {
		return nil, utils.LavaFormatWarning("invalid relay request, internal fields are nil", nil)
	}
	ctx = utils.AppendUniqueIdentifier(ctx, lavaprotocol.GetSalt(request.RelayData))
	utils.LavaFormatDebug("Provider got relay request",
		utils.Attribute{Key: "GUID", Value: ctx},
		utils.Attribute{Key: "request.SessionId", Value: request.RelaySession.SessionId},
		utils.Attribute{Key: "request.relayNumber", Value: request.RelaySession.RelayNum},
		utils.Attribute{Key: "request.cu", Value: request.RelaySession.CuSum},
		utils.Attribute{Key: "relay_timeout", Value: common.GetRemainingTimeoutFromContext(ctx)},
	)

	// Init relay
	relaySession, consumerAddress, chainMessage, err := rpcps.initRelay(ctx, request)
	if err != nil {
		return nil, rpcps.handleRelayErrorStatus(err)
	}

	// Try sending relay
	reply, err := rpcps.TryRelay(ctx, request, consumerAddress, chainMessage)

	if err != nil || common.ContextOutOfTime(ctx) {
		// failed to send relay. we need to adjust session state. cuSum and relayNumber.
		relayFailureError := rpcps.providerSessionManager.OnSessionFailure(relaySession, request.RelaySession.RelayNum)
		if relayFailureError != nil {
			var extraInfo string
			if err != nil {
				extraInfo = err.Error()
			}
			err = sdkerrors.Wrapf(relayFailureError, "On relay failure: "+extraInfo)
		}
		err = utils.LavaFormatError("TryRelay Failed", err,
			utils.Attribute{Key: "request.SessionId", Value: request.RelaySession.SessionId},
			utils.Attribute{Key: "request.userAddr", Value: consumerAddress},
			utils.Attribute{Key: "GUID", Value: ctx},
			utils.Attribute{Key: "timed_out", Value: common.ContextOutOfTime(ctx)},
		)
		go rpcps.metrics.AddError()
	} else {
		// On successful relay
		pairingEpoch := relaySession.PairingEpoch
		sendRewards := relaySession.IsPayingRelay() // when consumer mismatch causes this relay not to provide cu
		go rpcps.metrics.AddRelay(consumerAddress.String(), relaySession.LatestRelayCu, request.RelaySession.QosReport)
		relayError := rpcps.providerSessionManager.OnSessionDone(relaySession, request.RelaySession.RelayNum)
		if relayError != nil {
			utils.LavaFormatError("OnSession Done failure: ", relayError)
		} else if sendRewards {
			// SendProof gets the request copy, as in the case of data reliability enabled the request.blockNumber is changed.
			// Therefore the signature changes, so we need the original copy to extract the address from it.
			// we want this code to run in parallel so it doesn't stop the flow

			go rpcps.SendProof(ctx, pairingEpoch, request, consumerAddress, chainMessage.GetApiCollection().CollectionData.ApiInterface)
			utils.LavaFormatDebug("Provider Finished Relay Successfully",
				utils.Attribute{Key: "request.SessionId", Value: request.RelaySession.SessionId},
				utils.Attribute{Key: "request.relayNumber", Value: request.RelaySession.RelayNum},
				utils.Attribute{Key: "GUID", Value: ctx},
			)
		}
	}
	utils.LavaFormatDebug("Provider returned a relay response",
		utils.Attribute{Key: "GUID", Value: ctx},
		utils.Attribute{Key: "request.SessionId", Value: request.RelaySession.SessionId},
		utils.Attribute{Key: "request.relayNumber", Value: request.RelaySession.RelayNum},
		utils.Attribute{Key: "request.cu", Value: request.RelaySession.CuSum},
		utils.Attribute{Key: "relay_timeout", Value: common.GetRemainingTimeoutFromContext(ctx)},
	)
	return reply, rpcps.handleRelayErrorStatus(err)
}

func (rpcps *RPCProviderServer) initRelay(ctx context.Context, request *pairingtypes.RelayRequest) (relaySession *lavasession.SingleProviderSession, consumerAddress sdk.AccAddress, chainMessage chainlib.ChainMessage, err error) {
	relaySession, consumerAddress, err = rpcps.verifyRelaySession(ctx, request)
	if err != nil {
		return nil, nil, nil, err
	}
	// parse the message to extract the cu and chainMessage for sending it
	chainMessage, err = rpcps.chainParser.ParseMsg(request.RelayData.ApiUrl, request.RelayData.Data, request.RelayData.ConnectionType)
	if err != nil {
		return nil, nil, nil, err
	}
	relayCU := chainMessage.GetApi().ComputeUnits
	err = relaySession.PrepareSessionForUsage(ctx, relayCU, request.RelaySession.CuSum, rpcps.allowedMissingCUThreshold)
	if err != nil {
		// If PrepareSessionForUsage, session lose sync.
		// We then wrap the error with the SessionOutOfSyncError that has a unique error code.
		// The consumer knows the session lost sync using the code and will create a new session.
		return nil, nil, nil, utils.LavaFormatError("Session Out of sync", lavasession.SessionOutOfSyncError, utils.Attribute{Key: "PrepareSessionForUsage_Error", Value: err.Error()}, utils.Attribute{Key: "GUID", Value: ctx})
	}
	return relaySession, consumerAddress, chainMessage, nil
}

func (rpcps *RPCProviderServer) RelaySubscribe(request *pairingtypes.RelayRequest, srv pairingtypes.Relayer_RelaySubscribeServer) error {
	if request.RelayData == nil || request.RelaySession == nil {
		return utils.LavaFormatError("invalid relay subscribe request, internal fields are nil", nil)
	}
	ctx := utils.AppendUniqueIdentifier(context.Background(), lavaprotocol.GetSalt(request.RelayData))
	utils.LavaFormatDebug("Provider got relay subscribe request",
		utils.Attribute{Key: "request.SessionId", Value: request.RelaySession.SessionId},
		utils.Attribute{Key: "request.relayNumber", Value: request.RelaySession.RelayNum},
		utils.Attribute{Key: "request.cu", Value: request.RelaySession.CuSum},
		utils.Attribute{Key: "GUID", Value: ctx},
	)
	relaySession, consumerAddress, chainMessage, err := rpcps.initRelay(ctx, request)
	if err != nil {
		return rpcps.handleRelayErrorStatus(err)
	}
	subscribed, err := rpcps.TryRelaySubscribe(ctx, uint64(request.RelaySession.Epoch), srv, chainMessage, consumerAddress, relaySession, request.RelaySession.RelayNum) // this function does not return until subscription ends
	if subscribed {
		// meaning we created a subscription and used it for at least a message
		pairingEpoch := relaySession.PairingEpoch
		relayError := rpcps.providerSessionManager.OnSessionDone(relaySession, request.RelaySession.RelayNum) // TODO: when we pay as u go on subscription this will need to change
		if relayError != nil {
			return rpcps.handleRelayErrorStatus(relayError)
		} else {
			go rpcps.SendProof(ctx, pairingEpoch, request, consumerAddress, chainMessage.GetApiCollection().CollectionData.ApiInterface)
			utils.LavaFormatDebug("Provider Finished Relay Successfully",
				utils.Attribute{Key: "request.SessionId", Value: request.RelaySession.SessionId},
				utils.Attribute{Key: "request.relayNumber", Value: request.RelaySession.RelayNum},
				utils.Attribute{Key: "GUID", Value: ctx},
			)
			err = nil // we don't want to return an error here
		}
	} else {
		// we didn't even manage to subscribe
		relayFailureError := rpcps.providerSessionManager.OnSessionFailure(relaySession, request.RelaySession.RelayNum)
		if relayFailureError != nil {
			err = utils.LavaFormatError("failed subscribing", lavasession.SubscriptionInitiationError, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "onSessionFailureError", Value: relayFailureError.Error()}, utils.Attribute{Key: "error", Value: err})
		} else {
			err = utils.LavaFormatError("failed subscribing", lavasession.SubscriptionInitiationError, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "error", Value: err})
		}
	}
	return rpcps.handleRelayErrorStatus(err)
}

func (rpcps *RPCProviderServer) SendProof(ctx context.Context, epoch uint64, request *pairingtypes.RelayRequest, consumerAddress sdk.AccAddress, apiInterface string) error {
	storedCU, updatedWithProof := rpcps.rewardServer.SendNewProof(ctx, request.RelaySession, epoch, consumerAddress.String(), apiInterface)
	if !updatedWithProof && storedCU > request.RelaySession.CuSum {
		rpcps.providerSessionManager.UpdateSessionCU(consumerAddress.String(), epoch, request.RelaySession.SessionId, storedCU)
		err := utils.LavaFormatError("Cu in relay smaller than existing proof", lavasession.ProviderConsumerCuMisMatch, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "existing_proof_cu", Value: storedCU})
		return rpcps.handleRelayErrorStatus(err)
	}
	return nil
}

func (rpcps *RPCProviderServer) TryRelaySubscribe(ctx context.Context, requestBlockHeight uint64, srv pairingtypes.Relayer_RelaySubscribeServer, chainMessage chainlib.ChainMessage, consumerAddress sdk.AccAddress, relaySession *lavasession.SingleProviderSession, relayNumber uint64) (subscribed bool, errRet error) {
	var reply *pairingtypes.RelayReply
	var clientSub *rpcclient.ClientSubscription
	var subscriptionID string
	subscribeRepliesChan := make(chan interface{})
	reply, subscriptionID, clientSub, err := rpcps.chainProxy.SendNodeMsg(ctx, subscribeRepliesChan, chainMessage)
	if err != nil {
		return false, utils.LavaFormatError("Subscription failed", err, utils.Attribute{Key: "GUID", Value: ctx})
	}
	subscription := &lavasession.RPCSubscription{
		Id:                   subscriptionID,
		Sub:                  clientSub,
		SubscribeRepliesChan: subscribeRepliesChan,
	}
	err = rpcps.providerSessionManager.ReleaseSessionAndCreateSubscription(relaySession, subscription, consumerAddress.String(), requestBlockHeight, relayNumber)
	if err != nil {
		return false, err
	}
	rpcps.rewardServer.SubscribeStarted(consumerAddress.String(), requestBlockHeight, subscriptionID)
	processSubscribeMessages := func() (subscribed bool, errRet error) {
		err = srv.Send(reply) // this reply contains the RPC ID
		if err != nil {
			utils.LavaFormatError("Error getting RPC ID", err, utils.Attribute{Key: "GUID", Value: ctx})
		} else {
			subscribed = true
		}

		for {
			select {
			case <-clientSub.Err():
				utils.LavaFormatError("client sub", err, utils.Attribute{Key: "GUID", Value: ctx})
				// delete this connection from the subs map

				return subscribed, err
			case subscribeReply := <-subscribeRepliesChan:
				data, err := json.Marshal(subscribeReply)
				if err != nil {
					return subscribed, utils.LavaFormatError("client sub unmarshal", err, utils.Attribute{Key: "GUID", Value: ctx})
				}

				err = srv.Send(
					&pairingtypes.RelayReply{
						Data: data,
					},
				)
				if err != nil {
					// usually triggered when client closes connection
					if strings.Contains(err.Error(), "Canceled desc = context canceled") {
						err = utils.LavaFormatWarning("Client closed connection", err, utils.Attribute{Key: "GUID", Value: ctx})
					} else {
						err = utils.LavaFormatError("srv.Send", err, utils.Attribute{Key: "GUID", Value: ctx})
					}
					return subscribed, err
				} else {
					subscribed = true
				}

				utils.LavaFormatDebug("Sending data", utils.Attribute{Key: "data", Value: string(data)}, utils.Attribute{Key: "GUID", Value: ctx})
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
	valid := rpcps.providerSessionManager.IsValidEpoch(uint64(request.RelaySession.Epoch))
	if !valid {
		latestBlock := rpcps.stateTracker.LatestBlock()
		errorMessage := "user reported invalid lava block height"
		if request.RelaySession.Epoch > latestBlock {
			errorMessage = "provider is behind user's block height"
		}
		return nil, nil, utils.LavaFormatWarning(errorMessage, lavasession.EpochMismatchError,
			utils.Attribute{Key: "current lava block", Value: latestBlock},
			utils.Attribute{Key: "requested lava block", Value: request.RelaySession.Epoch},
			utils.Attribute{Key: "threshold", Value: rpcps.providerSessionManager.GetBlockedEpochHeight()},
			utils.Attribute{Key: "GUID", Value: ctx},
		)
	}

	// Check data
	err = rpcps.verifyRelayRequestMetaData(ctx, request.RelaySession)
	if err != nil {
		return nil, nil, utils.LavaFormatWarning("did not pass relay validation", err, utils.Attribute{Key: "GUID", Value: ctx})
	}
	// check signature
	extractedConsumerAddress, err = sigs.ExtractSignerAddress(request.RelaySession)
	if err != nil {
		return nil, nil, utils.LavaFormatWarning("extract signer address from relay", err, utils.Attribute{Key: "GUID", Value: ctx})
	}

	// handle non data reliability relays
	consumerAddressString := extractedConsumerAddress.String()

	singleProviderSession, err = rpcps.getSingleProviderSession(ctx, request.RelaySession, consumerAddressString)
	return singleProviderSession, extractedConsumerAddress, err
}

func (rpcps *RPCProviderServer) getSingleProviderSession(ctx context.Context, request *pairingtypes.RelaySession, consumerAddressString string) (*lavasession.SingleProviderSession, error) {
	// regular session, verifies pairing epoch and relay number
	singleProviderSession, err := rpcps.providerSessionManager.GetSession(ctx, consumerAddressString, uint64(request.Epoch), request.SessionId, request.RelayNum)
	if err != nil {
		if lavasession.ConsumerNotRegisteredYet.Is(err) {
			valid, pairedProviders, verifyPairingError := rpcps.stateTracker.VerifyPairing(ctx, consumerAddressString, rpcps.providerAddress.String(), uint64(request.Epoch), request.SpecId)
			if verifyPairingError != nil {
				return nil, utils.LavaFormatError("Failed to VerifyPairing after ConsumerNotRegisteredYet", verifyPairingError,
					utils.Attribute{Key: "GUID", Value: ctx},
					utils.Attribute{Key: "sessionID", Value: request.SessionId},
					utils.Attribute{Key: "consumer", Value: consumerAddressString},
					utils.Attribute{Key: "provider", Value: rpcps.providerAddress},
					utils.Attribute{Key: "relayNum", Value: request.RelayNum},
				)
			}
			if !valid {
				return nil, utils.LavaFormatError("VerifyPairing, this consumer address is not valid with this provider", nil,
					utils.Attribute{Key: "GUID", Value: ctx},
					utils.Attribute{Key: "epoch", Value: request.Epoch},
					utils.Attribute{Key: "sessionID", Value: request.SessionId},
					utils.Attribute{Key: "consumer", Value: consumerAddressString},
					utils.Attribute{Key: "provider", Value: rpcps.providerAddress},
					utils.Attribute{Key: "relayNum", Value: request.RelayNum},
				)
			}
			maxCuForConsumer, getMaxCuError := rpcps.stateTracker.GetMaxCuForUser(ctx, consumerAddressString, request.SpecId, uint64(request.Epoch))
			if getMaxCuError != nil {
				return nil, utils.LavaFormatError("ConsumerNotRegisteredYet: GetMaxCuForUser failed", getMaxCuError,
					utils.Attribute{Key: "GUID", Value: ctx},
					utils.Attribute{Key: "epoch", Value: request.Epoch},
					utils.Attribute{Key: "sessionID", Value: request.SessionId},
					utils.Attribute{Key: "consumer", Value: consumerAddressString},
					utils.Attribute{Key: "provider", Value: rpcps.providerAddress},
					utils.Attribute{Key: "relayNum", Value: request.RelayNum},
				)
			}
			// After validating the consumer we can register it with provider session manager.
			singleProviderSession, err = rpcps.providerSessionManager.RegisterProviderSessionWithConsumer(ctx, consumerAddressString, uint64(request.Epoch), request.SessionId, request.RelayNum, maxCuForConsumer, pairedProviders)
			if err != nil {
				return nil, utils.LavaFormatError("Failed to RegisterProviderSessionWithConsumer", err,
					utils.Attribute{Key: "GUID", Value: ctx},
					utils.Attribute{Key: "sessionID", Value: request.SessionId},
					utils.Attribute{Key: "consumer", Value: consumerAddressString},
					utils.Attribute{Key: "relayNum", Value: request.RelayNum},
				)
			}
		} else {
			return nil, utils.LavaFormatError("Failed to get a provider session", err,
				utils.Attribute{Key: "GUID", Value: ctx},
				utils.Attribute{Key: "sessionID", Value: request.SessionId},
				utils.Attribute{Key: "consumer", Value: consumerAddressString},
				utils.Attribute{Key: "relayNum", Value: request.RelayNum},
			)
		}
	}
	return singleProviderSession, nil
}

func (rpcps *RPCProviderServer) verifyRelayRequestMetaData(ctx context.Context, requestSession *pairingtypes.RelaySession) error {
	providerAddress := rpcps.providerAddress.String()
	if requestSession.Provider != providerAddress {
		return utils.LavaFormatError("request had the wrong provider", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "providerAddress", Value: providerAddress}, utils.Attribute{Key: "request_provider", Value: requestSession.Provider})
	}
	if requestSession.SpecId != rpcps.rpcProviderEndpoint.ChainID {
		return utils.LavaFormatError("request had the wrong specID", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "request_specID", Value: requestSession.SpecId}, utils.Attribute{Key: "chainID", Value: rpcps.rpcProviderEndpoint.ChainID})
	}
	if requestSession.LavaChainId != rpcps.lavaChainID {
		return utils.LavaFormatError("request had the wrong lava chain ID", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "request_lavaChainID", Value: requestSession.LavaChainId}, utils.Attribute{Key: "lava chain id", Value: rpcps.lavaChainID})
	}
	return nil
}

func (rpcps *RPCProviderServer) handleRelayErrorStatus(err error) error {
	if err == nil {
		return nil
	}
	if lavasession.SessionOutOfSyncError.Is(err) {
		err = status.Error(codes.Code(lavasession.SessionOutOfSyncError.ABCICode()), err.Error())
	} else if lavasession.EpochMismatchError.Is(err) {
		err = status.Error(codes.Code(lavasession.EpochMismatchError.ABCICode()), err.Error())
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
		latestBlock, requestedHashes, err = rpcps.reliabilityManager.GetLatestBlockData(fromBlock, toBlock, request.RelayData.RequestBlock)
		if err != nil {
			if chaintracker.InvalidRequestedSpecificBlock.Is(err) {
				// specific block is invalid, try again without specific block
				latestBlock, requestedHashes, err = rpcps.reliabilityManager.GetLatestBlockData(fromBlock, toBlock, spectypes.NOT_APPLICABLE)
				if err != nil {
					return nil, utils.LavaFormatError("error getting range even without specific block", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "fromBlock", Value: fromBlock}, utils.Attribute{Key: "latestBlock", Value: latestBlock}, utils.Attribute{Key: "toBlock", Value: toBlock})
				}
			} else {
				return nil, utils.LavaFormatError("Could not guarantee data reliability", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "requestedBlock", Value: request.RelayData.RequestBlock}, utils.Attribute{Key: "latestBlock", Value: latestBlock}, utils.Attribute{Key: "fromBlock", Value: fromBlock}, utils.Attribute{Key: "toBlock", Value: toBlock})
			}
		}
		request.RelayData.RequestBlock = lavaprotocol.ReplaceRequestedBlock(request.RelayData.RequestBlock, latestBlock)
		for _, block := range requestedHashes {
			if block.Block == request.RelayData.RequestBlock {
				requestedBlockHash = []byte(block.Hash)
				if int64(len(requestedHashes)) == (toBlock - fromBlock + 1) {
					finalizedBlockHashes[block.Block] = block.Hash
				}
			} else {
				finalizedBlockHashes[block.Block] = block.Hash
			}
		}
		if requestedBlockHash == nil && request.RelayData.RequestBlock != spectypes.NOT_APPLICABLE {
			// avoid using cache, but can still service
			utils.LavaFormatWarning("no hash data for requested block", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "requestedBlock", Value: request.RelayData.RequestBlock}, utils.Attribute{Key: "latestBlock", Value: latestBlock})
		}

		// TODO: add a mechanism to handle this
		// if request.RelayData.RequestBlock > latestBlock {
		// 	// consumer asked for a block that is newer than our state tracker, we cant sign this for DR
		// 	return nil, utils.LavaFormatError("Requested a block that is too new", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "requestedBlock", Value: request.RelayData.RequestBlock}, utils.Attribute{Key: "latestBlock", Value: latestBlock})
		// }

		finalized = spectypes.IsFinalizedBlock(request.RelayData.RequestBlock, latestBlock, blockDistanceToFinalization)
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
			utils.LavaFormatWarning("cache not connected", err, utils.Attribute{Key: "GUID", Value: ctx})
		}
		// cache miss or invalid
		reply, _, _, err = rpcps.chainProxy.SendNodeMsg(ctx, nil, chainMsg)
		if err != nil {
			return nil, utils.LavaFormatError("Sending chainMsg failed", err, utils.Attribute{Key: "GUID", Value: ctx})
		}
		if requestedBlockHash != nil || finalized {
			err := cache.SetEntry(ctx, request, rpcps.rpcProviderEndpoint.ApiInterface, requestedBlockHash, rpcps.rpcProviderEndpoint.ChainID, consumerAddr.String(), reply, finalized)
			if err != nil && !performance.NotInitialisedError.Is(err) && request.RelaySession.Epoch != spectypes.NOT_APPLICABLE {
				utils.LavaFormatWarning("error updating cache with new entry", err, utils.Attribute{Key: "GUID", Value: ctx})
			}
		}
	}

	apiName := chainMsg.GetApi().Name
	if reqMsg != nil && strings.Contains(apiName, "unsubscribe") {
		err := rpcps.processUnsubscribe(ctx, apiName, consumerAddr, reqParams, uint64(request.RelayData.RequestBlock))
		if err != nil {
			return nil, err
		}
	}
	// TODO: verify that the consumer still listens, if it took to much time to get the response we cant update the CU.

	jsonStr, err := json.Marshal(finalizedBlockHashes)
	if err != nil {
		return nil, utils.LavaFormatError("failed unmarshaling finalizedBlockHashes", err, utils.Attribute{Key: "GUID", Value: ctx},
			utils.Attribute{Key: "finalizedBlockHashes", Value: finalizedBlockHashes})
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

func (rpcps *RPCProviderServer) processUnsubscribe(ctx context.Context, apiName string, consumerAddr sdk.AccAddress, reqParams interface{}, epoch uint64) error {
	var subscriptionID string
	switch reqParamsCasted := reqParams.(type) {
	case []interface{}:
		var ok bool
		subscriptionID, ok = reqParamsCasted[0].(string)
		if !ok {
			return utils.LavaFormatError("processUnsubscribe - p[0].(string) - type assertion failed", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "type", Value: reqParamsCasted[0]})
		}
	case map[string]interface{}:
		if apiName == "unsubscribe" {
			var ok bool
			subscriptionID, ok = reqParamsCasted["query"].(string)
			if !ok {
				return utils.LavaFormatError("processUnsubscribe - p['query'].(string) - type assertion failed", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "type", Value: reqParamsCasted["query"]})
			}
		}
	}
	return rpcps.providerSessionManager.ProcessUnsubscribe(apiName, subscriptionID, consumerAddr.String(), epoch)
}
