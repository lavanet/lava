package metrics

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
	"github.com/lavanet/lava/v4/protocol/parser"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/rand"
	"github.com/newrelic/go-agent/v3/newrelic"
	"google.golang.org/grpc/metadata"
)

var ReturnMaskedErrors = "false"

const (
	webSocketCloseMessage = "websocket: close "
	RefererHeaderKey      = "Referer"
	OriginHeaderKey       = "Origin"
	UserAgentHeaderKey    = "User-Agent"
)

type RPCConsumerLogs struct {
	newRelicApplication        *newrelic.Application
	MetricService              *MetricService
	StoreMetricData            bool
	excludeMetricsReferrers    string
	excludedUserAgent          []string
	consumerMetricsManager     *ConsumerMetricsManager
	consumerRelayServerClient  *ConsumerRelayServerClient
	consumerOptimizerQoSClient *ConsumerOptimizerQoSClient
}

func NewRPCConsumerLogs(consumerMetricsManager *ConsumerMetricsManager, consumerRelayServerClient *ConsumerRelayServerClient, consumerOptimizerQoSClient *ConsumerOptimizerQoSClient) (*RPCConsumerLogs, error) {
	err := godotenv.Load()
	if err != nil {
		utils.LavaFormatInfo("New relic missing environment file")
		return &RPCConsumerLogs{consumerMetricsManager: consumerMetricsManager, consumerRelayServerClient: consumerRelayServerClient}, nil // newRelicApplication is nil safe to use
	}

	newRelicAppName := os.Getenv("NEW_RELIC_APP_NAME")
	newRelicLicenseKey := os.Getenv("NEW_RELIC_LICENSE_KEY")
	if newRelicAppName == "" || newRelicLicenseKey == "" {
		utils.LavaFormatInfo("New relic missing environment variables")
		return &RPCConsumerLogs{
			consumerMetricsManager:     consumerMetricsManager,
			consumerRelayServerClient:  consumerRelayServerClient,
			consumerOptimizerQoSClient: consumerOptimizerQoSClient,
		}, nil
	}

	newRelicApplication, err := newrelic.NewApplication(
		newrelic.ConfigAppName(newRelicAppName),
		newrelic.ConfigLicense(newRelicLicenseKey),
		func(cfg *newrelic.Config) {
			// Set specific Config fields inside a custom ConfigOption.
			sMaxSamplesStored, ok := os.LookupEnv("NEW_RELIC_TRANSACTION_EVENTS_MAX_SAMPLES_STORED")
			if ok {
				utils.LavaFormatDebug("Setting NEW_RELIC_TRANSACTION_EVENTS_MAX_SAMPLES_STORED", utils.Attribute{Key: "sMaxSamplesStored", Value: sMaxSamplesStored})
				maxSamplesStored, err := strconv.Atoi(sMaxSamplesStored)
				if err != nil {
					utils.LavaFormatError("Failed converting sMaxSamplesStored to number", err, utils.Attribute{Key: "sMaxSamplesStored", Value: sMaxSamplesStored})
				} else {
					cfg.TransactionEvents.MaxSamplesStored = maxSamplesStored
				}
			} else {
				utils.LavaFormatDebug("Did not find NEW_RELIC_TRANSACTION_EVENTS_MAX_SAMPLES_STORED in env")
			}
		},
		newrelic.ConfigFromEnvironment(),
	)

	rpcConsumerLogs := &RPCConsumerLogs{newRelicApplication: newRelicApplication, StoreMetricData: false, consumerMetricsManager: consumerMetricsManager, consumerRelayServerClient: consumerRelayServerClient}
	isMetricEnabled, _ := strconv.ParseBool(os.Getenv("IS_METRICS_ENABLED"))
	if isMetricEnabled {
		rpcConsumerLogs.StoreMetricData = true
		rpcConsumerLogs.MetricService = NewMetricService()
		rpcConsumerLogs.excludeMetricsReferrers = os.Getenv("TO_EXCLUDE_METRICS_REFERRERS")
		agentsValue := os.Getenv("TO_EXCLUDE_METRICS_AGENTS")
		if len(agentsValue) > 0 {
			rpcConsumerLogs.excludedUserAgent = strings.Split(agentsValue, ";")
		}
	}
	return rpcConsumerLogs, err
}

func (rpccl *RPCConsumerLogs) SetLoLResponse(success bool) {
	if rpccl == nil {
		return
	}
	rpccl.consumerMetricsManager.SetLoLResponse(success)
}

func (rpccl *RPCConsumerLogs) SetWebSocketConnectionActive(chainId string, apiInterface string, add bool) {
	rpccl.consumerMetricsManager.SetWebSocketConnectionActive(chainId, apiInterface, add)
}

func (rpccl *RPCConsumerLogs) SetRelaySentToProviderMetric(providerAddress, chainId, apiInterface string) {
	rpccl.consumerMetricsManager.SetRelaySentToProviderMetric(chainId, apiInterface)
	rpccl.consumerOptimizerQoSClient.SetRelaySentToProvider(providerAddress, chainId)
}

func (rpccl *RPCConsumerLogs) SetRelayNodeErrorMetric(providerAddress, chainId, apiInterface string) {
	if providerAddress == "" {
		// skip if provider address is empty
		return
	}

	rpccl.consumerMetricsManager.SetRelayNodeErrorMetric(chainId, apiInterface)
	rpccl.consumerOptimizerQoSClient.SetNodeErrorToProvider(providerAddress, chainId)
}

func (rpccl *RPCConsumerLogs) SetNodeErrorRecoveredSuccessfullyMetric(chainId string, apiInterface string, attempt string) {
	rpccl.consumerMetricsManager.SetNodeErrorRecoveredSuccessfullyMetric(chainId, apiInterface, attempt)
}

func (rpccl *RPCConsumerLogs) SetNodeErrorAttemptMetric(chainId string, apiInterface string) {
	rpccl.consumerMetricsManager.SetNodeErrorAttemptMetric(chainId, apiInterface)
}

func (rpccl *RPCConsumerLogs) GetMessageSeed() string {
	return "GUID_" + strconv.Itoa(rand.Intn(10000000000))
}

// Input will be masked with a random GUID if returnMaskedErrors is set to true
func (rpccl *RPCConsumerLogs) GetUniqueGuidResponseForError(responseError error, msgSeed string) string {
	type ErrorData struct {
		Error_GUID string `json:"Error_GUID"`
		Error      string `json:"Error,omitempty"`
	}

	data := ErrorData{
		Error_GUID: msgSeed,
	}
	if ReturnMaskedErrors == "false" {
		data.Error = responseError.Error()
	}

	utils.LavaFormatError("UniqueGuidResponseForError", responseError, utils.Attribute{Key: "msgSeed", Value: msgSeed})

	ret, _ := json.Marshal(data)

	return string(ret)
}

// Websocket healthy disconnections throw "websocket: close 1005 (no status)" error,
// We don't want to alert error monitoring for that purpses.
func (rpccl *RPCConsumerLogs) AnalyzeWebSocketErrorAndGetFormattedMessage(webSocketAddr string, err error, msgSeed string, msg []byte, rpcType string, timeTaken time.Duration) []byte {
	if err != nil {
		errMessage := err.Error()
		if strings.Contains(errMessage, webSocketCloseMessage) {
			utils.LavaFormatDebug("Websocket connection closed by the user, " + errMessage)
			return nil
		}
		rpccl.LogRequestAndResponse(rpcType+" ws msg", true, "ws", webSocketAddr, string(msg), "", msgSeed, timeTaken, err)

		jsonResponse, err := json.Marshal(fiber.Map{
			"Error_Received": rpccl.GetUniqueGuidResponseForError(err, msgSeed),
		})
		if err != nil {
			utils.LavaFormatError("AnalyzeWebSocketErrorAndGetFormattedMessage unexpected behavior, failed marshalling json response", err, utils.LogAttr("seed", msgSeed))
		}
		return jsonResponse
	}
	return nil
}

func (rpccl *RPCConsumerLogs) LogRequestAndResponse(module string, hasError bool, method, path, req, resp, msgSeed string, timeTaken time.Duration, err error) {
	if hasError && err != nil {
		utils.LavaFormatError(module, err, []utils.Attribute{{Key: "GUID", Value: msgSeed}, {Key: "timeTaken", Value: timeTaken}, {Key: "request", Value: req}, {Key: "response", Value: parser.CapStringLen(resp)}, {Key: "method", Value: method}, {Key: "path", Value: path}, {Key: "HasError", Value: hasError}}...)
		return
	}
	utils.LavaFormatDebug(module, []utils.Attribute{{Key: "GUID", Value: msgSeed}, {Key: "timeTaken", Value: timeTaken}, {Key: "request", Value: req}, {Key: "response", Value: parser.CapStringLen(resp)}, {Key: "method", Value: method}, {Key: "path", Value: path}, {Key: "HasError", Value: hasError}}...)
}

func (rpccl *RPCConsumerLogs) LogStartTransaction(name string) func() {
	if rpccl.newRelicApplication == nil {
		return func() {
		}
	}

	tx := rpccl.newRelicApplication.StartTransaction(name)

	return func() {
		if tx != nil {
			tx.End()
		}
	}
}

// AddMetricForProcessingLatencyBeforeProvider adds a time calculation metric for the consumer's processing time before sending a relay to a provider
// it returns whether the latency was added or not
func (rpccl *RPCConsumerLogs) AddMetricForProcessingLatencyBeforeProvider(analytics *RelayMetrics, chainId string, apiInterface string) {
	if analytics != nil && analytics.ProcessingTimestamp.Before(time.Now()) {
		go rpccl.consumerMetricsManager.SetRelayProcessingLatencyBeforeProvider(time.Since(analytics.ProcessingTimestamp), chainId, apiInterface)
	}
}

func (rpccl *RPCConsumerLogs) AddMetricForProcessingLatencyAfterProvider(analytics *RelayMetrics, chainId string, apiInterface string) {
	if analytics != nil && analytics.MeasureAfterProviderProcessingTime && analytics.ProcessingTimestamp.Before(time.Now()) {
		go rpccl.consumerMetricsManager.SetRelayProcessingLatencyAfterProvider(time.Since(analytics.ProcessingTimestamp), chainId, apiInterface)
	}
}

func (rpccl *RPCConsumerLogs) AddMetricForHttp(data *RelayMetrics, err error, headers map[string][]string) {
	rpccl.consumerMetricsManager.SetRelayMetrics(data, err)
	rpccl.consumerRelayServerClient.SetRelayMetrics(data)
	refererHeaderValue := strings.Join(headers[RefererHeaderKey], ", ")
	userAgentHeaderValue := strings.Join(headers[UserAgentHeaderKey], ", ")
	if rpccl.StoreMetricData && rpccl.shouldCountMetrics(refererHeaderValue, userAgentHeaderValue) {
		originHeaderValue := headers[OriginHeaderKey]
		rpccl.SendMetrics(data, err, strings.Join(originHeaderValue, ", "))
	}
}

func (rpccl *RPCConsumerLogs) AddMetricForWebSocket(data *RelayMetrics, err error, c *websocket.Conn) {
	rpccl.consumerMetricsManager.SetRelayMetrics(data, err)
	rpccl.consumerRelayServerClient.SetRelayMetrics(data)
	refererHeaderValue, _ := c.Locals(RefererHeaderKey).(string)
	userAgentHeaderValue, _ := c.Locals(UserAgentHeaderKey).(string)
	if rpccl.StoreMetricData && rpccl.shouldCountMetrics(refererHeaderValue, userAgentHeaderValue) {
		originHeaderValue, _ := c.Locals(OriginHeaderKey).(string)
		rpccl.SendMetrics(data, err, originHeaderValue)
	}
}

func (rpccl *RPCConsumerLogs) AddMetricForGrpc(data *RelayMetrics, err error, metadataValues *metadata.MD) {
	getMetadataHeaderOrDefault := func(headerKey string) string {
		headerValues := metadataValues.Get(headerKey)
		headerValue := ""
		if len(headerValues) > 0 {
			headerValue = headerValues[0]
		}
		return headerValue
	}
	rpccl.consumerMetricsManager.SetRelayMetrics(data, err)
	rpccl.consumerRelayServerClient.SetRelayMetrics(data)
	refererHeaderValue := getMetadataHeaderOrDefault(RefererHeaderKey)
	userAgentHeaderValue := getMetadataHeaderOrDefault(UserAgentHeaderKey)
	if rpccl.StoreMetricData && rpccl.shouldCountMetrics(refererHeaderValue, userAgentHeaderValue) {
		originHeaderValue := getMetadataHeaderOrDefault(OriginHeaderKey)
		rpccl.SendMetrics(data, err, originHeaderValue)
	}
}

func (rpccl *RPCConsumerLogs) shouldCountMetrics(refererHeaderValue string, userAgentHeaderValue string) bool {
	if len(rpccl.excludeMetricsReferrers) > 0 && len(refererHeaderValue) > 0 {
		if strings.Contains(refererHeaderValue, rpccl.excludeMetricsReferrers) {
			return false
		}
	}

	if len(userAgentHeaderValue) > 0 {
		for _, excludedAgent := range rpccl.excludedUserAgent {
			if strings.Contains(userAgentHeaderValue, excludedAgent) {
				return false
			}
		}
	}
	return true
}

func (rpccl *RPCConsumerLogs) SetRelaySentByNewBatchTickerMetric(chainId string, apiInterface string) {
	rpccl.consumerMetricsManager.SetRelaySentByNewBatchTickerMetric(chainId, apiInterface)
}

func (rpccl *RPCConsumerLogs) SendMetrics(data *RelayMetrics, err error, origin string) {
	data.Success = err == nil
	data.Origin = origin
	rpccl.MetricService.SendData(*data)
}

func (rpccl *RPCConsumerLogs) LogTestMode(fiberCtx *fiber.Ctx) {
	headers := fiberCtx.GetReqHeaders()
	st := "Test Mode Log: new request\n"
	st += "Full URI: " + fiberCtx.Request().URI().String() + "\n"
	for header, HeaderVal := range headers {
		st += fmt.Sprintf("Header %16s HeaderVal: %s\n", header, HeaderVal)
	}
	utils.LavaFormatInfo(st)
}
