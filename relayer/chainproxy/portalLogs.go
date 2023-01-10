package chainproxy

import (
	"context"
	"fmt"
	"github.com/lavanet/lava/relayer/common"
	"math/rand"
	"os"
	"strconv"

	"github.com/lavanet/lava/relayer/lavasession"

	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/utils"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var ReturnMaskedErrors = "false"

const webSocketCloseMessage = "websocket: close 1005 (no status)"

type PortalLogs struct {
	newRelicApplication *newrelic.Application
}

func NewPortalLogs() (*PortalLogs, error) {
	err := godotenv.Load()
	if err != nil {
		utils.LavaFormatInfo("New relic missing environment file", nil)

		return &PortalLogs{nil}, nil
	}

	NEW_RELIC_APP_NAME := os.Getenv("NEW_RELIC_APP_NAME")
	NEW_RELIC_LICENSE_KEY := os.Getenv("NEW_RELIC_LICENSE_KEY")
	if NEW_RELIC_APP_NAME == "" || NEW_RELIC_LICENSE_KEY == "" {
		utils.LavaFormatInfo("New relic missing environment variables", nil)
		return &PortalLogs{nil}, nil
	}
	newRelicApplication, err := newrelic.NewApplication(
		newrelic.ConfigAppName(NEW_RELIC_APP_NAME),
		newrelic.ConfigLicense(NEW_RELIC_LICENSE_KEY),
		newrelic.ConfigFromEnvironment(),
	)
	return &PortalLogs{newRelicApplication}, err
}

func (cp *PortalLogs) GetMessageSeed() string {
	return "GUID_" + strconv.Itoa(rand.Intn(10000000000))
}

// Input will be masked with a random GUID if returnMaskedErrors is set to true
func (cp *PortalLogs) GetUniqueGuidResponseForError(ctx context.Context, responseError error, msgSeed string) string {
	var ret string
	ret = "Error GUID: " + msgSeed
	if ReturnMaskedErrors == "false" {
		ret += fmt.Sprintf(", Error: %v", responseError)
	}

	// extract epoch error allow-list from context
	epochErrorAllowlist, ok := ctx.Value("epochErrorAllowList").(*common.EpochErrorAllowlist)
	if !ok {
		utils.LavaFormatError("UniqueGuidResponseForError", responseError, &map[string]string{"msgSeed": msgSeed})
		return ret
	}

	// Only log error if it is not allow-listed for the current epoch
	if responseError != nil && !epochErrorAllowlist.IsErrorSet(responseError.Error()) {
		utils.LavaFormatError("UniqueGuidResponseForError", responseError, &map[string]string{"msgSeed": msgSeed})
	}

	return ret
}

// Websocket healthy disconnections throw "websocket: close 1005 (no status)" error,
// We dont want to alert error monitoring for that purpses.
func (cp *PortalLogs) AnalyzeWebSocketErrorAndWriteMessage(ctx context.Context, c *websocket.Conn, mt int, err error, msgSeed string, msg []byte, rpcType string) {
	if err != nil {
		if err.Error() == webSocketCloseMessage {
			utils.LavaFormatInfo("Websocket connection closed by the user, "+err.Error(), nil)
			return
		}
		cp.LogRequestAndResponse(ctx, rpcType+" ws msg", true, "ws", c.LocalAddr().String(), string(msg), "", msgSeed, err)
		c.WriteMessage(mt, []byte("Error Received: "+cp.GetUniqueGuidResponseForError(ctx, err, msgSeed)))
	}
}

func (cp *PortalLogs) LogRequestAndResponse(ctx context.Context, module string, hasError bool, method string, path string, req string, resp string, msgSeed string, err error) {
	if hasError && err != nil {
		// extract epoch error allow-list from context
		epochErrorAllowlist, ok := ctx.Value("epochErrorAllowList").(*common.EpochErrorAllowlist)
		if !ok {
			// If epochErrorAllowList doesn't exists in context just print log
			utils.LavaFormatError(module, err, &map[string]string{"GUID": msgSeed, "request": req, "response": parser.CapStringLen(resp), "method": method, "path": path, "HasError": strconv.FormatBool(hasError)})
			return
		}

		// Only log error if it is not allow-listed for the current epoch
		if !epochErrorAllowlist.IsErrorSet(err.Error()) {
			utils.LavaFormatError(module, err, &map[string]string{"GUID": msgSeed, "request": req, "response": parser.CapStringLen(resp), "method": method, "path": path, "HasError": strconv.FormatBool(hasError)})
		}

		// On first occurrence of no pairing list error add it to the allow-listed
		if err.Error() == lavasession.PairingListEmptyError.Error() && !epochErrorAllowlist.IsErrorSet(lavasession.PairingListEmptyError.Error()) {
			epochErrorAllowlist.SetError(lavasession.PairingListEmptyError.Error())
		}

		return
	}
	utils.LavaFormatDebug(module, &map[string]string{"GUID": msgSeed, "request": req, "response": parser.CapStringLen(resp), "method": method, "path": path, "HasError": strconv.FormatBool(hasError)})
}

func (cp *PortalLogs) LogStartTransaction(name string) {
	if cp.newRelicApplication != nil {
		txn := cp.newRelicApplication.StartTransaction(name)
		defer txn.End()
	}
}
