# Error Categorization & Standardized Logging Plan

## 1. Design: Error Layer Taxonomy

Four layers, each with a dedicated Lava error code range.

> **Retryable** means: retrying the same relay (with the same parameters) to a different endpoint/provider has a chance of succeeding. `Yes` = retry on a different provider. `No` = do not retry.

### Reserved Codes

| Code | Name | Description | Retryable |
|------|------|-------------|-----------|
| 0 | `UNKNOWN_ERROR` | Unclassified error — no matcher matched | Yes |

### Layer A: Protocol Errors (`PROTOCOL_*` — range 1000-1999) — Category: Internal
Errors raised from within the Lava protocol itself — not from nodes or chains.

| Code | Name | Description | Retryable |
|------|------|-------------|-----------|
| 1001 | `PROTOCOL_CONNECTION_TIMEOUT` | Network operation timed out connecting to provider | Yes |
| 1002 | `PROTOCOL_CONNECTION_REFUSED` | Provider connection refused | Yes |
| 1003 | `PROTOCOL_DNS_FAILURE` | DNS resolution failed | Yes |
| 1004 | `PROTOCOL_TLS_MISMATCH` | HTTP/HTTPS protocol mismatch | No |
| 1005 | `PROTOCOL_CONNECTION_RESET` | Connection reset by peer | Yes |
| 1006 | `PROTOCOL_CONNECTION_CLOSED` | Connection closed (EOF) | Yes |
| 1007 | `PROTOCOL_CONTEXT_DEADLINE` | Caller's context.Context deadline expired before the relay completed | Yes |
| 1008 | `PROTOCOL_CONTEXT_CANCELED` | Request context was canceled (client disconnect or relay race resolved) | No |
| 1009 | `PROTOCOL_NETWORK_UNREACHABLE` | Network or host unreachable (no route) | Yes |
| 1010 | `PROTOCOL_NO_PROVIDERS` | No providers/pairings available | No |
| 1011 | `PROTOCOL_ALL_ENDPOINTS_DISABLED` | All provider endpoints disabled | No |
| 1012 | `PROTOCOL_PROVIDER_UNAVAILABLE` | Provider service unavailable (gRPC UNAVAILABLE) | Yes |
| 1013 | `PROTOCOL_PROVIDER_ABORTED` | Provider aborted (gRPC ABORTED) | Yes |
| 1014 | `PROTOCOL_PROVIDER_DATA_LOSS` | Provider data loss (gRPC DATA_LOSS) | Yes |
| 1020 | `PROTOCOL_RATE_LIMITED` | Lava-side rate limit exceeded (SubCategoryRateLimit) | No |
| 1021 | `PROTOCOL_MAX_CU_EXCEEDED` | Maximum compute units exceeded for session | No |
| 1022 | `PROTOCOL_BATCH_SIZE_EXCEEDED` | Batch request size exceeded limit (SubCategoryUserError) | No |
| 1030 | `PROTOCOL_SESSION_NOT_FOUND` | Session does not exist | No |
| 1031 | `PROTOCOL_EPOCH_MISMATCH` | Epoch mismatch or too old | No |
| 1032 | `PROTOCOL_CONSUMER_BLOCKED` | Consumer is blocklisted | No |
| 1033 | `PROTOCOL_CONSUMER_NOT_REGISTERED` | Consumer not registered | No |
| 1034 | `PROTOCOL_RELAY_NUMBER_MISMATCH` | Relay number mismatch | No |
| 1035 | `PROTOCOL_SESSION_OUT_OF_SYNC` | Session out of sync | No |
| 1040 | `PROTOCOL_FINALIZATION_ERROR` | Provider finalization data incorrect | Yes |
| 1041 | `PROTOCOL_CONSISTENCY_ERROR` | Response consistency validation failed | Yes |
| 1042 | `PROTOCOL_HASH_CONSENSUS_ERROR` | Conflicting response hashes detected | Yes |
| 1043 | `PROTOCOL_NO_RESPONSE_TIMEOUT` | Relay race timeout — no provider returned a response within the protocol deadline | Yes |
| 1050 | `PROTOCOL_SUBSCRIPTION_NOT_FOUND` | Subscription not found | No |
| 1051 | `PROTOCOL_SUBSCRIPTION_INIT_FAILED` | Failed to initialize subscription | No |
| 1052 | `PROTOCOL_WEBSOCKET_IDLE_TIMEOUT` | WebSocket idle timeout | No |
| 1053 | `PROTOCOL_SUBSCRIPTION_ALREADY_EXISTS` | Subscription already exists for this consumer/key | No |

### Layer B: Node Errors (`NODE_*` — range 2000-2999) — Category: External
Errors returned by the blockchain node itself (not execution/state errors).

| Code | Name | Description | Retryable | Standard Code |
|------|------|-------------|-----------|---------------|
| **Generic Node Errors (2000-2099)** |||||
| 2001 | `NODE_METHOD_NOT_FOUND` | Method does not exist on this node (unknown to the API surface); non-retryable (SubCategoryUnsupportedMethod) | No | JSON-RPC -32601 |
| 2002 | `NODE_METHOD_NOT_SUPPORTED` | Method exists but is DISABLED on this specific node (provider tier / policy / admin config). Retryable on a different provider. | Yes | JSON-RPC -32004 |
| 2003 | `NODE_INTERNAL_ERROR` | Internal node error | Yes | JSON-RPC -32603 |
| 2004 | `NODE_SERVER_ERROR` | Generic server error | Yes | JSON-RPC -32000 |
| 2005 | `NODE_RATE_LIMITED` | Rate limited by node (SubCategoryRateLimit) | Yes | HTTP 429 / MessageContains("rate limit") |
| 2006 | `NODE_SERVICE_UNAVAILABLE` | Node temporarily unavailable | Yes | HTTP 503 |
| 2007 | `NODE_SYNCING` | Node is syncing/catching up | Yes | MessageContains("node is syncing" / "catching up to the chain") |
| 2008 | `NODE_UNIMPLEMENTED` | gRPC method unimplemented (SubCategoryUnsupportedMethod) | No | gRPC 12 |
| 2009 | `NODE_ENDPOINT_NOT_FOUND` | REST endpoint not found (SubCategoryUnsupportedMethod) | No | HTTP 404 |
| 2010 | `NODE_METHOD_NOT_ALLOWED` | REST method not allowed (SubCategoryUnsupportedMethod) | No | HTTP 405 |
| 2011 | `NODE_LIMIT_EXCEEDED` | Request exceeds node limit (e.g., eth_getLogs range) (SubCategoryRateLimit) | No | JSON-RPC -32005 |
| 2012 | `NODE_RESOURCE_NOT_FOUND` | Resource not found at node level | Yes | JSON-RPC -32001 |
| 2013 | `NODE_RESOURCE_UNAVAILABLE` | Resource exists but unavailable | Yes | JSON-RPC -32002 |
| 2014 | `NODE_GATEWAY_TIMEOUT` | Gateway timeout (HTTP 504 from provider) | Yes | HTTP 504 |
| 2015 | `NODE_BAD_GATEWAY` | Bad gateway (HTTP 502 from provider) | Yes | HTTP 502 |
| **Bitcoin/UTXO Node Errors (2100-2149)** |||||
| 2101 | `NODE_BITCOIN_WARMUP` | Node still warming up (Bitcoin -28 / RPC_IN_WARMUP) | Yes | — |
| 2102 | `NODE_BITCOIN_INITIAL_DOWNLOAD` | Node in initial block download (Bitcoin -10 / RPC_CLIENT_IN_INITIAL_DOWNLOAD) | Yes | — |
| 2103 | `NODE_BITCOIN_NOT_CONNECTED` | Node has no peers (Bitcoin -9 / RPC_CLIENT_NOT_CONNECTED) | Yes | — |
| **Solana Node Errors (2150-2169)** |||||
| 2150 | `NODE_SOLANA_UNHEALTHY` | Solana node behind/unhealthy (-32005) | Yes | — |

### Layer C: Blockchain Errors (`CHAIN_*` — range 3000-3999) — Category: External
Errors from the blockchain execution/state layer — transaction failures, state queries, etc.

Uses a **tiered classification system** (see Section 3 for details):
- **Tier 1 (Generic):** Semantic codes covering all chains (e.g., `CHAIN_NONCE_TOO_LOW`)
- **Tier 2 (Chain-specific):** Distinct codes ONLY where retryability differs from the generic pattern

| Code | Name | Description | Retryable | Chains |
|------|------|-------------|-----------|--------|
| **Transaction Errors (3000-3099)** |||||
| 3001 | `CHAIN_NONCE_TOO_LOW` | Nonce/sequence too low | No | EVM, Cosmos, Starknet, XRP, NEAR |
| 3002 | `CHAIN_NONCE_TOO_HIGH` | Nonce too high | No | EVM |
| 3003 | `CHAIN_INSUFFICIENT_FUNDS` | Insufficient funds for transfer/gas | No | Universal |
| 3004 | `CHAIN_GAS_TOO_LOW` | Intrinsic gas too low | No | EVM |
| 3005 | `CHAIN_GAS_LIMIT_EXCEEDED` | Exceeds block gas limit | No | EVM |
| 3006 | `CHAIN_TX_UNDERPRICED` | Transaction gas price too low | No | EVM |
| 3007 | `CHAIN_TX_ALREADY_KNOWN` | Transaction already in mempool | No | EVM, Starknet, XRP |
| 3008 | `CHAIN_TX_REPLACEMENT_UNDERPRICED` | Replacement tx gas too low | No | EVM |
| 3009 | `CHAIN_MEMPOOL_FULL` | Mempool/tx pool is full | No | EVM, Cosmos |
| 3010 | `CHAIN_TX_TOO_LARGE` | Transaction exceeds size limit | No | EVM, Solana |
| 3011 | `CHAIN_MAX_FEE_BELOW_BASE` | Max fee per gas below base fee | No | EVM (EIP-1559) |
| 3012 | `CHAIN_INVALID_SEQUENCE` | Invalid sequence (Cosmos nonce equivalent) | No | Cosmos |
| 3013 | `CHAIN_INSUFFICIENT_FEE` | Insufficient fee | No | Cosmos |
| 3014 | `CHAIN_TX_REJECTED` | Transaction rejected by network rules | No | Universal |
| 3015 | `CHAIN_DOUBLE_SPEND` | Double spend / UTXO already spent | No | Bitcoin/UTXO |
| 3016 | `CHAIN_INVALID_SIGNATURE` | Invalid transaction signature | No | Universal |
| **Execution Errors (3100-3199)** |||||
| 3101 | `CHAIN_EXECUTION_REVERTED` | Smart contract execution reverted | No | EVM, Starknet, NEAR, TON |
| 3102 | `CHAIN_OUT_OF_GAS` | Out of gas during execution | No | EVM, Cosmos, TON |
| 3103 | `CHAIN_STACK_OVERFLOW` | Stack limit reached | No | EVM, TON |
| 3104 | `CHAIN_INVALID_OPCODE` | Invalid opcode encountered | No | EVM, TON |
| 3105 | `CHAIN_WRITE_PROTECTION` | Write in STATICCALL context | No | EVM |
| 3106 | `CHAIN_CONTRACT_SIZE_EXCEEDED` | Contract bytecode exceeds 24KB EIP-170 limit (Geth: 'max code size exceeded') | No | EVM |
| 3107 | `CHAIN_ACCOUNT_NOT_FOUND` | Account/contract does not exist | No | Cosmos |
| 3108 | `CHAIN_ZKEVM_OUT_OF_COUNTERS` | Polygon zkEVM prover exceeded circuit counter budget | No | EVM (Polygon zkEVM) |
| **State/Data Errors (3200-3299)** |||||
| 3201 | `CHAIN_BLOCK_NOT_FOUND` | Block not found | Yes | Universal |
| 3202 | `CHAIN_TX_NOT_FOUND` | Transaction not found | Yes | Universal |
| 3203 | `CHAIN_RECEIPT_NOT_FOUND` | Transaction receipt not found | Yes | EVM (Cosmos-EVM variant) |
| 3204 | `CHAIN_STATE_PRUNED` | State pruned/missing trie node | Yes | EVM |
| 3205 | `CHAIN_DATA_NOT_AVAILABLE` | Historical data not available | Yes | Universal |
| 3206 | `CHAIN_BLOCK_TOO_OLD` | Block results only for recent blocks | Yes | Cosmos |
| 3207 | `CHAIN_LOG_RESPONSE_TOO_LARGE` | Log query returned too many results | No | EVM |
| **Solana-Specific (3300-3319) — Tier 2** |||||
| 3302 | `CHAIN_SOLANA_MISSING_LONG_TERM` | Slot missing in long-term storage (-32009) | No | Solana |
| 3303 | `CHAIN_SOLANA_LEDGER_JUMP` | Missing due to ledger jump/snapshot (-32007) | Yes | Solana |
| 3304 | `CHAIN_SOLANA_BLOCKHASH_NOT_FOUND` | Blockhash not found/expired | No | Solana |
| 3305 | `CHAIN_SOLANA_SIMULATION_FAILED` | Transaction simulation failed (-32002) | No | Solana |
| 3306 | `CHAIN_SOLANA_SIGNATURE_VERIFY_FAILED` | Signature verification failure (-32003) | No | Solana |
| 3307 | `CHAIN_SOLANA_EXCLUDED_FROM_INDEX` | Excluded from account secondary indexes (-32010) | No | Solana |
| 3308 | `CHAIN_SOLANA_SIGNATURE_LENGTH_MISMATCH` | Signature length mismatch (-32013) | No | Solana |
| 3309 | `CHAIN_SOLANA_BLOCK_STATUS_UNAVAILABLE` | Block status unavailable (-32014) | No | Solana |
| 3310 | `CHAIN_SOLANA_TX_VERSION_UNSUPPORTED` | Transaction version not supported (-32015) | No | Solana |
| 3311 | `CHAIN_SOLANA_MIN_CONTEXT_SLOT_NOT_REACHED` | Minimum context slot not reached (-32016) | No | Solana |
| **Starknet-Specific (3320-3339) — Tier 2** |||||
| 3320 | `CHAIN_STARKNET_FAILED_TO_RECEIVE_TX` | Failed to receive tx (code 1) | No | Starknet |
| 3321 | `CHAIN_STARKNET_CLASS_NOT_FOUND` | Class hash not found (code 28) | No | Starknet |
| 3322 | `CHAIN_STARKNET_COMPILATION_FAILED` | Sierra to CASM compilation failed (code 56) | No | Starknet |
| 3323 | `CHAIN_STARKNET_CLASS_ALREADY_DECLARED` | Class already declared (code 51) | No | Starknet |
| 3324 | `CHAIN_STARKNET_CONTRACT_ERROR` | Contract error during execution (code 40) | No | Starknet |
| 3325 | `CHAIN_STARKNET_TX_EXEC_ERROR` | Tx exec error (code 41) | No | Starknet |
| 3326 | `CHAIN_STARKNET_INVALID_NONCE` | Invalid nonce (code 52) | No | Starknet |
| 3327 | `CHAIN_STARKNET_INSUFFICIENT_FEE` | Insufficient fee (code 53) | No | Starknet |
| 3328 | `CHAIN_STARKNET_INSUFFICIENT_BALANCE` | Insufficient balance (code 54) | No | Starknet |
| 3329 | `CHAIN_STARKNET_VALIDATION_FAILURE` | Validation failure (code 55) | No | Starknet |
| 3330 | `CHAIN_STARKNET_CONTRACT_NOT_FOUND` | Contract not found (code 20) | No | Starknet |
| 3331 | `CHAIN_STARKNET_BLOCK_NOT_FOUND` | Block not found (code 24) | No | Starknet |
| 3332 | `CHAIN_STARKNET_TX_HASH_NOT_FOUND` | Tx hash not found (code 29) | No | Starknet |
| 3333 | `CHAIN_STARKNET_DUPLICATE_TX` | Duplicate tx (code 59) | No | Starknet |
| 3334 | `CHAIN_STARKNET_TX_VERSION_UNSUPPORTED` | Unsupported tx version (code 61) | No | Starknet |
| 3335 | `CHAIN_STARKNET_UNEXPECTED_ERROR` | Unexpected error (code 63) | No | Starknet |
| **Bitcoin/UTXO-Specific (3340-3359) — Tier 2** |||||
| 3341 | `CHAIN_BITCOIN_VERIFY_ERROR` | Transaction verification failed (-25 / RPC_VERIFY_ERROR) | No | Bitcoin/UTXO |
| 3342 | `CHAIN_BITCOIN_VERIFY_REJECTED` | Transaction rejected by rules (-26 / RPC_VERIFY_REJECTED) | No | Bitcoin/UTXO |
| 3343 | `CHAIN_BITCOIN_ALREADY_IN_CHAIN` | Transaction already confirmed (-27 / RPC_VERIFY_ALREADY_IN_CHAIN) | No | Bitcoin/UTXO |
| 3344 | `CHAIN_BITCOIN_WALLET_INSUFFICIENT_FUNDS` | Wallet UTXO coin selection failed (-6 / RPC_WALLET_INSUFFICIENT_FUNDS). Distinct from EVM CHAIN_INSUFFICIENT_FUNDS (3003) which is a tx submission failure. | No | Bitcoin/UTXO |
| **NEAR-Specific (3360-3379) — Tier 2** |||||
| 3360 | `CHAIN_NEAR_UNKNOWN_BLOCK` | Block not found or garbage-collected (UNKNOWN_BLOCK) | Yes | NEAR |
| 3361 | `CHAIN_NEAR_UNKNOWN_CHUNK` | Chunk not found (UNKNOWN_CHUNK) | Yes | NEAR |
| 3362 | `CHAIN_NEAR_INVALID_SHARD_ID` | Shard ID does not exist (INVALID_SHARD_ID) | No | NEAR |
| 3363 | `CHAIN_NEAR_NOT_SYNCED_YET` | Node still syncing (NOT_SYNCED_YET) | Yes | NEAR |

### Layer D: User Errors (`USER_*` — range 4000-4999) — Category: External
Errors caused by malformed or invalid client requests — classified by nature of error, regardless of where caught (pre-forwarding by Lava or returned by node).

| Code | Name | Description | Retryable | Standard Code |
|------|------|-------------|-----------|---------------|
All Layer D codes carry `SubCategoryUserError` — the consumer hot path short-circuits retries and charges zero CU for invalid client input.

| Code | Name | Description | Retryable | Standard Code |
|------|------|-------------|-----------|---------------|
| 4001 | `USER_PARSE_ERROR` | Invalid JSON in request | No | JSON-RPC -32700 |
| 4002 | `USER_INVALID_REQUEST` | Request is not a valid JSON-RPC/REST/gRPC object | No | JSON-RPC -32600 |
| 4003 | `USER_INVALID_PARAMS` | Invalid method parameters | No | JSON-RPC -32602 |
| 4004 | `USER_INVALID_BLOCK_FORMAT` | Invalid block number format (e.g., non-hex) | No | MessageContains("hex string without 0x prefix" / "invalid block number" / "invalid block hash") |
| 4005 | `USER_INVALID_ADDRESS` | Invalid address format | No | MessageContains("bad address checksum" / "invalid address") |
| 4006 | `USER_REQUEST_TOO_LARGE` | Request body exceeds size limit | No | HTTP 413 |
| 4007 | `USER_INVALID_HEX` | Invalid hex encoding | No | MessageContains("hex string has odd length" / "invalid hex") |

---

## 2. Design: Error Code Format

Each Lava error has:
- **Lava Code** (uint32): Internal code for logging, metrics, counting (1001, 2001, etc.)
- **Name** (string): Human-readable constant name (`PROTOCOL_CONNECTION_TIMEOUT`)
- **Category** (enum): `Internal` (Lava-introduced) or `External` (pass-through)
- **SubCategory** (enum): Finer classification (e.g., `SubCategoryUnsupportedMethod`)
- **Retryable** (bool): Whether the error warrants retry on a different provider
- **Description** (string): Human-readable explanation

### Visibility Scope
Lava error codes are **internal only** — they live within the Lava protocol layer:
- **Smart Router path:** Between router and endpoint (user and node never see Lava codes)
- **Decentralized path:** Between consumer and provider (user and node never see Lava codes)
- External responses always use standard protocol codes (JSON-RPC, gRPC, HTTP)
- Lava codes appear in **logs and metrics only**

---

## 3. Design: Tiered Error Classification System

Errors are classified using a two-level lookup with chain-family awareness.

### Chain Families

The authoritative map lives in `protocol/common/error_registry.go` (`chainFamilyMap`). `GetChainFamilyOrDefault` returns `ChainFamilyUnknown` (a dedicated sentinel, NOT `EVM`) when a chain ID is not registered — Tier-2 lookups against the sentinel intentionally miss so classification falls through to transport-scoped Tier-1 matchers rather than silently inheriting another family's semantics.

Two helper functions (`common.IsSolanaFamily` etc.) delegate to this map so there is a single source of truth.

| ChainFamily | Chain IDs |
|-------------|-----------|
| `EVM` | ETH1, SEP1, HOL1, ARBITRUM, POLYGON, BASE, OPTM, AVAX, BSC, BLAST, FTM250, SONIC, ... |
| `Solana` | SOLANA, SOLANAT, KOII, KOIIT |
| `Bitcoin` | BTC, BTCT, LTC, LTCT, DOGE, DOGET, BCH, BCHT |
| `CosmosSDK` | COSMOSHUB, LAVA, LAV1, AXELAR, EVMOS, OSMOSIS, JUN1, CELESTIA, ... |
| `Starknet` | STRK, STRKS |
| `Aptos` | APT1 |
| `Sui` | SUIT |
| `NEAR` | NEAR, NEART |
| `XRP` | XRP, XRPT |
| `Stellar` | XLM, XLMT |
| `TON` | TON, TONT |
| `Tron` | TRX, TRXT |
| `Cardano` | CARDANO, CARDANOT |
| `PolygonZkEVM` | (no chain IDs currently map to this family — matchers live in Tier-1 EVM) |
| `Unknown` | sentinel — Tier-2 lookups miss and fall through to Tier-1 |

### Classification Logic

```go
func ClassifyError(connectionError *LavaError, chainFamily ChainFamily, transport TransportType, errorCode int, errorMessage string) *LavaError {
    // Step 0: If caller already identified a connection-level error, use it directly
    if connectionError != nil {
        return connectionError
    }

    // Step 1: Check chain-specific mappings (Tier 2)
    // These override generic mappings where retryability differs
    if chainMappings, ok := ChainErrorMappings[chainFamily]; ok {
        for _, mapping := range chainMappings {
            if mapping.Matcher.Matches(errorCode, errorMessage) {
                return mapping.LavaError
            }
        }
    }

    // Step 2: Fall back to generic semantic mappings (Tier 1)
    // Only check matchers applicable to this transport type
    // (e.g., EVM/JSON-RPC chains skip gRPC matchers, Cosmos/gRPC chains skip HTTP matchers)
    for _, mapping := range GenericErrorMappings[transport] {
        if mapping.Matcher.Matches(errorCode, errorMessage) {
            return mapping.LavaError
        }
    }

    // Step 3: Unknown error
    return UNKNOWN_ERROR
}

// ClassifyMessage is a convenience wrapper for when the transport is genuinely unknown.
// It tries all transports in order (JsonRPC → REST → gRPC) and returns the first
// non-unknown classification. Prefer ClassifyError with an explicit transport when
// the transport is known, to avoid false matches across transport boundaries.
func ClassifyMessage(code int, message string) *LavaError {
    for _, transport := range []TransportType{TransportJsonRPC, TransportREST, TransportGRPC} {
        if c := ClassifyError(nil, -1, transport, code, message); c != LavaErrorUnknown {
            return c
        }
    }
    return LavaErrorUnknown
}
```

### Transport Types

```go
type TransportType int

const (
    TransportJsonRPC TransportType = iota // EVM, Solana, Bitcoin, etc. (includes WebSocket subscriptions)
    TransportREST                          // Aptos, Stellar, some Cosmos endpoints
    TransportGRPC                          // Cosmos SDK chains
)
```

Generic matchers are partitioned by transport so that `ClassifyError` only evaluates relevant matchers for the given chain's protocol.

### Matcher Types

```go
type ErrorMatcher interface {
    Matches(errorCode int, errorMessage string) bool
}

// Concrete matchers:
// CodeEquals(-32009)              — exact error code match
// MessageContains("nonce too low") — substring match in error message
// MessageRegex(`missing.*storage`) — regex match
// HTTPStatusEquals(429)           — HTTP status code match
// GRPCCodeEquals(codes.Unavailable) — gRPC status code match
```

### Connection Error Pre-Detection

Before calling `ClassifyError`, callers must call `DetectConnectionError(err)` on the raw Go error and pass the result as `connectionError`. This handles errors that can't be detected via code/message matching alone (e.g. `context.Canceled`, `context.DeadlineExceeded`, `net.Error` timeouts, `ECONNREFUSED`). It also falls back to string matching for errors wrapped without `%w` where `errors.Is` can't traverse the chain.

```go
func DetectConnectionError(err error) *LavaError {
    // errors.Is checks (properly wrapped errors)
    // net.Error timeout check
    // String fallback for non-%w wrapped errors ("context deadline exceeded", "context canceled")
    // ECONNREFUSED via net.OpError
}
```

`ClassifyError` Step 0 returns `connectionError` immediately if non-nil — so a detected connection error **always** takes precedence and can never produce `UNKNOWN_ERROR`.

### Matching Rules

1. **`GenericErrorMappings` is evaluated in declaration order — first match wins.** Matchers MUST be ordered most-specific first. A broader matcher placed before a narrower one will shadow it silently.
2. **Safety nets (required in Phase 1):**
   - **Shadow detection test:** A unit test that iterates every pair of matchers and fails if a broader matcher appears before a narrower one it would shadow.
   - **Real-world fixture tests:** A `testdata/` directory containing actual error responses captured from each node client (Geth, Erigon, Nethermind, Solana validator, Bitcoin Core, etc.). Table-driven tests run every fixture through `ClassifyError` and assert the expected Lava error code. When a new client variant surfaces in production as `UNKNOWN_ERROR`, its message is added to the fixtures.

### Mapping Registration

All mappings live in the central registry file:

```go
// Tier 2: Chain-specific (checked first, overrides generic)
var ChainErrorMappings = map[ChainFamily][]ChainErrorMapping{
    Solana: {
        {CodeEquals(-32009), CHAIN_SOLANA_MISSING_LONG_TERM},   // non-retryable
        {CodeEquals(-32007), CHAIN_SOLANA_LEDGER_JUMP},         // retryable
        {CodeEquals(-32002), NODE_SOLANA_UNHEALTHY},             // retryable
    },
    Bitcoin: {
        {CodeEquals(-13),    NODE_BITCOIN_WARMUP},              // retryable
        {CodeEquals(-25),    NODE_BITCOIN_INITIAL_DOWNLOAD},    // retryable
        {CodeEquals(-10),    CHAIN_BITCOIN_VERIFY_ERROR},       // non-retryable
        {CodeEquals(-11),    CHAIN_BITCOIN_VERIFY_REJECTED},    // non-retryable
        {CodeEquals(-100),   CHAIN_BITCOIN_INSUFFICIENT_FUNDS}, // non-retryable
    },
    Starknet: {
        {CodeEquals(28),     CHAIN_STARKNET_CLASS_NOT_FOUND},
        {CodeEquals(51),     CHAIN_STARKNET_CLASS_ALREADY_DECLARED},
        {CodeEquals(56),     CHAIN_STARKNET_COMPILATION_FAILED},
        {CodeEquals(40),     CHAIN_STARKNET_CONTRACT_ERROR},
    },
    // ... other chains
}

// Tier 1: Generic semantic (fallback), partitioned by transport type
var GenericErrorMappings = map[TransportType][]GenericMapping{
    TransportJsonRPC: {
        {MessageContains("nonce too low"),          CHAIN_NONCE_TOO_LOW},
        {MessageContains("insufficient funds"),     CHAIN_INSUFFICIENT_FUNDS},
        {MessageContains("execution reverted"),     CHAIN_EXECUTION_REVERTED},
        {MessageContains("already known"),          CHAIN_TX_ALREADY_KNOWN},
        {MessageContains("missing trie node"),      CHAIN_STATE_PRUNED},
        {MessageContains("historical state"),       CHAIN_STATE_PRUNED},
        {MessageContains("block not found"),        CHAIN_BLOCK_NOT_FOUND},
        {MessageRegex(`(?i)block #?\w+ not found`), CHAIN_BLOCK_NOT_FOUND},
        {CodeEquals(-32601),                        NODE_METHOD_NOT_FOUND},
        {CodeEquals(-32602),                        USER_INVALID_PARAMS},
        {CodeEquals(-32700),                        USER_PARSE_ERROR},
        {CodeEquals(-32000),                        NODE_SERVER_ERROR},   // generic fallback — specific messages matched above
        // ...
    },
    TransportREST: {
        {HTTPStatusEquals(429),                     NODE_RATE_LIMITED},
        {HTTPStatusEquals(503),                     NODE_SERVICE_UNAVAILABLE},
        {HTTPStatusEquals(404),                     NODE_ENDPOINT_NOT_FOUND},
        {HTTPStatusEquals(405),                     NODE_METHOD_NOT_ALLOWED},
        // ...
    },
    TransportGRPC: {
        {GRPCCodeEquals(codes.Unavailable),         NODE_SERVICE_UNAVAILABLE},
        {GRPCCodeEquals(codes.Unimplemented),       NODE_UNIMPLEMENTED},
        // HTTP status message matchers also appended at init() — HTTP status strings
        // can appear in gRPC error messages when the underlying transport is HTTP
        // (e.g. provider relay errors arriving via ClassifyLegacyError)
        // ...
    },
}
```

### Adding a New Chain
1. **If it uses standard protocols (EVM, JSON-RPC):** Assign a `ChainFamily` → generic mappings handle it automatically. Zero code changes.
2. **If it has unique errors with different retryability:** Add entries to `ChainErrorMappings` and define new `LavaError` constants in the registry.

---

## 4. Design: Central Error Registry

Single file: `protocol/common/error_registry.go`

```go
// ErrorCategory — top-level grouping: internal (Lava-introduced) vs external (pass-through)
type ErrorCategory int

const (
    Internal ErrorCategory = iota // Errors introduced by Lava — user would never see these without Lava
    External                      // Pass-through errors — user would get the same error talking to the node directly
)

// ErrorSubCategory — finer classification within each category.
// Subcategories carry behavioral implications the consumer hot path branches on
// (retries, CU charging, caching, endpoint health scoring).
type ErrorSubCategory int

const (
    SubCategoryNone              ErrorSubCategory = iota
    SubCategoryUnsupportedMethod                         // zero retries, zero CU, cached response, no provider scoring
    SubCategoryUserError                                 // invalid client input: zero retries, zero CU, no provider scoring (not cached)
    SubCategoryRateLimit                                 // endpoint is healthy but busy; apply backoff, do not mark unhealthy
)

func (sc ErrorSubCategory) IsUnsupportedMethod() bool { return sc == SubCategoryUnsupportedMethod }
func (sc ErrorSubCategory) IsUserError() bool         { return sc == SubCategoryUserError }
func (sc ErrorSubCategory) IsRateLimit() bool         { return sc == SubCategoryRateLimit }

// IsNonRetryableUserFacing combines the subcategories whose contract is
// "don't retry, don't charge CU". Consumer/retry sites call this instead of
// checking individual subcategories so adding a new non-retryable-user-facing
// subcategory only needs to update this one predicate.
func (sc ErrorSubCategory) IsNonRetryableUserFacing() bool {
    return sc.IsUnsupportedMethod() || sc.IsUserError()
}

// LavaError is the central error definition
type LavaError struct {
    Code        uint32
    Name        string
    Category    ErrorCategory
    SubCategory ErrorSubCategory
    Description string
    Retryable   bool
}

// Registry: all errors defined in one place (unexported — access via lookup
// helpers). Populated at package-init time and never mutated at runtime, so
// readers access it lock-free on the hot classification path.
var errorRegistry = map[uint32]*LavaError{...}

// Internal lookup helpers (unexported — callers should use ClassifyError or
// the subcategory predicates rather than raw registry lookups):
func getLavaError(code uint32) *LavaError
func getLavaErrorByName(name string) *LavaError

// Public chain-family helpers
func GetChainFamily(chainID string) (ChainFamily, bool) // ok=false when unknown
func GetChainFamilyOrDefault(chainID string) ChainFamily // returns ChainFamilyUnknown sentinel when unknown

// Public classification entry points
func ClassifyError(connErr *LavaError, family ChainFamily, transport TransportType, code int, msg string) *LavaError
func ClassifyMessage(code int, msg string) *LavaError // transport + chain unknown
func IsUnsupportedMethodError(chainID string, statusCode int, message string) bool
func IsUserInputError(chainID string, statusCode int, message string) bool

// Metrics callback registration (single-writer, atomic-pointer reads on hot path)
func SetErrorMetricsCallback(cb ErrorMetricsCallback)
func EmitErrorMetric(lavaError *LavaError, chainID string) // metric only, no log

// Structured logging entry points (fire metric + emit log)
func LogCodedError(description string, err error, lavaError *LavaError, chainID string, chainErrorCode int, chainErrorMessage string, attrs ...utils.Attribute) error
func LogCodedWarning(description string, err error, lavaError *LavaError, chainID string, chainErrorCode int, chainErrorMessage string, attrs ...utils.Attribute) error
```

---

## 5. Design: Standardized Logging Integration

Extend existing `LavaFormatError` with a coded error helper:

```go
// Dedicated coded error helper — auto-populates structured fields
utils.LavaFormatCodedError(PROTOCOL_CONNECTION_TIMEOUT, err,
    utils.LogAttr("provider", providerAddr),
)
```

Log output automatically includes:
- `error_code`: numeric code (1001)
- `error_name`: string name (PROTOCOL_CONNECTION_TIMEOUT)
- `error_category`: layer (protocol/node/blockchain/user)
- `retryable`: bool
- `chain_error_code`: original chain error code (e.g., -32009) — for Tier 1 generic codes
- `chain_error_message`: original chain error message — for debugging
- Standard structured attributes (provider, chainId, method, etc.)

---

## 6. Implementation Checklist

### Phase 1: Foundation
- [x] Create `protocol/common/error_registry.go` with `LavaError` struct, `ErrorCategory` enum, and all error code constants
- [x] Define all error codes from the taxonomy (Layers A-D) in the registry
- [x] Implement `ChainFamily` enum and chain ID → family mapping
- [x] Implement `TransportType` enum and chain ID → transport mapping
- [x] Implement `ErrorMatcher` interface with concrete matchers (`CodeEquals`, `MessageContains`, `MessageRegex`, `HTTPStatusEquals`, `GRPCCodeEquals`)
- [x] Implement `ClassifyError` function with two-tier lookup (chain-specific first, generic fallback)
- [x] Define all `ChainErrorMappings` (Tier 2) and `GenericErrorMappings` (Tier 1)
- [x] Add lookup helpers (`GetError`, `GetErrorByName`, `IsRetryable`, `GetCategory`)
- [x] Write unit tests for the registry and classification logic
- [x] Write shadow detection test to verify no broader matcher shadows a narrower one in `GenericErrorMappings`
- [x] Create `testdata/` directory with real error response fixtures from each node client (Geth, Erigon, Nethermind, Solana validator, Bitcoin Core, Starknet, etc.)
- [x] Write table-driven fixture tests that run every fixture through `ClassifyError` and assert expected Lava error code

### Phase 2: Logging Integration
- [x] Add `LavaFormatCodedError` helper to `utils/lavalog.go` that takes a `LavaError` code
- [x] Ensure coded errors emit `error_code`, `error_name`, `error_category`, `retryable`, `chain_error_code`, `chain_error_message` fields in structured logs
- [x] Add Prometheus counter that auto-increments per error code (`lava_errors_total{code, name, category, retryable, chain_id}`)
- [x] Write unit tests for coded error logging

### Phase 3: Migrate Existing Errors — Protocol Layer
- [x] Map existing `protocol/lavaprotocol/protocolerrors/errors.go` codes to new registry
- [x] Map existing `protocol/lavasession/errors.go` (consumer + provider) to new registry
- [x] Map existing `protocol/chaintracker/errors.go` to new registry
- [x] Map existing `protocol/common/errors.go` to new registry
- [x] Map existing `protocol/chainlib/common.go` errors to new registry
- [x] Map existing `protocol/performance/errors.go` to new registry
- [ ] Map existing `ecosystem/cache/handlers.go` errors to new registry _(intentionally deferred — cache layer has no production call site for ClassifyError; revisit if cache errors need structured metrics)_
- [x] Update `protocol/chainlib/node_error_handler.go` to use `ClassifyError` and registry codes
- [x] Replace `IsUnsupportedMethodError()` pattern matching with `LavaError.SubCategory.IsUnsupportedMethod()` check
- [x] Replace `IsUnsupportedMethodMessage()` in `protocol/common/errors.go` with registry-based classification
- [x] Update `protocol/rpcsmartrouter/error_mapper.go` to use `ClassifyError` and registry codes
- [x] Migrate `relayInnerDirect()` in `protocol/rpcsmartrouter/rpcsmartrouter_server.go` to use `LavaError` classification for endpoint health decisions (replace ad-hoc 5xx/429/timeout checks with `LavaError.Category` and `LavaError.Retryable`)

### Phase 4: Migrate Existing Errors — API Interface Layer
- [x] Update `protocol/common/return_errors.go` to use registry for JSON-RPC/REST error responses
- [x] Update JSON-RPC error handler to classify and log with codes
- [x] Update REST error handler to classify and log with codes
- [x] Update gRPC error handler to classify and log with codes
- [x] Update TendermintRPC error handler to classify and log with codes

### Phase 5: Migrate Existing Errors — Relay Path
- [x] Add `LavaError *LavaError` field to `RelayError` struct in `protocol/relaycore/relay_errors.go`
- [x] Call `ClassifyError` when creating `RelayError` in `results_manager.go` (`setErrorResponse` and `setValidResponse`) — decentralized path
- [x] Populate `RelayError.LavaError` in the smart-router path (`direct_rpc_relay.go` → pass classification from `ClassifyDirectRPCError` into the relay response flow)
- [x] Update `GetBestErrorMessageForUser` to prefer external errors (`CHAIN_*`, `NODE_*`) over internal (`PROTOCOL_*`) when selecting the best error for the user
- [x] Update `protocol/relaycore/relay_processor.go` to propagate codes
- [x] Update consumer server (`rpcconsumer/rpcconsumer_server.go`) to log with codes
- [x] Update provider server (`rpcprovider/rpcprovider_server.go`) to log with codes

### Phase 6: Metrics & Observability
- [x] Verify Prometheus counter `lava_errors_total{code, name, category, retryable}` works end-to-end
- [x] Update `protocol/metrics/consumer_metrics_manager.go` to use error codes (lava_errors_total auto-fires via LogCodedError — existing incident metrics kept for backwards compat)
- [x] Update `protocol/metrics/rpcconsumer_logs.go` to use error codes (same — LogCodedError handles it)
- [x] Verify error codes appear in existing dashboards/alerts (lava_errors_total emits all labels needed for dashboards)

### Phase 7: Refactor — Replace Legacy Errors with LavaError
- [x] Remove `UnsupportedMethodError` / `SolanaNonRetryableError` custom types, replace with `LavaWrappedError` + `LavaError.SubCategory` / `LavaError.Retryable`
- [x] Update `ShouldRetryError()` in `node_error_handler.go` to use registry's `Retryable` field
- [x] Make `LavaError` implement `error` interface with `Error()`, `Is()`, `ABCICode()` for drop-in replacement
- [x] Add `LavaWrappedError` + `NewLavaError()` for wrapping errors with classification that supports `errors.Is`

### Phase 8: Protocol Upgrade — Full sdkerrors Removal (future PR)
_Blocked on protocol upgrade: sdkerrors carry ABCI codes used in the gRPC wire format between consumer and provider. Changing them requires coordinated upgrade across all network participants._
- [ ] Replace `sdkerrors.Register` error variables with `LavaError`-based equivalents
- [ ] Update all `errors.Is(err, SomeOldError)` callsites to use `LavaError`-based checks
- [ ] Delete old error packages / re-exports after all consumers are migrated
- [ ] Verify no remaining imports of old error definitions

---

## 7. Decisions Made

1. **User Error boundary**: `USER_*` errors include cases detected both pre-forwarding by Lava AND returned by the node (e.g., `-32602 Invalid params`). Classification is by nature of error, not where it's caught.

2. **Error code visibility**: Lava error codes are **internal only** — visible in logs and metrics. They live within the Lava protocol layer (Smart Router: between router and endpoint; Decentralized: between consumer and provider). Users and nodes never see Lava codes. External responses use standard protocol codes (JSON-RPC, gRPC, HTTP).

3. **Chain-specific codes**: **Tiered approach (Option A hybrid)**. Distinct Lava codes exist for chain-specific errors where retryability differs from the generic pattern (Solana -32009 vs -32007, Bitcoin warmup, Starknet class errors, etc.). All other errors use generic semantic codes with chain detail in log attributes.

4. **x/ module errors**: Left as-is — governed by Cosmos SDK conventions, only relevant on-chain.

5. **`LavaError` is a classification struct that also implements `error`.** It implements `Error()`, `Is()`, and `ABCICode()`. It is metadata *about* an error — used for logging, metrics, and retry decisions — but it can also participate in `errors.Is` chains. To attach classification to a real error (so both the original message and the classification travel together), use `LavaWrappedError` via `NewLavaError(classified, originalErr.Error())`. Callers use `errors.As(err, &LavaWrappedError{})` to extract the `*LavaError` from a wrapped error. _(Updated in Phase 7 — original design had LavaError as pure metadata; reversed to enable retry/health decisions via errors.Is.)_

6. **Transport-scoped generic matching**: `ClassifyError` accepts a `TransportType` parameter. Generic (Tier 1) matchers are partitioned by transport (JSON-RPC, REST, gRPC) so that EVM/JSON-RPC chains never evaluate gRPC matchers and vice versa.

7. **Unsupported methods use `SubCategoryUnsupportedMethod`.** Codes 2001, 2008, 2009, and 2010 have `SubCategory: SubCategoryUnsupportedMethod`. This replaces the current pattern-matching approach (`IsUnsupportedMethodError`, `IsUnsupportedMethodMessage`) with a subcategory check via `LavaError.SubCategory.IsUnsupportedMethod()`. The special behavior (zero retries, zero CU, cached response, no provider scoring) is derived from the subcategory. _(Note: 2002 `NODE_METHOD_NOT_SUPPORTED` was removed from this list during Phase 7 review — it represents a method that exists but is disabled on this node, which is a retryable condition on a different provider. It has `Retryable: true` and no subcategory.)_

8. **Two-level error grouping: Category + SubCategory.** Category is `Internal` (errors Lava introduces — protocol layer) vs `External` (errors the user would get regardless of Lava — node, chain, user input). SubCategory provides finer classification within each category (e.g., UnsupportedMethod, Connection, Session, ChainExecution, ChainState, UserInput). SubCategories to be finalized before Phase 1 implementation.

9. **Transparent hop: original errors pass through unchanged.** The router/consumer is a transparent hop — the user always receives the original error from the node, unmodified. `LavaError` classification is metadata for internal use only (logging, metrics, endpoint health). Unknown/unmatched errors default to `CategoryExternal` because they are node pass-throughs. _(Clarification added in Phase 7: `handleAndClassify` wraps classified errors in `LavaWrappedError` on the **Go error return path** — this is internal plumbing for retry/health decisions and never reaches the user. The actual node response body travels separately and is always returned unmodified to the user. The "transparent hop" principle applies to the response body, not the internal Go error return.)_

## 8. Chains Analyzed

| Chain Family | Unique Error System? | Tier 2 Codes Needed? | Details |
|---|---|---|---|
| **EVM** (ETH, Arbitrum, Optimism, Base, Polygon, Avalanche, Blast, Sonic) | No — all use standard -32000 range | No (message-based differentiation only) | L2 sequencer errors differ by message, not code |
| **Polygon zkEVM** | Partial — `out of counters` is unique | Yes (1 code) | Prover constraint, non-retryable, no equivalent elsewhere |
| **Solana** | Yes — codes -32001 to -32011 | Yes (4 codes in Layer C, 1 in Layer B) | -32009 vs -32007 have different retryability |
| **Bitcoin/UTXO** | Yes — codes -1 to -111 | Yes (5 codes) | Warmup, initial download, verify errors, UTXO-specific |
| **Starknet** | Yes — 25+ codes (1-63) | Yes (4 codes) | Class system entirely unique, compilation errors |
| **NEAR** | Yes — hierarchical error types | Yes (2 codes in Layer C, 1 in Layer B) | Shard/chunk unavailability retryable on diff provider |
| **XRP/Ripple** | Yes — tec/tef/tel/tem/ter system | Yes (3 codes) | Most complex; prefix determines retryability |
| **TON** | Partial — TVM exit codes + custom HTTP | Yes (3 codes) | Lite-server timeout retryable, message expired retryable |
| **Aptos** | Partial — custom REST error format | No (generic codes sufficient) | Error format differs but semantics map to generic codes |
| **Stellar** | Yes — typed error URIs + result_codes | No (generic codes sufficient) | REST-based, HTTP codes sufficient for retry decisions |
| **Cosmos SDK** | Standard — tx_response.code | No (generic codes sufficient) | Standard patterns, well-covered by generic tier |
