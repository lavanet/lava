package common

// ---------------------------------------------------------------------------
// Tier 2: Chain-specific error mappings (checked first, overrides generic)
// ---------------------------------------------------------------------------

var chainErrorMappings = map[ChainFamily][]errorMapping{
	ChainFamilySolana: {
		// Source: Solana RPC custom error codes (agave rpc-client-api/src/custom_error.rs)
		{CodeEquals(-32001), LavaErrorChainStatePruned},                    // "Block cleaned up, does not exist on node"
		{CodeEquals(-32002), LavaErrorChainSolanaSimulationFailed},         // "Transaction simulation failed"
		{CodeEquals(-32003), LavaErrorChainSolanaSignatureVerifyFailed},    // "Signature verification failure"
		{CodeEquals(-32004), LavaErrorChainBlockNotFound},                  // "Block not available for slot N"
		{CodeEquals(-32005), LavaErrorNodeSolanaUnhealthy},                 // "Node is unhealthy" / "Node is behind by N slots"
		{CodeEquals(-32007), LavaErrorChainSolanaLedgerJump},               // "Slot skipped or missing"
		{CodeEquals(-32009), LavaErrorChainSolanaMissingLongTerm},          // "Slot missing in long-term storage"
		{CodeEquals(-32010), LavaErrorChainSolanaExcludedFromIndex},        // "Excluded from account secondary indexes"
		{CodeEquals(-32013), LavaErrorChainSolanaSignatureLengthMismatch},  // "Signature length mismatch"
		{CodeEquals(-32014), LavaErrorChainSolanaBlockStatusUnavailable},   // "Block status unavailable"
		{CodeEquals(-32015), LavaErrorChainSolanaTxVersionUnsupported},     // "Transaction version not supported"
		{CodeEquals(-32016), LavaErrorChainSolanaMinContextSlotNotReached}, // "Minimum context slot not reached"
		{CodeEquals(-32602), LavaErrorUserInvalidParams},                   // "Invalid params" (Solana-specific override)
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
		{MessageContains("method does not exist"), LavaErrorNodeMethodNotFound},

		// Rate limiting
		{MessageContains("rate limit"), LavaErrorNodeRateLimited},

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
		{MessageContains("block not found"), LavaErrorChainBlockNotFound},

		// Node server/generic errors — broadest matchers last
		{CodeEquals(-32000), LavaErrorNodeServerError},

		// HTTP status codes — JSON-RPC is transported over HTTP, so status codes appear in errors
		{HTTPStatusContains(429), LavaErrorNodeRateLimited},
		{HTTPStatusContains(500), LavaErrorNodeInternalError},
		{HTTPStatusContains(502), LavaErrorNodeBadGateway},
		{HTTPStatusContains(503), LavaErrorNodeServiceUnavailable},
		{HTTPStatusContains(504), LavaErrorNodeGatewayTimeout},
	},

	TransportREST: {
		// HTTP status matchers
		{HTTPStatusContains(404), LavaErrorNodeEndpointNotFound},
		{HTTPStatusContains(405), LavaErrorNodeMethodNotAllowed},
		{HTTPStatusContains(413), LavaErrorUserRequestTooLarge},
		{HTTPStatusContains(429), LavaErrorNodeRateLimited},
		{HTTPStatusContains(500), LavaErrorNodeInternalError},
		{HTTPStatusContains(502), LavaErrorNodeBadGateway},
		{HTTPStatusContains(503), LavaErrorNodeServiceUnavailable},
		{HTTPStatusContains(504), LavaErrorNodeGatewayTimeout},
	},

	TransportGRPC: {
		// gRPC status code matchers (codes from google.golang.org/grpc/codes)
		{GRPCCodeEquals(12), LavaErrorNodeUnimplemented},      // codes.Unimplemented
		{GRPCCodeEquals(14), LavaErrorNodeServiceUnavailable}, // codes.Unavailable
	},
}

// ---------------------------------------------------------------------------
// ClassifyError — the central classification function
// ---------------------------------------------------------------------------

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
