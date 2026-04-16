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

func TestRegistry_NoDuplicateCodes(t *testing.T) {
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

func TestRegistry_NoDuplicateNames(t *testing.T) {
	seen := make(map[string]uint32)
	for _, le := range errorRegistry {
		if existingCode, ok := seen[le.Name]; ok {
			t.Errorf("duplicate name %s: codes %d and %d", le.Name, existingCode, le.Code)
		}
		seen[le.Name] = le.Code
	}
}

func TestRegistry_CodeRanges(t *testing.T) {
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

func TestRegistry_AllErrorCodesRegistered(t *testing.T) {
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

func TestRegisterError_RejectsReservedCodeZero(t *testing.T) {
	assert.PanicsWithValue(t,
		"error code 0 is reserved for LavaErrorUnknown; MY_NEW_ERROR must use a non-zero code",
		func() {
			registerError(&LavaError{Code: 0, Name: "MY_NEW_ERROR"})
		},
	)
}

func TestRegisterError_DuplicateCodeMentionsExistingName(t *testing.T) {
	// LavaErrorConnectionTimeout has code 1001 — re-registering under the same
	// code but a different name must panic with a message that names the existing
	// owner so the offender can find the collision fast.
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic on duplicate code")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T: %v", r, r)
		}
		assert.Contains(t, msg, "duplicate error code: 1001")
		assert.Contains(t, msg, "PROTOCOL_CONNECTION_TIMEOUT") // existing owner
		assert.Contains(t, msg, "COLLIDING_NAME")              // new entrant
	}()
	registerError(&LavaError{Code: 1001, Name: "COLLIDING_NAME"})
}

func TestRegisterError_DuplicateNameMentionsExistingCode(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic on duplicate name")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T: %v", r, r)
		}
		assert.Contains(t, msg, "duplicate error name: PROTOCOL_CONNECTION_TIMEOUT")
		assert.Contains(t, msg, "existing code 1001")
	}()
	registerError(&LavaError{Code: 99999, Name: "PROTOCOL_CONNECTION_TIMEOUT"})
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

func TestErrorCategory_String(t *testing.T) {
	assert.Equal(t, "internal", CategoryInternal.String())
	assert.Equal(t, "external", CategoryExternal.String())
	assert.Equal(t, "unknown", ErrorCategory(99).String())
}

func TestErrorSubCategory_String(t *testing.T) {
	assert.Equal(t, "none", SubCategoryNone.String())
	assert.Equal(t, "unsupported_method", SubCategoryUnsupportedMethod.String())
}

func TestErrorSubCategory_UnsupportedMethodTagging(t *testing.T) {
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

// TestLayerDErrors_NoSpecialSubCategory ensures Layer D codes (4000-4999) do NOT
// carry SubCategoryUnsupportedMethod — they are user-input errors that charge
// normal CU (the provider does real work on every call because the response is
// not cached).
func TestLayerDErrors_NoSpecialSubCategory(t *testing.T) {
	layerDFound := 0
	for code, le := range errorRegistry {
		if code >= 4000 && code < 5000 {
			layerDFound++
			assert.Equal(t, SubCategoryNone, le.SubCategory,
				"Layer D code %d (%s) must NOT carry a special SubCategory — user errors charge normal CU", code, le.Name)
			assert.False(t, le.Retryable,
				"Layer D code %d (%s) must be non-retryable", code, le.Name)
		}
	}
	require.Positive(t, layerDFound, "Layer D range should contain at least one registered error")
}

// TestErrorSubCategory_RateLimitTagging enforces that every rate-limit code carries the
// unified subcategory, and that IsRateLimited() is driven by it.
func TestErrorSubCategory_RateLimitTagging(t *testing.T) {
	rateLimitCodes := []*LavaError{
		LavaErrorRateLimited,       // 1020 protocol
		LavaErrorNodeRateLimited,   // 2005 node
		LavaErrorNodeLimitExceeded, // 2011 node limit (eth_getLogs range, etc.)
	}
	for _, le := range rateLimitCodes {
		assert.True(t, le.SubCategory.IsRateLimit(),
			"%s (%d) must be SubCategoryRateLimit", le.Name, le.Code)
		assert.True(t, le.IsRateLimited(),
			"%s (%d) must report IsRateLimited()", le.Name, le.Code)
	}

	// Exclusion spot-checks
	exclusions := []*LavaError{
		LavaErrorConnectionTimeout, // internal transport
		LavaErrorNodeInternalError, // retryable but not rate-limit
		LavaErrorUserParseError,    // user-input, not rate-limit
	}
	for _, le := range exclusions {
		assert.False(t, le.SubCategory.IsRateLimit(),
			"%s (%d) must NOT be SubCategoryRateLimit", le.Name, le.Code)
		assert.False(t, le.IsRateLimited())
	}
}

// TestClassifyError_UserInputMatchers asserts Layer D (user input) matchers classify
// representative messages: invalid block format, invalid address, and
// invalid hex encoding. The -32700/-32600/-32602/413 code paths are covered
// separately by TestClassifyError_GenericJsonRPCMappings.
func TestClassifyError_UserInputMatchers(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected *LavaError
	}{
		// USER_INVALID_BLOCK_FORMAT
		{"geth hex without 0x", "hex string without 0x prefix", LavaErrorUserInvalidBlockFormat},
		{"invalid block number", "invalid block number: expected hex", LavaErrorUserInvalidBlockFormat},
		{"invalid block hash", "invalid block hash: wrong length", LavaErrorUserInvalidBlockFormat},
		// USER_INVALID_ADDRESS
		{"bad checksum", "bad address checksum: 0xDEADbeef", LavaErrorUserInvalidAddress},
		{"bare invalid address", "invalid address: too short", LavaErrorUserInvalidAddress},
		// USER_INVALID_HEX
		{"geth odd length", "hex string has odd length", LavaErrorUserInvalidHex},
		{"bare invalid hex", "invalid hex encoding in parameter", LavaErrorUserInvalidHex},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(nil, ChainFamilyEVM, TransportJsonRPC, 0, tt.message)
			assert.Equal(t, tt.expected, result, "message %q", tt.message)
		})
	}
}

// TestClassifyError_ChainExecutionMatchers asserts execution-range matchers (codes
// 3100-3199) classify representative messages: contract size, account not
// found, zkEVM out of counters, and Bitcoin wallet-insufficient-funds.
func TestClassifyError_ChainExecutionMatchers(t *testing.T) {
	t.Run("contract size exceeded (EVM Tier-1)", func(t *testing.T) {
		result := ClassifyError(nil, ChainFamilyEVM, TransportJsonRPC, 0, "max code size exceeded")
		assert.Equal(t, LavaErrorChainContractSizeExceeded, result)
	})

	t.Run("account not found (Cosmos Tier-2)", func(t *testing.T) {
		result := ClassifyError(nil, ChainFamilyCosmosSDK, TransportJsonRPC, 0, "account not found: lava1abc")
		assert.Equal(t, LavaErrorChainAccountNotFound, result)
	})

	t.Run("account not found does NOT match on EVM (Cosmos-scoped)", func(t *testing.T) {
		// Cosmos Tier-2 scoping must not leak into EVM, where "account not
		// found" would collide with unfunded-address contract calls.
		result := ClassifyError(nil, ChainFamilyEVM, TransportJsonRPC, 0, "account not found")
		assert.Equal(t, LavaErrorUnknown, result)
	})

	t.Run("zkevm out of counters (generic Tier-1, EVM-routed)", func(t *testing.T) {
		result := ClassifyError(nil, ChainFamilyEVM, TransportJsonRPC, 0, "batch: out of counters: arithmetic")
		assert.Equal(t, LavaErrorChainZkEVMOutOfCounters, result)
	})

	t.Run("bitcoin wallet insufficient funds by code", func(t *testing.T) {
		// Code 3344 is the stable identity; the Name disambiguates from EVM
		// CHAIN_INSUFFICIENT_FUNDS (3003) which is a tx submission failure.
		result := ClassifyError(nil, ChainFamilyBitcoin, TransportJsonRPC, -6, "")
		assert.Equal(t, LavaErrorChainBitcoinWalletInsufficientFunds, result)
		assert.Equal(t, uint32(3344), result.Code)
		assert.Equal(t, "CHAIN_BITCOIN_WALLET_INSUFFICIENT_FUNDS", result.Name)
	})
}

// TestClassifyError_NodeSyncingMatcher asserts NODE_SYNCING is produced by the Tier-1
// JSON-RPC matchers for chains that surface a generic string message.
func TestClassifyError_NodeSyncingMatcher(t *testing.T) {
	cases := []string{
		"error: node is syncing, try again later",
		"node is still syncing",
		"rejected: catching up to the chain head",
	}
	for _, msg := range cases {
		t.Run(msg, func(t *testing.T) {
			result := ClassifyError(nil, ChainFamilyEVM, TransportJsonRPC, 0, msg)
			assert.Equal(t, LavaErrorNodeSyncing, result)
		})
	}
}

// TestClassifyError_UnexpectedEndOfJSONInputIsConnectionReset locks in that the truncated-
// response symptom is classified as a transport-layer connection failure,
// not a node-level internal error.
func TestClassifyError_UnexpectedEndOfJSONInputIsConnectionReset(t *testing.T) {
	result := ClassifyError(nil, ChainFamilyEVM, TransportJsonRPC, 0, "unexpected end of JSON input")
	assert.Equal(t, LavaErrorConnectionReset, result)
}

// TestClassifyError_ChainMatchers asserts Layer C (chain) matchers classify
// representative messages across Tier-1 tx/state matchers and Tier-2 Cosmos,
// Bitcoin, and Solana families.
func TestClassifyError_ChainMatchers(t *testing.T) {
	// Generic Tier-1 matchers (EVM family)
	evmCases := []struct {
		name     string
		message  string
		expected *LavaError
	}{
		{"geth oversized data", "oversized data", LavaErrorChainTxTooLarge},
		{"erigon tx size exceeds", "transaction size exceeds 128 kB", LavaErrorChainTxTooLarge},
		{"tx too large", "tx too large", LavaErrorChainTxTooLarge},
		{"invalid signature bare", "invalid signature: recovery failed", LavaErrorChainInvalidSignature},
		{"signature verification failed", "signature verification failed", LavaErrorChainInvalidSignature},
		// Layer C state
		{"transaction not found", "transaction not found: 0xabcd", LavaErrorChainTxNotFound},
		{"receipt not found", "receipt not found for hash 0xabcd", LavaErrorChainReceiptNotFound},
	}
	for _, tt := range evmCases {
		t.Run("EVM/"+tt.name, func(t *testing.T) {
			result := ClassifyError(nil, ChainFamilyEVM, TransportJsonRPC, 0, tt.message)
			assert.Equal(t, tt.expected, result, "message %q", tt.message)
		})
	}

	// Cosmos Tier-2
	cosmosCases := []struct {
		name     string
		message  string
		expected *LavaError
	}{
		{"account sequence mismatch", "account sequence mismatch, expected 7, got 5", LavaErrorChainInvalidSequence},
		{"insufficient fees (plural)", "insufficient fees; got: 100ulava required: 200ulava", LavaErrorChainInsufficientFee},
		{"insufficient fee (singular)", "insufficient fee for tx", LavaErrorChainInsufficientFee},
	}
	for _, tt := range cosmosCases {
		t.Run("Cosmos/"+tt.name, func(t *testing.T) {
			result := ClassifyError(nil, ChainFamilyCosmosSDK, TransportJsonRPC, 0, tt.message)
			assert.Equal(t, tt.expected, result, "message %q", tt.message)
		})
	}

	// Bitcoin Tier-2 — double-spend variants
	btcCases := []struct {
		name     string
		message  string
		expected *LavaError
	}{
		{"already spent", "utxo already spent", LavaErrorChainDoubleSpend},
		{"double spend", "double spend detected", LavaErrorChainDoubleSpend},
		{"txn-mempool-conflict", "txn-mempool-conflict: input already spent in mempool", LavaErrorChainDoubleSpend},
	}
	for _, tt := range btcCases {
		t.Run("Bitcoin/"+tt.name, func(t *testing.T) {
			result := ClassifyError(nil, ChainFamilyBitcoin, TransportJsonRPC, 0, tt.message)
			assert.Equal(t, tt.expected, result, "message %q", tt.message)
		})
	}

	// Solana Tier-2 — blockhash expiry
	t.Run("Solana/blockhash not found", func(t *testing.T) {
		result := ClassifyError(nil, ChainFamilySolana, TransportJsonRPC, 0, "Blockhash not found")
		assert.Equal(t, LavaErrorChainSolanaBlockhashNotFound, result)
	})

	// Cross-family precedence: Solana's own "invalid signature" code (3306) must
	// still win for the Solana family even though generic Tier-1 now has an
	// "invalid signature" matcher.
	t.Run("Solana signature code wins over generic message", func(t *testing.T) {
		result := ClassifyError(nil, ChainFamilySolana, TransportJsonRPC, -32003, "signature verification failed")
		assert.Equal(t, LavaErrorChainSolanaSignatureVerifyFailed, result,
			"Tier-2 Solana mapping must win over Tier-1 generic 'invalid signature'")
	})
}

// TestClassifyError_LayerDCodesHaveMatchers enforces the invariant that every registered
// Layer D code is produced by at least one matcher. The probe map is the
// contract for adding new Layer D errors: add a probe or delete the code.
func TestClassifyError_LayerDCodesHaveMatchers(t *testing.T) {
	// Known mapping: Layer D code → a representative (transport, code,
	// message) that should classify to it. Extending this list is the
	// "contract" for adding a new Layer D error.
	probes := map[uint32]struct {
		transport TransportType
		code      int
		msg       string
	}{
		4001: {TransportJsonRPC, -32700, ""},                        // USER_PARSE_ERROR
		4002: {TransportJsonRPC, -32600, ""},                        // USER_INVALID_REQUEST
		4003: {TransportJsonRPC, -32602, ""},                        // USER_INVALID_PARAMS
		4004: {TransportJsonRPC, 0, "hex string without 0x prefix"}, // USER_INVALID_BLOCK_FORMAT
		4005: {TransportJsonRPC, 0, "bad address checksum"},         // USER_INVALID_ADDRESS
		// 4006 is only wired on REST transport (HTTP 413 status code),
		// so probe with REST rather than JSON-RPC.
		4006: {TransportREST, 413, ""},                           // USER_REQUEST_TOO_LARGE
		4007: {TransportJsonRPC, 0, "hex string has odd length"}, // USER_INVALID_HEX
	}

	for code, le := range errorRegistry {
		if code < 4000 || code >= 5000 {
			continue
		}
		probe, ok := probes[code]
		require.True(t, ok,
			"Layer D code %d (%s) has no probe in TestClassifyError_LayerDCodesHaveMatchers — "+
				"either wire a matcher and add a probe, or delete the code",
			code, le.Name)
		result := ClassifyError(nil, ChainFamilyEVM, probe.transport, probe.code, probe.msg)
		assert.Equal(t, le, result,
			"Layer D code %d (%s) probe (%v, %d, %q) must classify to itself",
			code, le.Name, probe.transport, probe.code, probe.msg)
	}
}

// TestLayerDErrors_AreNonRetryable verifies that Layer D user-input errors
// are classified as non-retryable but do NOT carry SubCategoryUnsupportedMethod
// (which would zero out CU — providers should be paid for processing invalid input).
func TestLayerDErrors_AreNonRetryable(t *testing.T) {
	// JSON-RPC parse error code → USER_PARSE_ERROR → non-retryable
	le := ClassifyError(nil, ChainFamilyEVM, TransportJsonRPC, -32700, "parse error")
	assert.Equal(t, LavaErrorUserParseError, le)
	assert.False(t, le.Retryable)
	assert.False(t, le.SubCategory.IsUnsupportedMethod())

	// JSON-RPC invalid params → USER_INVALID_PARAMS → non-retryable
	le = ClassifyError(nil, ChainFamilyEVM, TransportJsonRPC, -32602, "invalid params")
	assert.Equal(t, LavaErrorUserInvalidParams, le)
	assert.False(t, le.Retryable)
	assert.False(t, le.SubCategory.IsUnsupportedMethod())
}

// TestClassifyNodeErrorForRetry_DerivesAllFlagsFromSingleLookup verifies the
// single-pass helper the consumer hot path uses. Each row asserts all three
// flags at once because the consumer relies on their mutual consistency —
// IsUnsupportedMethod must always imply IsNonRetryable, and unknown messages
// must fail open (all flags false → retryable).
func TestClassifyNodeErrorForRetry_DerivesAllFlagsFromSingleLookup(t *testing.T) {
	cases := []struct {
		name             string
		family           ChainFamily
		transport        TransportType
		errorCode        int
		message          string
		wantNonRetryable bool
		wantUnsupported  bool
	}{
		{
			name:             "EVM execution reverted → terminal, not unsupported",
			family:           ChainFamilyEVM,
			transport:        TransportJsonRPC,
			errorCode:        3,
			message:          "execution reverted",
			wantNonRetryable: true,
		},
		{
			name:             "EVM method not found → terminal + unsupported",
			family:           ChainFamilyEVM,
			transport:        TransportJsonRPC,
			errorCode:        -32601,
			message:          "method not found",
			wantNonRetryable: true,
			wantUnsupported:  true,
		},
		{
			name:             "JSON-RPC body code -32700 parse error → terminal, not unsupported",
			family:           ChainFamilyEVM,
			transport:        TransportJsonRPC,
			errorCode:        -32700,
			message:          "parse error",
			wantNonRetryable: true,
		},
		{
			name:             "generic transient → retryable (all flags false)",
			family:           ChainFamilyEVM,
			transport:        TransportJsonRPC,
			errorCode:        502,
			message:          "bad gateway",
			wantNonRetryable: false,
		},
		{
			name:             "unknown classification fails open",
			family:           ChainFamilyUnknown,
			transport:        TransportJsonRPC,
			errorCode:        0,
			message:          "totally unfamiliar garbage",
			wantNonRetryable: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyNodeErrorForRetry(tc.family, tc.transport, tc.errorCode, tc.message)
			assert.Equal(t, tc.wantNonRetryable, got.IsNonRetryable, "IsNonRetryable")
			assert.Equal(t, tc.wantUnsupported, got.IsUnsupportedMethod, "IsUnsupportedMethod")
			if got.IsUnsupportedMethod {
				assert.True(t, got.IsNonRetryable, "IsUnsupportedMethod implies IsNonRetryable")
			}
		})
	}
}

// TestApiInterfaceToTransport maps the spec API interface strings to the
// TransportType consumed by the error registry. tendermintrpc and unknown
// inputs fall back to JSON-RPC — the consumer and smart router rely on this
// default to avoid threading a null transport through hot paths.
func TestApiInterfaceToTransport(t *testing.T) {
	cases := []struct {
		apiInterface string
		want         TransportType
	}{
		{apiInterface: "jsonrpc", want: TransportJsonRPC},
		{apiInterface: "tendermintrpc", want: TransportJsonRPC},
		{apiInterface: "rest", want: TransportREST},
		{apiInterface: "grpc", want: TransportGRPC},
		{apiInterface: "", want: TransportJsonRPC},
		{apiInterface: "anything-else", want: TransportJsonRPC},
	}
	for _, tc := range cases {
		t.Run(tc.apiInterface, func(t *testing.T) {
			assert.Equal(t, tc.want, ApiInterfaceToTransport(tc.apiInterface))
		})
	}
}

// TestIsNonRetryableNodeErrorWithContext_JSONRPCBodyCode documents that the
// JSON-RPC Tier-1 code matchers only fire when callers pass the body error
// code (e.g. -32700) rather than the HTTP status that wraps it. This is the
// invariant the consumer and smart-router JSON-RPC paths rely on when they
// call ExtractJSONRPCErrorCode before classification.
func TestIsNonRetryableNodeErrorWithContext_JSONRPCBodyCode(t *testing.T) {
	// Using HTTP 200 (the wrapper status for JSON-RPC node errors) the
	// registry never reaches the body-code matcher.
	assert.False(t, IsNonRetryableNodeErrorWithContext(ChainFamilyEVM, TransportJsonRPC, 200, ""))
	// Passing the body code unlocks the Tier-1 JSON-RPC matcher for USER_*.
	assert.True(t, IsNonRetryableNodeErrorWithContext(ChainFamilyEVM, TransportJsonRPC, -32700, ""))
	assert.True(t, IsNonRetryableNodeErrorWithContext(ChainFamilyEVM, TransportJsonRPC, -32602, ""))
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

func TestMatcher_CodeEquals(t *testing.T) {
	m := CodeEquals(-32601)
	assert.True(t, m.Matches(-32601, "anything"))
	assert.False(t, m.Matches(-32600, "anything"))
	assert.False(t, m.Matches(0, "anything"))
}

func TestMatcher_MessageContains(t *testing.T) {
	m := MessageContains("nonce too low")
	assert.True(t, m.Matches(0, "nonce too low"))
	assert.True(t, m.Matches(0, "Error: Nonce Too Low for account"))
	assert.True(t, m.Matches(0, "NONCE TOO LOW"))
	assert.False(t, m.Matches(0, "nonce is fine"))
}

func TestMatcher_MessageRegex(t *testing.T) {
	m := MessageRegex(`missing.*trie node`)
	assert.True(t, m.Matches(0, "missing trie node abc123"))
	assert.True(t, m.Matches(0, "missing intermediate trie node"))
	assert.False(t, m.Matches(0, "trie node missing"))
}

func TestMatcher_HTTPStatusContains(t *testing.T) {
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

func TestMatcher_GRPCCodeEquals(t *testing.T) {
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

// TestDetectConnectionError_StringFallbackTable exercises every row in
// stringConnectionFallbacks to lock in precedence order and the carve-outs.
// Each case lists the input message and the expected LavaError (or nil).
func TestDetectConnectionError_StringFallbackTable(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected *LavaError
	}{
		// Layer 2 row 1: local context deadline (wrapped without %w)
		{"local context deadline", "operation failed: context deadline exceeded", LavaErrorContextDeadline},
		// Layer 2 row 1 carve-out: remote gRPC DeadlineExceeded MUST NOT be caught
		{"remote grpc deadline exceeded", "rpc error: code = DeadlineExceeded desc = context deadline exceeded", nil},
		// Layer 2 row 2: local context cancel
		{"local context canceled", "operation failed: context canceled", LavaErrorContextCanceled},
		// Layer 2 row 2 carve-out: remote gRPC Canceled MUST NOT be caught
		{"remote grpc canceled", "rpc error: code = Canceled desc = context canceled", nil},
		// Row 3: HTTP/2 GOAWAY → connection reset
		{"goaway protocol_error", `http2: server sent goaway and closed the connection; errcode=protocol_error`, LavaErrorConnectionReset},
		// Row 3 carve-out: ENHANCE_YOUR_CALM → falls through (rate-limit, not connection)
		{"goaway enhance_your_calm", `http2: server sent goaway; errcode=enhance_your_calm`, nil},
		// Row 4: HTTP/2 RST_STREAM
		{"stream error", "stream error: stream id 77; internal_error; received from peer", LavaErrorConnectionReset},
		// Row 5: connection refused (envoy wrap)
		{"connection refused", "upstream: delayed connect error: connection refused", LavaErrorConnectionRefused},
		// Rows 6–8: network unreachable variants
		{"network is unreachable", "dial tcp 1.2.3.4:443: connect: network is unreachable", LavaErrorNetworkUnreachable},
		{"host is unreachable", "dial tcp 1.2.3.4:443: connect: host is unreachable", LavaErrorNetworkUnreachable},
		{"no route to host", "dial tcp 1.2.3.4:443: connect: no route to host", LavaErrorNetworkUnreachable},
		// Row 9: connection reset (generic)
		{"connection reset", "read tcp: connection reset by peer", LavaErrorConnectionReset},
		// Row 10: envoy connection termination (distinct from refused)
		{"connection termination", "upstream: connection termination", LavaErrorConnectionReset},
		// Negative case: nothing matches
		{"unrelated error", "some unrelated node error", nil},

		// Precedence: "connection refused" and "connection reset" both present —
		// the "refused" row comes first in the table, so refused wins.
		{"refused+reset both present", "connection refused; connection reset", LavaErrorConnectionRefused},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectConnectionErrorFromString(strings.ToLower(tt.message))
			assert.Equal(t, tt.expected, result,
				"message %q expected %v got %v", tt.message, tt.expected, result)
		})
	}
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
		{"Bitcoin wallet insufficient funds -6", ChainFamilyBitcoin, -6, "", LavaErrorChainBitcoinWalletInsufficientFunds},
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
		{"Starknet tx version unsupported", ChainFamilyStarknet, 61, "", LavaErrorChainStarknetTxVersionUnsupported},
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

// TestClassifyError_UnsupportedMethodMatchersAreNarrow locks in the tight scope
// of the zero-CU unsupported-method matchers. A false positive here silently
// stops retries and bills nothing, so these cases must stay precise.
func TestClassifyError_UnsupportedMethodMatchersAreNarrow(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected *LavaError
	}{
		// "<keyword> ... is not available" — must match only when a method-ish
		// keyword is present in the same clause.
		{"method is not available (bare)", "method eth_getProof is not available", LavaErrorNodeMethodNotSupported},
		{"rpc is not available", "the rpc method eth_getLogs is not available on this node", LavaErrorNodeMethodNotSupported},
		{"data is not available — must NOT match", "data is not available for this block", LavaErrorUnknown},
		{"feature is not available — must NOT match", "feature is not available on this plan", LavaErrorUnknown},
		{"block is not available (Solana) — must NOT match", "Block is not available for slot 1234567", LavaErrorUnknown},

		// "invalid method" — must match only when terminal, colon/quote-followed,
		// or followed by "name".
		{"invalid method (bare)", "invalid method", LavaErrorNodeMethodNotFound},
		{"invalid method colon", "invalid method: eth_foo", LavaErrorNodeMethodNotFound},
		{"invalid method single-quote", "invalid method 'eth_foo'", LavaErrorNodeMethodNotFound},
		{"invalid method double-quote", `invalid method "eth_foo"`, LavaErrorNodeMethodNotFound},
		{"invalid method name", "invalid method name: eth_foo", LavaErrorNodeMethodNotFound},
		{"invalid method argument — must NOT match", "invalid method argument at position 2", LavaErrorUnknown},
		{"invalid method parameters — must NOT match", "invalid method parameters for call", LavaErrorUnknown},
		{"invalid method signature — must NOT match", "invalid method signature: expected 3 got 2", LavaErrorUnknown},

		// "blocked" provider-tier gating — must require method/rpc proximity.
		{"blocked method", "method eth_traceCall is blocked on your plan", LavaErrorNodeMethodNotSupported},
		{"blocked rpc", "blocked rpc eth_traceCall, upgrade required", LavaErrorNodeMethodNotSupported},
		{"blocked external request — must NOT match", "blocked external request by firewall", LavaErrorUnknown},
		{"blocked by policy request — must NOT match", "blocked by policy: request denied", LavaErrorUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(nil, ChainFamilyEVM, TransportJsonRPC, 0, tt.message)
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

func TestClassifyError_NoShadowedMatchers(t *testing.T) {
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
