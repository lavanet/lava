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

// INVARIANT: chainErrorMappings is populated exclusively at package-init time
// via this literal declaration. It is never mutated at runtime, which is what
// lets ClassifyError read it lock-free on the hot classification path. If you
// need to support dynamic registration, add explicit synchronisation — do NOT
// silently mutate this map from elsewhere.
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
		// Blockhash expiry — extremely common under load when clients submit tx
		// with a recent blockhash that has since rolled off the 150-slot window.
		// Placed before generic Tier-1 matchers would see it; case-insensitive
		// via the matcher pre-lowering.
		{MessageContains("blockhash not found"), LavaErrorChainSolanaBlockhashNotFound},
	},
	ChainFamilyBitcoin: {
		// Source: Bitcoin Core src/rpc/protocol.h
		{CodeEquals(-28), LavaErrorNodeBitcoinWarmup},                  // RPC_IN_WARMUP: "Loading block index..."
		{CodeEquals(-10), LavaErrorNodeBitcoinInitialDownload},         // RPC_CLIENT_IN_INITIAL_DOWNLOAD
		{CodeEquals(-9), LavaErrorNodeBitcoinNotConnected},             // RPC_CLIENT_NOT_CONNECTED: "Shutting down"
		{CodeEquals(-25), LavaErrorChainBitcoinVerifyError},            // RPC_VERIFY_ERROR
		{CodeEquals(-26), LavaErrorChainBitcoinVerifyRejected},         // RPC_VERIFY_REJECTED
		{CodeEquals(-27), LavaErrorChainBitcoinAlreadyInChain},         // RPC_VERIFY_ALREADY_IN_CHAIN
		{CodeEquals(-6), LavaErrorChainBitcoinWalletInsufficientFunds}, // RPC_WALLET_INSUFFICIENT_FUNDS
		// Message-based: UTXO already spent / double-spend detection.
		// Bitcoin Core returns these through sendrawtransaction rejections.
		{MessageContains("already spent"), LavaErrorChainDoubleSpend},
		{MessageContains("double spend"), LavaErrorChainDoubleSpend},
		{MessageContains("txn-mempool-conflict"), LavaErrorChainDoubleSpend}, // Bitcoin Core specific
	},
	ChainFamilyCosmosSDK: {
		// Cosmos SDK transaction errors. These must be Tier-2 (not Tier-1)
		// because "insufficient fee" also collides with Starknet's dedicated
		// CHAIN_STARKNET_INSUFFICIENT_FEE (code 53) which is handled in the
		// Starknet block below — Tier-2 scoping keeps each chain's semantics
		// from bleeding into the other.
		{MessageContains("account sequence mismatch"), LavaErrorChainInvalidSequence}, // Cosmos SDK x/auth
		{MessageContains("insufficient fees"), LavaErrorChainInsufficientFee},         // Cosmos SDK x/auth plural
		{MessageContains("insufficient fee"), LavaErrorChainInsufficientFee},          // Singular variant
		// "account not found" is Cosmos SDK x/auth for a missing signer account.
		// Generic Tier-1 doesn't carry this matcher because EVM uses different
		// phrasing and would false-match contract calls against unfunded addresses.
		{MessageContains("account not found"), LavaErrorChainAccountNotFound},
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
		{CodeEquals(61), LavaErrorChainStarknetTxVersionUnsupported},
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

// INVARIANT: genericErrorMappings is populated at package-init time — first by
// this literal, then extended by the init() function below which appends
// shared HTTP status matchers per transport. After init returns it is never
// mutated, so ClassifyError can read it lock-free on the hot path. Any future
// dynamic registration must add explicit synchronisation.
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

		// Unsupported methods (message-based fallback for nodes that use -32000).
		// Matchers here must be tightly scoped: NodeMethodNotFound carries
		// SubCategoryUnsupportedMethod (zero CU, no retry), so a false positive
		// silently stops retries and bills nothing. When in doubt, require the
		// literal word "method" to appear alongside the trigger phrase.
		{MessageContains("method not found"), LavaErrorNodeMethodNotFound},
		{MessageContains("method not supported"), LavaErrorNodeMethodNotSupported},
		{MessageContains("unknown method"), LavaErrorNodeMethodNotFound},
		{MessageContains("method does not exist"), LavaErrorNodeMethodNotFound},
		{MessageRegex(`(?i)method .* does not exist`), LavaErrorNodeMethodNotFound},
		// "<keyword> ... is not available" — require a method/rpc/endpoint/api keyword
		// in the same clause so chain-native messages like Solana's "Block is not
		// available for slot N" or generic "data is not available for block" don't
		// get pinned as unsupported method.
		{MessageRegex(`(?i)\b(?:method|rpc|endpoint|api)\b[^,;.]*\bis not available\b`), LavaErrorNodeMethodNotSupported},
		// "invalid method" — match only when the phrase is terminal, quoted, or
		// followed by "name"/colon. This rejects "invalid method argument",
		// "invalid method parameters", "invalid method signature", etc., which
		// would otherwise trip the zero-CU path on a user-input error.
		{MessageRegex(`(?i)invalid method(?:\s*$|\s*[:'"]|\s+name\b)`), LavaErrorNodeMethodNotFound},
		// Provider-disabled methods (e.g. QuickNode paid tier). Require "method"
		// or "rpc" near the "blocked" token so unrelated firewall/proxy messages
		// ("blocked external request") aren't misclassified.
		{MessageRegex(`(?i)blocked[^,]*\b(?:method|rpc)\b|(?i)\b(?:method|rpc)\b[^,]*\bblocked\b`), LavaErrorNodeMethodNotSupported},

		// --- User input validation (Layer D) ---
		// These must be ordered BEFORE the broader chain-transaction matchers so
		// "invalid address" / "invalid block number" don't leak into -32000 catch-all.
		// Matchers are deliberately tight: NodeMethodNotFound carries
		// SubCategoryUserError (zero retries, zero CU), so false positives would
		// silently swallow real chain errors.
		//
		// Block format: "hex string without 0x prefix" is Geth's exact phrase;
		// "invalid block number" and "invalid block hash" are generic.
		{MessageContains("hex string without 0x prefix"), LavaErrorUserInvalidBlockFormat}, // Geth
		{MessageContains("invalid block number"), LavaErrorUserInvalidBlockFormat},
		{MessageContains("invalid block hash"), LavaErrorUserInvalidBlockFormat},
		// Address format: "bad address checksum" is Geth; bare "invalid address"
		// covers most EVM variants.
		{MessageContains("bad address checksum"), LavaErrorUserInvalidAddress},
		{MessageContains("invalid address"), LavaErrorUserInvalidAddress},
		// Hex encoding: "hex string has odd length" is Geth's exact phrase; bare
		// "invalid hex" is a common catch-all but placed last so the more
		// specific phrases (block, address) win first.
		{MessageContains("hex string has odd length"), LavaErrorUserInvalidHex}, // Geth
		{MessageContains("invalid hex"), LavaErrorUserInvalidHex},

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
		// Node is still catching up to the chain head. NEAR carries its own
		// Tier-2 matcher (CHAIN_NEAR_NOT_SYNCED_YET), so this Tier-1 matcher
		// only fires for chains that surface a generic string message. The
		// phrase is tightly bounded to avoid matching unrelated "syncing"
		// contexts (e.g. a smart contract whose name contains "sync").
		{MessageContains("node is syncing"), LavaErrorNodeSyncing},
		{MessageContains("node is still syncing"), LavaErrorNodeSyncing},
		{MessageContains("catching up to the chain"), LavaErrorNodeSyncing},

		// Tx size limits — Geth/Erigon variants
		{MessageContains("oversized data"), LavaErrorChainTxTooLarge},           // Geth
		{MessageContains("transaction size exceeds"), LavaErrorChainTxTooLarge}, // Erigon
		{MessageContains("tx too large"), LavaErrorChainTxTooLarge},
		// Invalid signature — Tier-2 Solana (code 3306) runs first and wins
		// for the Solana family, so this generic matcher only fires for
		// non-Solana chains that use a free-form error message.
		{MessageContains("invalid signature"), LavaErrorChainInvalidSignature},
		{MessageContains("signature verification failed"), LavaErrorChainInvalidSignature},

		// Chain execution errors
		{MessageContains("execution reverted"), LavaErrorChainExecutionReverted},
		{MessageContains("out of gas"), LavaErrorChainOutOfGas},
		{MessageContains("stack limit reached"), LavaErrorChainStackOverflow},
		{MessageContains("invalid opcode"), LavaErrorChainInvalidOpcode},
		{MessageContains("write protection"), LavaErrorChainWriteProtection},
		// EIP-170 contract bytecode size limit. Geth/Erigon emit this as
		// "max code size exceeded" during contract creation.
		{MessageContains("max code size exceeded"), LavaErrorChainContractSizeExceeded},
		// Polygon zkEVM prover exceeded the circuit counter budget
		// (arithmetic, keccak, storage, etc.). zkEVMs speak EVM JSON-RPC
		// so this matcher sits in the generic Tier-1 block rather than a
		// family-scoped Tier-2 entry.
		{MessageContains("out of counters"), LavaErrorChainZkEVMOutOfCounters},

		// Chain state/data errors
		{MessageContains("missing trie node"), LavaErrorChainStatePruned},
		{MessageContains("historical state"), LavaErrorChainStatePruned},
		{MessageContains("block not found"), LavaErrorChainBlockNotFound},
		{MessageRegex(`(?i)block #?\w+ not found`), LavaErrorChainBlockNotFound},
		{MessageContains("transaction not found"), LavaErrorChainTxNotFound},  // some nodes return an error instead of null
		{MessageContains("receipt not found"), LavaErrorChainReceiptNotFound}, // Cosmos-EVM variant
		{MessageContains("response is too big"), LavaErrorChainLogResponseTooLarge},
		{MessageContains("exceeded max limit"), LavaErrorChainLogResponseTooLarge},

		// Truncated node responses — historically mapped to NODE_INTERNAL_ERROR
		// but this is a transport-layer symptom (connection closed mid-body /
		// reset by proxy) rather than the node signaling an application error.
		// Classifying it alongside PROTOCOL_CONNECTION_RESET lets endpoint-
		// health tracking treat it as a retryable connection failure, which
		// matches the production traces observed in commit 3136d4f35.
		{MessageContains("unexpected end of JSON input"), LavaErrorConnectionReset},

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

// init is the ONLY place allowed to mutate genericErrorMappings. It runs before
// any reader can observe the map, so the runtime invariant ("read-only after
// package init") holds. Do not add mutations outside this function.
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

// classifySubCategoryAcrossTransports tries every transport, returns the
// SubCategory of the first non-Unknown classification, or SubCategoryNone.
// Used by IsUnsupportedMethodError and IsUserInputError below.
//
// TRANSPORT ORDER IS SEMANTICALLY SIGNIFICANT. We try JSON-RPC first because
// it is the most common transport in production, then REST, then gRPC. The
// function returns on the first non-Unknown match — it does NOT aggregate
// across transports. Callers that know their transport exactly should call
// ClassifyError directly and avoid this helper; it is intended only for the
// narrow case where the transport is genuinely unknown.
//
// Edge case: a chain that returns gRPC status codes inside HTTP response
// bodies (e.g. grpc-web proxies) may match under the JSON-RPC bucket first
// with a subtly wrong classification. If that becomes a production issue,
// prefer threading the real transport through from the call site over
// changing the iteration order here.
func classifySubCategoryAcrossTransports(chainID string, statusCode int, message string) ErrorSubCategory {
	family := ChainFamilyUnknown
	if chainID != "" {
		family = GetChainFamilyOrDefault(chainID)
	}
	for _, transport := range []TransportType{TransportJsonRPC, TransportREST, TransportGRPC} {
		if c := ClassifyError(nil, family, transport, statusCode, message); c != LavaErrorUnknown {
			return c.SubCategory
		}
	}
	return SubCategoryNone
}

// IsUnsupportedMethodError returns true when the error identified by statusCode and
// message is classified as an unsupported method (the node does not recognise the
// method at all).
//
// chainID is used to consult chain-specific Tier-2 matchers first, so chain-native
// messages like Solana's "Block is not available for slot N" don't accidentally
// match a broad Tier-1 substring rule (e.g. "is not available") and get pinned as
// unsupported-method — which would skip retries and zero out CU charging.
// Pass "" when the chain is genuinely unknown; Tier-2 is then skipped.
func IsUnsupportedMethodError(chainID string, statusCode int, message string) bool {
	return classifySubCategoryAcrossTransports(chainID, statusCode, message).IsUnsupportedMethod()
}

// IsUserInputError returns true when the error identified by statusCode and
// message is classified as invalid client input (Layer D USER_* codes or
// batch-size-exceeded). The contract mirrors IsUnsupportedMethodError: the
// chainID enables chain-specific Tier-2 lookups so chain-native messages
// aren't misclassified as user input. Pass "" when the chain is unknown.
//
// The consumer hot path uses this symmetric to IsUnsupportedMethodError to
// short-circuit retries and charge zero CU for invalid client input.
func IsUserInputError(chainID string, statusCode int, message string) bool {
	return classifySubCategoryAcrossTransports(chainID, statusCode, message).IsUserError()
}

// ClassifyMessage classifies an error from just a message string and an optional
// numeric code, trying all transport types in order (JsonRPC → REST → gRPC) and
// returning the first non-unknown classification.
//
// Use this only when the transport AND chain are genuinely unknown. Tier-2
// (chain-specific) matchers are unconditionally skipped here, so a chain-native
// message that happens to contain a broad Tier-1 substring can be misclassified.
// If the chain is known, call ClassifyError directly (or use a helper like
// IsUnsupportedMethodError that takes a chainID) to avoid false matches.
func ClassifyMessage(code int, message string) *LavaError {
	for _, transport := range []TransportType{TransportJsonRPC, TransportREST, TransportGRPC} {
		if c := ClassifyError(nil, ChainFamilyUnknown, transport, code, message); c != LavaErrorUnknown {
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

	// Lower the message once per classification. Tier-1 has ~60 case-insensitive
	// substring matchers; re-lowering for each one allocates a fresh KB-scale copy
	// on the hot retry path. The loweredMessageMatcher fast path lets matchers
	// consume the pre-lowered form directly.
	loweredMessage := strings.ToLower(errorMessage)

	// Step 1: Check chain-specific mappings (Tier 2)
	if chainMappings, ok := chainErrorMappings[chainFamily]; ok {
		for _, mapping := range chainMappings {
			if matchMapping(mapping.Matcher, errorCode, errorMessage, loweredMessage) {
				return mapping.LavaError
			}
		}
	}

	// Step 2: Fall back to generic semantic mappings (Tier 1), scoped by transport
	if transportMappings, ok := genericErrorMappings[transport]; ok {
		for _, mapping := range transportMappings {
			if matchMapping(mapping.Matcher, errorCode, errorMessage, loweredMessage) {
				return mapping.LavaError
			}
		}
	}

	// Step 3: Unknown
	return LavaErrorUnknown
}

// matchMapping dispatches to the loweredMessageMatcher fast path when available,
// falling back to the standard ErrorMatcher interface otherwise.
func matchMapping(m ErrorMatcher, errorCode int, errorMessage, loweredMessage string) bool {
	if fast, ok := m.(loweredMessageMatcher); ok {
		return fast.matchesLowered(errorCode, loweredMessage)
	}
	return m.Matches(errorCode, errorMessage)
}

// DetectConnectionError inspects err for connection-level failures and returns the
// corresponding LavaError, or nil if the error is not connection-related.
// This is the single place for connection detection — callers pass the result as the
// connectionError argument to ClassifyError.
//
// Detection happens in three ordered layers:
//  1. Structured checks via errors.Is / errors.As (context, net.Error timeouts).
//  2. String-fallback table (detectConnectionErrorFromString) for wrapped errors
//     that lose their sentinel chain (e.g. fmt.Errorf without %w).
//  3. net.OpError unwrap for raw syscall errno codes.
//
// The string fallback is deliberately second, so structured detection wins when
// it can. Within the string fallback, the match order is explicit in the
// stringConnectionFallbacks data table to make precedence easy to audit.
func DetectConnectionError(err error) *LavaError {
	if err == nil {
		return nil
	}
	// Layer 1: structured sentinel checks
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

	// Layer 2: string fallback for errors wrapped without %w.
	if le := detectConnectionErrorFromString(strings.ToLower(err.Error())); le != nil {
		return le
	}

	// Layer 3: net.OpError with a raw syscall errno.
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var syscallErr *syscall.Errno
		if errors.As(opErr.Err, &syscallErr) {
			switch *syscallErr {
			case syscall.ECONNREFUSED:
				return LavaErrorConnectionRefused
			case syscall.ECONNRESET:
				return LavaErrorConnectionReset
			case syscall.ENETUNREACH, syscall.EHOSTUNREACH:
				return LavaErrorNetworkUnreachable
			}
		}
	}
	return nil
}

// stringConnectionFallback is one row of the string-fallback table used by
// detectConnectionErrorFromString. The first matching row wins, so ordering
// matters and is audited in-place.
type stringConnectionFallback struct {
	// substrings that must all appear in the lowered message for the row to match.
	mustContain []string
	// substrings that, if any are present, disqualify the row. Used to carve
	// exceptions out of a broad match (e.g. GOAWAY except ENHANCE_YOUR_CALM).
	mustNotContain []string
	// lava error returned when the row matches.
	result *LavaError
}

func (r stringConnectionFallback) matches(msg string) bool {
	for _, sub := range r.mustContain {
		if !strings.Contains(msg, sub) {
			return false
		}
	}
	for _, sub := range r.mustNotContain {
		if strings.Contains(msg, sub) {
			return false
		}
	}
	return true
}

// stringConnectionFallbacks is the precedence-ordered table used when the
// structured errors.Is/errors.As checks did not identify a connection error.
//
// Ordering rules:
//  1. Context cancel/deadline rows come first but are guarded so remote gRPC
//     status errors ("rpc error: code = DeadlineExceeded desc = ...") do NOT
//     match here — those carry a *remote* deadline and must fall through to
//     transport-scoped Tier-1 classification.
//  2. HTTP/2 GOAWAY is checked before the broader "connection reset" catch so
//     the ENHANCE_YOUR_CALM carve-out lands on the right row.
//  3. Refused / unreachable / reset / termination are mutually exclusive in
//     practice — each maps a distinct proxy/sidecar shape — but ordering is
//     preserved explicitly here so a future contributor doesn't have to infer
//     it from control-flow order.
//
// INVARIANT: written at package-init time (declaration), read-only thereafter.
var stringConnectionFallbacks = []stringConnectionFallback{
	// Remote gRPC status errors start with "rpc error"; those carry a *remote*
	// deadline/cancel and must not be classified as a local context error.
	// The guard is encoded as a mustNotContain on the rpc-error prefix marker.
	{
		mustContain:    []string{"context deadline exceeded"},
		mustNotContain: []string{"rpc error"},
		result:         LavaErrorContextDeadline,
	},
	{
		mustContain:    []string{"context canceled"},
		mustNotContain: []string{"rpc error"},
		result:         LavaErrorContextCanceled,
	},
	// HTTP/2 GOAWAY closes the whole connection. Exclude ENHANCE_YOUR_CALM —
	// that's a server-side rate-limit signal handled by the transport matchers.
	{
		mustContain:    []string{"goaway"},
		mustNotContain: []string{"enhance_your_calm"},
		result:         LavaErrorConnectionReset,
	},
	// HTTP/2 RST_STREAM appears as "stream error: stream ID ...; ...".
	// This is a narrow hazard — "stream error" could in principle appear in
	// unrelated messages — but in practice the gRPC/HTTP2 stack is the only
	// known producer, and losing this signal would break retry of RST_STREAM.
	{mustContain: []string{"stream error"}, result: LavaErrorConnectionReset},
	// Proxy / sidecar variants — Envoy wraps upstream failures with verbose
	// bodies that still contain these phrases. Order is not load-bearing here
	// because the phrases are mutually exclusive in practice, but keeping the
	// explicit order avoids accidental churn during future reordering.
	{mustContain: []string{"connection refused"}, result: LavaErrorConnectionRefused},
	{mustContain: []string{"network is unreachable"}, result: LavaErrorNetworkUnreachable},
	{mustContain: []string{"host is unreachable"}, result: LavaErrorNetworkUnreachable},
	{mustContain: []string{"no route to host"}, result: LavaErrorNetworkUnreachable},
	{mustContain: []string{"connection reset"}, result: LavaErrorConnectionReset},
	// Envoy "connection termination" — proxy/sidecar closed an established
	// upstream stream (distinct from "connection refused" which is a connect
	// failure before any stream was established).
	{mustContain: []string{"connection termination"}, result: LavaErrorConnectionReset},
}

// detectConnectionErrorFromString walks stringConnectionFallbacks in order and
// returns the first matching result, or nil if no row matches. msg must
// already be lowercased by the caller.
func detectConnectionErrorFromString(msg string) *LavaError {
	for _, row := range stringConnectionFallbacks {
		if row.matches(msg) {
			return row.result
		}
	}
	return nil
}

// IsClientCancellation reports whether err represents a request that was
// cancelled by the client / relay orchestration, as opposed to an upstream
// (provider/endpoint) fault. Two scenarios reach this path:
//
//  1. Relay race: multiple goroutines race in parallel. When one wins, the
//     parent context's cancel() fires and the losing goroutines observe
//     context.Canceled as their result — no provider is at fault.
//  2. Client disconnect: the upstream caller closed the connection before we
//     responded, producing context.Canceled on the in-flight request.
//
// The rule is: the error must be context.Canceled AND the context itself must
// now carry an error. A standalone context.Canceled without ctx.Err() would
// be suspicious — it should never happen in practice, but the conjunction
// guards against misclassifying a provider-emitted "context canceled" string
// as a client cancellation.
//
// Call this at endpoint-health / refusal-counter decision points instead of
// hand-rolling an errors.Is(err, context.Canceled) check — having one rule in
// one place avoids drift between the consumer session layer, the smart router
// health tracker, and the connection-refusal counter.
func IsClientCancellation(err error, ctx context.Context) bool {
	if err == nil || ctx == nil {
		return false
	}
	return errors.Is(err, context.Canceled) && ctx.Err() != nil
}
