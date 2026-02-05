package chainlib

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	sdkerrors "cosmossdk.io/errors"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	common "github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"google.golang.org/grpc/metadata"
)

const (
	ContextUserValueKeyDappID  = "dappID"
	RetryListeningInterval     = 10 // seconds
	debug                      = false
	relayMsgLogMaxChars        = 200
	RPCProviderNodeAddressHash = "Lava-Provider-Node-Address-Hash"
	RPCProviderNodeExtension   = "Lava-Provider-Node-Extension"
	RpcProviderLoadRateHeader  = "Lava-Provider-Load-Rate"
	RpcProviderUniqueIdHeader  = "Lava-Provider-Unique-Id"
	WebSocketExtension         = "websocket"
)

var (
	TrailersToAddToHeaderResponse      = []string{RPCProviderNodeExtension, RpcProviderLoadRateHeader}
	InvalidResponses                   = []string{"null", "", "nil", "undefined"}
	FailedSendingSubscriptionToClients = sdkerrors.New("failed Sending Subscription To Clients", 1015, "Failed Sending Subscription To Clients connection might have been closed by the user")
	NoActiveSubscriptionFound          = sdkerrors.New("failed finding an active subscription on provider side", 1016, "no active subscriptions for hashed params.")
	MaxBatchRequestSize                = 0 // configured via --max-batch-request-size flag, 0 means unlimited
	ErrBatchRequestSizeExceeded        = sdkerrors.New("batch request size exceeded", 1017, "batch request size exceeded the configured limit")
)

type RelayReplyWrapper struct {
	StatusCode int
	RelayReply *pairingtypes.RelayReply
}

type VerificationKey struct {
	Extension string
	Addon     string
}

type VerificationContainer struct {
	InternalPath   string
	ConnectionType string
	Name           string
	ParseDirective spectypes.ParseDirective
	Value          string
	LatestDistance uint64
	Severity       spectypes.ParseValue_VerificationSeverity
	VerificationKey
}

func (vc *VerificationContainer) IsActive() bool {
	if vc.Value == "" && vc.LatestDistance == 0 {
		return false
	}
	return true
}

type TaggedContainer struct {
	Parsing       *spectypes.ParseDirective
	ApiCollection *spectypes.ApiCollection
}

type ApiContainer struct {
	api           *spectypes.Api
	collectionKey CollectionKey
}

type ApiKey struct {
	Name           string
	ConnectionType string
	InternalPath   string
}

type CollectionKey struct {
	ConnectionType string
	InternalPath   string
	Addon          string
}

type BaseChainProxy struct {
	ErrorHandler
	averageBlockTime time.Duration
	NodeUrl          common.NodeUrl
	ChainID          string
	HashedNodeUrl    string
}

// returns the node url and chain id for that proxy.
func (bcp *BaseChainProxy) GetChainProxyInformation() (common.NodeUrl, string) {
	return bcp.NodeUrl, bcp.ChainID
}

func (bcp *BaseChainProxy) CapTimeoutForSend(ctx context.Context, chainMessage ChainMessageForSend) (context.Context, context.CancelFunc) {
	relayTimeout := GetRelayTimeout(chainMessage, bcp.averageBlockTime)
	processingTimeout := common.GetTimeoutForProcessing(relayTimeout, GetTimeoutInfo(chainMessage))
	connectCtx, cancel := bcp.NodeUrl.LowerContextTimeout(ctx, processingTimeout)
	return connectCtx, cancel
}

func extractDappIDFromFiberContext(c *fiber.Ctx) (dappID string) {
	// Read the dappID from the headers
	dappID = c.Get("dapp-id")
	if dappID == "" {
		dappID = generateNewDappID()
	}
	return dappID
}

// extractDappIDFromGrpcHeader extracts dappID from GRPC header
func extractDappIDFromGrpcHeader(metadataValues metadata.MD) string {
	dappId := generateNewDappID()
	if values, ok := metadataValues["dapp-id"]; ok && len(values) > 0 {
		dappId = values[0]
	}
	return dappId
}

// generateNewDappID generates default dappID
// In future we can also implement unique dappID generation
func generateNewDappID() string {
	return "DefaultDappID"
}

func constructFiberCallbackWithHeaderAndParameterExtraction(callbackToBeCalled fiber.Handler, isMetricEnabled bool) fiber.Handler {
	webSocketCallback := callbackToBeCalled
	handler := func(c *fiber.Ctx) error {
		// Extract dappID from headers
		dappID := extractDappIDFromFiberContext(c)

		// Store dappID in the local context
		c.Locals("dapp-id", dappID)

		if isMetricEnabled {
			c.Locals(metrics.RefererHeaderKey, c.Get(metrics.RefererHeaderKey, ""))
			c.Locals(metrics.UserAgentHeaderKey, c.Get(metrics.UserAgentHeaderKey, ""))
			c.Locals(metrics.OriginHeaderKey, c.Get(metrics.OriginHeaderKey, ""))
		}
		return webSocketCallback(c) // uses external dappID
	}
	return handler
}

func checkBTCResponseAndFixReply(chainID string, replyData []byte) string {
	response := string(replyData)
	if chainID == "BTC" || chainID == "BTCT" || chainID == "LTC" || chainID == "LTCT" || chainID == "DOGE" || chainID == "DOGET" {
		var jsonMsg *rpcclient.JsonrpcMessage
		if err := json.Unmarshal(replyData, &jsonMsg); err == nil {
			btcResponse := &rpcclient.BTCResponse{
				Version: jsonMsg.Version,
				ID:      jsonMsg.ID,
				Method:  jsonMsg.Method,
				Error:   jsonMsg.Error,
				Result:  jsonMsg.Result,
			}
			if marshaledRes, err := json.Marshal(btcResponse); err == nil {
				response = string(marshaledRes)
			}
		}
	}
	return response
}

func addHeadersAndSendString(c *fiber.Ctx, metaData []pairingtypes.Metadata, data string) error {
	for _, value := range metaData {
		c.Set(value.Name, value.Value)
	}

	return c.SendString(data)
}

// addHeadersAndSendBytes sends response bytes directly without string conversion.
// This is more memory-efficient for large responses as it avoids []byte to string allocation.
func addHeadersAndSendBytes(c *fiber.Ctx, metaData []pairingtypes.Metadata, data []byte) error {
	for _, value := range metaData {
		c.Set(value.Name, value.Value)
	}

	return c.Send(data)
}

// jsonRPCMethodRequest is a minimal struct to extract only the method field from JSON-RPC request
type jsonRPCMethodRequest struct {
	Method string `json:"method"`
}

// extractJSONRPCMethodFromRequest extracts the method name from a JSON-RPC request body.
// Returns empty string if parsing fails (non-JSON-RPC request or invalid JSON).
func extractJSONRPCMethodFromRequest(requestBody []byte) string {
	var req jsonRPCMethodRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		return ""
	}
	return req.Method
}

// isPassthroughMethod returns true if the given method should use passthrough mode.
// Passthrough mode skips response body logging and uses direct byte sending
// to reduce memory allocations for large responses.
func isPassthroughMethod(method string) bool {
	return IsPassthroughMethod(method)
}

// IsPassthroughMethod returns true if the given method should use passthrough mode.
// Passthrough mode skips expensive operations like decompression, response body logging,
// and string conversions to reduce memory allocations for large responses.
// This is exported so it can be used by other packages like rpcsmartrouter.
func IsPassthroughMethod(method string) bool {
	// Currently only debug_traceTransaction is in passthrough mode
	return method == "debug_traceTransaction"
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

func addAttributeToError(key, value, errorMessage string) string {
	return errorMessage + fmt.Sprintf(`, "%v": "%v"`, key, value)
}

func validateEndpoints(endpoints []common.NodeUrl, apiInterface string) {
	for _, endpoint := range endpoints {
		common.ValidateEndpoint(endpoint.Url, apiInterface)
	}
}

func ListenWithRetry(app *fiber.App, address string, chosenAddrCh *common.SafeChannelSender[string]) {
	for {
		ln, err := net.Listen("tcp", address)
		if err != nil {
			utils.LavaFormatError("net.Listen(tcp, address)", err, utils.LogAttr("address", address))
		} else {
			chosenAddrCh.Send(ln.Addr().String())

			err = app.Listener(ln)
			if err != nil {
				utils.LavaFormatError("app.Listen(listenAddr)", err)
			}
		}
		time.Sleep(RetryListeningInterval * time.Second)
	}
}

func GetListenerWithRetryGrpc(protocol, addr string) net.Listener {
	for {
		lis, err := net.Listen(protocol, addr)
		if err == nil {
			return lis
		}
		utils.LavaFormatError("failure setting up listener, net.Listen(protocol, addr)", err, utils.Attribute{Key: "listenAddr", Value: addr})
		time.Sleep(RetryListeningInterval * time.Second)
		utils.LavaFormatWarning("Attempting connection retry", nil)
	}
}

// GetHeaderFromCachedMap extracts a header value from a cached headers map.
// Returns the first value if present, or the defaultValue if not found.
// This avoids repeated calls to fiberCtx.Get() which has overhead.
func GetHeaderFromCachedMap(headers map[string][]string, key string, defaultValue string) string {
	if values, ok := headers[key]; ok && len(values) > 0 {
		return values[0]
	}
	return defaultValue
}

// rest request headers are formatted like map[string]string
func convertToMetadataMap(md map[string][]string) []pairingtypes.Metadata {
	metadata := make([]pairingtypes.Metadata, len(md))
	indexer := 0
	for k, v := range md {
		metadata[indexer] = pairingtypes.Metadata{Name: k, Value: strings.Join(v, ", ")}
		indexer += 1
	}
	return metadata
}

// rest response headers / grpc headers are formatted like map[string][]string
func convertToMetadataMapOfSlices(md map[string][]string) []pairingtypes.Metadata {
	metadata := make([]pairingtypes.Metadata, len(md))
	indexer := 0
	for k, v := range md {
		metadata[indexer] = pairingtypes.Metadata{Name: k, Value: v[0]}
		indexer += 1
	}
	return metadata
}

func convertRelayMetaDataToMDMetaData(md []pairingtypes.Metadata) metadata.MD {
	responseMetaData := make(metadata.MD)
	for _, v := range md {
		responseMetaData[v.Name] = append(responseMetaData[v.Name], v.Value)
	}
	return responseMetaData
}

// split two requested blocks to the most advanced and most behind
// the hierarchy is as follows:
// NOT_APPLICABLE
// LATEST_BLOCK
// PENDING_BLOCK
// SAFE
// FINALIZED
// numeric value (descending)
// EARLIEST
func CompareRequestedBlockInBatch(currentLatestRequestedBlock, currentEarliestRequestedBlock, parsedBlock int64) (latestCombinedBlock int64, earliestCombinedBlock int64) {
	latestCallback := func(currentLatest int64, parsedBlock int64) int64 {
		if currentLatest < 0 && parsedBlock < 0 {
			return utils.Max(currentLatest, parsedBlock)
		}

		if currentLatest > 0 && parsedBlock < 0 && parsedBlock != spectypes.EARLIEST_BLOCK {
			return parsedBlock
		}

		if currentLatest < 0 && parsedBlock > 0 && currentLatest != spectypes.EARLIEST_BLOCK {
			return currentLatest
		}

		return utils.Max(currentLatest, parsedBlock)
	}

	earliestCallback := func(currentEarliest int64, parsedBlock int64) int64 {
		if currentEarliest == spectypes.EARLIEST_BLOCK || parsedBlock == spectypes.EARLIEST_BLOCK {
			return spectypes.EARLIEST_BLOCK
		}

		if currentEarliest == spectypes.NOT_APPLICABLE || parsedBlock == spectypes.NOT_APPLICABLE {
			return spectypes.NOT_APPLICABLE
		}

		if currentEarliest < 0 && parsedBlock < 0 {
			return utils.Min(currentEarliest, parsedBlock)
		}

		if currentEarliest > 0 && parsedBlock < 0 {
			return currentEarliest
		}

		if currentEarliest < 0 && parsedBlock > 0 {
			return parsedBlock
		}

		return utils.Min(currentEarliest, parsedBlock)
	}

	return latestCallback(currentLatestRequestedBlock, parsedBlock), earliestCallback(currentEarliestRequestedBlock, parsedBlock)
}

func GetRelayTimeout(chainMessage ChainMessageForSend, averageBlockTime time.Duration) time.Duration {
	if chainMessage.TimeoutOverride() != 0 {
		return chainMessage.TimeoutOverride()
	}
	// Calculate extra RelayTimeout
	extraRelayTimeout := time.Duration(0)
	if IsHangingApi(chainMessage) {
		extraRelayTimeout = averageBlockTime * 2
	}
	relayTimeAddition := common.GetTimePerCu(GetComputeUnits(chainMessage))
	if chainMessage.GetApi().TimeoutMs > 0 {
		relayTimeAddition = time.Millisecond * time.Duration(chainMessage.GetApi().TimeoutMs)
	}
	// Set relay timout, increase it every time we fail a relay on timeout
	return extraRelayTimeout + relayTimeAddition
}

// setup a common preflight and cors configuration allowing wild cards and preflight caching.
func createAndSetupBaseAppListener(cmdFlags common.ConsumerCmdFlags, healthCheckPath string, healthReporter HealthReporter) *fiber.App {
	app := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})
	app.Use(favicon.New())
	app.Use(compress.New(compress.Config{Level: compress.LevelBestSpeed}))
	app.Use(func(c *fiber.Ctx) error {
		// we set up wild card by default.
		c.Set("Access-Control-Allow-Origin", cmdFlags.OriginFlag)
		// Handle preflight requests directly
		if c.Method() == "OPTIONS" {
			// set up all allowed methods.
			c.Set("Access-Control-Allow-Methods", cmdFlags.MethodsFlag)
			// allow headers
			c.Set("Access-Control-Allow-Headers", cmdFlags.HeadersFlag)
			// allow credentials
			c.Set("Access-Control-Allow-Credentials", cmdFlags.CredentialsFlag)
			// Cache preflight request for 24 hours (in seconds)
			c.Set("Access-Control-Max-Age", cmdFlags.CDNCacheDuration)
			return c.SendStatus(fiber.StatusNoContent)
		}
		if c.Method() == "DELETE" {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	})

	app.Get(healthCheckPath, func(fiberCtx *fiber.Ctx) error {
		if healthReporter.IsHealthy() {
			fiberCtx.Status(http.StatusOK)
			return fiberCtx.SendString("Health status OK")
		} else {
			fiberCtx.Status(http.StatusServiceUnavailable)
			return fiberCtx.SendString("Health status Failure")
		}
	})

	return app
}

func truncateAndPadString(s string, maxLength int) string {
	// Truncate to a maximum length
	if len(s) > maxLength {
		s = s[:maxLength]
	}

	// Pad with empty strings if the length is less than the specified maximum length
	s = fmt.Sprintf("%-*s", maxLength, s)

	return s
}

// return if response is valid or not - true
func ValidateNilResponse(responseString string) error {
	return nil // this feature was disabled in version 0.35.8 due to some nodes request this response.
	// after the timeout features we can add support for this filtering as it would be parsed and
	// returned to the user if multiple providers returned the same type of response

	// Removed on 0.35.8
	// if slices.Contains(InvalidResponses, responseString) {
	// 	return fmt.Errorf("response returned an empty value: %s", responseString)
	// }
	// return nil
}

func GetTimeoutInfo(chainMessage ChainMessageForSend) common.TimeoutInfo {
	return common.TimeoutInfo{
		CU:       chainMessage.GetApi().ComputeUnits,
		Hanging:  IsHangingApi(chainMessage),
		Stateful: GetStateful(chainMessage),
	}
}

func IsUrlWebSocket(urlToParse string) (bool, error) {
	u, err := url.Parse(urlToParse)
	if err != nil {
		return false, err
	}

	return u.Scheme == "ws" || u.Scheme == "wss", nil
}
