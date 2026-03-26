# Consumer Metrics (`lava_consumer_*`)

Emitted by the **rpcconsumer** process. Every metric tracks activity from the consumer's perspective — what it requests, which providers serve it, and how they perform.

**Source:** `protocol/metrics/consumer_metrics_manager.go`

---

## Key Concepts

**Relay** — a single RPC request dispatched to one provider. One user-facing request may produce multiple relays (retries, hedges, cross-validation).

**Hedging** — when a relay has not returned within a deadline, a second relay is fired to another provider speculatively. The first response wins, the other is discarded. Hedging improves p99 latency at the cost of extra load.

**Consistency (seenBlock enforcement)** — the consumer tracks the highest block it has seen per chain. When a method requires up-to-date state, the consumer only selects providers that have caught up to that block. Requests that go through this filter are counted as "consistency" requests.

**Cross-validation** — for sensitive requests, the consumer sends the same query to multiple providers and compares responses. Consensus is reached when a majority return the same answer. Providers that disagree are flagged. This catches stale or incorrect provider responses.

**Node error** — the provider responded, but the underlying blockchain node returned an error (e.g., `execution reverted`, method not found). This is a provider-side issue, not a transport failure.

**Protocol error** — the transport or session layer failed (connection refused, timeout, GRPC error, session mismatch). The provider was not successfully reached.

**CU (Compute Unit)** — the cost weight of a request as defined by the chain spec. Used for rate-limiting and billing.

**Virtual epoch** — an emergency-mode epoch counter used when the standard epoch cannot be determined (e.g., Lava chain is unreachable). The consumer increments it on a timer to keep session rotation going.

---

## Counting Invariants

These relationships hold by design and are useful for building dashboards and alerts:

```
requests_success_total + requests_failed_total  == requests_total   (every request succeeds or fails)
requests_batch_total   + requests_read_total
                       + requests_write_total   == requests_total   (batch is mutually exclusive with read/write)
```

`requests_debug_trace_total` and `requests_archive_total` are **sub-categories** of read/write, not a separate partition — their sum is not expected to equal `requests_total`.

When a request is served from the local cache, `provider_address` is set to the literal string **`"Cached"`** instead of a provider address.

---

## WebSocket

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_consumer_total_websocket_connections_active` | Gauge | `spec`, `apiInterface` | Number of currently open WebSocket connections from users to this consumer. Incremented on open, decremented on close. |
| `lava_consumer_total_ws_subscription_requests` | Counter | `spec`, `apiInterface` | Total WebSocket subscription requests received, including those that subsequently fail. |
| `lava_consumer_total_failed_ws_subscription_requests` | Counter | `spec`, `apiInterface` | WebSocket subscription requests that failed to establish a subscription with the provider. |

---

## Requests

Counters incremented once per completed user-facing request. The `provider_address` label is `"Cached"` when the response was served from the local response cache without hitting any provider.

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_consumer_requests_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Total requests processed. Every request increments exactly one bin here. |
| `lava_consumer_requests_success_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Requests that returned a successful response to the caller. |
| `lava_consumer_requests_failed_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Requests that returned an error to the caller. |
| `lava_consumer_requests_read_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Read (stateless, `stateful=0`) requests. Mutually exclusive with `batch`. |
| `lava_consumer_requests_write_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Write (stateful, `stateful=1`) requests. Mutually exclusive with `batch`. |
| `lava_consumer_requests_debug_trace_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Requests using the debug/trace addon. Subset of read or write — does **not** partition `requests_total`. |
| `lava_consumer_requests_archive_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Requests targeting archive data. Subset of read or write — does **not** partition `requests_total`. |
| `lava_consumer_requests_batch_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Batch (multi-call) requests. Mutually exclusive with `read` and `write`. |

---

## Incidents

Incident metrics track abnormal events during relay processing — failures, retries, hedges, and consistency enforcement. A single request can appear in multiple incident counters (e.g., it could be retried AND hedged).

### Errors

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_consumer_node_errors_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Node-level errors received from a provider (e.g., `execution reverted`, unknown method). Indicates a problem with the provider's node, not the transport. |
| `lava_consumer_protocol_errors_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Protocol/transport-layer failures (connection refused, timeout, GRPC error, session mismatch). The provider was not reachable or the session was invalid. |

### Retries

When a relay fails, the consumer may retry it against the same or a different provider. Each extra attempt is one retry.

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_consumer_retries_total` | Counter | `spec`, `apiInterface`, `method` | Total retry attempts triggered (extra relay attempts beyond the first). |
| `lava_consumer_retries_success_total` | Counter | `spec`, `apiInterface`, `method` | Retry sequences that eventually returned a successful response. |
| `lava_consumer_retries_failed_total` | Counter | `spec`, `apiInterface`, `method` | Retry sequences that exhausted all attempts without success. |
| `lava_consumer_retry_attempts` | Histogram | `spec`, `apiInterface`, `method` | Distribution of how many provider attempts were made per retried request. Buckets: 1–10. Useful for understanding retry depth (e.g., most retries resolve in 2 attempts vs. needing 5+). |

### Consistency Enforcement

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_consumer_consistency_total` | Counter | `spec`, `apiInterface`, `method` | Requests that required seenBlock enforcement — i.e., the consumer filtered providers to those that have seen at least the block the consumer has seen. High values indicate providers are lagging behind. |
| `lava_consumer_consistency_success_total` | Counter | `spec`, `apiInterface`, `method` | Consistency-enforced requests that succeeded. |
| `lava_consumer_consistency_failed_total` | Counter | `spec`, `apiInterface`, `method` | Consistency-enforced requests that failed (no provider was up-to-date enough, or the attempt failed). |

### Hedging

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_consumer_hedge_total` | Counter | `spec`, `apiInterface`, `method` | Requests that triggered at least one hedge relay (a speculative parallel relay fired when the first is slow). |
| `lava_consumer_hedge_success_total` | Counter | `spec`, `apiInterface`, `method` | Hedged requests where at least one of the parallel relays succeeded. |
| `lava_consumer_hedge_failed_total` | Counter | `spec`, `apiInterface`, `method` | Hedged requests where all parallel relays failed. |
| `lava_consumer_hedge_attempts` | Histogram | `spec`, `apiInterface`, `method` | Distribution of how many hedge relays were fired per hedged request. Buckets: 1–10. A value of 1 means one hedge (two relays total); 2 means two hedges (three relays total). |

---

## Latency

All latency histograms share the same bucket set: **1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000 ms**.

Use `histogram_quantile(0.99, rate(...[5m]))` in PromQL to compute percentiles.

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_consumer_end_to_end_latency_milliseconds` | Histogram | `spec`, `apiInterface`, `method` | Full request latency as seen by the consumer — from when the relay starts to when the result is returned to the caller. Includes cache miss handling, provider selection, network RTT, and response deserialization. |
| `lava_consumer_provider_latency_milliseconds` | Histogram | `spec`, `apiInterface`, `provider_address`, `method` | Network round-trip to a specific provider only — from relay dispatch to response receipt. Subtract this from `end_to_end_latency` to estimate consumer-side overhead. |

---

## Cache

The consumer maintains a local response cache for read requests. Requests are looked up in the cache before a provider is selected.

> **Note:** `lava_consumer_cache_latency_milliseconds` is only recorded on **cache hits**. Misses still increment `cache_failed_total` but do not add a latency observation.

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_consumer_cache_requests_total` | Counter | `spec`, `apiInterface`, `method` | Total cache lookups attempted. |
| `lava_consumer_cache_success_total` | Counter | `spec`, `apiInterface`, `method` | Cache hits — a cached response was found and returned. These requests also appear in `requests_total` with `provider_address="Cached"`. |
| `lava_consumer_cache_failed_total` | Counter | `spec`, `apiInterface`, `method` | Cache misses — no cached response; the request proceeds to provider selection. |
| `lava_consumer_cache_latency_milliseconds` | Histogram | `spec`, `apiInterface`, `method` | Time spent in the cache lookup on hit, in milliseconds. Buckets: same shared set as latency metrics. |

**Derived:** `cache_success_total / cache_requests_total` = cache hit rate.

---

## Cross-Validation

For sensitive queries the consumer dispatches the same request to several providers simultaneously and compares responses. Providers whose response hash matches the majority are "agreeing"; those that differ are "disagreeing". This is separate from retries — all providers are contacted in parallel.

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_consumer_cross_validation_requests_total` | Counter | `spec`, `apiInterface`, `method` | Total cross-validation rounds initiated. |
| `lava_consumer_cross_validation_success_total` | Counter | `spec`, `apiInterface`, `method` | Rounds that reached consensus (majority agreement on response hash). |
| `lava_consumer_cross_validation_failed_total` | Counter | `spec`, `apiInterface`, `method` | Rounds that failed to reach consensus (providers split or insufficient responses). |
| `lava_consumer_cross_validation_provider_agreements_total` | Counter | `spec`, `apiInterface`, `method`, `provider_address` | Times a specific provider's response matched the consensus hash. High values = reliable provider. |
| `lava_consumer_cross_validation_provider_disagreements_total` | Counter | `spec`, `apiInterface`, `method`, `provider_address` | Times a specific provider's response conflicted with the consensus hash. High values = unreliable / stale provider. |

---

## Health & Chain State

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_consumer_overall_health` | Gauge | — | `1` if at least one endpoint across all chains is healthy; `0` if all endpoints are down. Also exposed as HTTP: `GET /metrics/overall-health` returns `200 OK` or `503`. |
| `lava_consumer_overall_health_breakdown` | Gauge | `spec`, `apiInterface` | Per-chain/interface health: `1` = at least one provider is healthy for this chain+interface, `0` = none are. |
| `lava_consumer_latest_block` | Gauge | `spec` | Latest block number seen by the consumer for this chain (always labeled `spec=lava` for the Lava chain itself). |
| `lava_consumer_virtual_epoch` | Gauge | `spec` | Current virtual epoch counter for this chain (always labeled `spec=lava`). Increments on a timer when the Lava chain is unreachable, to keep session rotation alive. Normally stays at 0. |
| `lava_consumer_protocol_version` | Gauge | `version` | Running `lavap` binary version encoded as `major×1,000,000 + minor×1,000 + patch`. The `version` label carries the human-readable string (e.g. `"6.1.0"`). |

---

## Provider State

Metrics that describe the current state and behavior of each connected provider.

> **Optional label `provider_endpoint`:** `lava_consumer_provider_blocked`, `lava_consumer_selection_stats`, and `lava_consumer_latest_provider_block` include a `provider_endpoint` label **only** when the process is started with `--show-provider-endpoint-in-metrics`. By default this label is absent to reduce cardinality.

### Liveness & Blocking

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_consumer_provider_liveness` | Gauge | `spec`, `provider_address`, `provider_endpoint` | Result of the periodic liveness probe sent to each provider endpoint. `1` = probe succeeded, `0` = probe failed. Useful for distinguishing "provider is down" from "provider is slow". |
| `lava_consumer_provider_blocked` | Gauge | `spec`, `apiInterface`, `provider_address` | `1` if the provider is currently blocked by the consumer's optimizer (e.g., too many failures), `0` otherwise. Reset at each session boundary. |

### Selection & Scores

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_consumer_provider_selections` | Counter | `spec`, `provider_address` | Number of times this provider was selected as the target for a relay (before the request is attempted). Unequal distribution across providers indicates optimizer bias toward better providers. |
| `lava_consumer_selection_stats` | Gauge | `spec`, `apiInterface`, `provider_address`, `selection_metric` | Normalized selection score (0–1) used by the provider selection optimizer, broken down by dimension. The `selection_metric` label takes one of five values: `availability`, `latency`, `sync`, `stake`, `composite`. `composite` is the combined score that drives selection. Updated periodically via a background goroutine. Reset at each session boundary. |

### Block Tracking

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_consumer_latest_provider_block` | Gauge | `spec`, `provider_address`, `apiInterface` | Latest block number reported by this provider in its relay response. Used to assess how far behind a provider is relative to `lava_consumer_latest_block`. |
| `lava_consumer_latest_provider_relay_time` | Gauge | `spec`, `provider_address`, `apiInterface` | Unix timestamp (seconds) of the last relay sent to this provider. Useful for detecting providers that have not been contacted recently. |

### Traffic & Cost

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_consumer_total_cu_requested` | Counter | `spec`, `apiInterface` | Cumulative Compute Units consumed across all requests for this chain/interface. CU weight is defined per-method in the chain spec. |

---

## Debug / Internal

> These metrics are registered and available but are primarily intended for internal debugging. They are not recommended for dashboards or alerts.

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_consumer_selection_rng_value` | Gauge | `spec` | Last random float used in the provider selection weighted-random draw. Useful for debugging determinism issues in provider selection. Changes every relay — do not alert on this. |

---

## Useful PromQL Examples

```promql
# Error rate per chain
rate(lava_consumer_requests_failed_total[5m]) / rate(lava_consumer_requests_total[5m])

# p99 end-to-end latency per chain
histogram_quantile(0.99,
  sum by (spec, le) (rate(lava_consumer_end_to_end_latency_milliseconds_bucket[5m]))
)

# Cache hit rate per method
rate(lava_consumer_cache_success_total[5m]) / rate(lava_consumer_cache_requests_total[5m])

# Provider disagreement rate (potential data quality issue)
rate(lava_consumer_cross_validation_provider_disagreements_total[5m])

# Providers currently blocked
lava_consumer_provider_blocked == 1

# Retry depth: fraction of retried requests needing more than 3 attempts
1 - histogram_quantile(0.5,
  rate(lava_consumer_retry_attempts_bucket[5m])
)

# Block lag per provider (relative to consumer's known head)
lava_consumer_latest_block - on(spec) group_right lava_consumer_latest_provider_block
```
