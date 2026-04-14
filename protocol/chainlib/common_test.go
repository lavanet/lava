package chainlib

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	websocket2 "github.com/gorilla/websocket"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/common"
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

func TestCheckUTXOResponseAndFixReply(t *testing.T) {
	t.Run("single_response_preserves_error_null", func(t *testing.T) {
		// BTC-family nodes return "error":null; the relay pipeline uses omitempty which strips it
		input := `{"jsonrpc":"2.0","id":"1","result":"abc","error":null}`
		result := checkUTXOResponseAndFixReply("DOGE", []byte(input))
		// Should preserve error:null and strip jsonrpc (BTC uses JSON-RPC 1.0)
		var parsed map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(result), &parsed))
		_, hasError := parsed["error"]
		require.True(t, hasError, "error field must be present (even when null)")
		_, hasJsonrpc := parsed["jsonrpc"]
		require.False(t, hasJsonrpc, "jsonrpc field should be stripped for BTC-family chains")
	})

	t.Run("batch_response_preserves_error_null", func(t *testing.T) {
		// Multi-element batch: relay pipeline reconstructs with jsonrpc:"2.0" and omitempty on error
		input := `[{"jsonrpc":"2.0","id":"1","result":"hash1"},{"jsonrpc":"2.0","id":"2","result":"hash2"}]`
		result := checkUTXOResponseAndFixReply("DOGE", []byte(input))
		var parsed []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(result), &parsed))
		require.Len(t, parsed, 2)
		for i, elem := range parsed {
			_, hasError := elem["error"]
			require.True(t, hasError, "batch element %d must have error field", i)
			_, hasJsonrpc := elem["jsonrpc"]
			require.False(t, hasJsonrpc, "batch element %d should not have jsonrpc field", i)
		}
	})

	t.Run("single_element_batch_response", func(t *testing.T) {
		// Single-element batch must stay as array
		input := `[{"jsonrpc":"2.0","id":"1773768178254-0","result":"23699c7e"}]`
		result := checkUTXOResponseAndFixReply("DOGE", []byte(input))
		var parsed []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(result), &parsed), "single-element batch must remain an array")
		require.Len(t, parsed, 1)
		require.Equal(t, "1773768178254-0", parsed[0]["id"])
		_, hasError := parsed[0]["error"]
		require.True(t, hasError, "error field must be present")
		_, hasJsonrpc := parsed[0]["jsonrpc"]
		require.False(t, hasJsonrpc, "jsonrpc field should be stripped")
	})

	t.Run("non_btc_chain_passthrough", func(t *testing.T) {
		input := `{"jsonrpc":"2.0","id":1,"result":"abc"}`
		result := checkUTXOResponseAndFixReply("ETH1", []byte(input))
		require.Equal(t, input, result, "non-BTC chains should pass through unchanged")
	})

	t.Run("btc_with_actual_error", func(t *testing.T) {
		input := `{"id":"1","error":{"code":-8,"message":"Block height out of range"},"result":null}`
		result := checkUTXOResponseAndFixReply("BTC", []byte(input))
		var parsed map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(result), &parsed))
		errorField := parsed["error"]
		require.NotNil(t, errorField, "error field must be preserved when not null")
	})
}

func TestStripBrotliAcceptEncoding(t *testing.T) {
	cases := []struct {
		name, input, want string
		// wantPresent distinguishes "header absent" (fasthttp.Peek == nil) from
		// "header present with empty value" — the two cases are semantically
		// different for downstream content negotiation.
		wantPresent bool
	}{
		{"br_only_deletes_header", "br", "", false},
		{"br_with_gzip_and_deflate", "br, gzip, deflate", " gzip, deflate", true},
		{"br_with_qvalues", "br;q=1.0, gzip;q=0.8", " gzip;q=0.8", true},
		{"gzip_only_no_op", "gzip, deflate", "gzip, deflate", true},
		{"uppercase_BR_stripped", "BR, gzip", " gzip", true},
		{"br_not_a_token_preserved", "gzip, xbr, deflate", "gzip, xbr, deflate", true},
		{"empty_header_no_op", "", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			var seen string
			var present bool
			app.Use(stripBrotliAcceptEncoding)
			app.Get("/", func(c *fiber.Ctx) error {
				seen = c.Get(fiber.HeaderAcceptEncoding)
				present = c.Request().Header.Peek(fiber.HeaderAcceptEncoding) != nil
				return c.SendStatus(fiber.StatusOK)
			})

			req := httptest.NewRequest(fiber.MethodGet, "/", nil)
			if tc.input != "" {
				req.Header.Set(fiber.HeaderAcceptEncoding, tc.input)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, tc.want, seen)
			require.Equal(t, tc.wantPresent, present, "header presence mismatch (absent vs empty-value)")
		})
	}
}

func TestApplyResponseCompression(t *testing.T) {
	// Payload large enough to exceed fasthttp's built-in minimum compression threshold.
	payload := []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":"%s"}`, strings.Repeat("a", 4096)))

	cases := []struct {
		name             string
		mode             string
		acceptEncoding   string
		wantEncoding     string // "" means no Content-Encoding header
		wantBodyPassThru bool   // true means response body should equal payload byte-for-byte
	}{
		{"off_mode_no_compression", common.ResponseCompressionOff, "br, gzip, deflate", "", true},
		{"brotli_mode_encodes_br_when_advertised", common.ResponseCompressionBrotli, "br, gzip", "br", false},
		{"brotli_mode_falls_back_to_gzip_when_br_absent", common.ResponseCompressionBrotli, "gzip, deflate", "gzip", false},
		{"gzip_mode_strips_br_and_falls_back_to_gzip", common.ResponseCompressionGzip, "br, gzip, deflate", "gzip", false},
		{"gzip_mode_with_no_client_br_still_uses_gzip", common.ResponseCompressionGzip, "gzip", "gzip", false},
		{"unknown_mode_defaults_to_gzip", "something-unknown", "br, gzip", "gzip", false},
		{"empty_mode_defaults_to_gzip", "", "br, gzip", "gzip", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			applyResponseCompression(app, tc.mode)
			app.Get("/", func(c *fiber.Ctx) error {
				c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
				return c.Send(payload)
			})

			req := httptest.NewRequest(fiber.MethodGet, "/", nil)
			req.Header.Set(fiber.HeaderAcceptEncoding, tc.acceptEncoding)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, tc.wantEncoding, resp.Header.Get(fiber.HeaderContentEncoding),
				"unexpected Content-Encoding")

			if tc.wantBodyPassThru {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.Equal(t, payload, body, "off mode must return raw bytes")
			}
		})
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
