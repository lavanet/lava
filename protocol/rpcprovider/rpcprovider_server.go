package rpcprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/relayer/performance"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
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
	SendNewProof(ctx context.Context, singleProviderSession *lavasession.SingleProviderSession, epoch uint64, consumerAddr string)
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
	// relaySession, consumerAddress, err := rpcps.initRelay(ctx, request)
	// if err != nil {
	// 	return nil, rpcps.handleRelayErrorStatus(err)
	// }

	// reply, err := rpcps.TryRelay(ctx, request, userAddr, nodeMsg)
	// if err != nil && request.DataReliability == nil { // we ignore data reliability because its not checking/adding cu/relaynum.
	// 	// failed to send relay. we need to adjust session state. cuSum and relayNumber.
	// 	relayFailureError := s.onRelayFailure(userSessions, relaySession, nodeMsg)
	// 	if relayFailureError != nil {
	// 		err = sdkerrors.Wrapf(relayFailureError, "On relay failure: "+err.Error())
	// 	}
	// 	utils.LavaFormatError("TryRelay Failed", err, &map[string]string{
	// 		"request.SessionId": strconv.FormatUint(request.SessionId, 10),
	// 		"request.userAddr":  userAddr.String(),
	// 	})
	// } else {
	// 	utils.LavaFormatDebug("Provider Finished Relay Successfully", &map[string]string{
	// 		"request.SessionId":   strconv.FormatUint(request.SessionId, 10),
	// 		"request.relayNumber": strconv.FormatUint(request.RelayNum, 10),
	// 	})
	// }
	// return reply, s.handleRelayErrorStatus(err)
	return nil, fmt.Errorf("not implemented")
}
func (rpcps *RPCProviderServer) RelaySubscribe(request *pairingtypes.RelayRequest, srv pairingtypes.Relayer_RelaySubscribeServer) error {
	return fmt.Errorf("not implemented")
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

	// Checks
	err = rpcps.verifyRelayRequestMetaData(request)
	if err != nil {
		return nil, nil, utils.LavaFormatError("did not pass relay validation", err, nil)
	}
	consumerBytes, err := lavaprotocol.ExtractSignerAddress(request)
	if err != nil {
		return nil, nil, utils.LavaFormatError("extract signer address from relay", err, nil)
	}
	extractedConsumerAddress, err = sdk.AccAddressFromHex(consumerBytes.String())
	if err != nil {
		return nil, nil, utils.LavaFormatError("get relay consumer address", err, nil)
	}

	if request.DataReliability == nil {
		//regular session
		singleProviderSession, err = rpcps.providerSessionManager.GetSession(extractedConsumerAddress.String(), uint64(request.BlockHeight), request.RelayNum, request.SessionId)
		if err != nil {
			return nil, nil, utils.LavaFormatError("failed to get a provider session", err, &map[string]string{"sessionID": strconv.FormatUint(request.SessionId, 10), "consumer": extractedConsumerAddress.String(), "relayNum": strconv.FormatUint(request.RelayNum, 10)})
		}
		return singleProviderSession, extractedConsumerAddress, nil
	}

	// handle data reliability session verifications
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
