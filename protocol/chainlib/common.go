package chainlib

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	ContextUserValueKeyDappID = "dappID"
)

func extractDappIDFromFiberContext(c *fiber.Ctx) (dappID string) {
	if len(c.Route().Params) > 1 {
		dappID = c.Route().Params[1]
		dappID = strings.ReplaceAll(dappID, "*", "")
		return
	}
	return "NoDappID"
}

func constructFiberCallbackWithDappIDExtraction(callbackToBeCalled fiber.Handler) fiber.Handler {
	webSocketCallback := callbackToBeCalled
	handler := func(c *fiber.Ctx) error {
		// dappID := ""
		// if len(c.Route().Params) > 1 {
		// 	dappID = c.Route().Params[1]
		// 	dappID = strings.ReplaceAll(dappID, "*", "")
		// }
		return webSocketCallback(c) // uses external dappID
	}
	return handler
}

func convertToJsonError(errorMsg string) string {
	jsonResponse, err := json.Marshal(fiber.Map{
		"error": errorMsg,
	})
	if err != nil {
		return `{"error": "Failed to marshal error response to json"}`
	}

	return string(jsonResponse)
}

func extractDappIDFromWebsocketConnection(c *websocket.Conn) string {
	dappIDLocal := c.Locals(ContextUserValueKeyDappID)
	if dappID, ok := dappIDLocal.(string); ok {
		// zeroallocation policy for fiber.Ctx
		buffer := make([]byte, len(dappID))
		copy(buffer, dappID)
		return string(buffer)
	}
	return "NoDappID"
}

func addAttributeToError(key string, value string, errorMessage string) string {
	return errorMessage + fmt.Sprintf(", %v: %v", key, value)
}

func getServiceApis(spec spectypes.Spec, rpcInterface string) (retServerApis map[string]spectypes.ServiceApi, retTaggedApis map[string]spectypes.ServiceApi) {
	serverApis := map[string]spectypes.ServiceApi{}
	taggedApis := map[string]spectypes.ServiceApi{}
	if spec.Enabled {
		for _, api := range spec.Apis {
			if !api.Enabled {
				continue
			}
			//
			// TODO: find a better spot for this (more optimized, precompile regex, etc)
			for _, apiInterface := range api.ApiInterfaces {
				if apiInterface.Interface != rpcInterface {
					// spec will contain many api interfaces, we only need those that belong to the apiInterface of this sentry
					continue
				}
				if apiInterface.Interface == spectypes.APIInterfaceRest {
					re := regexp.MustCompile(`{[^}]+}`)
					processedName := string(re.ReplaceAll([]byte(api.Name), []byte("replace-me-with-regex")))
					processedName = regexp.QuoteMeta(processedName)
					processedName = strings.ReplaceAll(processedName, "replace-me-with-regex", `[^\/\s]+`)
					serverApis[processedName] = api
				} else {
					serverApis[api.Name] = api
				}

				if api.Parsing.GetFunctionTag() != "" {
					taggedApis[api.Parsing.GetFunctionTag()] = api
				}
			}
		}
	}
	return serverApis, taggedApis
}
