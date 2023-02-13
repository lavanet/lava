package common

import (
	"encoding/json"
	"math/rand"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
	"github.com/lavanet/lava/relayer/metrics"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/utils"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var ReturnMaskedErrors = "false"

const webSocketCloseMessage = "websocket: close 1005 (no status)"

type RPCConsumerLogs struct {
	newRelicApplication *newrelic.Application
	MetricService       *metrics.MetricService
	StoreMetricData     bool
}

func NewRPCConsumerLogs() (*RPCConsumerLogs, error) {
	err := godotenv.Load()
	if err != nil {
		utils.LavaFormatInfo("New relic missing environment file", nil)
		return &RPCConsumerLogs{}, nil
	}

	newRelicAppName := os.Getenv("NEW_RELIC_APP_NAME")
	newRelicLicenseKey := os.Getenv("NEW_RELIC_LICENSE_KEY")
	if newRelicAppName == "" || newRelicLicenseKey == "" {
		utils.LavaFormatInfo("New relic missing environment variables", nil)
		return &RPCConsumerLogs{}, nil
	}
	newRelicApplication, err := newrelic.NewApplication(
		newrelic.ConfigAppName(newRelicAppName),
		newrelic.ConfigLicense(newRelicLicenseKey),
		newrelic.ConfigFromEnvironment(),
	)
	portal := &RPCConsumerLogs{newRelicApplication: newRelicApplication, StoreMetricData: false}
	isMetricEnabled, _ := strconv.ParseBool(os.Getenv("IS_METRICS_ENABLED"))
	if isMetricEnabled {
		portal.StoreMetricData = true
		portal.MetricService = metrics.NewMetricService()
	}
	return portal, err
}

func (pl *RPCConsumerLogs) GetMessageSeed() string {
	return "GUID_" + strconv.Itoa(rand.Intn(10000000000))
}

// Input will be masked with a random GUID if returnMaskedErrors is set to true
func (pl *RPCConsumerLogs) GetUniqueGuidResponseForError(responseError error, msgSeed string) string {
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
func (pl *RPCConsumerLogs) AnalyzeWebSocketErrorAndWriteMessage(c *websocket.Conn, mt int, err error, msgSeed string, msg []byte, rpcType string) {
	if err != nil {
		if err.Error() == webSocketCloseMessage {
			utils.LavaFormatInfo("Websocket connection closed by the user, "+err.Error(), nil)
			return
		}
		pl.LogRequestAndResponse(rpcType+" ws msg", true, "ws", c.LocalAddr().String(), string(msg), "", msgSeed, err)

		jsonResponse, _ := json.Marshal(fiber.Map{
			"Error_Received": pl.GetUniqueGuidResponseForError(err, msgSeed),
		})

		c.WriteMessage(mt, jsonResponse)
	}
}

func (pl *RPCConsumerLogs) LogRequestAndResponse(module string, hasError bool, method string, path string, req string, resp string, msgSeed string, err error) {
	if hasError && err != nil {
		utils.LavaFormatError(module, err, &map[string]string{"GUID": msgSeed, "request": req, "response": parser.CapStringLen(resp), "method": method, "path": path, "HasError": strconv.FormatBool(hasError)})
		return
	}
	utils.LavaFormatDebug(module, &map[string]string{"GUID": msgSeed, "request": req, "response": parser.CapStringLen(resp), "method": method, "path": path, "HasError": strconv.FormatBool(hasError)})
}

func (pl *RPCConsumerLogs) LogStartTransaction(name string) {
	if pl.newRelicApplication != nil {
		txn := pl.newRelicApplication.StartTransaction(name)
		defer txn.End()
	}
}

func (pl *RPCConsumerLogs) AddMetric(data *metrics.RelayMetrics, isNotSuccessful bool) {
	if pl.StoreMetricData {
		data.Success = !isNotSuccessful
		pl.MetricService.SendData(*data)
	}
}
