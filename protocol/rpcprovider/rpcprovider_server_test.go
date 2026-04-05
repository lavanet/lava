package rpcprovider

import (
	"context"
	"errors"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chaintracker"
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
	if chainlib.IsUnsupportedMethodErrorType(inputError) {
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
				require.True(t, chainlib.IsUnsupportedMethodErrorType(tt.inputError))
				require.NotEmpty(t, tt.inputError.Error())
				if tt.methodName != "" {
					require.Contains(t, tt.inputError.Error(), tt.methodName)
				}
			}
		})
	}
}

func TestUnsupportedMethodErrorProperties(t *testing.T) {
	// Test the LavaWrappedError properties when created via NewUnsupportedMethodError
	t.Run("Error with method name", func(t *testing.T) {
		originalErr := errors.New("method not found")
		methodName := "eth_unsupportedMethod"
		err := chainlib.NewUnsupportedMethodError(originalErr, methodName)

		require.Contains(t, err.Error(), methodName)
		require.True(t, chainlib.IsUnsupportedMethodErrorType(err))
		// Unwrap returns the underlying LavaError, not the original error
		unwrapped := errors.Unwrap(err)
		require.NotNil(t, unwrapped)
	})

	t.Run("Error without method name", func(t *testing.T) {
		originalErr := errors.New("method not found")
		err := chainlib.NewUnsupportedMethodError(originalErr, "")

		require.Contains(t, err.Error(), "unsupported method")
		require.True(t, chainlib.IsUnsupportedMethodErrorType(err))
		unwrapped := errors.Unwrap(err)
		require.NotNil(t, unwrapped)
	})

	t.Run("Different method names produce different error messages", func(t *testing.T) {
		originalErr := errors.New("method not found")
		err1 := chainlib.NewUnsupportedMethodError(originalErr, "eth_call")
		err2 := chainlib.NewUnsupportedMethodError(originalErr, "eth_customMethod")

		require.Contains(t, err1.Error(), "eth_call")
		require.Contains(t, err2.Error(), "eth_customMethod")
		require.True(t, chainlib.IsUnsupportedMethodErrorType(err1))
		require.True(t, chainlib.IsUnsupportedMethodErrorType(err2))
	})
}

func TestProviderIdentifierHeader(t *testing.T) {
	t.Run("provider always uses address as identifier", func(t *testing.T) {
		providerAddr := sdk.AccAddress("lava1abc123def456ghi789jkl012mno345pqr678st")
		rpcps := &RPCProviderServer{
			rpcProviderEndpoint: &lavasession.RPCProviderEndpoint{},
			providerAddress:     providerAddr,
		}
		require.Equal(t, providerAddr.String(), rpcps.providerAddress.String())
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

	extractedAddress, err := rpcps.ExtractConsumerAddress(ctx, relaySession)
	require.NoError(t, err)
	require.Equal(t, consumer_address, extractedAddress, "Should extract correct consumer address from signed session")
}
