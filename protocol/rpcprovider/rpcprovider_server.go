package rpcprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	"github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	chainRouter               chainlib.ChainRouter
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
	GetLatestBlockData(fromBlock, toBlock, specificBlock int64) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, err error)
	GetLatestBlockNum() int64
}

type RewardServerInf interface {
	SendNewProof(ctx context.Context, proof *pairingtypes.RelaySession, epoch uint64, consumerAddr, apiInterface string) (existingCU uint64, updatedWithProof bool)
	SubscribeStarted(consumer string, epoch uint64, subscribeID string)
	SubscribeEnded(consumer string, epoch uint64, subscribeID string)
}

type StateTrackerInf interface {
	LatestBlock() int64
	GetMaxCuForUser(ctx context.Context, consumerAddress, chainID string, epocu uint64) (maxCu uint64, err error)
	VerifyPairing(ctx context.Context, consumerAddress, providerAddress string, epoch uint64, chainID string) (valid bool, total int64, projectId string, err error)
}

func (rpcps *RPCProviderServer) ServeRPCRequests(
	ctx context.Context, rpcProviderEndpoint *lavasession.RPCProviderEndpoint,
	chainParser chainlib.ChainParser,
	rewardServer RewardServerInf,
	providerSessionManager *lavasession.ProviderSessionManager,
	reliabilityManager ReliabilityManagerInf,
	privKey *btcec.PrivateKey,
	cache *performance.Cache,
	chainRouter chainlib.ChainRouter,
	stateTracker StateTrackerInf,
	providerAddress sdk.AccAddress,
	lavaChainID string,
	allowedMissingCUThreshold float64,
	providerMetrics *metrics.ProviderMetrics,
) {
	rpcps.cache = cache
	rpcps.chainRouter = chainRouter
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
		utils.Attribute{Key: "relay addon", Value: request.RelayData.Addon},
		utils.Attribute{Key: "relay extensions", Value: request.RelayData.GetExtensions()},
	)

	if request.RelaySession.QosExcellenceReport != nil {
		utils.LavaFormatDebug("DEBUG",
			utils.Attribute{Key: "qosExc latency", Value: request.RelaySession.GetQosExcellenceReport().Latency.String()},
		)
	}
	if request.RelaySession.QosReport != nil {
		utils.LavaFormatDebug("DEBUG",
			utils.Attribute{Key: "qos latency", Value: request.RelaySession.GetQosReport().Latency.String()},
		)
	}

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
		replyBlock := reply.LatestBlock
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
				utils.Attribute{Key: "requestedBlock", Value: request.RelayData.RequestBlock},
				utils.Attribute{Key: "replyBlock", Value: replyBlock},
				utils.Attribute{Key: "method", Value: chainMessage.GetApi().Name},
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
	defer func(relaySession *lavasession.SingleProviderSession) {
		// if we error in here until PrepareSessionForUsage was called successfully we can't call OnSessionFailure
		if err != nil {
			relaySession.DisbandSession()
		}
	}(relaySession) // lock in the session address
	// parse the message to extract the cu and chainMessage for sending it
	chainMessage, err = rpcps.chainParser.ParseMsg(request.RelayData.ApiUrl, request.RelayData.Data, request.RelayData.ConnectionType, request.RelayData.GetMetadata(), 0)
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

func (rpcps *RPCProviderServer) ValidateAddonsExtensions(addon string, extensions []string, chainMessage chainlib.ChainMessage) error {
	// this validates all of the values are handled by chainParser
	_, _, err := rpcps.chainParser.SeparateAddonsExtensions(append(extensions, addon))
	if err != nil {
		return err
	}
	apiCollection := chainMessage.GetApiCollection()
	if apiCollection.CollectionData.AddOn != addon {
		return utils.LavaFormatWarning("invalid addon in relay, parsed addon is not the same as requested", nil, utils.Attribute{Key: "requested addon", Value: addon[0]}, utils.Attribute{Key: "parsed addon", Value: chainMessage.GetApiCollection().CollectionData.AddOn})
	}
	if !rpcps.chainRouter.ExtensionsSupported(extensions) {
		return utils.LavaFormatWarning("requested extensions are unsupported in chainRouter", nil, utils.Attribute{Key: "requested extensions", Value: extensions})
	}
	return nil
}

func (rpcps *RPCProviderServer) ValidateRequest(chainMessage chainlib.ChainMessage, request *pairingtypes.RelayRequest, ctx context.Context) error {
	// TODO: remove this if case, the reason its here is because lava-sdk does't have data reliability + block parsing.
	// this is a temporary solution until we have a working block parsing in lava-sdk
	if request.RelayData.RequestBlock == spectypes.NOT_APPLICABLE {
		return nil
	}
	reqBlock, _ := chainMessage.RequestedBlock()
	if reqBlock != request.RelayData.RequestBlock {
		// the consumer either configured an invalid value or is modifying the requested block as part of a data reliability message
		// see if this modification is supported
		providerRequestedBlockPreUpdate := reqBlock
		chainMessage.UpdateLatestBlockInMessage(request.RelayData.RequestBlock, true)
		// if after UpdateLatestBlockInMessage it's not aligned we have a problem
		reqBlock, _ = chainMessage.RequestedBlock()
		if reqBlock != request.RelayData.RequestBlock {
			return utils.LavaFormatError("requested block mismatch between consumer and provider", nil, utils.Attribute{Key: "provider_parsed_block_pre_update", Value: providerRequestedBlockPreUpdate}, utils.Attribute{Key: "provider_requested_block", Value: reqBlock}, utils.Attribute{Key: "consumer_requested_block", Value: request.RelayData.RequestBlock}, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "metadata", Value: request.RelayData.Metadata})
		}
	}
	return nil
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
		// no need to perform on session done as we did it in try relay subscribe
		go rpcps.SendProof(ctx, pairingEpoch, request, consumerAddress, chainMessage.GetApiCollection().CollectionData.ApiInterface)
		utils.LavaFormatDebug("Provider Finished Relay Successfully",
			utils.Attribute{Key: "request.SessionId", Value: request.RelaySession.SessionId},
			utils.Attribute{Key: "request.relayNumber", Value: request.RelaySession.RelayNum},
			utils.Attribute{Key: "GUID", Value: ctx},
		)
		err = nil // we don't want to return an error here
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
	reply, subscriptionID, clientSub, err := rpcps.chainRouter.SendNodeMsg(ctx, subscribeRepliesChan, chainMessage, nil)
	if err != nil {
		return false, utils.LavaFormatError("Subscription failed", err, utils.Attribute{Key: "GUID", Value: ctx})
	}
	reply.Metadata, _, _ = rpcps.chainParser.HandleHeaders(reply.Metadata, chainMessage.GetApiCollection(), spectypes.Header_pass_reply)
	if clientSub == nil {
		// failed subscription, but not an error. (probably a node error)
		// return the response to the user, and close the session.
		relayError := rpcps.providerSessionManager.OnSessionDone(relaySession, relayNumber) // subscription failed due to node error mark session as done and return
		if relayError != nil {
			utils.LavaFormatError("Error OnSessionDone", relayError)
		}
		err = srv.Send(reply) // this reply contains the error to the subscription
		if err != nil {
			utils.LavaFormatError("Error returning response", err)
		}
		return true, nil // we already returned the error to the user so no need to return another error.
	}

	// if we got a node error clientSub will be nil and also err will be nil. in that case we need to check for clientSub
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
		} else if request.RelaySession.Epoch == 0 {
			errorMessage = "user reported lava block 0, either it's test rpcprovider or a consumer that has no node access"
		}
		utils.LavaFormatInfo(errorMessage,
			utils.Attribute{Key: "Info Type", Value: lavasession.EpochMismatchError},
			utils.Attribute{Key: "current lava block", Value: latestBlock},
			utils.Attribute{Key: "requested lava block", Value: request.RelaySession.Epoch},
			utils.Attribute{Key: "threshold", Value: rpcps.providerSessionManager.GetBlockedEpochHeight()},
			utils.Attribute{Key: "GUID", Value: ctx},
		)
		return nil, nil, lavasession.EpochMismatchError
	}

	// Check data
	err = rpcps.verifyRelayRequestMetaData(ctx, request.RelaySession, request.RelayData)
	if err != nil {
		return nil, nil, utils.LavaFormatWarning("did not pass relay validation", err, utils.Attribute{Key: "GUID", Value: ctx})
	}

	// check signature
	extractedConsumerAddress, err = rpcps.ExtractConsumerAddress(ctx, request.RelaySession)
	if err != nil {
		return nil, nil, err
	}
	consumerAddressString := extractedConsumerAddress.String()

	// validate & fetch badge to send into provider session manager
	err = rpcps.validateBadgeSession(ctx, request.RelaySession)
	if err != nil {
		return nil, nil, utils.LavaFormatWarning("badge validation err", err, utils.Attribute{Key: "GUID", Value: ctx})
	}

	singleProviderSession, err = rpcps.getSingleProviderSession(ctx, request.RelaySession, consumerAddressString)
	return singleProviderSession, extractedConsumerAddress, err
}

func (rpcps *RPCProviderServer) ExtractConsumerAddress(ctx context.Context, relaySession *pairingtypes.RelaySession) (extractedConsumerAddress sdk.AccAddress, err error) {
	if relaySession.Badge != nil {
		extractedConsumerAddress, err = sigs.ExtractSignerAddress(*relaySession.Badge)
		if err != nil {
			return nil, err
		}
	} else {
		extractedConsumerAddress, err = sigs.ExtractSignerAddress(relaySession)
		if err != nil {
			return nil, utils.LavaFormatWarning("extract signer address from relay", err, utils.Attribute{Key: "GUID", Value: ctx})
		}
	}
	return extractedConsumerAddress, nil
}

func (rpcps *RPCProviderServer) validateBadgeSession(ctx context.Context, relaySession *pairingtypes.RelaySession) error {
	if relaySession.Badge == nil { // not a badge session
		return nil
	}
	// validating badge signer
	badgeUserSigner, err := sigs.ExtractSignerAddress(relaySession)
	if err != nil {
		return utils.LavaFormatWarning("cannot extract badge user from relay", err, utils.Attribute{Key: "GUID", Value: ctx})
	}
	if badgeUserSigner.String() != relaySession.Badge.Address {
		return utils.LavaFormatWarning("did not pass badge signer validation", err, utils.Attribute{Key: "GUID", Value: ctx})
	}
	// validating badge lavaChainId
	if relaySession.LavaChainId != relaySession.Badge.LavaChainId {
		return utils.LavaFormatWarning("mismatch in badge lavaChainId", err, utils.Attribute{Key: "GUID", Value: ctx})
	}
	return nil
}

func (rpcps *RPCProviderServer) getSingleProviderSession(ctx context.Context, request *pairingtypes.RelaySession, consumerAddressString string) (*lavasession.SingleProviderSession, error) {
	// regular session, verifies pairing epoch and relay number
	singleProviderSession, err := rpcps.providerSessionManager.GetSession(ctx, consumerAddressString, uint64(request.Epoch), request.SessionId, request.RelayNum, request.Badge)
	if err != nil {
		if lavasession.ConsumerNotRegisteredYet.Is(err) {
			valid, pairedProviders, projectId, verifyPairingError := rpcps.stateTracker.VerifyPairing(ctx, consumerAddressString, rpcps.providerAddress.String(), uint64(request.Epoch), request.SpecId)
			if verifyPairingError != nil {
				return nil, utils.LavaFormatInfo("Failed to VerifyPairing for new consumer",
					utils.Attribute{Key: "Error", Value: verifyPairingError},
					utils.Attribute{Key: "GUID", Value: ctx},
					utils.Attribute{Key: "sessionID", Value: request.SessionId},
					utils.Attribute{Key: "consumer", Value: consumerAddressString},
					utils.Attribute{Key: "provider", Value: rpcps.providerAddress},
					utils.Attribute{Key: "relayNum", Value: request.RelayNum},
					utils.Attribute{Key: "Providers block", Value: rpcps.stateTracker.LatestBlock()},
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
			singleProviderSession, err = rpcps.providerSessionManager.RegisterProviderSessionWithConsumer(ctx, consumerAddressString, uint64(request.Epoch), request.SessionId, request.RelayNum, maxCuForConsumer, pairedProviders, projectId, request.Badge)
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

func (rpcps *RPCProviderServer) verifyRelayRequestMetaData(ctx context.Context, requestSession *pairingtypes.RelaySession, relayData *pairingtypes.RelayPrivateData) error {
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

	if !bytes.Equal(requestSession.ContentHash, sigs.HashMsg(relayData.GetContentHashData())) {
		return utils.LavaFormatError("content hash mismatch between consumer and provider", nil,
			utils.Attribute{Key: "ApiInterface", Value: relayData.ApiInterface},
			utils.Attribute{Key: "ApiUrl", Value: relayData.ApiUrl},
			utils.Attribute{Key: "RequestBlock", Value: relayData.RequestBlock},
			utils.Attribute{Key: "ConnectionType", Value: relayData.ConnectionType},
			utils.Attribute{Key: "Metadata", Value: relayData.Metadata},
			utils.Attribute{Key: "GUID", Value: ctx})
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
	errV := rpcps.ValidateRequest(chainMsg, request, ctx)
	if errV != nil {
		return nil, errV
	}

	errV = rpcps.ValidateAddonsExtensions(request.RelayData.Addon, request.RelayData.Extensions, chainMsg)
	if errV != nil {
		return nil, errV
	}
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
		specificBlock := request.RelayData.RequestBlock
		if specificBlock < spectypes.LATEST_BLOCK {
			// GetLatestBlockData only supports latest relative queries or specific block numbers
			specificBlock = spectypes.NOT_APPLICABLE
		}
		latestBlock, requestedHashes, err = rpcps.reliabilityManager.GetLatestBlockData(fromBlock, toBlock, specificBlock)
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

		chainMsg.UpdateLatestBlockInMessage(latestBlock, true)

		request.RelayData.RequestBlock = lavaprotocol.ReplaceRequestedBlock(request.RelayData.RequestBlock, latestBlock)
		requestedBlockHash, finalizedBlockHashes = chaintracker.FindRequestedBlockHash(requestedHashes, request.RelayData.RequestBlock, toBlock, fromBlock, finalizedBlockHashes)
		finalized = spectypes.IsFinalizedBlock(request.RelayData.RequestBlock, latestBlock, blockDistanceToFinalization)
		if !finalized && requestedBlockHash == nil && request.RelayData.RequestBlock != spectypes.NOT_APPLICABLE {
			// avoid using cache, but can still service
			reqHashesAttr := utils.Attribute{Key: "hashes", Value: requestedHashes}
			elementsToTake := 10
			if len(requestedHashes) > elementsToTake {
				reqHashesAttr = utils.Attribute{Key: "hashes", Value: requestedHashes[len(requestedHashes)-elementsToTake:]}
			}
			utils.LavaFormatWarning("no hash data for requested block", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "fromBlock", Value: fromBlock}, utils.Attribute{Key: "toBlock", Value: toBlock}, utils.Attribute{Key: "requestedBlock", Value: request.RelayData.RequestBlock}, utils.Attribute{Key: "latestBlock", Value: latestBlock}, reqHashesAttr)
		}

		// TODO: add a mechanism to handle this
		// if request.RelayData.RequestBlock > latestBlock {
		// 	// consumer asked for a block that is newer than our state tracker, we cant sign this for DR
		// 	return nil, utils.LavaFormatError("Requested a block that is too new", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "requestedBlock", Value: request.RelayData.RequestBlock}, utils.Attribute{Key: "latestBlock", Value: latestBlock})
		// }
	}
	cache := rpcps.cache
	// TODO: handle cache on fork for dataReliability = false
	var reply *pairingtypes.RelayReply = nil
	var err error = nil
	ignoredMetadata := []pairingtypes.Metadata{}
	if requestedBlockHash != nil || finalized {
		var cacheReply *pairingtypes.CacheRelayReply
		cacheReply, err = cache.GetEntry(ctx, request.RelayData, requestedBlockHash, rpcps.rpcProviderEndpoint.ChainID, finalized, rpcps.providerAddress.String())
		reply = cacheReply.GetReply()
		ignoredMetadata = cacheReply.GetOptionalMetadata()
		if err != nil && performance.NotConnectedError.Is(err) {
			utils.LavaFormatWarning("cache not connected", err, utils.Attribute{Key: "GUID", Value: ctx})
		}
	}
	if err != nil || reply == nil {
		// cache miss or invalid
		reply, _, _, err = rpcps.chainRouter.SendNodeMsg(ctx, nil, chainMsg, request.RelayData.Extensions)
		if err != nil {
			return nil, utils.LavaFormatError("Sending chainMsg failed", err, utils.Attribute{Key: "GUID", Value: ctx})
		}
		reply.Metadata, _, ignoredMetadata = rpcps.chainParser.HandleHeaders(reply.Metadata, chainMsg.GetApiCollection(), spectypes.Header_pass_reply)
		// TODO: use overwriteReqBlock on the reply metadata to set the correct latest block
		if requestedBlockHash != nil || finalized {
			// TODO: we do not add ignoredMetadata to the cache response
			err := cache.SetEntry(ctx, request.RelayData, requestedBlockHash, rpcps.rpcProviderEndpoint.ChainID, reply, finalized, rpcps.providerAddress.String(), ignoredMetadata)
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
	reply.Metadata = append(reply.Metadata, ignoredMetadata...) // appended here only after signing
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

func (rpcps *RPCProviderServer) Probe(ctx context.Context, probeReq *pairingtypes.ProbeRequest) (*pairingtypes.ProbeReply, error) {
	probeReply := &pairingtypes.ProbeReply{
		Guid:                  probeReq.GetGuid(),
		LatestBlock:           rpcps.reliabilityManager.GetLatestBlockNum(),
		FinalizedBlocksHashes: []byte{},
		LavaEpoch:             rpcps.providerSessionManager.GetCurrentEpoch(),
		LavaLatestBlock:       uint64(rpcps.stateTracker.LatestBlock()),
	}
	return probeReply, nil
}
