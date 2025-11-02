package chainlib

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	websocket2 "github.com/gorilla/websocket"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchSpecApiByName(t *testing.T) {
	t.Parallel()
	connectionType := ""
	testTable := []struct {
		name        string
		serverApis  map[ApiKey]ApiContainer
		inputName   string
		expectedApi spectypes.Api
		expectedOk  bool
	}{
		{
			name: "test1",
			serverApis: map[ApiKey]ApiContainer{
				{Name: "/blocks/[^\\/\\s]+", ConnectionType: connectionType}: {
					api: &spectypes.Api{
						Name: "/blocks/{height}",
						BlockParsing: spectypes.BlockParser{
							ParserArg:  []string{"0"},
							ParserFunc: spectypes.PARSER_FUNC_PARSE_BY_ARG,
						},
						ComputeUnits: 10,
						Enabled:      true,
						Category:     spectypes.SpecCategory{Deterministic: true},
					},
					collectionKey: CollectionKey{ConnectionType: connectionType},
				},
			},
			inputName:   "/blocks/10",
			expectedApi: spectypes.Api{Name: "/blocks/{height}"},
			expectedOk:  true,
		},
		{
			name: "test2",
			serverApis: map[ApiKey]ApiContainer{
				{Name: "/cosmos/base/tendermint/v1beta1/blocks/[^\\/\\s]+", ConnectionType: connectionType}: {
					api: &spectypes.Api{
						Name: "/cosmos/base/tendermint/v1beta1/blocks/{height}",
						BlockParsing: spectypes.BlockParser{
							ParserArg:  []string{"0"},
							ParserFunc: spectypes.PARSER_FUNC_PARSE_BY_ARG,
						},
						ComputeUnits: 10,
						Enabled:      true,
						Category:     spectypes.SpecCategory{Deterministic: true},
					},
					collectionKey: CollectionKey{ConnectionType: connectionType},
				},
			},
			inputName:   "/cosmos/base/tendermint/v1beta1/blocks/10",
			expectedApi: spectypes.Api{Name: "/cosmos/base/tendermint/v1beta1/blocks/{height}"},
			expectedOk:  true,
		},
		{
			name: "test3",
			serverApis: map[ApiKey]ApiContainer{
				{Name: "/cosmos/base/tendermint/v1beta1/blocks/latest", ConnectionType: connectionType}: {
					api: &spectypes.Api{
						Name: "/cosmos/base/tendermint/v1beta1/blocks/latest",
						BlockParsing: spectypes.BlockParser{
							ParserArg:  []string{"0"},
							ParserFunc: spectypes.PARSER_FUNC_DEFAULT,
						},
						ComputeUnits: 10,
						Enabled:      true,
						Category:     spectypes.SpecCategory{Deterministic: true},
					},
					collectionKey: CollectionKey{ConnectionType: connectionType},
				},
			},
			inputName:   "/cosmos/base/tendermint/v1beta1/blocks/latest",
			expectedApi: spectypes.Api{Name: "/cosmos/base/tendermint/v1beta1/blocks/latest"},
			expectedOk:  true,
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			api, ok := matchSpecApiByName(testCase.inputName, connectionType, testCase.serverApis)
			if ok != testCase.expectedOk {
				t.Fatalf("expected ok value %v, but got %v", testCase.expectedOk, ok)
			}
			if api.api.Name != testCase.expectedApi.Name {
				t.Fatalf("expected api %v, but got %v", testCase.expectedApi.Name, api.api.Name)
			}
		})
	}
}

func TestConvertToJsonError(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		name     string
		errorMsg string
		expected string
	}{
		{
			name:     "valid json",
			errorMsg: "some error message",
			expected: `{"error":"some error message"}`,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := convertToJsonError(testCase.errorMsg)
			if result != testCase.expected {
				t.Errorf("Expected result to be %s, but got %s", testCase.expected, result)
			}
		})
	}
}

func TestFormatErrorForAPIInterface(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		name           string
		errMasking     string
		apiInterface   string
		requestID      json.RawMessage
		chainID        string
		expectedOutput map[string]interface{} // Use map for flexible JSON comparison
	}{
		{
			name:         "JSON-RPC with numeric ID",
			errMasking:   `{"Error_GUID":"abc123","Error":"relay failed"}`,
			apiInterface: "jsonrpc",
			requestID:    json.RawMessage("1"),
			chainID:      "ETH1",
			expectedOutput: map[string]interface{}{
				"jsonrpc": "2.0",
				"error": map[string]interface{}{
					"code":    float64(-32603),
					"message": "relay failed",
					"data":    "GUID:abc123",
				},
				"id": float64(1),
			},
		},
		{
			name:         "JSON-RPC with string ID",
			errMasking:   `{"Error_GUID":"abc123","Error":"relay failed"}`,
			apiInterface: "jsonrpc",
			requestID:    json.RawMessage(`"req-123"`),
			chainID:      "ETH1",
			expectedOutput: map[string]interface{}{
				"jsonrpc": "2.0",
				"error": map[string]interface{}{
					"code":    float64(-32603),
					"message": "relay failed",
					"data":    "GUID:abc123",
				},
				"id": "req-123",
			},
		},
		{
			name:         "JSON-RPC with null ID",
			errMasking:   `{"Error_GUID":"abc123","Error":"relay failed"}`,
			apiInterface: "jsonrpc",
			requestID:    json.RawMessage("null"),
			chainID:      "ETH1",
			expectedOutput: map[string]interface{}{
				"jsonrpc": "2.0",
				"error": map[string]interface{}{
					"code":    float64(-32603),
					"message": "relay failed",
					"data":    "GUID:abc123",
				},
				"id": nil,
			},
		},
		{
			name:         "JSON-RPC with nil request ID",
			errMasking:   `{"Error_GUID":"abc123","Error":"relay failed"}`,
			apiInterface: "jsonrpc",
			requestID:    nil,
			chainID:      "ETH1",
			expectedOutput: map[string]interface{}{
				"jsonrpc": "2.0",
				"error": map[string]interface{}{
					"code":    float64(-32603),
					"message": "relay failed",
					"data":    "GUID:abc123",
				},
				"id": nil,
			},
		},
		{
			name:         "REST standard chain",
			errMasking:   `{"Error_GUID":"abc123","Error":"relay failed"}`,
			apiInterface: "rest",
			requestID:    nil,
			chainID:      "LAV1",
			expectedOutput: map[string]interface{}{
				"code":    float64(13),
				"message": "relay failed",
				"details": []interface{}{
					map[string]interface{}{
						"@type": RestErrorDetailsType,
						"guid":  "abc123",
					},
				},
			},
		},
		{
			name:         "REST Aptos chain",
			errMasking:   `{"Error_GUID":"abc123","Error":"relay failed"}`,
			apiInterface: "rest",
			requestID:    nil,
			chainID:      "APT1",
			expectedOutput: map[string]interface{}{
				"message":       "relay failed",
				"error_code":    "internal_error",
				"vm_error_code": nil,
				"guid":          "abc123",
			},
		},
		{
			name:         "TendermintRPC (uses JSON-RPC 2.0 format)",
			errMasking:   `{"Error_GUID":"abc123","Error":"relay failed"}`,
			apiInterface: "tendermintrpc",
			requestID:    json.RawMessage("-1"),
			chainID:      "LAV1",
			expectedOutput: map[string]interface{}{
				"jsonrpc": "2.0",
				"error": map[string]interface{}{
					"code":    float64(-32603),
					"message": "relay failed",
					"data":    "GUID:abc123",
				},
				"id": float64(-1),
			},
		},
		{
			name:         "JSON-RPC without Error field (masked errors)",
			errMasking:   `{"Error_GUID":"abc123"}`,
			apiInterface: "jsonrpc",
			requestID:    json.RawMessage("1"),
			chainID:      "ETH1",
			expectedOutput: map[string]interface{}{
				"jsonrpc": "2.0",
				"error": map[string]interface{}{
					"code":    float64(-32603),
					"message": "Internal error",
					"data":    "GUID:abc123",
				},
				"id": float64(1),
			},
		},
		{
			name:         "JSON-RPC with detailed error message",
			errMasking:   `{"Error_GUID":"xyz789","Error":"failed relay, insufficient results: rpc error: code = Unknown desc = did not pass relay validation"}`,
			apiInterface: "jsonrpc",
			requestID:    json.RawMessage("42"),
			chainID:      "ETH1",
			expectedOutput: map[string]interface{}{
				"jsonrpc": "2.0",
				"error": map[string]interface{}{
					"code":    float64(-32603),
					"message": "failed relay, insufficient results: rpc error: code = Unknown desc = did not pass relay validation",
					"data":    "GUID:xyz789",
				},
				"id": float64(42),
			},
		},
		{
			name:         "JSON-RPC with structured error (code and clean message)",
			errMasking:   `{"Error_GUID":"abc123","Error":"rpc error: code = Code(3370) desc = relayReceiver is disabled","Error_Code":"3370","Error_Message":"relayReceiver is disabled"}`,
			apiInterface: "jsonrpc",
			requestID:    json.RawMessage("1"),
			chainID:      "ETH1",
			expectedOutput: map[string]interface{}{
				"jsonrpc": "2.0",
				"error": map[string]interface{}{
					"code":    float64(-32603),
					"message": "relayReceiver is disabled", // Clean message used
					"data":    "GUID:abc123|Code:3370",     // Error code included
				},
				"id": float64(1),
			},
		},
		{
			name:         "REST with structured error (code and clean message)",
			errMasking:   `{"Error_GUID":"def456","Error":"rpc error: code = Code(3370) desc = relayReceiver is disabled","Error_Code":"3370","Error_Message":"relayReceiver is disabled"}`,
			apiInterface: "rest",
			requestID:    nil,
			chainID:      "LAV1",
			expectedOutput: map[string]interface{}{
				"code":    float64(13),
				"message": "relayReceiver is disabled", // Clean message used
				"details": []interface{}{
					map[string]interface{}{
						"@type":      RestErrorDetailsType,
						"guid":       "def456",
						"error_code": "3370", // Error code included
					},
				},
			},
		},
		{
			name:         "REST Aptos with structured error",
			errMasking:   `{"Error_GUID":"ghi789","Error":"rpc error: code = Code(4001) desc = insufficient funds","Error_Code":"4001","Error_Message":"insufficient funds"}`,
			apiInterface: "rest",
			requestID:    nil,
			chainID:      "APT1",
			expectedOutput: map[string]interface{}{
				"message":         "insufficient funds", // Clean message used
				"error_code":      "internal_error",
				"vm_error_code":   nil,
				"lava_error_code": "4001", // Lava error code included
				"guid":            "ghi789",
			},
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := formatErrorForAPIInterface(
				testCase.errMasking,
				testCase.apiInterface,
				testCase.requestID,
				testCase.chainID,
			)

			// Validate JSON structure
			var resultJSON interface{}
			err := json.Unmarshal([]byte(result), &resultJSON)
			require.NoError(t, err, "Result should be valid JSON")

			// Compare as maps for flexible comparison
			require.Equal(t, testCase.expectedOutput, resultJSON)
		})
	}
}

func TestAddAttributeToError(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		name         string
		key          string
		value        string
		errorMessage string
		expected     string
	}{
		{
			name:         "Valid conversion",
			key:          "key1",
			value:        "value1",
			errorMessage: `"errorKey": "error_value"`,
			expected:     `"errorKey": "error_value", "key1": "value1"`,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result := addAttributeToError(testCase.key, testCase.value, testCase.errorMessage)
			if result != testCase.expected {
				t.Errorf("addAttributeToError(%q, %q, %q) = %q; expected %q", testCase.key, testCase.value, testCase.errorMessage, result, testCase.expected)
			}
		})
	}
}

func TestExtractDappIDFromWebsocketConnection(t *testing.T) {
	testCases := []struct {
		name     string
		route    string
		headers  map[string][]string
		expected string
	}{
		{
			name:     "dappId exists in params",
			route:    "/ws",
			headers:  map[string][]string{"dapp-id": {"DappID123"}},
			expected: "DappID123",
		},
		{
			name:     "dappId does not exist in params",
			route:    "/ws",
			headers:  map[string][]string{},
			expected: "DefaultDappID",
		},
	}

	app := fiber.New()

	webSocketCallback := websocket.New(func(websockConn *websocket.Conn) {
		mt, _, _ := websockConn.ReadMessage()
		dappID, ok := websockConn.Locals("dapp-id").(string)
		if !ok {
			t.Fatalf("Unable to extract dappID")
		}
		websockConn.WriteMessage(mt, []byte(dappID))
	})

	app.Get("/ws", constructFiberCallbackWithHeaderAndParameterExtraction(webSocketCallback, false))

	go app.Listen("127.0.0.1:3000")
	defer func() {
		app.Shutdown()
	}()
	time.Sleep(time.Millisecond * 20) // let the server go up
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			url := "ws://127.0.0.1:3000" + testCase.route
			dialer := &websocket2.Dialer{}
			conn, _, err := dialer.Dial(url, testCase.headers)
			if err != nil {
				t.Fatalf("Error dialing websocket connection: %s", err)
			}
			defer conn.Close()

			err = conn.WriteMessage(websocket.TextMessage, []byte("test"))
			if err != nil {
				t.Fatalf("Error writing message to websocket connection: %s", err)
			}

			_, response, err := conn.ReadMessage()
			if err != nil {
				t.Fatalf("Error reading message from websocket connection: %s", err)
			}

			responseString := string(response)
			if responseString != testCase.expected {
				t.Errorf("Expected %s but got %s", testCase.expected, responseString)
			}
		})
	}
}

func TestExtractDappIDFromFiberContext(t *testing.T) {
	testCases := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{
			name:     "dappId exists in headers",
			headers:  map[string]string{"dapp-id": "DappID123"},
			expected: "DappID123",
		},
		{
			name:     "dappId does not exist in headers",
			headers:  map[string]string{},
			expected: "DefaultDappID",
		},
	}

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		dappID := extractDappIDFromFiberContext(c)
		return c.SendString(dappID)
	})

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			for key, value := range testCase.headers {
				req.Header.Set(key, value)
			}
			resp, _ := app.Test(req)
			body, _ := io.ReadAll(resp.Body)
			responseString := string(body)
			if responseString != testCase.expected {
				t.Errorf("Expected %s but got %s", testCase.expected, responseString)
			}
		})
	}
}

func TestParsedMessage_GetServiceApi(t *testing.T) {
	pm := baseChainMessageContainer{
		api: &spectypes.Api{},
	}
	assert.Equal(t, &spectypes.Api{}, pm.GetApi())
}

func TestParsedMessage_GetApiCollection(t *testing.T) {
	pm := baseChainMessageContainer{
		apiCollection: &spectypes.ApiCollection{},
	}
	assert.Equal(t, &spectypes.ApiCollection{}, pm.GetApiCollection())
}

func TestParsedMessage_RequestedBlock(t *testing.T) {
	pm := baseChainMessageContainer{
		latestRequestedBlock: 123,
	}
	requestedBlock, _ := pm.RequestedBlock()
	assert.Equal(t, int64(123), requestedBlock)
}

func TestParsedMessage_GetRPCMessage(t *testing.T) {
	rpcInput := &mockRPCInput{}

	pm := baseChainMessageContainer{
		msg: rpcInput,
	}
	assert.Equal(t, rpcInput, pm.GetRPCMessage())
}

type mockRPCInput struct {
	chainproxy.BaseMessage
}

func (m *mockRPCInput) SubscriptionIdExtractor(reply *rpcclient.JsonrpcMessage) string {
	return ""
}

func (m *mockRPCInput) GetRawRequestHash() ([]byte, error) {
	return nil, fmt.Errorf("test")
}

func (m *mockRPCInput) GetParams() interface{} {
	return nil
}

func (m *mockRPCInput) GetResult() json.RawMessage {
	return nil
}

func (m *mockRPCInput) UpdateLatestBlockInMessage(uint64, bool) bool {
	return false
}

func (m *mockRPCInput) ParseBlock(block string) (int64, error) {
	return 0, nil
}

func TestGetServiceApis(t *testing.T) {
	spec := spectypes.Spec{
		Enabled: true,
		ApiCollections: []*spectypes.ApiCollection{
			{
				Enabled: true,
				CollectionData: spectypes.CollectionData{
					ApiInterface: spectypes.APIInterfaceRest,
				},
				Apis: []*spectypes.Api{
					{
						Enabled: true,
						Name:    "test-api",
					},
					{
						Enabled: true,
						Name:    "test-api-2",
					},
					{
						Enabled: false,
						Name:    "test-api-disabled",
					},
					{
						Enabled: true,
						Name:    "test-api-3",
					},
				},
			},
			{
				Enabled: true,
				CollectionData: spectypes.CollectionData{
					ApiInterface: spectypes.APIInterfaceGrpc,
				},
				Apis: []*spectypes.Api{
					{
						Enabled: true,
						Name:    "gtest-api",
					},
					{
						Enabled: true,
						Name:    "gtest-api-2",
					},
					{
						Enabled: false,
						Name:    "gtest-api-disabled",
					},
					{
						Enabled: true,
						Name:    "gtest-api-3",
					},
				},
			},
		},
	}

	rpcInterface := spectypes.APIInterfaceRest
	_, serverApis, _, _, _, _ := getServiceApis(spec, rpcInterface)

	// Test serverApis
	if len(serverApis) != 3 {
		t.Errorf("Expected serverApis length to be 3, but got %d", len(serverApis))
	}
}

func TestCompareRequestedBlockInBatch(t *testing.T) {
	playbook := []struct {
		latest           int64
		earliest         int64
		parsed           int64
		expectedLatest   int64
		expectedEarliest int64
	}{
		{
			latest:           spectypes.LATEST_BLOCK,
			earliest:         spectypes.LATEST_BLOCK,
			parsed:           spectypes.LATEST_BLOCK,
			expectedLatest:   spectypes.LATEST_BLOCK,
			expectedEarliest: spectypes.LATEST_BLOCK,
		},
		{
			latest:           10,
			earliest:         5,
			parsed:           7,
			expectedLatest:   10,
			expectedEarliest: 5,
		},
		{
			latest:           10,
			earliest:         5,
			parsed:           2,
			expectedLatest:   10,
			expectedEarliest: 2,
		},
		{
			latest:           10,
			earliest:         5,
			parsed:           12,
			expectedLatest:   12,
			expectedEarliest: 5,
		},
		{
			latest:           spectypes.LATEST_BLOCK,
			earliest:         5,
			parsed:           10,
			expectedLatest:   spectypes.LATEST_BLOCK,
			expectedEarliest: 5,
		},
		{
			latest:           10,
			earliest:         5,
			parsed:           spectypes.LATEST_BLOCK,
			expectedLatest:   spectypes.LATEST_BLOCK,
			expectedEarliest: 5,
		},
		{
			latest:           10,
			earliest:         5,
			parsed:           spectypes.LATEST_BLOCK,
			expectedLatest:   spectypes.LATEST_BLOCK,
			expectedEarliest: 5,
		},
		{
			latest:           10,
			earliest:         spectypes.EARLIEST_BLOCK,
			parsed:           2,
			expectedLatest:   10,
			expectedEarliest: spectypes.EARLIEST_BLOCK,
		},
		{
			latest:           10,
			earliest:         5,
			parsed:           spectypes.EARLIEST_BLOCK,
			expectedLatest:   10,
			expectedEarliest: spectypes.EARLIEST_BLOCK,
		},
		{
			latest:           spectypes.LATEST_BLOCK,
			earliest:         spectypes.EARLIEST_BLOCK,
			parsed:           5,
			expectedLatest:   spectypes.LATEST_BLOCK,
			expectedEarliest: spectypes.EARLIEST_BLOCK,
		},
		{
			latest:           spectypes.EARLIEST_BLOCK,
			earliest:         spectypes.LATEST_BLOCK,
			parsed:           5,
			expectedLatest:   5,
			expectedEarliest: 5,
		},
		{
			latest:           spectypes.LATEST_BLOCK,
			earliest:         spectypes.EARLIEST_BLOCK,
			parsed:           spectypes.NOT_APPLICABLE,
			expectedLatest:   spectypes.NOT_APPLICABLE,
			expectedEarliest: spectypes.EARLIEST_BLOCK,
		},
		{
			latest:           4,
			earliest:         spectypes.EARLIEST_BLOCK,
			parsed:           spectypes.NOT_APPLICABLE,
			expectedLatest:   spectypes.NOT_APPLICABLE,
			expectedEarliest: spectypes.EARLIEST_BLOCK,
		},
		{
			latest:           4,
			earliest:         2,
			parsed:           spectypes.NOT_APPLICABLE,
			expectedLatest:   spectypes.NOT_APPLICABLE,
			expectedEarliest: spectypes.NOT_APPLICABLE,
		},
	}

	for _, test := range playbook {
		testName := fmt.Sprintf("latest=%d_earliest=%d_parsed=%d", test.latest, test.earliest, test.parsed)
		t.Run(testName, func(t *testing.T) {
			latest, earliest := CompareRequestedBlockInBatch(test.latest, test.earliest, test.parsed)
			require.Equal(t, test.expectedLatest, latest, "latest")
			require.Equal(t, test.expectedEarliest, earliest, "earliest")
		})
	}
}
