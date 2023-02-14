package chainproxy

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc/metadata"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/lavanet/lava/relayer/metrics"

	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/utils"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var ReturnMaskedErrors = "false"

const webSocketCloseMessage = "websocket: close 1005 (no status)"
const refererHeaderKey = "Referer"

type PortalLogs struct {
	newRelicApplication       *newrelic.Application
	metricService             *metrics.MetricService
	storeMetricData           bool
	toExcludeMetricsReferrers string
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
	portal := &PortalLogs{newRelicApplication: newRelicApplication, storeMetricData: false}
	isMetricEnabled, _ := strconv.ParseBool(os.Getenv("IS_METRICS_ENABLED"))
	if isMetricEnabled {
		portal.storeMetricData = true
		portal.metricService = metrics.NewMetricService()
		portal.toExcludeMetricsReferrers = os.Getenv("TO_EXCLUDE_METRICS_REFERRERS")
	}
	return portal, err
}

func (pl *PortalLogs) GetMessageSeed() string {
	return "GUID_" + strconv.Itoa(rand.Intn(10000000000))
}

// Input will be masked with a random GUID if returnMaskedErrors is set to true
func (pl *PortalLogs) GetUniqueGuidResponseForError(responseError error, msgSeed string) string {
	var ret string
	ret = "Error GUID: " + msgSeed
	utils.LavaFormatError("UniqueGuidResponseForError", responseError, &map[string]string{"msgSeed": msgSeed})
	if ReturnMaskedErrors == "false" {
		ret += fmt.Sprintf(", Error: %v", responseError)
	}

	return ret
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
		c.WriteMessage(mt, []byte("Error Received: "+pl.GetUniqueGuidResponseForError(err, msgSeed)))
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

func (pl *PortalLogs) AddMetric(data *metrics.RelayMetrics, isSuccessful bool) {
	if pl.storeMetricData {
		data.Success = isSuccessful
		pl.metricService.SendData(*data)
	}
}

func (pl *PortalLogs) ShouldCountMetric(c *fiber.Ctx) bool {
	refererHeaderValue := c.Get(refererHeaderKey, "")
	if refererHeaderValue == "" {
		refererHeaderValue = c.Get(strings.ToLower(refererHeaderKey), "")
	}
	if len(pl.toExcludeMetricsReferrers) > 0 && refererHeaderKey != "" {
		return !strings.Contains(refererHeaderValue, pl.toExcludeMetricsReferrers)
	}
	return true
}

func (pl *PortalLogs) ShouldCountMetricForWebSocket(c *websocket.Conn) bool {
	refererHeaderValue := c.Locals(refererHeaderKey).(string)
	if len(pl.toExcludeMetricsReferrers) > 0 && len(refererHeaderValue) > 0 {
		return !strings.Contains(refererHeaderValue, pl.toExcludeMetricsReferrers)
	}
	return true
}

func (pl *PortalLogs) ShouldCountMetricForGrpc(ctx context.Context) bool {
	headersValues, ok := metadata.FromIncomingContext(ctx)
	if ok && len(pl.toExcludeMetricsReferrers) > 0 {
		refererHeaderValue := headersValues.Get(refererHeaderKey)
		result := len(refererHeaderValue) > 0 && strings.Contains(refererHeaderValue[0], pl.toExcludeMetricsReferrers)
		return !result
	}
	return true
}
