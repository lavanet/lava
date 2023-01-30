package chainlib

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/relayer/chainproxy/thirdparty"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/metrics"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type GrpcChainParser struct {
	spec       spectypes.Spec
	rwLock     sync.RWMutex
	serverApis map[string]spectypes.ServiceApi
	taggedApis map[string]spectypes.ServiceApi
}

// NewGrpcChainParser creates a new instance of GrpcChainParser
func NewGrpcChainParser() (chainParser *GrpcChainParser, err error) {
	return &GrpcChainParser{}, nil
}

// ParseMsg parses message data into chain message object
func (apip *GrpcChainParser) ParseMsg(url string, data []byte, connectionType string) (ChainMessage, error) {
	// Check API is supported and save it in nodeMsg.
	serviceApi, err := apip.getSupportedApi(url)
	if err != nil {
		return nil, utils.LavaFormatError("failed to getSupportedApi gRPC", err, nil)
	}

	var apiInterface *spectypes.ApiInterface = nil
	for i := range serviceApi.ApiInterfaces {
		if serviceApi.ApiInterfaces[i].Type == connectionType {
			apiInterface = &serviceApi.ApiInterfaces[i]
			break
		}
	}
	if apiInterface == nil {
		return nil, fmt.Errorf("could not find the interface %s in the service %s", connectionType, serviceApi.Name)
	}

	// TODO why we don't have requested block here?
	nodeMsg := &parsedMessage{
		serviceApi:     serviceApi,
		apiInterface:   apiInterface,
		requestedBlock: -2,
	}
	return nodeMsg, nil
}

// getSupportedApi fetches service api from spec by name
func (apip *GrpcChainParser) getSupportedApi(name string) (*spectypes.ServiceApi, error) {
	// Guard that the JsonRPCChainParser instance exists
	if apip == nil {
		return nil, errors.New("RestChainParser not defined")
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Fetch server api by name
	api, ok := matchSpecApiByName(name, apip.serverApis)

	// Return an error if spec does not exist
	if !ok {
		return nil, errors.New("JRPC api not supported")
	}

	// Return an error if api is disabled
	if !api.Enabled {
		return nil, errors.New("api is disabled")
	}

	return &api, nil
}

// SetSpec sets the spec for the GrpcChainParser
func (apip *GrpcChainParser) SetSpec(spec spectypes.Spec) {
	// Guard that the JsonRPCChainParser instance exists
	if apip == nil {
		return
	}

	// Add a read-write lock to ensure thread safety
	apip.rwLock.Lock()
	defer apip.rwLock.Unlock()

	// extract server and tagged apis from spec
	serverApis, taggedApis := getServiceApis(spec, grpcInterface)

	// Set the spec field of the JsonRPCChainParser object
	apip.spec = spec
	apip.serverApis = serverApis
	apip.taggedApis = taggedApis
}

// DataReliabilityParams returns data reliability params from spec (spec.enabled and spec.dataReliabilityThreshold)
func (apip *GrpcChainParser) DataReliabilityParams() (enabled bool, dataReliabilityThreshold uint32) {
	// Guard that the GrpcChainParser instance exists
	if apip == nil {
		return false, 0
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Return enabled and data reliability threshold from spec
	return apip.spec.Enabled, apip.spec.GetReliabilityThreshold()
}

// ChainBlockStats returns block stats from spec
// (spec.AllowedBlockLagForQosSync, spec.AverageBlockTime, spec.BlockDistanceForFinalizedData)
func (apip *GrpcChainParser) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData uint32) {
	// Guard that the GrpcChainParser instance exists
	if apip == nil {
		return 0, 0, 0
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Convert average block time from int64 -> time.Duration
	averageBlockTime = time.Duration(apip.spec.AverageBlockTime) * time.Second

	// Return allowedBlockLagForQosSync, averageBlockTime, blockDistanceForFinalizedData from spec
	return apip.spec.AllowedBlockLagForQosSync, averageBlockTime, apip.spec.BlockDistanceForFinalizedData
}

type GrpcChainListener struct {
	endpoint    *lavasession.RPCEndpoint
	relaySender RelaySender
	logger      *lavaprotocol.RPCConsumerLogs
}

func NewGrpcChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender, rpcConsumerLogs *lavaprotocol.RPCConsumerLogs) (chainListener *GrpcChainListener) {
	// Create a new instance of GrpcChainListener
	chainListener = &GrpcChainListener{
		listenEndpoint,
		relaySender,
		rpcConsumerLogs,
	}

	return chainListener
}

func (apil *GrpcChainListener) Serve(ctx context.Context) {
	utils.LavaFormatInfo("gRPC PortalStart", nil)

	lis, err := net.Listen("tcp", apil.endpoint.NetworkAddress)
	if err != nil {
		utils.LavaFormatFatal("provider failure setting up listener", err, &map[string]string{"listenAddr": apil.endpoint.NetworkAddress})
	}
	apiInterface := apil.endpoint.ApiInterface
	sendRelayCallback := func(ctx context.Context, method string, reqBody []byte) ([]byte, error) {
		msgSeed := apil.logger.GetMessageSeed()
		utils.LavaFormatInfo("GRPC Got Relay: "+method, nil)
		var relayReply *pairingtypes.RelayReply
		metricsData := metrics.NewRelayAnalytics("NoDappID", apil.endpoint.ChainID, apiInterface)
		if relayReply, _, err = apil.relaySender.SendRelay(ctx, method, string(reqBody), "", "NoDappID", metricsData); err != nil {
			go apil.logger.AddMetric(metricsData, err != nil)
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)
			apil.logger.LogRequestAndResponse("http in/out", true, method, string(reqBody), "", errMasking, msgSeed, err)
			return nil, utils.LavaFormatError("Failed to SendRelay", fmt.Errorf(errMasking), nil)
		}
		apil.logger.LogRequestAndResponse("http in/out", false, method, string(reqBody), "", "", msgSeed, nil)
		return relayReply.Data, nil
	}

	_, httpServer, err := thirdparty.RegisterServer(apil.endpoint.ChainID, sendRelayCallback)
	if err != nil {
		utils.LavaFormatFatal("provider failure RegisterServer", err, &map[string]string{"listenAddr": apil.endpoint.NetworkAddress})
	}

	utils.LavaFormatInfo("Server listening", &map[string]string{"Address": lis.Addr().String()})

	if err := httpServer.Serve(lis); !errors.Is(err, http.ErrServerClosed) {
		utils.LavaFormatFatal("Portal failed to serve", err, &map[string]string{"Address": lis.Addr().String(), "ChainID": apil.endpoint.ChainID})
	}
}
