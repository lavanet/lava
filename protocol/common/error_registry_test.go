package common

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Registry integrity tests
// ---------------------------------------------------------------------------

func TestRegistryNoDuplicateCodes(t *testing.T) {
	// registerError panics on duplicate codes at init time,
	// but let's also verify the registry is consistent.
	seen := make(map[uint32]string)
	for code, le := range errorRegistry {
		if existing, ok := seen[code]; ok {
			t.Errorf("duplicate code %d: %s and %s", code, existing, le.Name)
		}
		seen[code] = le.Name
		assert.Equal(t, code, le.Code, "registry key %d doesn't match error code %d (%s)", code, le.Code, le.Name)
	}
}

func TestRegistryNoDuplicateNames(t *testing.T) {
	seen := make(map[string]uint32)
	for _, le := range errorRegistry {
		if existingCode, ok := seen[le.Name]; ok {
			t.Errorf("duplicate name %s: codes %d and %d", le.Name, existingCode, le.Code)
		}
		seen[le.Name] = le.Code
	}
}

func TestRegistryCodeRanges(t *testing.T) {
	for _, le := range errorRegistry {
		if le.Code == 0 {
			continue // UNKNOWN_ERROR
		}
		switch {
		case le.Code >= 1000 && le.Code < 2000:
			assert.Equal(t, CategoryInternal, le.Category,
				"code %d (%s) is in protocol range but not CategoryInternal", le.Code, le.Name)
		case le.Code >= 2000 && le.Code < 5000:
			assert.Equal(t, CategoryExternal, le.Category,
				"code %d (%s) is in external range but not CategoryExternal", le.Code, le.Name)
		default:
			t.Errorf("code %d (%s) is outside valid ranges (1000-4999)", le.Code, le.Name)
		}
	}
}

func TestAllErrorCodesRegistered(t *testing.T) {
	// Spot-check that key error codes are in the registry
	// Verify UNKNOWN_ERROR is registered at code 0
	assert.Equal(t, LavaErrorUnknown, getLavaError(0))
	assert.Equal(t, "UNKNOWN_ERROR", getLavaError(0).Name)

	codes := []uint32{
		1001, // CONNECTION_TIMEOUT
		1002, // CONNECTION_REFUSED
		2001, // METHOD_NOT_FOUND
		2005, // RATE_LIMITED
		3001, // NONCE_TOO_LOW
		3101, // EXECUTION_REVERTED
		4001, // PARSE_ERROR
	}
	for _, code := range codes {
		le := getLavaError(code)
		assert.NotEqual(t, LavaErrorUnknown, le, "code %d should be registered", code)
		assert.Equal(t, code, le.Code)
	}
}

// ---------------------------------------------------------------------------
// Lookup helper tests
// ---------------------------------------------------------------------------

func TestErrorRegistry_GetLavaErrorByCode(t *testing.T) {
	le := getLavaError(1001)
	assert.Equal(t, "PROTOCOL_CONNECTION_TIMEOUT", le.Name)

	le = getLavaError(99999)
	assert.Equal(t, LavaErrorUnknown, le)
}

func TestErrorRegistry_GetLavaErrorByName(t *testing.T) {
	le := getLavaErrorByName("PROTOCOL_CONNECTION_TIMEOUT")
	assert.Equal(t, uint32(1001), le.Code)

	le = getLavaErrorByName("NONEXISTENT")
	assert.Equal(t, LavaErrorUnknown, le)
}

func TestErrorRegistry_IsRetryableStates(t *testing.T) {
	assert.True(t, isRetryable(1001))  // CONNECTION_TIMEOUT
	assert.False(t, isRetryable(1004)) // TLS_MISMATCH
	assert.False(t, isRetryable(3001)) // NONCE_TOO_LOW
	assert.True(t, isRetryable(0))     // UNKNOWN — retryable by default
}

func TestErrorRegistry_IsInternalExternalFlags(t *testing.T) {
	assert.True(t, IsInternal(1001))  // PROTOCOL_CONNECTION_TIMEOUT
	assert.False(t, IsExternal(1001)) // not external

	assert.True(t, IsExternal(2001))  // NODE_METHOD_NOT_FOUND
	assert.False(t, IsInternal(2001)) // not internal

	assert.True(t, IsExternal(3001)) // CHAIN_NONCE_TOO_LOW
	assert.True(t, IsExternal(4001)) // USER_PARSE_ERROR
	assert.True(t, IsExternal(0))    // UNKNOWN — external
}

// ---------------------------------------------------------------------------
// LavaError as error interface + errors.Is tests
// ---------------------------------------------------------------------------

func TestLavaError_String(t *testing.T) {
	assert.Equal(t, "[3001] CHAIN_NONCE_TOO_LOW", LavaErrorChainNonceTooLow.String())
}

func TestLavaError_ABCICode(t *testing.T) {
	assert.Equal(t, uint32(3001), LavaErrorChainNonceTooLow.ABCICode())
	assert.Equal(t, uint32(1001), LavaErrorConnectionTimeout.ABCICode())
	assert.Equal(t, uint32(0), LavaErrorUnknown.ABCICode())
}

func TestLavaError_IsNonMatch(t *testing.T) {
	// Is returns false for non-LavaError targets
	assert.False(t, LavaErrorChainNonceTooLow.Is(errors.New("not a LavaError")))
}

func TestLavaWrappedError_EmptyContext(t *testing.T) {
	wrapped := NewLavaError(LavaErrorChainNonceTooLow, "")
	assert.Contains(t, wrapped.Error(), "CHAIN_NONCE_TOO_LOW")
}

func TestLavaWrappedError_IsNonMatch(t *testing.T) {
	wrapped := NewLavaError(LavaErrorChainNonceTooLow, "context")
	// Is returns false for non-LavaError targets
	assert.False(t, errors.Is(wrapped, errors.New("not a LavaError")))
}

func TestLavaWrappedError_Unwrap(t *testing.T) {
	wrapped := NewLavaError(LavaErrorChainNonceTooLow, "context")
	unwrapped := errors.Unwrap(wrapped)
	require.NotNil(t, unwrapped)
	assert.Equal(t, LavaErrorChainNonceTooLow, unwrapped)
}

func TestLavaError_ErrorInterface(t *testing.T) {
	var err error = LavaErrorChainNonceTooLow
	assert.Contains(t, err.Error(), "CHAIN_NONCE_TOO_LOW")
}

func TestLavaError_ErrorsIs(t *testing.T) {
	// Direct match
	assert.True(t, errors.Is(LavaErrorChainNonceTooLow, LavaErrorChainNonceTooLow))
	assert.False(t, errors.Is(LavaErrorChainNonceTooLow, LavaErrorConnectionTimeout))

	// Wrapped with NewLavaError
	wrapped := NewLavaError(LavaErrorChainNonceTooLow, "tx failed")
	assert.True(t, errors.Is(wrapped, LavaErrorChainNonceTooLow))
	assert.False(t, errors.Is(wrapped, LavaErrorConnectionTimeout))
	assert.Contains(t, wrapped.Error(), "tx failed")

	// Wrapped with fmt.Errorf %w
	doubleWrapped := fmt.Errorf("relay error: %w", wrapped)
	assert.True(t, errors.Is(doubleWrapped, LavaErrorChainNonceTooLow))
}

// ---------------------------------------------------------------------------
// ErrorCategory / ErrorSubCategory tests
// ---------------------------------------------------------------------------

func TestErrorCategoryString(t *testing.T) {
	assert.Equal(t, "internal", CategoryInternal.String())
	assert.Equal(t, "external", CategoryExternal.String())
	assert.Equal(t, "unknown", ErrorCategory(99).String())
}

func TestErrorSubCategoryString(t *testing.T) {
	assert.Equal(t, "none", SubCategoryNone.String())
	assert.Equal(t, "unsupported_method", SubCategoryUnsupportedMethod.String())
}

func TestUnsupportedMethodSubCategory(t *testing.T) {
	// 2002 (NODE_METHOD_NOT_SUPPORTED) is intentionally excluded: it's retryable on another provider
	unsupportedCodes := []uint32{2001, 2008, 2009, 2010}
	for _, code := range unsupportedCodes {
		le := getLavaError(code)
		require.NotEqual(t, LavaErrorUnknown, le, "code %d not registered", code)
		assert.True(t, le.SubCategory.IsUnsupportedMethod(),
			"code %d (%s) should be SubCategoryUnsupportedMethod", code, le.Name)
	}

	// Non-unsupported codes should not be
	normalCodes := []uint32{1001, 2003, 2005, 3001, 4001}
	for _, code := range normalCodes {
		le := getLavaError(code)
		assert.False(t, le.SubCategory.IsUnsupportedMethod(),
			"code %d (%s) should NOT be SubCategoryUnsupportedMethod", code, le.Name)
	}
}

// ---------------------------------------------------------------------------
// ChainFamily tests
// ---------------------------------------------------------------------------

func TestChainFamily_GetByChainID(t *testing.T) {
	tests := []struct {
		chainID  string
		expected ChainFamily
		found    bool
	}{
		{"ETH1", ChainFamilyEVM, true},
		{"ARBITRUM", ChainFamilyEVM, true},
		{"SOLANA", ChainFamilySolana, true},
		{"BTC", ChainFamilyBitcoin, true},
		{"COSMOSHUB", ChainFamilyCosmosSDK, true},
		{"STRK", ChainFamilyStarknet, true},
		{"APT1", ChainFamilyAptos, true},
		{"NEAR", ChainFamilyNEAR, true},
		{"XRP", ChainFamilyXRP, true},
		{"XLM", ChainFamilyStellar, true},
		{"TON", ChainFamilyTON, true},
		{"CARDANO", ChainFamilyCardano, true},
		{"TRX", ChainFamilyTron, true},
		{"EVMOS", ChainFamilyCosmosSDK, true},
		// Testnets
		{"SOLANAT", ChainFamilySolana, true},
		{"BTCT", ChainFamilyBitcoin, true},
		{"NEART", ChainFamilyNEAR, true},
		// Unknown chain
		{"UNKNOWN_CHAIN", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.chainID, func(t *testing.T) {
			family, ok := GetChainFamily(tt.chainID)
			assert.Equal(t, tt.found, ok)
			if ok {
				assert.Equal(t, tt.expected, family)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Matcher tests
// ---------------------------------------------------------------------------

func TestCodeEqualsMatcher(t *testing.T) {
	m := CodeEquals(-32601)
	assert.True(t, m.Matches(-32601, "anything"))
	assert.False(t, m.Matches(-32600, "anything"))
	assert.False(t, m.Matches(0, "anything"))
}

func TestMessageContainsMatcher(t *testing.T) {
	m := MessageContains("nonce too low")
	assert.True(t, m.Matches(0, "nonce too low"))
	assert.True(t, m.Matches(0, "Error: Nonce Too Low for account"))
	assert.True(t, m.Matches(0, "NONCE TOO LOW"))
	assert.False(t, m.Matches(0, "nonce is fine"))
}

func TestMessageRegexMatcher(t *testing.T) {
	m := MessageRegex(`missing.*trie node`)
	assert.True(t, m.Matches(0, "missing trie node abc123"))
	assert.True(t, m.Matches(0, "missing intermediate trie node"))
	assert.False(t, m.Matches(0, "trie node missing"))
}

func TestHTTPStatusContainsMatcher(t *testing.T) {
	m := HTTPStatusContains(429)
	assert.True(t, m.Matches(0, "HTTP status 429: Too Many Requests"))
	assert.False(t, m.Matches(0, "HTTP status 200: OK"))

	// Boundary checks — false-positive defense
	m500 := HTTPStatusContains(500)
	assert.True(t, m500.Matches(0, "HTTP status 500: Internal Server Error"))
	assert.False(t, m500.Matches(0, "block 25001500"), "should not match 500 inside 25001500")
	assert.False(t, m500.Matches(0, "error code 5001"), "should not match 500 with trailing digit")
	assert.False(t, m500.Matches(0, "tx 1500 confirmed"), "should not match 500 with leading digit")
	assert.False(t, m500.Matches(0, "slot 15003 was skipped"), "should not match 500 embedded in 15003")
}

func TestGRPCCodeEqualsMatcher(t *testing.T) {
	m := GRPCCodeEquals(14) // codes.Unavailable
	assert.True(t, m.Matches(14, "anything"))
	assert.False(t, m.Matches(12, "anything"))
}

// ---------------------------------------------------------------------------
// ClassifyError tests
// ---------------------------------------------------------------------------

type timeoutNetError struct{}

func (e *timeoutNetError) Error() string   { return "mock net error" }
func (e *timeoutNetError) Timeout() bool   { return true }
func (e *timeoutNetError) Temporary() bool { return true }

func TestClassifyError_ConnectionErrorTakesPrecedence(t *testing.T) {
	result := ClassifyError(LavaErrorConnectionTimeout, ChainFamilyEVM, TransportJsonRPC, -32601, "method not found")
	assert.Equal(t, LavaErrorConnectionTimeout, result, "connection error should take precedence over message match")
}

func TestClassifyError_ConnectionErrorNeverUnknown(t *testing.T) {
	// A non-nil connectionError must NEVER produce LavaErrorUnknown, regardless
	// of what the message/code matchers would return for the same input.
	connectionErrors := []*LavaError{
		LavaErrorConnectionTimeout,
		LavaErrorConnectionRefused,
		LavaErrorContextDeadline,
		LavaErrorContextCanceled,
	}
	for _, connErr := range connectionErrors {
		result := ClassifyError(connErr, ChainFamilyEVM, TransportJsonRPC, 0, "some unrecognized error message")
		assert.NotEqual(t, LavaErrorUnknown, result,
			"ClassifyError with connectionError=%s must not return LavaErrorUnknown", connErr.Name)
		assert.Equal(t, connErr, result,
			"ClassifyError must return the connectionError unchanged")
	}
}

func TestDetectConnectionError_NeverProducesUnknown(t *testing.T) {
	// For any error that DetectConnectionError recognises, the full classification
	// pipeline must not produce LavaErrorUnknown.
	testCases := []struct {
		name string
		err  error
	}{
		{"context.Canceled", context.Canceled},
		{"context.DeadlineExceeded", context.DeadlineExceeded},
		{"string: context deadline exceeded", fmt.Errorf("http request failed: context deadline exceeded: Context deadline exceeded")},
		{"string: context canceled", fmt.Errorf("operation failed: context canceled")},
		{"net.Error timeout", &timeoutNetError{}},
		{"http2 GOAWAY PROTOCOL_ERROR", fmt.Errorf(`http request failed: Post "https://base.lava.build:443/": http2: server sent GOAWAY and closed the connection; LastStreamID=1857, ErrCode=PROTOCOL_ERROR, debug=""`)},
		{"http2 GOAWAY INTERNAL_ERROR", fmt.Errorf(`http2: server sent GOAWAY and closed the connection; ErrCode=INTERNAL_ERROR, debug=""`)},
		{"http2 RST_STREAM INTERNAL_ERROR", fmt.Errorf(`failed reading response: stream error: stream ID 77; INTERNAL_ERROR; received from peer`)},
		{"503 envoy connection refused", fmt.Errorf(`503 Service Unavailable: {"error":{"code":503,"message":"upstream connect error or disconnect/reset before headers. retried and the latest reset reason: remote connection failure, transport failure reason: delayed connect error: Connection refused","status":"UNAVAILABLE"}}`)},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			connErr := DetectConnectionError(tc.err)
			require.NotNil(t, connErr, "DetectConnectionError should recognise this error")
			result := ClassifyError(connErr, ChainFamilyEVM, TransportJsonRPC, 0, tc.err.Error())
			assert.NotEqual(t, LavaErrorUnknown, result,
				"classification pipeline must not produce UNKNOWN_ERROR for a recognised connection error")
		})
	}
}

func TestDetectConnectionError_GOAWAYEnhanceYourCalmNotCaught(t *testing.T) {
	// ENHANCE_YOUR_CALM GOAWAY is a rate-limit signal — DetectConnectionError must NOT catch it
	// so that transport-specific matchers can classify it as LavaErrorNodeRateLimited.
	err := fmt.Errorf(`http2: server sent GOAWAY and closed the connection; ErrCode=ENHANCE_YOUR_CALM, debug=""`)
	connErr := DetectConnectionError(err)
	assert.Nil(t, connErr, "ENHANCE_YOUR_CALM GOAWAY should not be caught by DetectConnectionError")
}

func TestClassifyError_Tier2BeforeTier1(t *testing.T) {
	// Solana -32005 should match Tier 2 (NODE_SOLANA_UNHEALTHY) not Tier 1 (NODE_LIMIT_EXCEEDED)
	result := ClassifyError(nil, ChainFamilySolana, TransportJsonRPC, -32005, "Node is unhealthy")
	assert.Equal(t, LavaErrorNodeSolanaUnhealthy, result)
}

func TestClassifyError_Tier1Fallback(t *testing.T) {
	// Non-Solana chain with -32601 should match Tier 1 generic
	result := ClassifyError(nil, ChainFamilyEVM, TransportJsonRPC, -32601, "method not found")
	assert.Equal(t, LavaErrorNodeMethodNotFound, result)
}

func TestClassifyError_UnknownFallback(t *testing.T) {
	result := ClassifyError(nil, ChainFamilyEVM, TransportJsonRPC, 0, "some random error")
	assert.Equal(t, LavaErrorUnknown, result)
}

func TestClassifyError_TransportScoping(t *testing.T) {
	// HTTP 429 in JSON-RPC transport should match (we added HTTP matchers to JSON-RPC)
	result := ClassifyError(nil, ChainFamilyEVM, TransportJsonRPC, 0, "HTTP status 429")
	assert.Equal(t, LavaErrorNodeRateLimited, result)

	// HTTP 429 in REST transport should also match
	result = ClassifyError(nil, ChainFamilyAptos, TransportREST, 0, "HTTP status 429")
	assert.Equal(t, LavaErrorNodeRateLimited, result)

	// gRPC code 14 (Unavailable) should match in gRPC transport
	result = ClassifyError(nil, ChainFamilyCosmosSDK, TransportGRPC, 14, "service unavailable")
	assert.Equal(t, LavaErrorNodeServiceUnavailable, result)

	// gRPC code should NOT match in JSON-RPC transport
	result = ClassifyError(nil, ChainFamilyEVM, TransportJsonRPC, 14, "service unavailable")
	assert.NotEqual(t, LavaErrorNodeServiceUnavailable, result)
}

func TestClassifyError_ChainSpecificMappings(t *testing.T) {
	tests := []struct {
		name     string
		family   ChainFamily
		code     int
		message  string
		expected *LavaError
	}{
		// Solana Tier 2 — source: agave rpc-client-api/src/custom_error.rs
		{"Solana -32009", ChainFamilySolana, -32009, "", LavaErrorChainSolanaMissingLongTerm},
		{"Solana -32007", ChainFamilySolana, -32007, "", LavaErrorChainSolanaLedgerJump},
		{"Solana -32005", ChainFamilySolana, -32005, "", LavaErrorNodeSolanaUnhealthy},
		{"Solana -32004", ChainFamilySolana, -32004, "", LavaErrorChainBlockNotFound},
		{"Solana -32001", ChainFamilySolana, -32001, "", LavaErrorChainStatePruned},
		// Bitcoin Tier 2 — source: Bitcoin Core src/rpc/protocol.h
		{"Bitcoin warmup -28", ChainFamilyBitcoin, -28, "", LavaErrorNodeBitcoinWarmup},
		{"Bitcoin initial download -10", ChainFamilyBitcoin, -10, "", LavaErrorNodeBitcoinInitialDownload},
		{"Bitcoin not connected -9", ChainFamilyBitcoin, -9, "", LavaErrorNodeBitcoinNotConnected},
		{"Bitcoin verify error -25", ChainFamilyBitcoin, -25, "", LavaErrorChainBitcoinVerifyError},
		{"Bitcoin verify rejected -26", ChainFamilyBitcoin, -26, "", LavaErrorChainBitcoinVerifyRejected},
		{"Bitcoin already in chain -27", ChainFamilyBitcoin, -27, "", LavaErrorChainBitcoinAlreadyInChain},
		{"Bitcoin insufficient funds -6", ChainFamilyBitcoin, -6, "", LavaErrorChainBitcoinInsufficientFunds},
		// Starknet Tier 2
		{"Starknet failed to receive tx", ChainFamilyStarknet, 1, "", LavaErrorChainStarknetFailedToReceiveTx},
		{"Starknet contract not found", ChainFamilyStarknet, 20, "", LavaErrorChainStarknetContractNotFound},
		{"Starknet block not found", ChainFamilyStarknet, 24, "", LavaErrorChainStarknetBlockNotFound},
		{"Starknet class not found", ChainFamilyStarknet, 28, "", LavaErrorChainStarknetClassNotFound},
		{"Starknet tx hash not found", ChainFamilyStarknet, 29, "", LavaErrorChainStarknetTxHashNotFound},
		{"Starknet contract error", ChainFamilyStarknet, 40, "", LavaErrorChainStarknetContractError},
		{"Starknet tx exec error", ChainFamilyStarknet, 41, "", LavaErrorChainStarknetTxExecError},
		{"Starknet class declared", ChainFamilyStarknet, 51, "", LavaErrorChainStarknetClassAlreadyDeclared},
		{"Starknet invalid nonce", ChainFamilyStarknet, 52, "", LavaErrorChainStarknetInvalidNonce},
		{"Starknet insufficient fee", ChainFamilyStarknet, 53, "", LavaErrorChainStarknetInsufficientFee},
		{"Starknet insufficient balance", ChainFamilyStarknet, 54, "", LavaErrorChainStarknetInsufficientBalance},
		{"Starknet validation failure", ChainFamilyStarknet, 55, "", LavaErrorChainStarknetValidationFailure},
		{"Starknet compilation", ChainFamilyStarknet, 56, "", LavaErrorChainStarknetCompilationFailed},
		{"Starknet duplicate tx", ChainFamilyStarknet, 59, "", LavaErrorChainStarknetDuplicateTx},
		{"Starknet unsupported tx version", ChainFamilyStarknet, 61, "", LavaErrorChainStarknetUnsupportedTxVersion},
		{"Starknet unexpected error", ChainFamilyStarknet, 63, "", LavaErrorChainStarknetUnexpectedError},
		// NEAR Tier 2 (message-based)
		{"NEAR unknown block", ChainFamilyNEAR, 0, "UNKNOWN_BLOCK", LavaErrorChainNEARUnknownBlock},
		{"NEAR unknown chunk", ChainFamilyNEAR, 0, "UNKNOWN_CHUNK", LavaErrorChainNEARUnknownChunk},
		{"NEAR invalid shard", ChainFamilyNEAR, 0, "INVALID_SHARD_ID", LavaErrorChainNEARInvalidShardID},
		{"NEAR not synced", ChainFamilyNEAR, 0, "NOT_SYNCED_YET", LavaErrorChainNEARNotSyncedYet},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(nil, tt.family, TransportJsonRPC, tt.code, tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyError_GenericJsonRPCMappings(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		message  string
		expected *LavaError
	}{
		// JSON-RPC error codes
		{"method not found code", -32601, "", LavaErrorNodeMethodNotFound},
		{"parse error code", -32700, "", LavaErrorUserParseError},
		{"invalid request code", -32600, "", LavaErrorUserInvalidRequest},
		{"invalid params code", -32602, "", LavaErrorUserInvalidParams},
		{"internal error code", -32603, "", LavaErrorNodeInternalError},
		{"server error code", -32000, "some error", LavaErrorNodeServerError},
		{"limit exceeded code", -32005, "query limit", LavaErrorNodeLimitExceeded},

		// Message-based matchers
		{"nonce too low", 0, "nonce too low", LavaErrorChainNonceTooLow},
		{"nonce too high", 0, "nonce too high", LavaErrorChainNonceTooHigh},
		{"insufficient funds", 0, "insufficient funds for gas", LavaErrorChainInsufficientFunds},
		{"execution reverted", 0, "execution reverted", LavaErrorChainExecutionReverted},
		{"already known", 0, "already known", LavaErrorChainTxAlreadyKnown},
		{"out of gas", 0, "out of gas", LavaErrorChainOutOfGas},
		{"rate limit message", 0, "rate limit exceeded", LavaErrorNodeRateLimited},
		{"missing trie node", 0, "missing trie node abcdef", LavaErrorChainStatePruned},
		{"block not found", 0, "block not found", LavaErrorChainBlockNotFound},
		{"underpriced", 0, "transaction underpriced", LavaErrorChainTxUnderpriced},
		{"replacement underpriced", 0, "replacement transaction underpriced", LavaErrorChainTxReplacementUnderpriced},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(nil, ChainFamilyEVM, TransportJsonRPC, tt.code, tt.message)
			assert.Equal(t, tt.expected, result, "expected %s but got %s", tt.expected.Name, result.Name)
		})
	}
}

func TestClassifyError_GenericRESTMappings(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected *LavaError
	}{
		{"404 not found", "HTTP 404: Not Found", LavaErrorNodeEndpointNotFound},
		{"405 method not allowed", "HTTP 405: Method Not Allowed", LavaErrorNodeMethodNotAllowed},
		{"413 too large", "HTTP 413: Request Entity Too Large", LavaErrorUserRequestTooLarge},
		{"429 rate limit", "HTTP 429: Too Many Requests", LavaErrorNodeRateLimited},
		{"500 internal", "HTTP 500: Internal Server Error", LavaErrorNodeInternalError},
		{"502 bad gateway", "HTTP 502: Bad Gateway", LavaErrorNodeBadGateway},
		{"503 unavailable", "HTTP 503: Service Unavailable", LavaErrorNodeServiceUnavailable},
		{"504 timeout", "HTTP 504: Gateway Timeout", LavaErrorNodeGatewayTimeout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(nil, ChainFamilyAptos, TransportREST, 0, tt.message)
			assert.Equal(t, tt.expected, result, "expected %s but got %s", tt.expected.Name, result.Name)
		})
	}
}

func TestClassifyError_GenericGRPCMappings(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected *LavaError
	}{
		{"unimplemented", 12, LavaErrorNodeUnimplemented},
		{"unavailable", 14, LavaErrorNodeServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(nil, ChainFamilyCosmosSDK, TransportGRPC, tt.code, "")
			assert.Equal(t, tt.expected, result, "expected %s but got %s", tt.expected.Name, result.Name)
		})
	}
}

// ---------------------------------------------------------------------------
// Shadow detection test
// ---------------------------------------------------------------------------

func TestNoShadowedMatchers(t *testing.T) {
	// Tier 1: generic matchers, partitioned by transport
	for transport, mappings := range genericErrorMappings {
		t.Run(transport.String(), func(t *testing.T) {
			checkMappingsForShadows(t, mappings)
		})
	}

	// Tier 2: chain-specific matchers, partitioned by chain family
	for family, mappings := range chainErrorMappings {
		t.Run(family.String(), func(t *testing.T) {
			checkMappingsForShadows(t, mappings)
		})
	}
}

func checkMappingsForShadows(t *testing.T, mappings []errorMapping) {
	t.Helper()
	for i := 0; i < len(mappings); i++ {
		for j := i + 1; j < len(mappings); j++ {
			a := mappings[i]
			b := mappings[j]
			// If both matchers would match the same input AND map to different errors,
			// the later one is shadowed.
			if a.LavaError == b.LavaError {
				continue // same target — not a shadow, just redundancy
			}
			checkShadow(t, i, j, a, b)
		}
	}
}

func checkShadow(t *testing.T, i, j int, a, b errorMapping) {
	t.Helper()
	// We can't generically test all possible inputs, but we can check
	// specific known-ambiguous patterns:

	// MessageContains shadowing: if a.substring is a substring of b.substring
	aMsg, aIsMsg := a.Matcher.(messageContainsMatcher)
	bMsg, bIsMsg := b.Matcher.(messageContainsMatcher)
	if aIsMsg && bIsMsg {
		// If a's substring is contained in b's substring, a would match anything b matches
		if len(aMsg.substring) < len(bMsg.substring) &&
			strings.Contains(bMsg.substring, aMsg.substring) {
			t.Errorf("matcher[%d] MessageContains(%q) shadows matcher[%d] MessageContains(%q) → %s would never match",
				i, aMsg.substring, j, bMsg.substring, b.LavaError.Name)
		}
	}

	// CodeEquals shadowing: two CodeEquals with same code but different targets
	aCode, aIsCode := a.Matcher.(codeEqualsMatcher)
	bCode, bIsCode := b.Matcher.(codeEqualsMatcher)
	if aIsCode && bIsCode && aCode.code == bCode.code {
		t.Errorf("matcher[%d] CodeEquals(%d)→%s shadows matcher[%d] CodeEquals(%d)→%s",
			i, aCode.code, a.LavaError.Name, j, bCode.code, b.LavaError.Name)
	}
}
