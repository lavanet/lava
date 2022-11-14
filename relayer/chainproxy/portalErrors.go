package chainproxy

import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/utils"
)

var ReturnMaskedErrors = "false"

const webSocketCloseMessage = "websocket: close 1005 (no status)"

// Input will be masked with a random GUID if returnMaskedErrors is set to true
func GetUniqueGuidResponseForError(responseError error) string {
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
func AnalyzeWebSocketErrorAndWriteMessage(c *websocket.Conn, mt int, err error, msgSeed string) {
	if err != nil {
		if err.Error() == webSocketCloseMessage {
			utils.LavaFormatInfo("Websocket connection closed by the user, "+err.Error(), nil)
			return
		}
		c.WriteMessage(mt, []byte("Error Received: "+GetUniqueGuidResponseForError(err)))
	}
}

// Logging the Request and Response to the stdout.
func LogRequestAndResponse(module string, hasError bool, method string, path string, req string, resp string, msgSeed string, err error) {
	if hasError && err != nil {
		utils.LavaFormatInfo(module, &map[string]string{"GUID": msgSeed, "request": req, "response": resp, "method": method, "path": path, "HasError": strconv.FormatBool(hasError), "error": err.Error()})
		return
	}
	utils.LavaFormatInfo(module, &map[string]string{"GUID": msgSeed, "request": req, "response": resp, "method": method, "path": path, "HasError": strconv.FormatBool(hasError)})
}
