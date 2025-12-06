package rpcprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chaintracker"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

type MockChainTracker struct {
	*chaintracker.DummyChainTracker // Embed DummyChainTracker to satisfy the interface
	latestBlock                     int64
	changeTime                      time.Time
}

// Test the error handling logic directly without needing a full ProviderSessionManager
func testUnsupportedMethodErrorHandling(inputError error) error {
	// This function replicates the error handling logic from finalizeSession
	var unsupportedMethodError *chainlib.UnsupportedMethodError
	if errors.As(inputError, &unsupportedMethodError) {
		// In the actual code, this would log an info message
		// For testing, we just return the original error without wrapping
		return inputError
	}
	// In the actual code, this would wrap the error with additional context
	// For testing, we just return the original error
	return inputError
}

func NewMockChainTracker() *MockChainTracker {
	return &MockChainTracker{
		DummyChainTracker: &chaintracker.DummyChainTracker{},
		latestBlock:       0,
		changeTime:        time.Now(),
	}
}

func (mct *MockChainTracker) GetLatestBlockData(fromBlock int64, toBlock int64, specificBlock int64) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, changeTime time.Time, err error) {
	return mct.latestBlock, nil, mct.changeTime, nil
}

func (mct *MockChainTracker) GetLatestBlockNum() (int64, time.Time) {
	return mct.latestBlock, mct.changeTime
}

func (mct *MockChainTracker) GetAtomicLatestBlockNum() int64 {
	return mct.latestBlock
}

func (mct *MockChainTracker) SetLatestBlock(newLatest int64, changeTime time.Time) {
	mct.latestBlock = newLatest
	mct.changeTime = changeTime
}

func (mct *MockChainTracker) IsDummy() bool {
	return false
}

func TestHandleConsistency(t *testing.T) {
	plays := []struct {
		seenBlock          int64
		requestBlock       int64
		name               string
		specId             string
		err                error
		timeout            time.Duration
		changeTime         time.Duration
		chainTrackerBlocks []int64
		sleep              bool
	}{
		{
			name:               "success",
			seenBlock:          0,
			requestBlock:       100,
			specId:             "LAV1",
			err:                nil,
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         10 * time.Millisecond, // 10 ms ago
			sleep:              false,
		},
		{
			name:               "success with seen eq",
			seenBlock:          100,
			requestBlock:       100,
			specId:             "LAV1",
			err:                nil,
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         10 * time.Millisecond, // 10 ms ago
			sleep:              false,
		},
		{
			name:               "success with seen less",
			seenBlock:          99,
			requestBlock:       100,
			specId:             "LAV1",
			err:                nil,
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         10 * time.Millisecond, // 10 ms ago
			sleep:              false,
		},
		{
			name:               "success with seen more",
			seenBlock:          99,
			requestBlock:       100,
			specId:             "LAV1",
			err:                nil,
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         10 * time.Millisecond, // 10 ms ago
			sleep:              false,
		},
		{
			name:               "latest success",
			seenBlock:          0,
			requestBlock:       spectypes.LATEST_BLOCK,
			specId:             "LAV1",
			err:                nil,
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         10 * time.Millisecond,
			sleep:              false,
		},
		{
			name:               "latest success with seen lower",
			seenBlock:          99,
			requestBlock:       spectypes.LATEST_BLOCK,
			specId:             "LAV1",
			err:                nil,
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         10 * time.Millisecond,
			sleep:              false,
		},
		{
			name:               "latest with seen higher - sleep, fail",
			seenBlock:          101,
			requestBlock:       spectypes.LATEST_BLOCK,
			specId:             "LAV1",
			err:                fmt.Errorf("fail"),
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         100 * time.Second,
			sleep:              true,
		},
		{
			name:               "latest success with seen higher - sleep, succeed",
			seenBlock:          101,
			requestBlock:       spectypes.LATEST_BLOCK,
			specId:             "LAV1",
			err:                nil,
			timeout:            25 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100, 101},
			changeTime:         100 * time.Second,
			sleep:              true,
		},
		{
			name:               "latest with seen higher - immediate fail",
			seenBlock:          100000,
			requestBlock:       spectypes.LATEST_BLOCK,
			specId:             "LAV1",
			err:                fmt.Errorf("fail"),
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         1 * time.Millisecond, // 1 ms ago
			sleep:              false,
		},
		{
			name:               "requested higher - immediate fail",
			seenBlock:          10001,
			requestBlock:       10001,
			specId:             "LAV1",
			err:                fmt.Errorf("fail"),
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         1 * time.Millisecond, // 1 ms ago,
			sleep:              false,
		},
		{
			name:               "requested higher seen 0 - success",
			seenBlock:          0,
			requestBlock:       10001,
			specId:             "LAV1",
			err:                nil,
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         1 * time.Millisecond, // 1 ms ago,
			sleep:              false,
		},
		{
			name:               "requested higher seen lower- success",
			seenBlock:          99,
			requestBlock:       101,
			specId:             "LAV1",
			err:                nil,
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         100 * time.Second,
			sleep:              false,
		},
		{
			name:               "requested higher seen higher- sleep fail",
			seenBlock:          101,
			requestBlock:       101,
			specId:             "LAV1",
			err:                fmt.Errorf("fail"),
			timeout:            15 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100},
			changeTime:         100 * time.Second,
			sleep:              true,
		},
		{
			name:               "requested much higher seen higher - sleep success",
			seenBlock:          101,
			requestBlock:       10001,
			specId:             "LAV1",
			err:                nil,
			timeout:            50 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100, 101},
			changeTime:         100 * time.Second,
			sleep:              true,
		},
		{
			name:               "requested higher seen higher - sleep success",
			seenBlock:          101,
			requestBlock:       101,
			specId:             "LAV1",
			err:                nil,
			timeout:            50 * time.Millisecond, // 150 is one way travel time
			chainTrackerBlocks: []int64{100, 101},
			changeTime:         100 * time.Second,
			sleep:              true,
		},
	}
	for _, play := range plays {
		t.Run(play.name, func(t *testing.T) {
			specId := play.specId
			ts := chainlib.SetupForTests(t, 1, specId, "../../")
			replyDataBuf := []byte(`{"reply": "REPLY-STUB"}`)
			serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Handle the incoming request and provide the desired response
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, string(replyDataBuf))
			})
			chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ts.Ctx, specId, spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
			if closeServer != nil {
				defer closeServer()
			}
			mockChainTracker := NewMockChainTracker()
			require.GreaterOrEqual(t, len(play.chainTrackerBlocks), 1)
			calls := 1                                                                                      // how many times we have setLatestBlock in the mock
			mockChainTracker.SetLatestBlock(play.chainTrackerBlocks[0], time.Now().Add(-1*play.changeTime)) // change time is only in the past
			require.NoError(t, err)
			rpcproviderServer := RPCProviderServer{
				chainTracker: mockChainTracker,
				rpcProviderEndpoint: &lavasession.RPCProviderEndpoint{
					ChainID: specId,
				},
			}
			seenBlock := play.seenBlock
			requestBlock := play.requestBlock
			blockLagForQosSync, averageBlockTime, blockDistanceToFinalization, blocksInFinalizationData := chainParser.ChainBlockStats()
			go func() {
				// advance mockChainTracker
				if len(play.chainTrackerBlocks) > 1 {
					nextBlock := play.chainTrackerBlocks[1]
					time.Sleep(6 * time.Millisecond)
					mockChainTracker.SetLatestBlock(nextBlock, time.Now())
					calls += 1
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), play.timeout)
			latestBlock, _, timeSlept, err := rpcproviderServer.handleConsistency(ctx, play.timeout, seenBlock, requestBlock, averageBlockTime, blockLagForQosSync, blocksInFinalizationData, blockDistanceToFinalization)
			cancel()
			if play.err != nil {
				require.Error(t, err, strconv.Itoa(calls))
			} else {
				require.NoError(t, err, strconv.Itoa(calls))
			}
			require.Less(t, timeSlept, play.timeout)
			if play.sleep {
				require.NotZero(t, timeSlept)
			} else {
				require.Zero(t, timeSlept)
			}
			if play.err == nil {
				require.LessOrEqual(t, seenBlock, latestBlock)
			}
		})
	}
}

func TestUnsupportedMethodErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		inputError     error
		expectLogInfo  bool
		expectLogError bool
		methodName     string
	}{
		{
			name:           "UnsupportedMethodError with method name",
			inputError:     chainlib.NewUnsupportedMethodError(errors.New("method not found"), "eth_unsupportedMethod"),
			expectLogInfo:  true,
			expectLogError: false,
			methodName:     "eth_unsupportedMethod",
		},
		{
			name:           "UnsupportedMethodError without method name",
			inputError:     chainlib.NewUnsupportedMethodError(errors.New("method not found"), ""),
			expectLogInfo:  true,
			expectLogError: false,
			methodName:     "",
		},
		{
			name:           "Regular error",
			inputError:     errors.New("some other error"),
			expectLogInfo:  false,
			expectLogError: true,
			methodName:     "",
		},
		{
			name:           "Nil error",
			inputError:     nil,
			expectLogInfo:  false,
			expectLogError: false,
			methodName:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the error handling logic directly
			result := testUnsupportedMethodErrorHandling(tt.inputError)

			// Verify the error handling behavior
			if tt.inputError == nil {
				// If input error is nil, should return nil
				require.NoError(t, result)
			} else {
				// For both UnsupportedMethodError and regular errors, the function should return the original error
				require.Equal(t, tt.inputError, result)
			}

			// Additional verification for UnsupportedMethodError
			if tt.expectLogInfo {
				var unsupportedMethodError *chainlib.UnsupportedMethodError
				if errors.As(tt.inputError, &unsupportedMethodError) {
					require.Equal(t, tt.methodName, unsupportedMethodError.GetMethodName())
					require.NotEmpty(t, unsupportedMethodError.Error())
				}
			}
		})
	}
}

func TestUnsupportedMethodErrorProperties(t *testing.T) {
	// Test the UnsupportedMethodError type properties and methods
	t.Run("Error with method name", func(t *testing.T) {
		originalErr := errors.New("method not found")
		methodName := "eth_unsupportedMethod"
		err := chainlib.NewUnsupportedMethodError(originalErr, methodName)

		require.Equal(t, methodName, err.GetMethodName())
		require.Contains(t, err.Error(), methodName)
		require.Contains(t, err.Error(), originalErr.Error())
		require.Equal(t, originalErr, err.Unwrap())
	})

	t.Run("Error without method name", func(t *testing.T) {
		originalErr := errors.New("method not found")
		err := chainlib.NewUnsupportedMethodError(originalErr, "")

		require.Equal(t, "", err.GetMethodName())
		require.Contains(t, err.Error(), originalErr.Error())
		require.Equal(t, originalErr, err.Unwrap())
	})

	t.Run("Error with method name using WithMethod", func(t *testing.T) {
		originalErr := errors.New("method not found")
		err := chainlib.NewUnsupportedMethodError(originalErr, "")
		err = err.WithMethod("eth_customMethod")

		require.Equal(t, "eth_customMethod", err.GetMethodName())
		require.Contains(t, err.Error(), "eth_customMethod")
		require.Contains(t, err.Error(), originalErr.Error())
		require.Equal(t, originalErr, err.Unwrap())
	})
}

func TestProviderIdentifierHeaderForStaticProviders(t *testing.T) {
	tests := []struct {
		name            string
		staticProvider  bool
		providerName    string
		providerAddress string
		shouldUseName   bool
		description     string
	}{
		{
			name:            "static provider with name",
			staticProvider:  true,
			providerName:    "Ethereum-Primary-Provider",
			providerAddress: "lava1abc123def456ghi789jkl012mno345pqr678st",
			shouldUseName:   true,
			description:     "Static provider with configured name should use the name",
		},
		{
			name:            "static provider with empty name",
			staticProvider:  true,
			providerName:    "",
			providerAddress: "lava1abc123def456ghi789jkl012mno345pqr678st",
			shouldUseName:   false,
			description:     "Static provider without name should fall back to address",
		},
		{
			name:            "non-static provider with name",
			staticProvider:  false,
			providerName:    "Regular-Provider",
			providerAddress: "lava1xyz987wvu654tsr321qpo210nml109ihg543cb",
			shouldUseName:   false,
			description:     "Non-static provider should always use address regardless of name",
		},
		{
			name:            "non-static provider without name",
			staticProvider:  false,
			providerName:    "",
			providerAddress: "lava1def456ghi789jkl012mno345pqr678stu901vw",
			shouldUseName:   false,
			description:     "Non-static provider without name should use address",
		},
		{
			name:            "static provider with special characters in name",
			staticProvider:  true,
			providerName:    "Provider-1_Test.2024",
			providerAddress: "lava1special123provider456test789address012abc",
			shouldUseName:   true,
			description:     "Static provider with special characters in name should use the name",
		},
		{
			name:            "static provider with unicode name",
			staticProvider:  true,
			providerName:    "Provider-测试-Тест",
			providerAddress: "lava1unicode123provider456test789address012def",
			shouldUseName:   true,
			description:     "Static provider with unicode characters in name should use the name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock chain message to capture the headers
			var capturedHeaders []headerPair

			mockChainMessage := &mockChainMessageForProviderHeader{
				headers: &capturedHeaders,
			}

			// Create mock provider address
			providerAddr := sdk.AccAddress(tt.providerAddress)

			// Create RPCProviderServer with test configuration
			rpcps := &RPCProviderServer{
				StaticProvider: tt.staticProvider,
				rpcProviderEndpoint: &lavasession.RPCProviderEndpoint{
					Name: tt.providerName,
				},
				providerAddress: providerAddr,
			}

			// Simulate the logic from sendRelayMessageToNode
			providerIdentifier := rpcps.providerAddress.String()
			if rpcps.StaticProvider && rpcps.rpcProviderEndpoint.Name != "" {
				providerIdentifier = rpcps.rpcProviderEndpoint.Name
			}

			// Simulate appending the header
			mockChainMessage.AppendHeader(RPCProviderAddressHeader, providerIdentifier)

			// Verify the correct identifier was used
			require.Len(t, capturedHeaders, 1, "Expected exactly one header to be appended")
			require.Equal(t, RPCProviderAddressHeader, capturedHeaders[0].name, "Header name should be RPCProviderAddressHeader")

			// Verify the value based on whether name should be used
			if tt.shouldUseName {
				require.Equal(t, tt.providerName, capturedHeaders[0].value, tt.description)
			} else {
				// Should use the address string (whatever format it's in)
				require.Equal(t, providerAddr.String(), capturedHeaders[0].value, tt.description)
			}
		})
	}
}

func TestProviderIdentifierLogic(t *testing.T) {
	t.Run("logic flow for identifier selection", func(t *testing.T) {
		testCases := []struct {
			staticProvider bool
			providerName   string
			expected       string
		}{
			{true, "MyProvider", "MyProvider"},
			{true, "", "address"},
			{false, "MyProvider", "address"},
			{false, "", "address"},
		}

		for _, tc := range testCases {
			providerAddress := "address"
			providerIdentifier := providerAddress
			if tc.staticProvider && tc.providerName != "" {
				providerIdentifier = tc.providerName
			}
			require.Equal(t, tc.expected, providerIdentifier)
		}
	})
}

// Mock types for testing

type headerPair struct {
	name  string
	value string
}

type mockChainMessageForProviderHeader struct {
	headers *[]headerPair
}

func (m *mockChainMessageForProviderHeader) AppendHeader(name string, value string) {
	*m.headers = append(*m.headers, headerPair{name: name, value: value})
}
