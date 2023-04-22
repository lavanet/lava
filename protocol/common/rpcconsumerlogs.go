package common

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/protocol/parser"
	"github.com/lavanet/lava/utils"
	"github.com/newrelic/go-agent/v3/newrelic"
	"google.golang.org/grpc/metadata"
)

var ReturnMaskedErrors = "false"

const (
	webSocketCloseMessage = "websocket: close 1005 (no status)"
	RefererHeaderKey      = "Referer"
	OriginHeaderKey       = "Origin"
	UserAgentHeaderKey    = "User-Agent"
)

type RPCConsumerLogs struct {
	newRelicApplication     *newrelic.Application
	MetricService           *metrics.MetricService
	StoreMetricData         bool
	excludeMetricsReferrers string
}

func NewRPCConsumerLogs() (*RPCConsumerLogs, error) {
	err := godotenv.Load()
	if err != nil {
		utils.LavaFormatInfo("New relic missing environment file")
		return &RPCConsumerLogs{}, nil
	}

	newRelicAppName := os.Getenv("NEW_RELIC_APP_NAME")
	newRelicLicenseKey := os.Getenv("NEW_RELIC_LICENSE_KEY")
	if newRelicAppName == "" || newRelicLicenseKey == "" {
		utils.LavaFormatInfo("New relic missing environment variables")
		return &RPCConsumerLogs{}, nil
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

	portal := &RPCConsumerLogs{newRelicApplication: newRelicApplication, StoreMetricData: false}
	isMetricEnabled, _ := strconv.ParseBool(os.Getenv("IS_METRICS_ENABLED"))
	if isMetricEnabled {
		portal.StoreMetricData = true
		portal.MetricService = metrics.NewMetricService()
		portal.excludeMetricsReferrers = os.Getenv("TO_EXCLUDE_METRICS_REFERRERS")
	}
	return portal, err
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
// We dont want to alert error monitoring for that purpses.
func (rpccl *RPCConsumerLogs) AnalyzeWebSocketErrorAndWriteMessage(c *websocket.Conn, mt int, err error, msgSeed string, msg []byte, rpcType string) {
	if err != nil {
		if err.Error() == webSocketCloseMessage {
			utils.LavaFormatInfo("Websocket connection closed by the user, " + err.Error())
			return
		}
		rpccl.LogRequestAndResponse(rpcType+" ws msg", true, "ws", c.LocalAddr().String(), string(msg), "", msgSeed, err)

		jsonResponse, _ := json.Marshal(fiber.Map{
			"Error_Received": rpccl.GetUniqueGuidResponseForError(err, msgSeed),
		})

		c.WriteMessage(mt, jsonResponse)
	}
}

func (rpccl *RPCConsumerLogs) LogRequestAndResponse(module string, hasError bool, method string, path string, req string, resp string, msgSeed string, err error) {
	if hasError && err != nil {
		utils.LavaFormatError(module, err, []utils.Attribute{{Key: "GUID", Value: msgSeed}, {Key: "request", Value: req}, {Key: "response", Value: parser.CapStringLen(resp)}, {Key: "method", Value: method}, {Key: "path", Value: path}, {Key: "HasError", Value: hasError}}...)
		return
	}
	utils.LavaFormatDebug(module, []utils.Attribute{{Key: "GUID", Value: msgSeed}, {Key: "request", Value: req}, {Key: "response", Value: parser.CapStringLen(resp)}, {Key: "method", Value: method}, {Key: "path", Value: path}, {Key: "HasError", Value: hasError}}...)
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

func (rpccl *RPCConsumerLogs) AddMetricForHttp(data *metrics.RelayMetrics, err error, headers map[string]string) {
	if rpccl.StoreMetricData && rpccl.shouldCountMetricForHttp(headers) {
		originHeaderValue := headers[OriginHeaderKey]
		rpccl.SendMetrics(data, err, originHeaderValue)
	}
}

func (rpccl *RPCConsumerLogs) AddMetricForWebSocket(data *metrics.RelayMetrics, err error, c *websocket.Conn) {
	if rpccl.StoreMetricData && rpccl.shouldCountMetricForWebSocket(c) {
		originValue, _ := c.Locals(OriginHeaderKey).(string)
		rpccl.SendMetrics(data, err, originValue)
	}
}

func (rpccl *RPCConsumerLogs) AddMetricForGrpc(data *metrics.RelayMetrics, err error, metadataValues *metadata.MD) {
	if rpccl.StoreMetricData && rpccl.shouldCountMetricForGrpc(metadataValues) {
		originValue := GetMetadataHeaderOrDefault(metadataValues, OriginHeaderKey)
		rpccl.SendMetrics(data, err, originValue)
	}
}

func (rpccl *RPCConsumerLogs) shouldCountMetricForHttp(headers map[string]string) bool {
	refererHeaderValue := headers[RefererHeaderKey]
	userAgentHeaderValue := headers[UserAgentHeaderKey]
	return rpccl.shouldCountMetrics(refererHeaderValue, userAgentHeaderValue)
}

// TODO add these when the initial call is made (Origin,UserAgent)
func (rpccl *RPCConsumerLogs) shouldCountMetricForWebSocket(c *websocket.Conn) bool {
	refererHeaderValue, _ := c.Locals(RefererHeaderKey).(string)
	userAgentHeaderValue, _ := c.Locals(UserAgentHeaderKey).(string)
	return rpccl.shouldCountMetrics(refererHeaderValue, userAgentHeaderValue)
}

func (rpccl *RPCConsumerLogs) shouldCountMetricForGrpc(metadataValues *metadata.MD) bool {
	if metadataValues != nil {
		refererHeaderValue := GetMetadataHeaderOrDefault(metadataValues, RefererHeaderKey)
		agentHeaderValue := GetMetadataHeaderOrDefault(metadataValues, UserAgentHeaderKey)
		return rpccl.shouldCountMetrics(refererHeaderValue, agentHeaderValue)
	}
	return true
}

func (rpccl *RPCConsumerLogs) shouldCountMetrics(refererHeaderValue string, userAgentHeaderValue string) bool {
	if len(rpccl.excludeMetricsReferrers) > 0 && len(refererHeaderValue) > 0 {
		return !strings.Contains(refererHeaderValue, rpccl.excludeMetricsReferrers)
	}

	if len(userAgentHeaderValue) == 0 {
		return false
	}
	excludedUserAgents := []string{"curl", "PostmanRuntime"}
	for _, excludedAgent := range excludedUserAgents {
		if strings.Contains(userAgentHeaderValue, excludedAgent) {
			return false
		}
	}
	return true
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

func (rpccl *RPCConsumerLogs) SendMetrics(data *metrics.RelayMetrics, err error, origin string) {
	data.Success = err == nil
	data.Origin = origin
	rpccl.MetricService.SendData(*data)
}

func GetMetadataHeaderOrDefault(mt *metadata.MD, headerKey string) string {
	headerValues := mt.Get(headerKey)
	headerValue := ""
	if len(headerValues) > 0 {
		headerValue = headerValues[0]
	}
	return headerValue
}
