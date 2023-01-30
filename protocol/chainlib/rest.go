package chainlib

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/metrics"
	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type RestChainParser struct {
	spec       spectypes.Spec
	rwLock     sync.RWMutex
	serverApis map[string]spectypes.ServiceApi
	taggedApis map[string]spectypes.ServiceApi
}

// NewRestChainParser creates a new instance of RestChainParser
func NewRestChainParser() (chainParser *RestChainParser, err error) {
	return &RestChainParser{}, nil
}

func (apip *RestChainParser) ParseMsg(url string, data []byte, connectionType string) (ChainMessage, error) {
	// Check api is supported and save it in nodeMsg
	serviceApi, err := apip.getSupportedApi(url)
	if err != nil {
		return nil, err
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
func (apip *RestChainParser) getSupportedApi(name string) (*spectypes.ServiceApi, error) {
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
		return nil, errors.New("rest api not supported")
	}

	// Return an error if api is disabled
	if !api.Enabled {
		return nil, errors.New("api is disabled")
	}

	return &api, nil
}

// SetSpec sets the spec for the RestChainParser
func (apip *RestChainParser) SetSpec(spec spectypes.Spec) {
	// Guard that the RestChainParser instance exists
	if apip == nil {
		return
	}

	// Add a read-write lock to ensure thread safety
	apip.rwLock.Lock()
	defer apip.rwLock.Unlock()

	// extract server and tagged apis from spec
	serverApis, taggedApis := getServiceApis(spec, restInterface)

	// Set the spec field of the RestChainParser object
	apip.spec = spec
	apip.serverApis = serverApis
	apip.taggedApis = taggedApis
}

// DataReliabilityParams returns data reliability params from spec (spec.enabled and spec.dataReliabilityThreshold)
func (apip *RestChainParser) DataReliabilityParams() (enabled bool, dataReliabilityThreshold uint32) {
	// Guard that the RestChainParser instance exists
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
func (apip *RestChainParser) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData uint32) {
	// Guard that the JsonRPCChainParser instance exists
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

type RestChainListener struct {
	endpoint    *lavasession.RPCEndpoint
	relaySender RelaySender
	logger      *lavaprotocol.RPCConsumerLogs
}

// NewRestChainListener creates a new instance of RestChainListener
func NewRestChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender, rpcConsumerLogs *lavaprotocol.RPCConsumerLogs) (chainListener *RestChainListener) {
	// Create a new instance of JsonRPCChainListener
	chainListener = &RestChainListener{
		listenEndpoint,
		relaySender,
		rpcConsumerLogs,
	}

	return chainListener
}

func (apil *RestChainListener) Serve(ctx context.Context) {
	// Setup HTTP Server
	app := fiber.New(fiber.Config{})

	app.Use(favicon.New())

	chainID := apil.endpoint.ChainID
	apiInterface := apil.endpoint.ApiInterface
	// Catch Post
	app.Post("/:dappId/*", func(c *fiber.Ctx) error {
		apil.logger.LogStartTransaction("rest-http")

		msgSeed := apil.logger.GetMessageSeed()

		path := "/" + c.Params("*")

		// TODO: handle contentType, in case its not application/json currently we set it to application/json in the Send() method
		// contentType := string(c.Context().Request.Header.ContentType())
		dappID := extractDappIDFromFiberContext(c)
		metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		utils.LavaFormatInfo("in <<<", &map[string]string{"path": path, "dappID": dappID, "msgSeed": msgSeed})
		requestBody := string(c.Body())
		reply, _, err := apil.relaySender.SendRelay(ctx, path, requestBody, http.MethodPost, dappID, metricsData)
		go apil.logger.AddMetric(metricsData, err != nil)
		if err != nil {
			// Get unique GUID response
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			apil.logger.LogRequestAndResponse("http in/out", true, http.MethodPost, path, requestBody, errMasking, msgSeed, err)

			// Set status to internal error
			c.Status(fiber.StatusInternalServerError)

			// Construct json response
			response := convertToJsonError(errMasking)

			// Return error json response
			return c.SendString(response)
		}
		// Log request and response
		apil.logger.LogRequestAndResponse("http in/out", false, http.MethodPost, path, requestBody, string(reply.Data), msgSeed, nil)

		// Return json response
		return c.SendString(string(reply.Data))
	})

	// Catch the others
	app.Use("/:dappId/*", func(c *fiber.Ctx) error {
		apil.logger.LogStartTransaction("rest-http")
		msgSeed := apil.logger.GetMessageSeed()

		query := "?" + string(c.Request().URI().QueryString())
		path := "/" + c.Params("*")
		dappID := ""
		if len(c.Route().Params) > 1 {
			dappID = c.Route().Params[1]
			dappID = strings.ReplaceAll(dappID, "*", "")
		}
		utils.LavaFormatInfo("in <<<", &map[string]string{"path": path, "dappID": dappID, "msgSeed": msgSeed})
		analytics := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)

		reply, _, err := apil.relaySender.SendRelay(ctx, path, query, http.MethodGet, dappID, analytics)
		go apil.logger.AddMetric(analytics, err != nil)
		if err != nil {
			// Get unique GUID response
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			apil.logger.LogRequestAndResponse("http in/out", true, http.MethodGet, path, "", errMasking, msgSeed, err)

			// Set status to internal error
			c.Status(fiber.StatusInternalServerError)

			// Construct json response
			response := convertToJsonError(errMasking)

			// Return error json response
			return c.SendString(response)
		}
		// Log request and response
		apil.logger.LogRequestAndResponse("http in/out", false, http.MethodGet, path, "", string(reply.Data), msgSeed, nil)

		// Return json response
		return c.SendString(string(reply.Data))
	})

	// Go
	err := app.Listen(apil.endpoint.NetworkAddress)
	if err != nil {
		utils.LavaFormatError("app.Listen(listenAddr)", err, nil)
	}
}
