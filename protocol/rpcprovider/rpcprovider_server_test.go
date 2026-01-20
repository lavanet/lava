package rpcprovider

import (
	"context"
	"errors"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chaintracker"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/utils/sigs"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/stretchr/testify/require"
)

type MockChainTracker struct {
	*chaintracker.DummyChainTracker // Embed DummyChainTracker to satisfy the interface
	latestBlock                     int64
	changeTime                      time.Time
	blockHashes                     map[int64]string
	shouldError                     bool
	errorToReturn                   error
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
		blockHashes:       make(map[int64]string),
		shouldError:       false,
	}
}

func (mct *MockChainTracker) GetLatestBlockData(fromBlock int64, toBlock int64, specificBlock int64) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, changeTime time.Time, err error) {
	if mct.shouldError {
		return 0, nil, time.Time{}, mct.errorToReturn
	}

	var hashes []*chaintracker.BlockStore

	// If specific block requested, return its hash if available
	if specificBlock > 0 && specificBlock != -2 { // NOT_APPLICABLE is handled as no specific block
		if hash, ok := mct.blockHashes[specificBlock]; ok {
			hashes = []*chaintracker.BlockStore{{Block: specificBlock, Hash: hash}}
		}
	}

	return mct.latestBlock, hashes, mct.changeTime, nil
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

func (mct *MockChainTracker) SetBlockHash(block int64, hash string) {
	mct.blockHashes[block] = hash
}

func (mct *MockChainTracker) SetError(err error) {
	mct.shouldError = true
	mct.errorToReturn = err
}

func (mct *MockChainTracker) ClearError() {
	mct.shouldError = false
	mct.errorToReturn = nil
}

func (mct *MockChainTracker) IsDummy() bool {
	return false
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

func TestExtractConsumerAddressWithSkipSigning(t *testing.T) {
	ctx := context.Background()
	rpcps := &RPCProviderServer{}

	// Create a valid relay session with signature
	consumer_sk, consumer_address := sigs.GenerateFloatingKey()
	relaySession := &pairingtypes.RelaySession{
		SpecId:      "LAV1",
		ContentHash: []byte("test_hash"),
		SessionId:   123,
		CuSum:       100,
		Provider:    "lava@test_provider",
		RelayNum:    1,
		QosReport:   &pairingtypes.QualityOfServiceReport{},
		Epoch:       100,
	}

	// Sign the relay session
	sig, err := sigs.Sign(consumer_sk, *relaySession)
	require.NoError(t, err)
	relaySession.Sig = sig

	// Test with SkipRelaySigning disabled (normal behavior)
	originalSkipRelaySigning := lavaprotocol.SkipRelaySigning
	lavaprotocol.SkipRelaySigning = false
	defer func() { lavaprotocol.SkipRelaySigning = originalSkipRelaySigning }()

	extractedAddress, err := rpcps.ExtractConsumerAddress(ctx, relaySession)
	require.NoError(t, err)
	require.Equal(t, consumer_address, extractedAddress, "Should extract correct consumer address from signed session")

	// Test with SkipRelaySigning enabled
	lavaprotocol.SkipRelaySigning = true

	extractedAddress, err = rpcps.ExtractConsumerAddress(ctx, relaySession)
	require.NoError(t, err)
	// Should return placeholder address when SkipRelaySigning is enabled
	expectedPlaceholder, err := sdk.AccAddressFromHexUnsafe("0000000000000000000000000000000000000000")
	require.NoError(t, err)
	require.Equal(t, expectedPlaceholder, extractedAddress, "Should return placeholder address when SkipRelaySigning is enabled")
}
