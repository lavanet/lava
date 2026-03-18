# Integrating failsafe-go into Lava: Architecture Analysis

## Executive Summary

**Overall difficulty: HIGH.** The lava repo already has a mature, battle-tested resilience stack deeply woven into its session management, relay state machines, and provider lifecycle. failsafe-go would provide cleaner abstractions and composability, but the current patterns carry significant domain-specific logic (blockchain reporting, epoch management, provider scoring, archive upgrades, cross-validation consensus) that failsafe-go cannot express natively. The integration is feasible but would be a **multi-sprint refactoring effort** with substantial regression risk.

**Recommendation**: Incremental, bottom-up adoption — start with the simplest, most isolated patterns (Timeout, WebSocket backoff) and work upward. Do **not** attempt a big-bang replacement.

---

## Pattern-by-Pattern Breakdown

### 1. Retry Logic → `retrypolicy.RetryPolicy`

**Current**: `ConsumerRelayStateMachine` / `SmartRouterRelayStateMachine`, `RelayRetriesManager`, `RelayProcessor.shouldRetryRelay()`

**Difficulty: HIGH**

| Aspect | Detail |
|--------|--------|
| **What it replaces** | The ticker-based retry loops in both state machines, the `RelayRetryLimit` constant, `RelayRetryBackoffDuration`, and the retry decision logic in `RelayProcessor.shouldRetryRelay()` |
| **What it does NOT replace** | `RelayRetriesManager` (hash-based 6h deduplication cache), provider selection/rotation via `UsedProviders`, archive extension upgrade on retry #1, epoch mismatch special handling (always retry), Solana/unsupported-method abort conditions |
| **How** | `retrypolicy.NewBuilder[*RelayResult]().WithMaxRetries(2).WithDelay(2*time.Millisecond).AbortIf(isNonRetryableError).HandleIf(isRetryableError).OnRetry(updateProviderSelection).Build()` |
| **Why it's hard** | The retry loop is not a simple "call → fail → retry same call." Each retry iteration involves: (a) selecting a *different* provider via `UsedProviders`, (b) potentially upgrading to archive extension on retry #1, (c) checking the hash dedup cache, (d) respecting different rules per selection mode (Stateless=retry, Stateful/CrossValidation=no retry). These are **side-effectful state transitions** between retries, not just backoff-and-replay. failsafe-go's `OnRetry` hook can handle some of this, but the provider-rotation-as-retry-mechanism is a fundamental architectural mismatch. |

### 2. Circuit Breaker → `circuitbreaker.CircuitBreaker`

**Current**: Session blocking (`BlockListed`), provider blocking (`_blockProvider`), `consecutivePairingErrors` (SmartRouter), `MaximumNumberOfFailuresAllowedPerConsumerSession`

**Difficulty: HIGH**

| Aspect | Detail |
|--------|--------|
| **What it replaces** | The consecutive error tracking and blocking decision matrix in `OnSessionFailure`, the `consecutivePairingErrors` circuit breaker in SmartRouter, session `BlockListed` flag |
| **What it does NOT replace** | Blockchain provider reporting (side effect of blocking), epoch-aware provider carry-over, `retrySecondChanceAfter` (3 min grace period), per-endpoint blocklisting |
| **How** | One `CircuitBreaker` per provider: `circuitbreaker.NewBuilder[*RelayResult]().WithFailureThreshold(MaxFailures).WithDelay(3*time.Minute).OnOpen(reportToBlockchain).Build()` |
| **Why it's hard** | The current circuit breaking is **multi-level** (session → endpoint → provider → epoch) and tightly coupled to `ConsumerSessionManager`'s internal maps. A failsafe-go circuit breaker is a standalone object per "resource" — you'd need to manage a map of `CircuitBreaker` instances keyed by provider address, and synchronize their state with the existing session lifecycle. The "second chance" mechanism maps conceptually to half-open state, but the recovery probe logic (checking if provider is back) is custom. The provider blocking also triggers blockchain reports, which is a side effect that goes beyond what a circuit breaker normally does. |

### 3. Timeout → `timeout.Timeout`

**Current**: `common/timeout.go` (`GetTimePerCu`, `LocalNodeTimePerCu`), `context.WithTimeout` calls, three-level timeout hierarchy

**Difficulty: LOW-MEDIUM**

| Aspect | Detail |
|--------|--------|
| **What it replaces** | The `context.WithTimeout(processingCtx, processingTimeout)` calls in both servers, the per-relay timeout logic in state machines |
| **What it does NOT replace** | The adaptive timeout *calculation* (`GetTimePerCu` based on CU, hanging API flag, stateful flag), subscription first-reply timeout (10s) |
| **How** | `timeout.New[*RelayResult](calculatedTimeout)` composed inside a retry policy: `failsafe.With(retryPolicy, timeoutPolicy).Get(sendRelay)` — this gives per-attempt timeouts with retry on timeout |
| **Why it's approachable** | The timeout calculation is cleanly separated from its enforcement. You keep `GetTimePerCu()` to compute the duration, then wrap it in a failsafe-go `Timeout` instead of raw `context.WithTimeout`. The main benefit: composing timeout *inside* retry gives you per-attempt timeouts automatically, which the current code achieves manually. |

### 4. WebSocket Reconnection Backoff → `retrypolicy.RetryPolicy`

**Current**: `websocket_backoff.go` (SmartRouter), `ExponentialBackoff` struct

**Difficulty: LOW**

| Aspect | Detail |
|--------|--------|
| **What it replaces** | The entire `ExponentialBackoff` struct and `NextBackoff()`/`Reset()`/`Clone()` methods, reconnection retry loops in `upstream_ws_pool.go` |
| **How** | `retrypolicy.NewBuilder[*ws.Conn]().WithBackoff(100*time.Millisecond, 30*time.Second).WithJitterFactor(0.3).WithMaxRetries(10).Build()` — drop-in replacement |
| **Why it's easy** | This is the **cleanest integration point**. The WebSocket backoff is self-contained, well-isolated, and maps 1:1 to failsafe-go's retry with exponential backoff. The subscription restoration after reconnect can go in an `OnSuccess` listener. |

### 5. Provider Failover/Selection → No direct failsafe-go equivalent

**Current**: `ProviderOptimizer`, `ConsumerSessionManager.GetSessions()`, `UsedProviders`, QoS scoring

**Difficulty: N/A (not replaceable)**

| Aspect | Detail |
|--------|--------|
| **Why** | failsafe-go has no concept of "choose a different backend on each attempt." Provider selection is domain logic (QoS scoring, stake weighting, T-Digest percentile normalization, strategy-based selection). This must remain as-is. failsafe-go's retry can *trigger* re-selection, but the selection logic itself stays. |

### 6. Hedging/Parallel Sends → `hedgepolicy.HedgePolicy`

**Current**: Stateful mode (send to all), CrossValidation (send to N, check consensus)

**Difficulty: MEDIUM**

| Aspect | Detail |
|--------|--------|
| **What it replaces** | The parallel goroutine fan-out in Stateful mode that sends to all top providers and returns the first result |
| **What it does NOT replace** | CrossValidation mode — failsafe-go's hedge cancels on *first success*, but cross-validation needs *agreement threshold* (quorum). This is fundamentally different from hedging. |
| **How** | For Stateful: `hedgepolicy.NewBuilderWithDelay[*RelayResult](0).WithMaxHedges(numProviders-1).Build()` — immediate parallel execution, first success wins |
| **Why it's medium** | Stateful mode maps well to hedging. CrossValidation does not — it requires consensus logic that failsafe-go doesn't support. You'd keep CrossValidation as custom code. |

### 7. Rate Limiting → `ratelimiter.RateLimiter`

**Current**: `WebsocketConnectionLimiter`, per-client subscription limits, `ClientRateLimiter`

**Difficulty: LOW-MEDIUM**

| Aspect | Detail |
|--------|--------|
| **What it replaces** | The `WebSocketRateLimit` enforcement, subscription-per-minute limits |
| **What it does NOT replace** | Per-IP connection tracking (stateful, needs IP-keyed map), ban duration logic |
| **How** | `ratelimiter.NewSmooth[any](maxRequests, time.Second)` per client or per IP |
| **Why it's medium** | The current rate limiting is per-IP with banning, which requires maintaining state keyed by IP. You'd need a map of `RateLimiter` instances per IP, plus custom ban logic on top. |

### 8. Bulkhead/Concurrency → `bulkhead.Bulkhead`

**Current**: `MaxCallsPerRelay = 50`, `max-concurrent-providers` flag

**Difficulty: LOW**

| Aspect | Detail |
|--------|--------|
| **What it replaces** | The implicit concurrency limits on parallel provider calls |
| **How** | `bulkhead.New[*RelayResult](maxConcurrentProviders)` |
| **Why it's easy** | Simple concurrency cap, maps directly. |

### 9. Fallback → `fallback.Fallback`

**Current**: Cache lookup before relay, backup provider tier, archive extension upgrade

**Difficulty: LOW-MEDIUM**

| Aspect | Detail |
|--------|--------|
| **What it replaces** | The cache-hit-returns-early pattern, backup provider fallback |
| **How** | `fallback.NewWithFunc(func(exec failsafe.Execution[*RelayResult]) (*RelayResult, error) { return cache.Lookup(key) })` |
| **Why** | Cache fallback is clean. Backup provider tier is more complex (requires session manager interaction). |

---

## Difficulty Summary

| Pattern | failsafe-go Policy | Difficulty | Risk |
|---------|-------------------|------------|------|
| WebSocket backoff | RetryPolicy | **LOW** | Low |
| Timeout enforcement | Timeout | **LOW-MEDIUM** | Low |
| Bulkhead | Bulkhead | **LOW** | Low |
| Cache fallback | Fallback | **LOW-MEDIUM** | Low |
| Rate limiting | RateLimiter | **LOW-MEDIUM** | Medium |
| Stateful hedging | HedgePolicy | **MEDIUM** | Medium |
| Relay retry logic | RetryPolicy | **HIGH** | High |
| Circuit breaker | CircuitBreaker | **HIGH** | High |
| Provider selection | N/A | **N/A** | — |
| CrossValidation | N/A | **N/A** | — |

---

## Recommended Adoption Strategy

### Phase 1 (Low risk, high value)

- Replace `ExponentialBackoff` in SmartRouter WebSocket with failsafe-go `RetryPolicy`
- Replace raw `context.WithTimeout` with failsafe-go `Timeout` (composability benefit)
- Add `Bulkhead` for concurrent provider calls

### Phase 2 (Medium risk)

- Introduce failsafe-go `Fallback` for cache-miss-to-relay pattern
- Replace Stateful mode parallel fan-out with `HedgePolicy`
- Add `RateLimiter` per-client (keep per-IP custom logic)

### Phase 3 (High risk, requires careful refactoring)

- Refactor relay state machines to use failsafe-go `RetryPolicy` with `OnRetry` hooks for provider rotation
- Introduce per-provider `CircuitBreaker` instances managed by `ConsumerSessionManager`
- Compose full policy stacks: `Fallback → Retry → CircuitBreaker → Timeout`

**Phase 3 is where the real complexity lives** — the relay state machines (`consumer_relay_state_machine.go` ~400 lines, `smartrouter_relay_state_machine.go` ~650 lines) are the heart of the resilience logic and carry significant domain-specific state transitions that will need careful decomposition.

---

## Key Risks

1. **Regression risk**: Both components are critical path for all RPC traffic. Any behavioral change in retry/timeout/failover semantics could cause outages.
2. **Semantic mismatch**: failsafe-go assumes "retry = call the same function again." Lava's retry means "call a *different provider* with *possibly different parameters*" (archive upgrade, extension changes). This impedance mismatch requires wrapping the provider selection inside the retried function.
3. **Testing gap**: The current resilience behavior is likely validated by integration tests and production experience, not unit tests of the resilience logic itself. Replacing it requires building a comprehensive test harness first.
4. **Two systems running in parallel**: During migration, you'll have some patterns using failsafe-go and others using the legacy approach, increasing cognitive load.

---

## Key Files Reference

### rpcconsumer

| File | Purpose |
|------|---------|
| `protocol/rpcconsumer/rpcconsumer_server.go` | HTTP/WebSocket handling, relay processing |
| `protocol/rpcconsumer/consumer_relay_state_machine.go` | Retry orchestration, selection mode logic |
| `protocol/lavasession/consumer_session_manager.go` | Provider pairing, session allocation, blocking |
| `protocol/lavasession/used_providers.go` | Provider rotation tracking |
| `protocol/provideroptimizer/provider_optimizer.go` | QoS scoring, weighted provider selection |
| `protocol/relaycore/relay_processor.go` | Response collection, consensus, error aggregation |
| `protocol/relaycore/relay_state.go` | Archive extension detection, retry hash caching |
| `protocol/common/timeout.go` | Adaptive timeout calculation |
| `protocol/lavaprotocol/relay_retries_manager.go` | Hash-based retry deduplication (6h TTL) |

### rpcsmartrouter

| File | Purpose |
|------|---------|
| `protocol/rpcsmartrouter/rpcsmartrouter_server.go` | Request handling, relay sending, health checks |
| `protocol/rpcsmartrouter/smartrouter_relay_state_machine.go` | Retry logic, circuit breaker, state transitions |
| `protocol/rpcsmartrouter/direct_ws_subscription_manager.go` | WebSocket subscriptions, rate limiting |
| `protocol/rpcsmartrouter/upstream_ws_pool.go` | Connection pooling, auto-scaling |
| `protocol/rpcsmartrouter/websocket_backoff.go` | Exponential backoff for WS reconnection |
| `protocol/rpcsmartrouter/error_mapper.go` | Error classification |
