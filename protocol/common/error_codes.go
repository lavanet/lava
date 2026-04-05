package common

// ---------------------------------------------------------------------------
// Layer A: Protocol Errors (Internal) — range 1000-1999
// Errors raised from within the Lava protocol itself — not from nodes or chains.
// ---------------------------------------------------------------------------

var (
	// Connection errors (1001-1009)
	LavaErrorConnectionTimeout = registerError(&LavaError{
		Code: 1001, Name: "PROTOCOL_CONNECTION_TIMEOUT", Category: CategoryInternal,
		Description: "Network operation timed out connecting to provider", Retryable: true,
	})
	LavaErrorConnectionRefused = registerError(&LavaError{
		Code: 1002, Name: "PROTOCOL_CONNECTION_REFUSED", Category: CategoryInternal,
		Description: "Provider connection refused", Retryable: true,
	})
	LavaErrorDNSFailure = registerError(&LavaError{
		Code: 1003, Name: "PROTOCOL_DNS_FAILURE", Category: CategoryInternal,
		Description: "DNS resolution failed", Retryable: true,
	})
	LavaErrorTLSMismatch = registerError(&LavaError{
		Code: 1004, Name: "PROTOCOL_TLS_MISMATCH", Category: CategoryInternal,
		Description: "HTTP/HTTPS protocol mismatch", Retryable: false,
	})
	LavaErrorConnectionReset = registerError(&LavaError{
		Code: 1005, Name: "PROTOCOL_CONNECTION_RESET", Category: CategoryInternal,
		Description: "Connection reset by peer", Retryable: true,
	})
	LavaErrorConnectionClosed = registerError(&LavaError{
		Code: 1006, Name: "PROTOCOL_CONNECTION_CLOSED", Category: CategoryInternal,
		Description: "Connection closed (EOF)", Retryable: true,
	})
	LavaErrorContextDeadline = registerError(&LavaError{
		Code: 1007, Name: "PROTOCOL_CONTEXT_DEADLINE", Category: CategoryInternal,
		Description: "Context deadline exceeded", Retryable: true,
	})
	LavaErrorContextCanceled = registerError(&LavaError{
		Code: 1008, Name: "PROTOCOL_CONTEXT_CANCELED", Category: CategoryInternal,
		Description: "Request context was canceled (client disconnect or relay race resolved)", Retryable: false,
	})

	// Provider availability (1010-1019)
	LavaErrorNoProviders = registerError(&LavaError{
		Code: 1010, Name: "PROTOCOL_NO_PROVIDERS", Category: CategoryInternal,
		Description: "No providers/pairings available", Retryable: false,
	})
	LavaErrorAllEndpointsDisabled = registerError(&LavaError{
		Code: 1011, Name: "PROTOCOL_ALL_ENDPOINTS_DISABLED", Category: CategoryInternal,
		Description: "All provider endpoints disabled", Retryable: false,
	})
	LavaErrorProviderUnavailable = registerError(&LavaError{
		Code: 1012, Name: "PROTOCOL_PROVIDER_UNAVAILABLE", Category: CategoryInternal,
		Description: "Provider service unavailable (gRPC UNAVAILABLE)", Retryable: true,
	})
	LavaErrorProviderAborted = registerError(&LavaError{
		Code: 1013, Name: "PROTOCOL_PROVIDER_ABORTED", Category: CategoryInternal,
		Description: "Provider aborted (gRPC ABORTED)", Retryable: true,
	})
	LavaErrorProviderDataLoss = registerError(&LavaError{
		Code: 1014, Name: "PROTOCOL_PROVIDER_DATA_LOSS", Category: CategoryInternal,
		Description: "Provider data loss (gRPC DATA_LOSS)", Retryable: true,
	})
	LavaErrorInsufficientProviders = registerError(&LavaError{
		Code: 1015, Name: "PROTOCOL_INSUFFICIENT_PROVIDERS", Category: CategoryInternal,
		Description: "Insufficient providers available for addon or cross-validation", Retryable: false,
	})

	// Rate limiting / CU (1020-1029)
	LavaErrorRateLimited = registerError(&LavaError{
		Code: 1020, Name: "PROTOCOL_RATE_LIMITED", Category: CategoryInternal,
		Description: "Lava-side rate limit exceeded", Retryable: false,
	})
	LavaErrorMaxCUExceeded = registerError(&LavaError{
		Code: 1021, Name: "PROTOCOL_MAX_CU_EXCEEDED", Category: CategoryInternal,
		Description: "Maximum compute units exceeded for session", Retryable: false,
	})
	LavaErrorBatchSizeExceeded = registerError(&LavaError{
		Code: 1022, Name: "PROTOCOL_BATCH_SIZE_EXCEEDED", Category: CategoryInternal,
		Description: "Batch request size exceeded limit", Retryable: false,
	})
	LavaErrorCUMismatch = registerError(&LavaError{
		Code: 1023, Name: "PROTOCOL_CU_MISMATCH", Category: CategoryInternal,
		Description: "CU accounting inconsistency or security violation", Retryable: false,
	})

	// Session errors (1030-1039)
	LavaErrorSessionNotFound = registerError(&LavaError{
		Code: 1030, Name: "PROTOCOL_SESSION_NOT_FOUND", Category: CategoryInternal,
		Description: "Session does not exist", Retryable: false,
	})
	LavaErrorEpochMismatch = registerError(&LavaError{
		Code: 1031, Name: "PROTOCOL_EPOCH_MISMATCH", Category: CategoryInternal,
		Description: "Epoch mismatch or too old", Retryable: false,
	})
	LavaErrorConsumerBlocked = registerError(&LavaError{
		Code: 1032, Name: "PROTOCOL_CONSUMER_BLOCKED", Category: CategoryInternal,
		Description: "Consumer is blocklisted", Retryable: false,
	})
	LavaErrorConsumerNotRegistered = registerError(&LavaError{
		Code: 1033, Name: "PROTOCOL_CONSUMER_NOT_REGISTERED", Category: CategoryInternal,
		Description: "Consumer not registered", Retryable: false,
	})
	LavaErrorRelayNumberMismatch = registerError(&LavaError{
		Code: 1034, Name: "PROTOCOL_RELAY_NUMBER_MISMATCH", Category: CategoryInternal,
		Description: "Relay number mismatch", Retryable: false,
	})
	LavaErrorSessionOutOfSync = registerError(&LavaError{
		Code: 1035, Name: "PROTOCOL_SESSION_OUT_OF_SYNC", Category: CategoryInternal,
		Description: "Session out of sync", Retryable: false,
	})
	LavaErrorInvalidRelayRequest = registerError(&LavaError{
		Code: 1036, Name: "PROTOCOL_INVALID_RELAY_REQUEST", Category: CategoryInternal,
		Description: "Relay request validation failed (wrong provider/specID/chainID/seen block/content hash)", Retryable: false,
	})
	LavaErrorRequestBlockMismatch = registerError(&LavaError{
		Code: 1037, Name: "PROTOCOL_REQUEST_BLOCK_MISMATCH", Category: CategoryInternal,
		Description: "Block height mismatch between consumer request and provider state", Retryable: true,
	})
	LavaErrorSessionAccountingFailed = registerError(&LavaError{
		Code: 1038, Name: "PROTOCOL_SESSION_ACCOUNTING_FAILED", Category: CategoryInternal,
		Description: "Session accounting (OnSessionDone/OnSessionFailure) failed", Retryable: false,
	})
	LavaErrorSubscriptionCleanupFailed = registerError(&LavaError{
		Code: 1039, Name: "PROTOCOL_SUBSCRIPTION_CLEANUP_FAILED", Category: CategoryInternal,
		Description: "Subscription consumer removal (RemoveConsumer) failed", Retryable: false,
	})

	// Verification / consensus (1040-1049)
	LavaErrorFinalizationError = registerError(&LavaError{
		Code: 1040, Name: "PROTOCOL_FINALIZATION_ERROR", Category: CategoryInternal,
		Description: "Provider finalization data incorrect", Retryable: true,
	})
	LavaErrorConsistencyError = registerError(&LavaError{
		Code: 1041, Name: "PROTOCOL_CONSISTENCY_ERROR", Category: CategoryInternal,
		Description: "Response consistency validation failed", Retryable: true,
	})
	LavaErrorHashConsensusError = registerError(&LavaError{
		Code: 1042, Name: "PROTOCOL_HASH_CONSENSUS_ERROR", Category: CategoryInternal,
		Description: "Conflicting response hashes detected", Retryable: true,
	})
	LavaErrorNoResponseTimeout = registerError(&LavaError{
		Code: 1043, Name: "PROTOCOL_NO_RESPONSE_TIMEOUT", Category: CategoryInternal,
		Description: "Timeout waiting for any provider response", Retryable: true,
	})
	LavaErrorRelayProcessingFailed = registerError(&LavaError{
		Code: 1044, Name: "PROTOCOL_RELAY_PROCESSING_FAILED", Category: CategoryInternal,
		Description: "Relay response processing failed on consumer side", Retryable: true,
	})

	// Subscriptions (1050-1059)
	LavaErrorSubscriptionNotFound = registerError(&LavaError{
		Code: 1050, Name: "PROTOCOL_SUBSCRIPTION_NOT_FOUND", Category: CategoryInternal,
		Description: "Subscription not found", Retryable: false,
	})
	LavaErrorSubscriptionInitFailed = registerError(&LavaError{
		Code: 1051, Name: "PROTOCOL_SUBSCRIPTION_INIT_FAILED", Category: CategoryInternal,
		Description: "Failed to initialize subscription", Retryable: false,
	})
	LavaErrorWebSocketIdleTimeout = registerError(&LavaError{
		Code: 1052, Name: "PROTOCOL_WEBSOCKET_IDLE_TIMEOUT", Category: CategoryInternal,
		Description: "WebSocket idle timeout", Retryable: false,
	})
)

// ---------------------------------------------------------------------------
// Layer B: Node Errors (External) — range 2000-2999
// Errors returned by the blockchain node itself (not execution/state errors).
// ---------------------------------------------------------------------------

var (
	// Generic node errors (2000-2099)
	LavaErrorNodeMethodNotFound = registerError(&LavaError{
		Code: 2001, Name: "NODE_METHOD_NOT_FOUND", Category: CategoryExternal,
		SubCategory: SubCategoryUnsupportedMethod,
		Description: "Method not found/supported", Retryable: false,
	})
	LavaErrorNodeMethodNotSupported = registerError(&LavaError{
		Code: 2002, Name: "NODE_METHOD_NOT_SUPPORTED", Category: CategoryExternal,
		Description: "Method exists but disabled on this node", Retryable: true,
	})
	LavaErrorNodeInternalError = registerError(&LavaError{
		Code: 2003, Name: "NODE_INTERNAL_ERROR", Category: CategoryExternal,
		Description: "Internal node error", Retryable: true,
	})
	LavaErrorNodeServerError = registerError(&LavaError{
		Code: 2004, Name: "NODE_SERVER_ERROR", Category: CategoryExternal,
		Description: "Generic server error", Retryable: true,
	})
	LavaErrorNodeRateLimited = registerError(&LavaError{
		Code: 2005, Name: "NODE_RATE_LIMITED", Category: CategoryExternal,
		Description: "Rate limited by node", Retryable: true,
	})
	LavaErrorNodeServiceUnavailable = registerError(&LavaError{
		Code: 2006, Name: "NODE_SERVICE_UNAVAILABLE", Category: CategoryExternal,
		Description: "Node temporarily unavailable", Retryable: true,
	})
	LavaErrorNodeSyncing = registerError(&LavaError{
		Code: 2007, Name: "NODE_SYNCING", Category: CategoryExternal,
		Description: "Node is syncing/catching up", Retryable: true,
	})
	LavaErrorNodeUnimplemented = registerError(&LavaError{
		Code: 2008, Name: "NODE_UNIMPLEMENTED", Category: CategoryExternal,
		SubCategory: SubCategoryUnsupportedMethod,
		Description: "gRPC method unimplemented", Retryable: false,
	})
	LavaErrorNodeEndpointNotFound = registerError(&LavaError{
		Code: 2009, Name: "NODE_ENDPOINT_NOT_FOUND", Category: CategoryExternal,
		SubCategory: SubCategoryUnsupportedMethod,
		Description: "REST endpoint not found", Retryable: false,
	})
	LavaErrorNodeMethodNotAllowed = registerError(&LavaError{
		Code: 2010, Name: "NODE_METHOD_NOT_ALLOWED", Category: CategoryExternal,
		SubCategory: SubCategoryUnsupportedMethod,
		Description: "REST method not allowed", Retryable: false,
	})
	LavaErrorNodeLimitExceeded = registerError(&LavaError{
		Code: 2011, Name: "NODE_LIMIT_EXCEEDED", Category: CategoryExternal,
		Description: "Request exceeds node limit (e.g., eth_getLogs range)", Retryable: false,
	})
	LavaErrorNodeResourceNotFound = registerError(&LavaError{
		Code: 2012, Name: "NODE_RESOURCE_NOT_FOUND", Category: CategoryExternal,
		Description: "Resource not found at node level", Retryable: true,
	})
	LavaErrorNodeResourceUnavailable = registerError(&LavaError{
		Code: 2013, Name: "NODE_RESOURCE_UNAVAILABLE", Category: CategoryExternal,
		Description: "Resource exists but unavailable", Retryable: true,
	})
	LavaErrorNodeGatewayTimeout = registerError(&LavaError{
		Code: 2014, Name: "NODE_GATEWAY_TIMEOUT", Category: CategoryExternal,
		Description: "Gateway timeout (HTTP 504 from provider)", Retryable: true,
	})
	LavaErrorNodeBadGateway = registerError(&LavaError{
		Code: 2015, Name: "NODE_BAD_GATEWAY", Category: CategoryExternal,
		Description: "Bad gateway (HTTP 502 from provider)", Retryable: true,
	})

	// Bitcoin/UTXO node errors (2100-2149)
	// Source: Bitcoin Core src/rpc/protocol.h
	LavaErrorNodeBitcoinWarmup = registerError(&LavaError{
		Code: 2101, Name: "NODE_BITCOIN_WARMUP", Category: CategoryExternal,
		Description: "Node still warming up (RPC_IN_WARMUP, Bitcoin -28)", Retryable: true,
	})
	LavaErrorNodeBitcoinInitialDownload = registerError(&LavaError{
		Code: 2102, Name: "NODE_BITCOIN_INITIAL_DOWNLOAD", Category: CategoryExternal,
		Description: "Node in initial block download (RPC_CLIENT_IN_INITIAL_DOWNLOAD, Bitcoin -10)", Retryable: true,
	})
	LavaErrorNodeBitcoinNotConnected = registerError(&LavaError{
		Code: 2103, Name: "NODE_BITCOIN_NOT_CONNECTED", Category: CategoryExternal,
		Description: "Node not connected / shutting down (RPC_CLIENT_NOT_CONNECTED, Bitcoin -9)", Retryable: true,
	})

	// Solana node errors (2150-2169)
	LavaErrorNodeSolanaUnhealthy = registerError(&LavaError{
		Code: 2150, Name: "NODE_SOLANA_UNHEALTHY", Category: CategoryExternal,
		Description: "Solana node behind/unhealthy (-32005)", Retryable: true,
	})

	// NEAR node errors (2170-2189)
	LavaErrorNodeNEARUnavailableShard = registerError(&LavaError{
		Code: 2170, Name: "NODE_NEAR_UNAVAILABLE_SHARD", Category: CategoryExternal,
		Description: "Node doesn't track requested shard", Retryable: true,
	})
)

// ---------------------------------------------------------------------------
// Layer C: Blockchain Errors (External) — range 3000-3999
// Errors from the blockchain execution/state layer.
// ---------------------------------------------------------------------------

var (
	// Transaction errors (3000-3099)
	LavaErrorChainNonceTooLow = registerError(&LavaError{
		Code: 3001, Name: "CHAIN_NONCE_TOO_LOW", Category: CategoryExternal,
		Description: "Nonce/sequence too low", Retryable: false,
	})
	LavaErrorChainNonceTooHigh = registerError(&LavaError{
		Code: 3002, Name: "CHAIN_NONCE_TOO_HIGH", Category: CategoryExternal,
		Description: "Nonce too high", Retryable: false,
	})
	LavaErrorChainInsufficientFunds = registerError(&LavaError{
		Code: 3003, Name: "CHAIN_INSUFFICIENT_FUNDS", Category: CategoryExternal,
		Description: "Insufficient funds for transfer/gas", Retryable: false,
	})
	LavaErrorChainGasTooLow = registerError(&LavaError{
		Code: 3004, Name: "CHAIN_GAS_TOO_LOW", Category: CategoryExternal,
		Description: "Intrinsic gas too low", Retryable: false,
	})
	LavaErrorChainGasLimitExceeded = registerError(&LavaError{
		Code: 3005, Name: "CHAIN_GAS_LIMIT_EXCEEDED", Category: CategoryExternal,
		Description: "Exceeds block gas limit", Retryable: false,
	})
	LavaErrorChainTxUnderpriced = registerError(&LavaError{
		Code: 3006, Name: "CHAIN_TX_UNDERPRICED", Category: CategoryExternal,
		Description: "Transaction gas price too low", Retryable: false,
	})
	LavaErrorChainTxAlreadyKnown = registerError(&LavaError{
		Code: 3007, Name: "CHAIN_TX_ALREADY_KNOWN", Category: CategoryExternal,
		Description: "Transaction already in mempool", Retryable: false,
	})
	LavaErrorChainTxReplacementUnderpriced = registerError(&LavaError{
		Code: 3008, Name: "CHAIN_TX_REPLACEMENT_UNDERPRICED", Category: CategoryExternal,
		Description: "Replacement tx gas too low", Retryable: false,
	})
	LavaErrorChainMempoolFull = registerError(&LavaError{
		Code: 3009, Name: "CHAIN_MEMPOOL_FULL", Category: CategoryExternal,
		Description: "Mempool/tx pool is full", Retryable: false,
	})
	LavaErrorChainTxTooLarge = registerError(&LavaError{
		Code: 3010, Name: "CHAIN_TX_TOO_LARGE", Category: CategoryExternal,
		Description: "Transaction exceeds size limit", Retryable: false,
	})
	LavaErrorChainMaxFeeBelowBase = registerError(&LavaError{
		Code: 3011, Name: "CHAIN_MAX_FEE_BELOW_BASE", Category: CategoryExternal,
		Description: "Max fee per gas below base fee", Retryable: false,
	})
	LavaErrorChainInvalidSequence = registerError(&LavaError{
		Code: 3012, Name: "CHAIN_INVALID_SEQUENCE", Category: CategoryExternal,
		Description: "Invalid sequence (Cosmos nonce equivalent)", Retryable: false,
	})
	LavaErrorChainInsufficientFee = registerError(&LavaError{
		Code: 3013, Name: "CHAIN_INSUFFICIENT_FEE", Category: CategoryExternal,
		Description: "Insufficient fee", Retryable: false,
	})
	LavaErrorChainTxRejected = registerError(&LavaError{
		Code: 3014, Name: "CHAIN_TX_REJECTED", Category: CategoryExternal,
		Description: "Transaction rejected by network rules", Retryable: false,
	})
	LavaErrorChainTxVersionUnsupported = registerError(&LavaError{
		Code: 3015, Name: "CHAIN_TX_VERSION_UNSUPPORTED", Category: CategoryExternal,
		Description: "Unsupported transaction version", Retryable: false,
	})
	LavaErrorChainDoubleSpend = registerError(&LavaError{
		Code: 3016, Name: "CHAIN_DOUBLE_SPEND", Category: CategoryExternal,
		Description: "Double spend / UTXO already spent", Retryable: false,
	})
	LavaErrorChainTxValidationFailed = registerError(&LavaError{
		Code: 3017, Name: "CHAIN_TX_VALIDATION_FAILED", Category: CategoryExternal,
		Description: "Transaction validation failed", Retryable: false,
	})
	LavaErrorChainInvalidSignature = registerError(&LavaError{
		Code: 3018, Name: "CHAIN_INVALID_SIGNATURE", Category: CategoryExternal,
		Description: "Invalid transaction signature", Retryable: false,
	})

	// Execution errors (3100-3199)
	LavaErrorChainExecutionReverted = registerError(&LavaError{
		Code: 3101, Name: "CHAIN_EXECUTION_REVERTED", Category: CategoryExternal,
		Description: "Smart contract execution reverted", Retryable: false,
	})
	LavaErrorChainOutOfGas = registerError(&LavaError{
		Code: 3102, Name: "CHAIN_OUT_OF_GAS", Category: CategoryExternal,
		Description: "Out of gas during execution", Retryable: false,
	})
	LavaErrorChainStackOverflow = registerError(&LavaError{
		Code: 3103, Name: "CHAIN_STACK_OVERFLOW", Category: CategoryExternal,
		Description: "Stack limit reached", Retryable: false,
	})
	LavaErrorChainInvalidOpcode = registerError(&LavaError{
		Code: 3104, Name: "CHAIN_INVALID_OPCODE", Category: CategoryExternal,
		Description: "Invalid opcode encountered", Retryable: false,
	})
	LavaErrorChainWriteProtection = registerError(&LavaError{
		Code: 3105, Name: "CHAIN_WRITE_PROTECTION", Category: CategoryExternal,
		Description: "Write in STATICCALL context", Retryable: false,
	})
	LavaErrorChainContractSizeExceeded = registerError(&LavaError{
		Code: 3106, Name: "CHAIN_CONTRACT_SIZE_EXCEEDED", Category: CategoryExternal,
		Description: "Contract code size exceeded", Retryable: false,
	})
	LavaErrorChainProgramError = registerError(&LavaError{
		Code: 3107, Name: "CHAIN_PROGRAM_ERROR", Category: CategoryExternal,
		Description: "Program/instruction error", Retryable: false,
	})
	LavaErrorChainVMError = registerError(&LavaError{
		Code: 3108, Name: "CHAIN_VM_ERROR", Category: CategoryExternal,
		Description: "VM execution error", Retryable: false,
	})
	LavaErrorChainAccountNotFound = registerError(&LavaError{
		Code: 3109, Name: "CHAIN_ACCOUNT_NOT_FOUND", Category: CategoryExternal,
		Description: "Account/contract does not exist", Retryable: false,
	})
	LavaErrorChainZkEVMOutOfCounters = registerError(&LavaError{
		Code: 3110, Name: "CHAIN_ZKEVM_OUT_OF_COUNTERS", Category: CategoryExternal,
		Description: "zkEVM prover constraint exceeded", Retryable: false,
	})

	// State/data errors (3200-3299)
	LavaErrorChainBlockNotFound = registerError(&LavaError{
		Code: 3201, Name: "CHAIN_BLOCK_NOT_FOUND", Category: CategoryExternal,
		Description: "Block not found", Retryable: true,
	})
	LavaErrorChainTxNotFound = registerError(&LavaError{
		Code: 3202, Name: "CHAIN_TX_NOT_FOUND", Category: CategoryExternal,
		Description: "Transaction not found", Retryable: true,
	})
	LavaErrorChainReceiptNotFound = registerError(&LavaError{
		Code: 3203, Name: "CHAIN_RECEIPT_NOT_FOUND", Category: CategoryExternal,
		Description: "Transaction receipt not found", Retryable: true,
	})
	LavaErrorChainStatePruned = registerError(&LavaError{
		Code: 3204, Name: "CHAIN_STATE_PRUNED", Category: CategoryExternal,
		Description: "State pruned/missing trie node", Retryable: true,
	})
	LavaErrorChainDataNotAvailable = registerError(&LavaError{
		Code: 3205, Name: "CHAIN_DATA_NOT_AVAILABLE", Category: CategoryExternal,
		Description: "Historical data not available", Retryable: true,
	})
	LavaErrorChainBlockTooOld = registerError(&LavaError{
		Code: 3206, Name: "CHAIN_BLOCK_TOO_OLD", Category: CategoryExternal,
		Description: "Block results only for recent blocks", Retryable: true,
	})
	LavaErrorChainBlockTooNew = registerError(&LavaError{
		Code: 3207, Name: "CHAIN_BLOCK_TOO_NEW", Category: CategoryExternal,
		Description: "Block ahead of node's latest", Retryable: false,
	})
	LavaErrorChainLogResponseTooLarge = registerError(&LavaError{
		Code: 3208, Name: "CHAIN_LOG_RESPONSE_TOO_LARGE", Category: CategoryExternal,
		Description: "Log query returned too many results", Retryable: false,
	})

	// Solana-specific (3300-3319) — Tier 2
	// Source: Solana RPC custom error codes (agave rpc-client-api/src/custom_error.rs)
	LavaErrorChainSolanaMissingLongTerm = registerError(&LavaError{
		Code: 3302, Name: "CHAIN_SOLANA_MISSING_LONG_TERM", Category: CategoryExternal,
		Description: "Slot missing in long-term storage (-32009)", Retryable: false,
	})
	LavaErrorChainSolanaLedgerJump = registerError(&LavaError{
		Code: 3303, Name: "CHAIN_SOLANA_LEDGER_JUMP", Category: CategoryExternal,
		Description: "Missing due to ledger jump/snapshot (-32007)", Retryable: true,
	})
	LavaErrorChainSolanaBlockhashNotFound = registerError(&LavaError{
		Code: 3304, Name: "CHAIN_SOLANA_BLOCKHASH_NOT_FOUND", Category: CategoryExternal,
		Description: "Blockhash not found/expired", Retryable: false,
	})
	LavaErrorChainSolanaSimulationFailed = registerError(&LavaError{
		Code: 3305, Name: "CHAIN_SOLANA_SIMULATION_FAILED", Category: CategoryExternal,
		Description: "Transaction simulation failed (-32002)", Retryable: false,
	})
	LavaErrorChainSolanaSignatureVerifyFailed = registerError(&LavaError{
		Code: 3306, Name: "CHAIN_SOLANA_SIGNATURE_VERIFY_FAILED", Category: CategoryExternal,
		Description: "Signature verification failure (-32003)", Retryable: false,
	})
	LavaErrorChainSolanaExcludedFromIndex = registerError(&LavaError{
		Code: 3307, Name: "CHAIN_SOLANA_EXCLUDED_FROM_INDEX", Category: CategoryExternal,
		Description: "Excluded from account secondary indexes (-32010)", Retryable: false,
	})
	LavaErrorChainSolanaSignatureLengthMismatch = registerError(&LavaError{
		Code: 3308, Name: "CHAIN_SOLANA_SIGNATURE_LENGTH_MISMATCH", Category: CategoryExternal,
		Description: "Incorrectly formatted signature (-32013)", Retryable: false,
	})
	LavaErrorChainSolanaBlockStatusUnavailable = registerError(&LavaError{
		Code: 3309, Name: "CHAIN_SOLANA_BLOCK_STATUS_UNAVAILABLE", Category: CategoryExternal,
		Description: "Block status unavailable (-32014)", Retryable: true,
	})
	LavaErrorChainSolanaTxVersionUnsupported = registerError(&LavaError{
		Code: 3310, Name: "CHAIN_SOLANA_TX_VERSION_UNSUPPORTED", Category: CategoryExternal,
		Description: "Transaction version not supported (-32015)", Retryable: false,
	})
	LavaErrorChainSolanaMinContextSlotNotReached = registerError(&LavaError{
		Code: 3311, Name: "CHAIN_SOLANA_MIN_CONTEXT_SLOT_NOT_REACHED", Category: CategoryExternal,
		Description: "Minimum context slot not reached (-32016)", Retryable: true,
	})

	// Starknet-specific (3320-3349) — Tier 2
	// Source: Starknet JSON-RPC spec error codes
	LavaErrorChainStarknetFailedToReceiveTx = registerError(&LavaError{
		Code: 3320, Name: "CHAIN_STARKNET_FAILED_TO_RECEIVE_TX", Category: CategoryExternal,
		Description: "Sequencer rejected transaction (code 1)", Retryable: false,
	})
	LavaErrorChainStarknetClassNotFound = registerError(&LavaError{
		Code: 3321, Name: "CHAIN_STARKNET_CLASS_NOT_FOUND", Category: CategoryExternal,
		Description: "Class hash not found (code 28)", Retryable: false,
	})
	LavaErrorChainStarknetCompilationFailed = registerError(&LavaError{
		Code: 3322, Name: "CHAIN_STARKNET_COMPILATION_FAILED", Category: CategoryExternal,
		Description: "Sierra to CASM compilation failed (code 56)", Retryable: false,
	})
	LavaErrorChainStarknetClassAlreadyDeclared = registerError(&LavaError{
		Code: 3323, Name: "CHAIN_STARKNET_CLASS_ALREADY_DECLARED", Category: CategoryExternal,
		Description: "Class already declared (code 51)", Retryable: false,
	})
	LavaErrorChainStarknetContractError = registerError(&LavaError{
		Code: 3324, Name: "CHAIN_STARKNET_CONTRACT_ERROR", Category: CategoryExternal,
		Description: "Contract error during execution (code 40)", Retryable: false,
	})
	LavaErrorChainStarknetTxExecError = registerError(&LavaError{
		Code: 3325, Name: "CHAIN_STARKNET_TX_EXEC_ERROR", Category: CategoryExternal,
		Description: "Transaction execution error (code 41)", Retryable: false,
	})
	LavaErrorChainStarknetInvalidNonce = registerError(&LavaError{
		Code: 3326, Name: "CHAIN_STARKNET_INVALID_NONCE", Category: CategoryExternal,
		Description: "Invalid transaction nonce (code 52)", Retryable: false,
	})
	LavaErrorChainStarknetInsufficientFee = registerError(&LavaError{
		Code: 3327, Name: "CHAIN_STARKNET_INSUFFICIENT_FEE", Category: CategoryExternal,
		Description: "Insufficient max fee (code 53)", Retryable: false,
	})
	LavaErrorChainStarknetInsufficientBalance = registerError(&LavaError{
		Code: 3328, Name: "CHAIN_STARKNET_INSUFFICIENT_BALANCE", Category: CategoryExternal,
		Description: "Insufficient account balance (code 54)", Retryable: false,
	})
	LavaErrorChainStarknetValidationFailure = registerError(&LavaError{
		Code: 3329, Name: "CHAIN_STARKNET_VALIDATION_FAILURE", Category: CategoryExternal,
		Description: "Account validation failed (code 55)", Retryable: false,
	})
	LavaErrorChainStarknetContractNotFound = registerError(&LavaError{
		Code: 3330, Name: "CHAIN_STARKNET_CONTRACT_NOT_FOUND", Category: CategoryExternal,
		Description: "Contract address not found (code 20)", Retryable: false,
	})
	LavaErrorChainStarknetBlockNotFound = registerError(&LavaError{
		Code: 3331, Name: "CHAIN_STARKNET_BLOCK_NOT_FOUND", Category: CategoryExternal,
		Description: "Block not found (code 24)", Retryable: true,
	})
	LavaErrorChainStarknetTxHashNotFound = registerError(&LavaError{
		Code: 3332, Name: "CHAIN_STARKNET_TX_HASH_NOT_FOUND", Category: CategoryExternal,
		Description: "Transaction hash not found (code 29)", Retryable: true,
	})
	LavaErrorChainStarknetDuplicateTx = registerError(&LavaError{
		Code: 3333, Name: "CHAIN_STARKNET_DUPLICATE_TX", Category: CategoryExternal,
		Description: "Duplicate transaction in mempool (code 59)", Retryable: false,
	})
	LavaErrorChainStarknetUnsupportedTxVersion = registerError(&LavaError{
		Code: 3334, Name: "CHAIN_STARKNET_UNSUPPORTED_TX_VERSION", Category: CategoryExternal,
		Description: "Unsupported transaction version (code 61)", Retryable: false,
	})
	LavaErrorChainStarknetUnexpectedError = registerError(&LavaError{
		Code: 3335, Name: "CHAIN_STARKNET_UNEXPECTED_ERROR", Category: CategoryExternal,
		Description: "Unexpected server error (code 63)", Retryable: true,
	})

	// Bitcoin/UTXO-specific (3340-3359) — Tier 2
	// Source: Bitcoin Core src/rpc/protocol.h
	LavaErrorChainBitcoinVerifyError = registerError(&LavaError{
		Code: 3341, Name: "CHAIN_BITCOIN_VERIFY_ERROR", Category: CategoryExternal,
		Description: "Transaction/block verification failed (RPC_VERIFY_ERROR, Bitcoin -25)", Retryable: false,
	})
	LavaErrorChainBitcoinVerifyRejected = registerError(&LavaError{
		Code: 3342, Name: "CHAIN_BITCOIN_VERIFY_REJECTED", Category: CategoryExternal,
		Description: "Transaction rejected by network rules (RPC_VERIFY_REJECTED, Bitcoin -26)", Retryable: false,
	})
	LavaErrorChainBitcoinAlreadyInChain = registerError(&LavaError{
		Code: 3343, Name: "CHAIN_BITCOIN_ALREADY_IN_CHAIN", Category: CategoryExternal,
		Description: "Transaction already confirmed (RPC_VERIFY_ALREADY_IN_CHAIN, Bitcoin -27)", Retryable: false,
	})
	LavaErrorChainBitcoinInsufficientFunds = registerError(&LavaError{
		Code: 3344, Name: "CHAIN_BITCOIN_INSUFFICIENT_FUNDS", Category: CategoryExternal,
		Description: "UTXO coin selection failed (RPC_WALLET_INSUFFICIENT_FUNDS, Bitcoin -6)", Retryable: false,
	})

	// NEAR-specific (3360-3379) — Tier 2
	// Source: NEAR RPC docs (error names in JSON-RPC error.cause.name)
	LavaErrorChainNEARUnknownBlock = registerError(&LavaError{
		Code: 3360, Name: "CHAIN_NEAR_UNKNOWN_BLOCK", Category: CategoryExternal,
		Description: "Block not found or garbage-collected (UNKNOWN_BLOCK)", Retryable: true,
	})
	LavaErrorChainNEARUnknownChunk = registerError(&LavaError{
		Code: 3361, Name: "CHAIN_NEAR_UNKNOWN_CHUNK", Category: CategoryExternal,
		Description: "Chunk not found (UNKNOWN_CHUNK)", Retryable: true,
	})
	LavaErrorChainNEARInvalidShardID = registerError(&LavaError{
		Code: 3362, Name: "CHAIN_NEAR_INVALID_SHARD_ID", Category: CategoryExternal,
		Description: "Shard ID does not exist (INVALID_SHARD_ID)", Retryable: false,
	})
	LavaErrorChainNEARGasPriceOverflow = registerError(&LavaError{
		Code: 3363, Name: "CHAIN_NEAR_GAS_PRICE_OVERFLOW", Category: CategoryExternal,
		Description: "Gas price computation overflowed", Retryable: false,
	})
	LavaErrorChainNEARNotSyncedYet = registerError(&LavaError{
		Code: 3364, Name: "CHAIN_NEAR_NOT_SYNCED_YET", Category: CategoryExternal,
		Description: "Node still syncing (NOT_SYNCED_YET)", Retryable: true,
	})

	// XRP-specific (3380-3399) — Tier 2
	LavaErrorChainXRPPathDry = registerError(&LavaError{
		Code: 3381, Name: "CHAIN_XRP_PATH_DRY", Category: CategoryExternal,
		Description: "Payment path has no liquidity (tec 128)", Retryable: false,
	})
	LavaErrorChainXRPAmendmentBlocked = registerError(&LavaError{
		Code: 3382, Name: "CHAIN_XRP_AMENDMENT_BLOCKED", Category: CategoryExternal,
		Description: "Server is amendment-blocked", Retryable: false,
	})
	LavaErrorChainXRPNoNetwork = registerError(&LavaError{
		Code: 3383, Name: "CHAIN_XRP_NO_NETWORK", Category: CategoryExternal,
		Description: "Not connected to XRP network", Retryable: true,
	})

	// TON-specific (3400-3419) — Tier 2
	LavaErrorChainTONMessageExpired = registerError(&LavaError{
		Code: 3401, Name: "CHAIN_TON_MESSAGE_EXPIRED", Category: CategoryExternal,
		Description: "External message TTL expired", Retryable: false,
	})
	LavaErrorChainTONMessageRejected = registerError(&LavaError{
		Code: 3402, Name: "CHAIN_TON_MESSAGE_REJECTED", Category: CategoryExternal,
		Description: "External message not accepted", Retryable: false,
	})
	LavaErrorChainTONLiteServerTimeout = registerError(&LavaError{
		Code: 3403, Name: "CHAIN_TON_LITE_SERVER_TIMEOUT", Category: CategoryExternal,
		Description: "Lite-server backend timeout", Retryable: true,
	})
)

// ---------------------------------------------------------------------------
// Layer D: User Errors (External) — range 4000-4999
// Errors caused by malformed or invalid client requests.
// ---------------------------------------------------------------------------

var (
	LavaErrorUserParseError = registerError(&LavaError{
		Code: 4001, Name: "USER_PARSE_ERROR", Category: CategoryExternal,
		Description: "Invalid JSON in request", Retryable: false,
	})
	LavaErrorUserInvalidRequest = registerError(&LavaError{
		Code: 4002, Name: "USER_INVALID_REQUEST", Category: CategoryExternal,
		Description: "Request is not a valid JSON-RPC/REST/gRPC object", Retryable: false,
	})
	LavaErrorUserInvalidParams = registerError(&LavaError{
		Code: 4003, Name: "USER_INVALID_PARAMS", Category: CategoryExternal,
		Description: "Invalid method parameters", Retryable: false,
	})
	LavaErrorUserInvalidBlockFormat = registerError(&LavaError{
		Code: 4006, Name: "USER_INVALID_BLOCK_FORMAT", Category: CategoryExternal,
		Description: "Invalid block number format (e.g., non-hex)", Retryable: false,
	})
	LavaErrorUserInvalidAddress = registerError(&LavaError{
		Code: 4007, Name: "USER_INVALID_ADDRESS", Category: CategoryExternal,
		Description: "Invalid address format", Retryable: false,
	})
	LavaErrorUserMissingRequiredParams = registerError(&LavaError{
		Code: 4008, Name: "USER_MISSING_REQUIRED_PARAMS", Category: CategoryExternal,
		Description: "Missing required parameters", Retryable: false,
	})
	LavaErrorUserBatchTooLarge = registerError(&LavaError{
		Code: 4009, Name: "USER_BATCH_TOO_LARGE", Category: CategoryExternal,
		Description: "Batch request exceeds size limit", Retryable: false,
	})
	LavaErrorUserRequestTooLarge = registerError(&LavaError{
		Code: 4010, Name: "USER_REQUEST_TOO_LARGE", Category: CategoryExternal,
		Description: "Request body exceeds size limit", Retryable: false,
	})
	LavaErrorUserInvalidSubscription = registerError(&LavaError{
		Code: 4011, Name: "USER_INVALID_SUBSCRIPTION", Category: CategoryExternal,
		Description: "Invalid subscription parameters", Retryable: false,
	})
	LavaErrorUserIDMismatch = registerError(&LavaError{
		Code: 4012, Name: "USER_ID_MISMATCH", Category: CategoryExternal,
		Description: "Request/response ID mismatch", Retryable: false,
	})
	LavaErrorUserInvalidHex = registerError(&LavaError{
		Code: 4013, Name: "USER_INVALID_HEX", Category: CategoryExternal,
		Description: "Invalid hex encoding", Retryable: false,
	})
)
