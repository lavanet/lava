package chainproxy

import (
	"fmt"
	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
	"github.com/lavanet/lava/utils"
	"github.com/newrelic/go-agent/v3/newrelic"
	"math/rand"
	"os"
	"strconv"
)

var ReturnMaskedErrors = "false"

const webSocketCloseMessage = "websocket: close 1005 (no status)"

type PortalLogs struct {
	newRelicApplication *newrelic.Application
}

func NewPortalLogs() (*PortalLogs, error) {
	err := godotenv.Load()
	if err != nil {
		return &PortalLogs{}, err
	}

	NEW_RELIC_APP_NAME := os.Getenv("NEW_RELIC_APP_NAME")
	NEW_RELIC_LICENSE_KEY := os.Getenv("NEW_RELIC_LICENSE_KEY")
	if NEW_RELIC_APP_NAME == "" || NEW_RELIC_LICENSE_KEY == "" {
		utils.LavaFormatInfo("New relic missing environment variables", nil)
		return &PortalLogs{}, nil
	}
	newRelicApplication, err := newrelic.NewApplication(
		newrelic.ConfigAppName(NEW_RELIC_APP_NAME),
		newrelic.ConfigLicense(NEW_RELIC_LICENSE_KEY),
		newrelic.ConfigFromEnvironment(),
	)
	return &PortalLogs{newRelicApplication}, nil
}

// Input will be masked with a random GUID if returnMaskedErrors is set to true
func (cp *PortalLogs) GetUniqueGuidResponseForError(responseError error) string {
	guID := fmt.Sprintf("GUID%d", rand.Int63())
	var ret string
	ret = "Error GUID: " + guID
	utils.LavaFormatError("UniqueGuidResponseForError", responseError, &map[string]string{"GUID": guID})
	if ReturnMaskedErrors == "false" {
		ret += fmt.Sprintf(", Error: %v", responseError)
	}
	return ret
}

// Websocket healthy disconnections throw "websocket: close 1005 (no status)" error,
// We dont want to alert error monitoring for that purpses.
func (cp *PortalLogs) AnalyzeWebSocketErrorAndWriteMessage(c *websocket.Conn, mt int, err error, msgSeed string) {

	if err != nil {
		if err.Error() == webSocketCloseMessage {
			utils.LavaFormatInfo("Websocket connection closed by the user, "+err.Error(), nil)
			return
		}
		c.WriteMessage(mt, []byte("Error Received: "+cp.GetUniqueGuidResponseForError(err)))
	}
}

func (cp *PortalLogs) LogRequestAndResponse(module string, hasError bool, method string, path string, req string, resp string, msgSeed string, err error) {
	if hasError && err != nil {
		utils.LavaFormatInfo(module, &map[string]string{"GUID": msgSeed, "request": req, "response": resp, "method": method, "path": path, "HasError": strconv.FormatBool(hasError), "error": err.Error()})
		return
	}
	utils.LavaFormatInfo(module, &map[string]string{"GUID": msgSeed, "request": req, "response": resp, "method": method, "path": path, "HasError": strconv.FormatBool(hasError)})
}

func (cp *PortalLogs) LogStartTransaction(name string) {
	if cp.newRelicApplication != nil {
		txn := cp.newRelicApplication.StartTransaction(name)
		defer txn.End()
	}
}
