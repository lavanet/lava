package chainproxy

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/newrelic/go-agent/v3/newrelic"
	"math/rand"
	"strconv"

	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/utils"
)

var ReturnMaskedErrors = "false"

const webSocketCloseMessage = "websocket: close 1005 (no status)"

// Input will be masked with a random GUID if returnMaskedErrors is set to true
//func GetUniqueGuidResponseForError(responseError error) string {
//	guID := fmt.Sprintf("GUID%d", rand.Int63())
//	var ret string
//	ret = "Error GUID: " + guID
//	utils.LavaFormatError("UniqueGuidResponseForError", responseError, &map[string]string{"GUID": guID})
//	if ReturnMaskedErrors == "false" {
//		ret += fmt.Sprintf(", Error: %v", responseError)
//	}
//	return ret
//}
//
//// Websocket healthy disconnections throw "websocket: close 1005 (no status)" error,
//// We dont want to alert error monitoring for that purpses.
//func AnalyzeWebSocketErrorAndWriteMessage(c *websocket.Conn, mt int, err error, msgSeed string) {
//	if err.Error() == webSocketCloseMessage {
//		utils.LavaFormatInfo("Websocket connection closed by the user, "+err.Error(), nil)
//		return
//	}
//	c.WriteMessage(mt, []byte("Error Received: "+GetUniqueGuidResponseForError(err)))
//}
//
//// Logging the Request and Response to the stdout.
//func LogRequestAndResponse(module string, hasError bool, method string, path string, req string, resp string, msgSeed string, err error) {
//	if hasError {
//		utils.LavaFormatInfo(module, &map[string]string{"GUID": msgSeed, "request": req, "response": resp, "method": method, "path": path, "HasError": strconv.FormatBool(hasError), "error": err.Error()})
//		return
//	}
//	utils.LavaFormatInfo(module, &map[string]string{"GUID": msgSeed, "request": req, "response": resp, "method": method, "path": path, "HasError": strconv.FormatBool(hasError)})
//}

//func LogNewRelicApp() {
//	var myEnv map[string]string
//	myEnv, err = godotenv.Read()
//
//	NEW_RELIC_APP_NAME := myEnv["NEW_RELIC_APP_NAME"]
//	NEW_RELIC_LICENSE_KEY := myEnv["NEW_RELIC_LICENSE_KEY"]
//
//	newrelicApp, err := newrelic.NewApplication(
//		newrelic.ConfigAppName(NEW_RELIC_APP_NAME),
//		newrelic.ConfigLicense(NEW_RELIC_LICENSE_KEY),
//		newrelic.ConfigAppLogEnabled(true),
//		newrelic.ConfigAppLogForwardingEnabled(true),
//		func(config *newrelic.Config) {
//			zerologlog.Debug().Enabled()
//		},
//	)
//	if myEnv == nil {
//		newrelicApp = nil
//	}
//
//
//}

type PortalLogs struct {
	newRelicApplication *newrelic.Application
}

func (cp *PortalLogs) CreateNewRelicApp() (err error) {
	myEnv, err := godotenv.Read()
	if err != nil {
		return
	}
	NEW_RELIC_APP_NAME := myEnv["NEW_RELIC_APP_NAME"]
	NEW_RELIC_LICENSE_KEY := myEnv["NEW_RELIC_LICENSE_KEY"]

	cp.newRelicApplication, err = newrelic.NewApplication(
		newrelic.ConfigAppName(NEW_RELIC_APP_NAME),
		newrelic.ConfigLicense(NEW_RELIC_LICENSE_KEY),
	)
	return
}

func (cp *PortalLogs) LogRequestAndResponse(module string, hasError bool, method string, path string, req string, resp string, msgSeed string, err error) {
	if hasError {
		utils.LavaFormatInfo(module, &map[string]string{"GUID": msgSeed, "request": req, "response": resp, "method": method, "path": path, "HasError": strconv.FormatBool(hasError), "error": err.Error()})
		return
	}
	utils.LavaFormatInfo(module, &map[string]string{"GUID": msgSeed, "request": req, "response": resp, "method": method, "path": path, "HasError": strconv.FormatBool(hasError)})
}

// Websocket healthy disconnections throw "websocket: close 1005 (no status)" error,
// We dont want to alert error monitoring for that purpses.
func (cp *PortalLogs) AnalyzeWebSocketErrorAndWriteMessage(c *websocket.Conn, mt int, err error, msgSeed string) {
	if err.Error() == webSocketCloseMessage {
		utils.LavaFormatInfo("Websocket connection closed by the user, "+err.Error(), nil)
		return
	}
	c.WriteMessage(mt, []byte("Error Received: "+cp.GetUniqueGuidResponseForError(err)))
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
func (cp *PortalLogs) LogStartTransaction(name string) {
	if cp.newRelicApplication != nil {
		txn := cp.newRelicApplication.StartTransaction(name)
		defer txn.End()
	}
}

// create a struct that manages all the logs f the portal (PortalLogs)
// change the package into portalLogs package
// the struct will have newRelic (type newRelicApp)
//
