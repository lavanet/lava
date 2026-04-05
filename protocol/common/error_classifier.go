package common

import (
	"context"
	"errors"
	"net"
	"strings"
	"syscall"
)

// ---------------------------------------------------------------------------
// Tier 2: Chain-specific error mappings (checked first, overrides generic)
// ---------------------------------------------------------------------------

var chainErrorMappings = map[ChainFamily][]errorMapping{
	ChainFamilySolana: {
		// Source: Solana RPC custom error codes (agave rpc-client-api/src/custom_error.rs)
		{CodeEquals(-32001), LavaErrorChainStatePruned},                                        // "Block cleaned up, does not exist on node"
		{CodeEquals(-32002), LavaErrorChainSolanaSimulationFailed},                             // "Transaction simulation failed"
		{CodeEquals(-32003), LavaErrorChainSolanaSignatureVerifyFailed},                        // "Signature verification failure"
		{CodeEquals(-32004), LavaErrorChainBlockNotFound},                                      // "Block not available for slot N"
		{CodeEquals(-32005), LavaErrorNodeSolanaUnhealthy},                                     // "Node is unhealthy" / "Node is behind by N slots"
		{CodeEquals(-32007), LavaErrorChainSolanaLedgerJump},                                   // "Slot skipped or missing"
		{CodeEquals(-32009), LavaErrorChainSolanaMissingLongTerm},                              // "Slot missing in long-term storage"
		{MessageContains("missing in long-term storage"), LavaErrorChainSolanaMissingLongTerm}, // message-based fallback
		{CodeEquals(-32010), LavaErrorChainSolanaExcludedFromIndex},                            // "Excluded from account secondary indexes"
		{CodeEquals(-32013), LavaErrorChainSolanaSignatureLengthMismatch},                      // "Signature length mismatch"
		{CodeEquals(-32014), LavaErrorChainSolanaBlockStatusUnavailable},                       // "Block status unavailable"
		{CodeEquals(-32015), LavaErrorChainSolanaTxVersionUnsupported},                         // "Transaction version not supported"
		{CodeEquals(-32016), LavaErrorChainSolanaMinContextSlotNotReached},                     // "Minimum context slot not reached"
		{CodeEquals(-32602), LavaErrorUserInvalidParams},                                       // "Invalid params" (Solana-specific override)
	},
	ChainFamilyBitcoin: {
		// Source: Bitcoin Core src/rpc/protocol.h
		{CodeEquals(-28), LavaErrorNodeBitcoinWarmup},            // RPC_IN_WARMUP: "Loading block index..."
		{CodeEquals(-10), LavaErrorNodeBitcoinInitialDownload},   // RPC_CLIENT_IN_INITIAL_DOWNLOAD
		{CodeEquals(-9), LavaErrorNodeBitcoinNotConnected},       // RPC_CLIENT_NOT_CONNECTED: "Shutting down"
		{CodeEquals(-25), LavaErrorChainBitcoinVerifyError},      // RPC_VERIFY_ERROR
		{CodeEquals(-26), LavaErrorChainBitcoinVerifyRejected},   // RPC_VERIFY_REJECTED
		{CodeEquals(-27), LavaErrorChainBitcoinAlreadyInChain},   // RPC_VERIFY_ALREADY_IN_CHAIN
		{CodeEquals(-6), LavaErrorChainBitcoinInsufficientFunds}, // RPC_WALLET_INSUFFICIENT_FUNDS
	},
	ChainFamilyStarknet: {
		// Source: Starknet JSON-RPC spec error codes
		{CodeEquals(1), LavaErrorChainStarknetFailedToReceiveTx},
		{CodeEquals(20), LavaErrorChainStarknetContractNotFound},
		{CodeEquals(24), LavaErrorChainStarknetBlockNotFound},
		{CodeEquals(28), LavaErrorChainStarknetClassNotFound},
		{CodeEquals(29), LavaErrorChainStarknetTxHashNotFound},
		{CodeEquals(40), LavaErrorChainStarknetContractError},
		{CodeEquals(41), LavaErrorChainStarknetTxExecError},
		{CodeEquals(51), LavaErrorChainStarknetClassAlreadyDeclared},
		{CodeEquals(52), LavaErrorChainStarknetInvalidNonce},
		{CodeEquals(53), LavaErrorChainStarknetInsufficientFee},
		{CodeEquals(54), LavaErrorChainStarknetInsufficientBalance},
		{CodeEquals(55), LavaErrorChainStarknetValidationFailure},
		{CodeEquals(56), LavaErrorChainStarknetCompilationFailed},
		{CodeEquals(59), LavaErrorChainStarknetDuplicateTx},
		{CodeEquals(61), LavaErrorChainStarknetUnsupportedTxVersion},
		{CodeEquals(63), LavaErrorChainStarknetUnexpectedError},
	},
	ChainFamilyNEAR: {
		// Source: NEAR RPC docs — matched by error.cause.name in message
		{MessageContains("UNKNOWN_BLOCK"), LavaErrorChainNEARUnknownBlock},
		{MessageContains("UNKNOWN_CHUNK"), LavaErrorChainNEARUnknownChunk},
		{MessageContains("INVALID_SHARD_ID"), LavaErrorChainNEARInvalidShardID},
		{MessageContains("NOT_SYNCED_YET"), LavaErrorChainNEARNotSyncedYet},
	},
}

// ---------------------------------------------------------------------------
// Tier 1: Generic error mappings, partitioned by transport type.
// Evaluated in declaration order — first match wins.
// Matchers MUST be ordered most-specific first.
// ---------------------------------------------------------------------------

var genericErrorMappings = map[TransportType][]errorMapping{
	TransportJsonRPC: {
		// --- Code-based matchers first (more precise than substring matching) ---

		// Standard JSON-RPC 2.0 codes
		{CodeEquals(-32601), LavaErrorNodeMethodNotFound},
		{CodeEquals(-32700), LavaErrorUserParseError},
		{CodeEquals(-32600), LavaErrorUserInvalidRequest},
		{CodeEquals(-32602), LavaErrorUserInvalidParams},
		{CodeEquals(-32603), LavaErrorNodeInternalError},

		// EIP-1474 server error codes
		{CodeEquals(-32001), LavaErrorNodeResourceNotFound},    // "Resource not found"
		{CodeEquals(-32002), LavaErrorNodeResourceUnavailable}, // "Resource unavailable"
		{CodeEquals(-32003), LavaErrorChainTxRejected},         // "Transaction rejected"
		{CodeEquals(-32004), LavaErrorNodeMethodNotSupported},  // "Method not supported"
		{CodeEquals(-32005), LavaErrorNodeLimitExceeded},       // "Limit exceeded"

		// --- Message-based matchers (for -32000 catch-all and codeless errors) ---

		// Unsupported methods (message-based fallback for nodes that use -32000)
		{MessageContains("method not found"), LavaErrorNodeMethodNotFound},
		{MessageContains("method not supported"), LavaErrorNodeMethodNotSupported},
		{MessageContains("unknown method"), LavaErrorNodeMethodNotFound},
		{MessageContains("method does not exist"), LavaErrorNodeMethodNotSupported},
		{MessageRegex(`(?i)method .* does not exist`), LavaErrorNodeMethodNotSupported},
		{MessageContains("is not available"), LavaErrorNodeMethodNotSupported},
		{MessageContains("invalid method"), LavaErrorNodeMethodNotFound},

		// Rate limiting
		{MessageContains("rate limit"), LavaErrorNodeRateLimited},
		{MessageContains("enhance_your_calm"), LavaErrorNodeRateLimited}, // HTTP/2 GOAWAY with ENHANCE_YOUR_CALM — server-side rate limit

		// Chain transaction errors — matchers cover Geth, Erigon, and Nethermind variants
		{MessageContains("nonce too low"), LavaErrorChainNonceTooLow},
		{MessageContains("nonce is too low"), LavaErrorChainNonceTooLow}, // Nethermind processor
		{MessageContains("nonce too high"), LavaErrorChainNonceTooHigh},
		{MessageContains("nonce is too high"), LavaErrorChainNonceTooHigh},                // Nethermind processor
		{MessageContains("insufficient funds"), LavaErrorChainInsufficientFunds},          // Geth + Erigon
		{MessageContains("insufficientfunds"), LavaErrorChainInsufficientFunds},           // Nethermind PascalCase
		{MessageContains("insufficient maxfeeper"), LavaErrorChainInsufficientFunds},      // Nethermind: "insufficient MaxFeePerGas..."
		{MessageContains("insufficient sender balance"), LavaErrorChainInsufficientFunds}, // Nethermind
		{MessageContains("intrinsic gas too low"), LavaErrorChainGasTooLow},               // Geth
		{MessageContains("intrinsicgas"), LavaErrorChainGasTooLow},                        // Erigon: "IntrinsicGas"
		{MessageContains("gas limit below intrinsic gas"), LavaErrorChainGasTooLow},       // Nethermind
		{MessageContains("exceeds block gas limit"), LavaErrorChainGasLimitExceeded},
		{MessageContains("replacement transaction underpriced"), LavaErrorChainTxReplacementUnderpriced},
		{MessageContains("could not replace existing tx"), LavaErrorChainTxReplacementUnderpriced}, // Erigon
		{MessageContains("transaction underpriced"), LavaErrorChainTxUnderpriced},                  // Geth
		{MessageContains("underpriced"), LavaErrorChainTxUnderpriced},                              // Erigon: bare "underpriced"
		{MessageContains("fee too low"), LavaErrorChainTxUnderpriced},                              // Erigon: "fee too low"
		{MessageContains("already known"), LavaErrorChainTxAlreadyKnown},                           // Geth + Erigon
		{MessageContains("alreadyknown"), LavaErrorChainTxAlreadyKnown},                            // Nethermind PascalCase
		{MessageContains("txpool is full"), LavaErrorChainMempoolFull},
		{MessageContains("max fee per gas less than block base fee"), LavaErrorChainMaxFeeBelowBase},

		// Chain execution errors
		{MessageContains("execution reverted"), LavaErrorChainExecutionReverted},
		{MessageContains("out of gas"), LavaErrorChainOutOfGas},
		{MessageContains("stack limit reached"), LavaErrorChainStackOverflow},
		{MessageContains("invalid opcode"), LavaErrorChainInvalidOpcode},
		{MessageContains("write protection"), LavaErrorChainWriteProtection},

		// Chain state/data errors
		{MessageContains("missing trie node"), LavaErrorChainStatePruned},
		{MessageContains("historical state"), LavaErrorChainStatePruned},
		{MessageContains("block not found"), LavaErrorChainBlockNotFound},
		{MessageRegex(`(?i)block #?\w+ not found`), LavaErrorChainBlockNotFound},
		{MessageContains("response is too big"), LavaErrorChainLogResponseTooLarge},
		{MessageContains("exceeded max limit"), LavaErrorChainLogResponseTooLarge},

		// Malformed/truncated node responses
		{MessageContains("unexpected end of JSON input"), LavaErrorNodeInternalError},

		// Node server/generic errors — broadest matchers last
		{MessageContains("all attempts exhausted"), LavaErrorNodeServerError},
		{CodeEquals(-32000), LavaErrorNodeServerError},

		// HTTP status matchers are appended by init() via httpStatusMessageMappings()
	},

	TransportREST: {
		// Message-based matchers for common REST error patterns
		{MessageContains("endpoint not found"), LavaErrorNodeEndpointNotFound},
		{MessageContains("route not found"), LavaErrorNodeEndpointNotFound},
		{MessageContains("path not found"), LavaErrorNodeEndpointNotFound},
		{MessageContains("method not allowed"), LavaErrorNodeMethodNotAllowed},
		// CodeEquals and HTTPStatusContains matchers are appended by init()
		// via httpStatusCodeMappings() and httpStatusMessageMappings()
	},

	TransportGRPC: {
		// gRPC status code matchers (codes from google.golang.org/grpc/codes)
		{GRPCCodeEquals(12), LavaErrorNodeUnimplemented},      // codes.Unimplemented
		{GRPCCodeEquals(14), LavaErrorNodeServiceUnavailable}, // codes.Unavailable
		// Message-based matchers for gRPC errors conveyed without status codes
		{MessageContains("rate limit"), LavaErrorNodeRateLimited},
		{MessageContains("enhance_your_calm"), LavaErrorNodeRateLimited}, // HTTP/2 GOAWAY ENHANCE_YOUR_CALM
		{MessageContains("unimplemented"), LavaErrorNodeUnimplemented},
		{MessageContains("not implemented"), LavaErrorNodeUnimplemented},
		{MessageContains("service not found"), LavaErrorNodeUnimplemented},
	},
}

// ---------------------------------------------------------------------------
// Shared HTTP status mappings — appended to JSON-RPC and REST at init time
// ---------------------------------------------------------------------------

// httpStatusCodeMappings returns CodeEquals matchers for common HTTP error status codes.
// These are used for REST transport where the error code is the HTTP status code itself.
func httpStatusCodeMappings() []errorMapping {
	return []errorMapping{
		{CodeEquals(404), LavaErrorNodeEndpointNotFound},
		{CodeEquals(405), LavaErrorNodeMethodNotAllowed},
		{CodeEquals(413), LavaErrorUserRequestTooLarge},
		{CodeEquals(429), LavaErrorNodeRateLimited},
		{CodeEquals(500), LavaErrorNodeInternalError},
		{CodeEquals(502), LavaErrorNodeBadGateway},
		{CodeEquals(503), LavaErrorNodeServiceUnavailable},
		{CodeEquals(504), LavaErrorNodeGatewayTimeout},
		// Cloudflare custom 5xx errors
		{CodeEquals(520), LavaErrorNodeServerError},    // Web server returned unknown error
		{CodeEquals(521), LavaErrorNodeServerError},    // Web server is down
		{CodeEquals(522), LavaErrorNodeGatewayTimeout}, // Connection timed out
		{CodeEquals(523), LavaErrorNodeServerError},    // Origin is unreachable
		{CodeEquals(524), LavaErrorNodeGatewayTimeout}, // A timeout occurred
		{CodeEquals(525), LavaErrorNodeServerError},    // SSL handshake failed
		{CodeEquals(526), LavaErrorNodeServerError},    // Invalid SSL certificate
		{CodeEquals(527), LavaErrorNodeServerError},    // Railgun error
		{CodeEquals(530), LavaErrorNodeServerError},    // Origin DNS error
	}
}

// httpStatusMessageMappings returns HTTPStatusContains matchers for common HTTP error status codes.
// These match status codes appearing as substrings in error messages (e.g., "HTTP status 429").
func httpStatusMessageMappings() []errorMapping {
	return []errorMapping{
		{HTTPStatusContains(404), LavaErrorNodeEndpointNotFound},
		{HTTPStatusContains(405), LavaErrorNodeMethodNotAllowed},
		{HTTPStatusContains(413), LavaErrorUserRequestTooLarge},
		{HTTPStatusContains(429), LavaErrorNodeRateLimited},
		{HTTPStatusContains(500), LavaErrorNodeInternalError},
		{HTTPStatusContains(502), LavaErrorNodeBadGateway},
		{HTTPStatusContains(503), LavaErrorNodeServiceUnavailable},
		{HTTPStatusContains(504), LavaErrorNodeGatewayTimeout},
		// Cloudflare custom 5xx errors
		{HTTPStatusContains(520), LavaErrorNodeServerError},
		{HTTPStatusContains(521), LavaErrorNodeServerError},
		{HTTPStatusContains(522), LavaErrorNodeGatewayTimeout},
		{HTTPStatusContains(523), LavaErrorNodeServerError},
		{HTTPStatusContains(524), LavaErrorNodeGatewayTimeout},
		{HTTPStatusContains(525), LavaErrorNodeServerError},
		{HTTPStatusContains(526), LavaErrorNodeServerError},
		{HTTPStatusContains(527), LavaErrorNodeServerError},
		{HTTPStatusContains(530), LavaErrorNodeServerError},
	}
}

func init() {
	// Append shared HTTP status message matchers to JSON-RPC transport
	genericErrorMappings[TransportJsonRPC] = append(genericErrorMappings[TransportJsonRPC], httpStatusMessageMappings()...)

	// Append both CodeEquals and HTTPStatusContains matchers to REST transport
	genericErrorMappings[TransportREST] = append(genericErrorMappings[TransportREST], httpStatusCodeMappings()...)
	genericErrorMappings[TransportREST] = append(genericErrorMappings[TransportREST], httpStatusMessageMappings()...)

	// Append HTTPStatusContains matchers to gRPC transport — HTTP status codes can appear
	// in gRPC error messages when the underlying transport is HTTP (e.g. provider relay errors)
	genericErrorMappings[TransportGRPC] = append(genericErrorMappings[TransportGRPC], httpStatusMessageMappings()...)
}

// ---------------------------------------------------------------------------
// ClassifyError — the central classification function
// ---------------------------------------------------------------------------

// ClassifyMessage classifies an error from just a message string and an optional numeric code,
// trying all transport types in order (JsonRPC → REST → gRPC) and returning the first
// non-unknown classification.
//
// Use this only when the transport is genuinely unknown. If the transport is known,
// call ClassifyError directly with the explicit transport to avoid false matches
// (e.g., a gRPC status code accidentally matching a JSON-RPC code matcher first).
func ClassifyMessage(code int, message string) *LavaError {
	for _, transport := range []TransportType{TransportJsonRPC, TransportREST, TransportGRPC} {
		if c := ClassifyError(nil, -1, transport, code, message); c != LavaErrorUnknown {
			return c
		}
	}
	return LavaErrorUnknown
}

// ClassifyError classifies an error into a LavaError for internal use (logging, metrics, endpoint health).
// The original error always passes through unchanged to the user (transparent hop).
//
// Parameters:
//   - connectionError: pre-detected connection-level error (timeout, refused, etc.), or nil
//   - chainFamily: the chain family for Tier 2 lookups, use -1 if unknown
//   - transport: the transport type for Tier 1 generic matcher partitioning
//   - errorCode: the numeric error code (e.g., JSON-RPC error code), or 0 if not applicable
//   - errorMessage: the error message string for substring/regex matching
func ClassifyError(connectionError *LavaError, chainFamily ChainFamily, transport TransportType, errorCode int, errorMessage string) *LavaError {
	// Step 0: If caller already identified a connection-level error, use it
	if connectionError != nil {
		return connectionError
	}

	// Step 1: Check chain-specific mappings (Tier 2)
	if chainMappings, ok := chainErrorMappings[chainFamily]; ok {
		for _, mapping := range chainMappings {
			if mapping.Matcher.Matches(errorCode, errorMessage) {
				return mapping.LavaError
			}
		}
	}

	// Step 2: Fall back to generic semantic mappings (Tier 1), scoped by transport
	if transportMappings, ok := genericErrorMappings[transport]; ok {
		for _, mapping := range transportMappings {
			if mapping.Matcher.Matches(errorCode, errorMessage) {
				return mapping.LavaError
			}
		}
	}

	// Step 3: Unknown
	return LavaErrorUnknown
}

// DetectConnectionError inspects err for connection-level failures and returns the
// corresponding LavaError, or nil if the error is not connection-related.
// This is the single place for connection detection — callers pass the result as the
// connectionError argument to ClassifyError.
func DetectConnectionError(err error) *LavaError {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return LavaErrorContextCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return LavaErrorContextDeadline
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return LavaErrorConnectionTimeout
	}
	// Fallback for errors wrapped without %w (errors.Is can't traverse those chains)
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "context deadline exceeded") {
		return LavaErrorContextDeadline
	}
	if strings.Contains(msg, "context canceled") {
		return LavaErrorContextCanceled
	}
	// HTTP/2 GOAWAY closes the connection at the transport level — treat as a connection reset.
	// Exclude ENHANCE_YOUR_CALM (rate-limit signal) which is handled by transport-specific matchers.
	if strings.Contains(msg, "goaway") && !strings.Contains(msg, "enhance_your_calm") {
		return LavaErrorConnectionReset
	}
	// HTTP/2 RST_STREAM (stream error) — single-stream reset by the peer, retryable connection-level error.
	if strings.Contains(msg, "stream error") {
		return LavaErrorConnectionReset
	}
	// "connection refused" appearing inside wrapped proxy/envoy errors (e.g. 503 + Envoy body).
	if strings.Contains(msg, "connection refused") {
		return LavaErrorConnectionRefused
	}
	if strings.Contains(msg, "connection reset") {
		return LavaErrorConnectionReset
	}
	// Envoy "connection termination" — proxy/sidecar closed the upstream connection.
	if strings.Contains(msg, "connection termination") {
		return LavaErrorConnectionReset
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var syscallErr *syscall.Errno
		if errors.As(opErr.Err, &syscallErr) {
			switch *syscallErr {
			case syscall.ECONNREFUSED:
				return LavaErrorConnectionRefused
			case syscall.ECONNRESET:
				return LavaErrorConnectionReset
			}
		}
	}
	return nil
}
