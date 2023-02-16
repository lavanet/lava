package chainproxy

import (
	"encoding/json"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"google.golang.org/grpc/metadata"

	"github.com/lavanet/lava/relayer/metrics"

	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/utils"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var ReturnMaskedErrors = "false"

const (
	webSocketCloseMessage = "websocket: close 1005 (no status)"
	RefererHeaderKey      = "Referer"
)

type PortalLogs struct {
	newRelicApplication     *newrelic.Application
	MetricService           *metrics.MetricService
	StoreMetricData         bool
	excludeMetricsReferrers string
}

func NewPortalLogs() (*PortalLogs, error) {
	err := godotenv.Load()
	if err != nil {
		utils.LavaFormatInfo("New relic missing environment file", nil)
		return &PortalLogs{}, nil
	}

	newRelicAppName := os.Getenv("NEW_RELIC_APP_NAME")
	newRelicLicenseKey := os.Getenv("NEW_RELIC_LICENSE_KEY")
	if newRelicAppName == "" || newRelicLicenseKey == "" {
		utils.LavaFormatInfo("New relic missing environment variables", nil)
		return &PortalLogs{}, nil
	}
	newRelicApplication, err := newrelic.NewApplication(
		newrelic.ConfigAppName(newRelicAppName),
		newrelic.ConfigLicense(newRelicLicenseKey),
		newrelic.ConfigFromEnvironment(),
	)
	portal := &PortalLogs{newRelicApplication: newRelicApplication, StoreMetricData: false}
	isMetricEnabled, _ := strconv.ParseBool(os.Getenv("IS_METRICS_ENABLED"))
	if isMetricEnabled {
		portal.StoreMetricData = true
		portal.MetricService = metrics.NewMetricService()
	}
	return portal, err
}

func (pl *PortalLogs) GetMessageSeed() string {
	return "GUID_" + strconv.Itoa(rand.Intn(10000000000))
}

// Input will be masked with a random GUID if returnMaskedErrors is set to true
func (pl *PortalLogs) GetUniqueGuidResponseForError(responseError error, msgSeed string) string {
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

	utils.LavaFormatError("UniqueGuidResponseForError", responseError, &map[string]string{"msgSeed": msgSeed})

	ret, _ := json.Marshal(data)

	return string(ret)
}

// Websocket healthy disconnections throw "websocket: close 1005 (no status)" error,
// We dont want to alert error monitoring for that purpses.
func (pl *PortalLogs) AnalyzeWebSocketErrorAndWriteMessage(c *websocket.Conn, mt int, err error, msgSeed string, msg []byte, rpcType string) {
	if err != nil {
		if err.Error() == webSocketCloseMessage {
			utils.LavaFormatInfo("Websocket connection closed by the user, "+err.Error(), nil)
			return
		}
		pl.LogRequestAndResponse(rpcType+" ws msg", true, "ws", c.LocalAddr().String(), string(msg), "", msgSeed, err)

		type ErrorResponse struct {
			ErrorReceived string `json:"Error_Received"`
		}

		jsonResponse, _ := json.Marshal(ErrorResponse{
			ErrorReceived: pl.GetUniqueGuidResponseForError(err, msgSeed),
		})

		c.WriteMessage(mt, jsonResponse)
	}
}

func (pl *PortalLogs) LogRequestAndResponse(module string, hasError bool, method string, path string, req string, resp string, msgSeed string, err error) {
	if hasError && err != nil {
		utils.LavaFormatError(module, err, &map[string]string{"GUID": msgSeed, "request": req, "response": parser.CapStringLen(resp), "method": method, "path": path, "HasError": strconv.FormatBool(hasError)})
		return
	}
	utils.LavaFormatDebug(module, &map[string]string{"GUID": msgSeed, "request": req, "response": parser.CapStringLen(resp), "method": method, "path": path, "HasError": strconv.FormatBool(hasError)})
}

func (pl *PortalLogs) LogStartTransaction(name string) {
	if pl.newRelicApplication != nil {
		txn := pl.newRelicApplication.StartTransaction(name)
		defer txn.End()
	}
}

func (pl *PortalLogs) AddMetricForHttp(data *metrics.RelayMetrics, err error, headers map[string]string) {
	if pl.StoreMetricData && pl.shouldCountMetricForHttp(headers) {
		data.Success = err == nil
		pl.MetricService.SendData(*data)
	}
}

func (pl *PortalLogs) AddMetricForWebSocket(data *metrics.RelayMetrics, err error, c *websocket.Conn) {
	if pl.StoreMetricData && pl.shouldCountMetricForWebSocket(c) {
		data.Success = err == nil
		pl.MetricService.SendData(*data)
	}
}

func (pl *PortalLogs) AddMetricForGrpc(data *metrics.RelayMetrics, err error, metadataValues *metadata.MD) {
	if pl.StoreMetricData && pl.shouldCountMetricForGrpc(metadataValues) {
		data.Success = err == nil
		pl.MetricService.SendData(*data)
	}
}

func (pl *PortalLogs) shouldCountMetricForHttp(headers map[string]string) bool {
	refererHeaderValue := headers[RefererHeaderKey]
	return pl.shouldCountMetrics(refererHeaderValue)
}

func (pl *PortalLogs) shouldCountMetricForWebSocket(c *websocket.Conn) bool {
	refererHeaderValue, isHeaderFound := c.Locals(RefererHeaderKey).(string)
	if !isHeaderFound {
		return true
	}
	return pl.shouldCountMetrics(refererHeaderValue)
}

func (pl *PortalLogs) shouldCountMetricForGrpc(metadataValues *metadata.MD) bool {
	if metadataValues != nil {
		refererHeaderValue := metadataValues.Get(RefererHeaderKey)
		result := len(refererHeaderValue) > 0 && pl.shouldCountMetrics(refererHeaderValue[0])
		return !result
	}
	return true
}

func (pl *PortalLogs) shouldCountMetrics(refererHeaderValue string) bool {
	if len(pl.excludeMetricsReferrers) > 0 && len(refererHeaderValue) > 0 {
		return !strings.Contains(refererHeaderValue, pl.excludeMetricsReferrers)
	}
	return true
}
