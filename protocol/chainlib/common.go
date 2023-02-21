package chainlib

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	common "github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	ContextUserValueKeyDappID = "dappID"
)

type BaseChainParser struct {
	taggedApis map[string]spectypes.ServiceApi
	rwLock     sync.RWMutex
}

func (bcp *BaseChainParser) SetTaggedApis(taggedApis map[string]spectypes.ServiceApi) {
	bcp.taggedApis = taggedApis
}

func (bcp *BaseChainParser) GetSpecApiByTag(tag string) (spectypes.ServiceApi, bool) {
	bcp.rwLock.RLock()
	defer bcp.rwLock.RUnlock()

	val, ok := bcp.taggedApis[tag]
	return val, ok
}

type parsedMessage struct {
	serviceApi     *spectypes.ServiceApi
	apiInterface   *spectypes.ApiInterface
	requestedBlock int64
	msg            parser.RPCInput
}

type BaseChainProxy struct {
	averageBlockTime time.Duration
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
	return pm.msg
}

func extractDappIDFromFiberContext(c *fiber.Ctx) (dappID string) {
	dappID = c.Params("dappId")
	if dappID == "" {
		dappID = "NoDappID"
	}
	return dappID
}

func constructFiberCallbackWithHeaderAndParameterExtraction(callbackToBeCalled fiber.Handler, isMetricEnabled bool) fiber.Handler {
	webSocketCallback := callbackToBeCalled
	handler := func(c *fiber.Ctx) error {
		if isMetricEnabled {
			c.Locals(common.RefererHeaderKey, c.Get(common.RefererHeaderKey, ""))
		}
		return webSocketCallback(c) // uses external dappID
	}
	return handler
}

func extractDappIDFromWebsocketConnection(c *websocket.Conn) string {
	dappId := c.Params("dappId")
	if dappId == "" {
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
	return errorMessage + fmt.Sprintf(`, "%v": "%v"`, key, value)
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

// matchSpecApiByName returns service api which match given name
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

// rpc default endpoint should be websocket. otherwise return an error
func verifyRPCEndpoint(endpoint string) {
	u, err := url.Parse(endpoint)
	if err != nil {
		utils.LavaFormatFatal("unparsable url", err, &map[string]string{"url": endpoint})
	}
	switch u.Scheme {
	case "ws", "wss":
		return
	default:
		utils.LavaFormatWarning("URL scheme should be websocket (ws/wss), got: "+u.Scheme, nil, nil)
	}
}

// rpc default endpoint should be websocket. otherwise return an error
func verifyTendermintEndpoint(endpoints []string) (websocketEndpoint string, httpEndpoint string) {
	for _, endpoint := range endpoints {
		u, err := url.Parse(endpoint)
		if err != nil {
			utils.LavaFormatFatal("unparsable url", err, &map[string]string{"url": endpoint})
		}
		switch u.Scheme {
		case "http", "https":
			httpEndpoint = endpoint
		case "ws", "wss":
			websocketEndpoint = endpoint
		default:
			utils.LavaFormatFatal("URL scheme should be websocket (ws/wss) or (http/https), got: "+u.Scheme, nil, nil)
		}
	}

	if websocketEndpoint == "" || httpEndpoint == "" {
		utils.LavaFormatFatal("Tendermint Provider was not provided with both http and websocket urls. please provide both", nil,
			&map[string]string{"websocket": websocketEndpoint, "http": httpEndpoint})
	}
	return websocketEndpoint, httpEndpoint
}

func GetApiInterfaceFromServiceApi(serviceApi *spectypes.ServiceApi, connectionType string) *spectypes.ApiInterface {
	var apiInterface *spectypes.ApiInterface = nil
	for i := range serviceApi.ApiInterfaces {
		if serviceApi.ApiInterfaces[i].Type == connectionType {
			apiInterface = &serviceApi.ApiInterfaces[i]
			break
		}
	}
	return apiInterface
}

func CraftChainMessage(serviceApi spectypes.ServiceApi, chainParser ChainParser) ChainMessageForSend {
	return chainParser.CraftMessage(serviceApi)
}
