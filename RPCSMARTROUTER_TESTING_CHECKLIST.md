# RPCSmartRouter Functionalities Testing Checklist
## Direct RPC (Centralized)

This document lists high-level test scenarios for `rpcsmartrouter`.
It is a centralized solution that relays directly to RPC endpoints, without providers or blockchain interactions.

---

## 1. Configuration & Endpoint Setup
- [ ] **Static endpoints only** - load `static-providers` and build direct connections
- [ ] **Protocol detection** - http/https/ws/wss/grpc based on URL or api-interface
- [ ] **Auth headers and auth path** - `auth-config` applied to requests
- [ ] **Per-endpoint timeout** - NodeUrl timeout overrides honored
- [ ] **IP forwarding** - `ip-forwarding` header set when enabled

---

## 2. API Interface Support

### 2.1 JSON-RPC
- [ ] **Basic relay** - JSON-RPC over HTTP/HTTPS (eth_blockNumber, eth_getBalance)
- [ ] **Batch requests** - multiple JSON-RPC calls in one request
- [ ] **Error handling** - correct JSON-RPC error codes and shapes
- [ ] **Block params** - latest/earliest/pending/specific block number

### 2.2 REST API
- [ ] **Basic relay** - GET/POST/PUT/DELETE to REST endpoints
- [ ] **Path and query params** - correct URL joining and encoding
- [ ] **Response formats** - JSON and non-JSON responses handled
- [ ] **Error responses** - HTTP status and body forwarded correctly

### 2.3 TendermintRPC
- [ ] **Basic relay** - Tendermint RPC method calls
- [ ] **Block height queries** - specific blocks by height
- [ ] **ABCI queries** - ABCI request handling

### 2.4 gRPC (Unary Only)
- [ ] **Unary calls** - request/response with dynamic descriptors
- [ ] **Metadata forwarding** - request and response metadata handling
- [ ] **Error codes** - gRPC status mapping
- [ ] **Descriptor source** - reflection + descriptor set fallback
- [ ] **Streaming not supported** - clear error on streaming methods

---

## 3. Direct Relay Behavior & Retry
- [ ] **Direct-only flow** - no provider-relay path reachable
- [ ] **Retry on 5xx/429/timeouts** - endpoint rotation with backoff
- [ ] **No retry on 4xx** - client errors are terminal
- [ ] **Timeout handling** - per-request timeout and overrides
- [ ] **Endpoint health** - mark unhealthy on node errors, reset on success

---

## 4. Cache & Consistency (If Enabled)
- [ ] **Cache bypass** - stateful and quorum requests skip cache
- [ ] **LATEST_BLOCK consistency** - seen-block based cache keys
- [ ] **Unsupported method caching** - cache unsupported method errors
- [ ] **No cache for node errors** - errors not stored as success
- [ ] **Force cache refresh** - header bypass works
- [ ] **Cache unavailability** - graceful degradation on cache failure

---

## 5. Quorum & Multi-Endpoint Behavior
- [ ] **Quorum selection** - multi-endpoint consensus when enabled
- [ ] **Best result selection** - prefer highest block / lowest latency
- [ ] **Partial success** - accept when at least one endpoint succeeds
- [ ] **Stateful fanout** - broadcast to all endpoints for stateful APIs

---

## 6. WebSocket Subscriptions
- [ ] **WS connection setup** - connect to ws/wss endpoints
- [ ] **Subscription create** - eth_subscribe/newHeads/logs
- [ ] **Message routing** - upstream events forwarded to clients
- [ ] **Cleanup** - disconnects remove upstream subscriptions
- [ ] **Reconnect/backoff** - retry with endpoint rotation

---

## 7. Response Metadata
- [ ] **Lava-Provider-Address** - endpoint identity (not provider address)
- [ ] **Lava-Retries** - retry count included
- [ ] **Provider-Latest-Block** - latest block header set
- [ ] **Lava-Guid** - request GUID propagated
- [ ] **lava-identified-node-error** - node error flag set correctly
- [ ] **lava-fast-tx-participants** - stateful fanout participants
- [ ] **lava-quorum-all-providers** - quorum participants list
- [ ] **Lava-Relay-Protocol** - transport protocol reported

---

## 8. Not Applicable in RPCSmartRouter
- [ ] **Provider pairing/epochs** - no pairing, no epoch updates
- [ ] **Relay signing/payments** - no signatures or payment flows
- [ ] **Conflict reporting** - no on-chain conflict detection/reporting
- [ ] **On-chain QoS** - no staking or reputation updates
