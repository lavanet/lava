package chainlib

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	ContextUserValueKeyDappID = "dappID"
)

type parsedMessage struct {
	serviceApi     *spectypes.ServiceApi
	apiInterface   *spectypes.ApiInterface
	requestedBlock int64
	msg            interface{}
}

func (pm parsedMessage) GetServiceApi() *spectypes.ServiceApi {
	return pm.serviceApi
}

func (pm parsedMessage) GetInterface() *spectypes.ApiInterface {
	return pm.apiInterface
}

func (pm parsedMessage) RequestedBlock() int64 {
	return pm.requestedBlock
}

func (pm parsedMessage) GetRPCMessage() parser.RPCInput {
	rpcInput, ok := pm.msg.(parser.RPCInput)
	if !ok {
		return nil
	}
	return rpcInput
}

func extractDappIDFromFiberContext(c *fiber.Ctx) (dappID string) {
	dappID = c.Params("dappId")
	if dappID == "" {
		dappID = "NoDappID"
	}
	return dappID
}

func constructFiberCallbackWithDappIDExtraction(callbackToBeCalled fiber.Handler) fiber.Handler {
	webSocketCallback := callbackToBeCalled
	handler := func(c *fiber.Ctx) error {
		dappId := extractDappIDFromFiberContext(c)
		c.Locals("dappId", dappId)
		return webSocketCallback(c) // uses external dappID
	}
	return handler
}

func extractDappIDFromWebsocketConnection(c *websocket.Conn) string {
	dappId, ok := c.Locals("dappId").(string)
	if !ok {
		dappId = "NoDappID"
	}
	return dappId
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

func matchSpecApiByName(name string, serverApis map[string]spectypes.ServiceApi) (spectypes.ServiceApi, bool) {
	// TODO: make it faster and better by not doing a regex instead using a better algorithm
	for apiName, api := range serverApis {
		re, err := regexp.Compile(apiName)
		if err != nil {
			utils.LavaFormatError("regex Compile api", err, &map[string]string{"apiName": apiName})
			continue
		}
		if re.Match([]byte(name)) {
			return api, true
		}
	}
	return spectypes.ServiceApi{}, false
}
