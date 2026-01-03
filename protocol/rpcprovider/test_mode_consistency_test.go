package rpcprovider

import (
	"context"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	types "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

// TestUnsupportedMethodResponseGeneration tests that the generateUnsupportedMethodResponse
// method creates appropriate responses for different API interfaces
func TestUnsupportedMethodResponseGeneration(t *testing.T) {
	psm := NewProviderStateMachine("test", nil, nil, 0, nil)

	testCases := []struct {
		name               string
		apiInterface       string
		methodName         string
		expectedInResponse string
		expectedStatusCode int
	}{
		{
			name:               "JSON-RPC unsupported method",
			apiInterface:       "jsonrpc",
			methodName:         "eth_unsupportedMethod",
			expectedInResponse: "-32601", // JSON-RPC method not found code
			expectedStatusCode: 200,
		},
		{
			name:               "REST unsupported endpoint",
			apiInterface:       "rest",
			methodName:         "unknown_endpoint",
			expectedInResponse: "Endpoint not found",
			expectedStatusCode: 404,
		},
		{
			name:               "gRPC unsupported method",
			apiInterface:       "grpc",
			methodName:         "UnknownService",
			expectedInResponse: "Method not implemented",
			expectedStatusCode: 500,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock chain message
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			chainMsg := chainlib.NewMockChainMessage(ctrl)
			apiCollection := &spectypes.ApiCollection{
				CollectionData: spectypes.CollectionData{
					ApiInterface: tc.apiInterface,
				},
			}

			chainMsg.EXPECT().GetApiCollection().Return(apiCollection).AnyTimes()

			// Test the response generation
			responseData, statusCode := psm.generateUnsupportedMethodResponse(chainMsg, tc.methodName, "")

			// Verify the response contains expected patterns
			require.Contains(t, responseData, tc.expectedInResponse,
				"Response should contain expected pattern for %s", tc.apiInterface)
			require.Equal(t, tc.expectedStatusCode, statusCode,
				"Status code should match expected value for %s", tc.apiInterface)
			require.Contains(t, responseData, tc.methodName,
				"Response should contain the method name")
		})
	}
}

// TestTestModeUnsupportedMethodDetection tests that test mode generates responses
// that will be properly detected by the existing unsupported method detection logic
func TestTestModeUnsupportedMethodDetection(t *testing.T) {
	// Create test mode configuration with unsupported method probability = 1.0
	testConfig := &TestModeConfig{
		TestMode: true,
		Responses: map[string]TestResponse{
			"unsupported_method": {
				UnsupportedProbability: 1.0, // Always generate unsupported response
				SuccessProbability:     0.0,
				ErrorProbability:       0.0,
				RateLimitProbability:   0.0,
			},
		},
	}

	psm := NewProviderStateMachine("test", lavaprotocol.NewRelayRetriesManager(), nil, 0, testConfig)

	testCases := []struct {
		name         string
		apiInterface string
		methodName   string
	}{
		{"JSON-RPC unsupported", "jsonrpc", "unsupported_method"},
		{"REST unsupported", "rest", "unsupported_method"},
		{"gRPC unsupported", "grpc", "unsupported_method"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock chain message
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			chainMsg := chainlib.NewMockChainMessage(ctrl)
			api := &spectypes.Api{Name: tc.methodName}
			apiCollection := &spectypes.ApiCollection{
				CollectionData: spectypes.CollectionData{
					ApiInterface: tc.apiInterface,
				},
			}

			chainMsg.EXPECT().GetApi().Return(api).AnyTimes()
			chainMsg.EXPECT().GetApiCollection().Return(apiCollection).AnyTimes()

			// Create test request
			request := &types.RelayRequest{
				RelaySession: &types.RelaySession{
					Provider:  "test_provider",
					SessionId: 123,
				},
			}

			// Generate test response
			ctx := context.WithValue(context.Background(), TestModeContextKey{}, true)
			response := psm.generateTestResponse(ctx, chainMsg, request)

			require.NotNil(t, response, "Response should not be nil")
			require.NotNil(t, response.RelayReply, "RelayReply should not be nil")

			// Verify that the response data contains patterns that will be detected
			// as unsupported methods by the existing detection logic
			responseData := string(response.RelayReply.Data)

			switch tc.apiInterface {
			case "jsonrpc", "tendermintrpc":
				// Should contain JSON-RPC error code -32601
				require.Contains(t, responseData, "-32601",
					"JSON-RPC response should contain method not found error code")
			case "rest":
				// Should have 404 status code for REST
				require.Equal(t, 404, response.StatusCode,
					"REST response should have 404 status code")
				require.Contains(t, responseData, "not found",
					"REST response should contain 'not found' text")
			case "grpc":
				// Should contain gRPC unimplemented patterns
				require.True(t,
					strings.Contains(responseData, "not implemented") ||
						strings.Contains(responseData, "unimplemented"),
					"gRPC response should contain unimplemented patterns")
			}

			// Most importantly, verify that chainlib.IsUnsupportedMethodErrorMessage
			// would detect this as an unsupported method
			if tc.apiInterface != "rest" { // REST relies on status code detection
				isUnsupported := chainlib.IsUnsupportedMethodErrorMessage(responseData)
				require.True(t, isUnsupported,
					"Generated response should be detected as unsupported method by chainlib.IsUnsupportedMethodErrorMessage")
			}
		})
	}
}

// TestConfiguredUnsupportedReply tests that custom configured unsupported replies work
func TestConfiguredUnsupportedReply(t *testing.T) {
	customReply := `{"custom":"unsupported method error"}`

	testConfig := &TestModeConfig{
		TestMode: true,
		Responses: map[string]TestResponse{
			"custom_method": {
				UnsupportedMethodReply: customReply,
				UnsupportedProbability: 1.0,
				SuccessProbability:     0.0,
			},
		},
	}

	psm := NewProviderStateMachine("test", nil, nil, 0, testConfig)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chainMsg := chainlib.NewMockChainMessage(ctrl)
	api := &spectypes.Api{Name: "custom_method"}
	chainMsg.EXPECT().GetApi().Return(api).AnyTimes()

	request := &types.RelayRequest{}

	ctx := context.WithValue(context.Background(), TestModeContextKey{}, true)
	response := psm.generateTestResponse(ctx, chainMsg, request)

	responseData := string(response.RelayReply.Data)
	require.Equal(t, customReply, responseData,
		"Should use configured custom reply when provided")
	require.Equal(t, 500, response.StatusCode,
		"Should use default 500 status code for custom replies")
}

func TestTestModeHeadOnFirstRequestThenGapApplied(t *testing.T) {
	testConfig := &TestModeConfig{
		TestMode:           true,
		HeadBlock:          1000,
		GapBlocks:          50,
		HeadOnFirstRequest: true,
		Responses: map[string]TestResponse{
			"eth_blockNumber": {
				SuccessReply:       `{"jsonrpc":"2.0","id":1,"result":"0x1"}`,
				SuccessProbability: 1.0,
			},
		},
	}

	psm := NewProviderStateMachine("test", lavaprotocol.NewRelayRetriesManager(), nil, 0, testConfig)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chainMsg := chainlib.NewMockChainMessage(ctrl)
	api := &spectypes.Api{Name: "eth_blockNumber"}
	apiCollection := &spectypes.ApiCollection{CollectionData: spectypes.CollectionData{ApiInterface: "jsonrpc"}}
	chainMsg.EXPECT().GetApi().Return(api).AnyTimes()
	chainMsg.EXPECT().GetApiCollection().Return(apiCollection).AnyTimes()

	ctx := context.WithValue(context.Background(), TestModeContextKey{}, true)

	resp1 := psm.generateTestResponse(ctx, chainMsg, &types.RelayRequest{RelaySession: &types.RelaySession{Provider: "p"}})
	require.NotNil(t, resp1)
	require.NotNil(t, resp1.RelayReply)
	require.Equal(t, int64(1000), resp1.RelayReply.LatestBlock)

	resp2 := psm.generateTestResponse(ctx, chainMsg, &types.RelayRequest{RelaySession: &types.RelaySession{Provider: "p"}})
	require.NotNil(t, resp2)
	require.NotNil(t, resp2.RelayReply)
	require.Equal(t, int64(950), resp2.RelayReply.LatestBlock)
}

func floatPtr(f float64) *float64 {
	return &f
}

func TestTestModeAvailabilityZeroAlwaysFailsWithGrpcUnavailable(t *testing.T) {
	testConfig := &TestModeConfig{
		TestMode: true,
		Responses: map[string]TestResponse{
			"eth_blockNumber": {
				Availability:       floatPtr(0.0),
				SuccessReply:       `{"jsonrpc":"2.0","id":1,"result":"0x1"}`,
				SuccessProbability: 1.0,
			},
		},
	}

	psm := NewProviderStateMachine("test", lavaprotocol.NewRelayRetriesManager(), nil, 0, testConfig)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chainMsg := chainlib.NewMockChainMessage(ctrl)
	chainMsg.EXPECT().GetRawRequestHash().Return([]byte{1, 2, 3}, nil).AnyTimes()
	api := &spectypes.Api{Name: "eth_blockNumber"}
	apiCollection := &spectypes.ApiCollection{CollectionData: spectypes.CollectionData{ApiInterface: "jsonrpc"}}
	chainMsg.EXPECT().GetApi().Return(api).AnyTimes()
	chainMsg.EXPECT().GetApiCollection().Return(apiCollection).AnyTimes()

	ctx := context.WithValue(context.Background(), TestModeContextKey{}, true)
	_, _, err := psm.SendNodeMessage(ctx, chainMsg, &types.RelayRequest{RelayData: &types.RelayPrivateData{Extensions: []string{}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "test mode availability failure")
}

func TestTestModeAvailabilityOneAlwaysReturnsReply(t *testing.T) {
	testConfig := &TestModeConfig{
		TestMode: true,
		Responses: map[string]TestResponse{
			"eth_blockNumber": {
				Availability:       floatPtr(1.0),
				SuccessReply:       `{"jsonrpc":"2.0","id":1,"result":"0x1"}`,
				SuccessProbability: 1.0,
			},
		},
	}

	psm := NewProviderStateMachine("test", lavaprotocol.NewRelayRetriesManager(), nil, 0, testConfig)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chainMsg := chainlib.NewMockChainMessage(ctrl)
	chainMsg.EXPECT().GetRawRequestHash().Return([]byte{1, 2, 3}, nil).AnyTimes()
	api := &spectypes.Api{Name: "eth_blockNumber"}
	apiCollection := &spectypes.ApiCollection{CollectionData: spectypes.CollectionData{ApiInterface: "jsonrpc"}}
	chainMsg.EXPECT().GetApi().Return(api).AnyTimes()
	chainMsg.EXPECT().GetApiCollection().Return(apiCollection).AnyTimes()
	chainMsg.EXPECT().CheckResponseError(gomock.Any(), gomock.Any()).Return(false, "").AnyTimes()

	ctx := context.WithValue(context.Background(), TestModeContextKey{}, true)
	reply, _, err := psm.SendNodeMessage(ctx, chainMsg, &types.RelayRequest{RelayData: &types.RelayPrivateData{Extensions: []string{}}})
	require.NoError(t, err)
	require.NotNil(t, reply)
	require.NotNil(t, reply.RelayReply)
	require.Contains(t, string(reply.RelayReply.Data), "\"result\"")
}
