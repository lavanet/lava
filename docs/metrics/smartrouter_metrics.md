# SmartRouter Metrics (`lava_rpcsmartrouter_*`)

Emitted by the **rpcsmartrouter** process. Tracks routing activity at the router level — aggregated across all backing RPC endpoints it manages.

**Source:** `protocol/metrics/smartrouter_metrics_manager.go`

> The SmartRouter also emits **per-endpoint** metrics (`lava_rpc_endpoint_*`). Those are documented separately in [`rpcendpoint_metrics.md`](./rpcendpoint_metrics.md). The `lava_rpcsmartrouter_*` metrics documented here reflect the router as a whole, aggregated across all its endpoints.

---

## Key Concepts

**SmartRouter** — a process that sits in front of one or more raw RPC endpoints (e.g., your own Ethereum nodes) and provides the same relay interface as a Lava consumer. It multiplexes, retries, hedges, caches, and cross-validates across those endpoints, and it exposes Prometheus metrics at two scopes: router-level (`lava_rpcsmartrouter_*`) and endpoint-level (`lava_rpc_endpoint_*`).

**Endpoint** — a single backing RPC URL managed by the SmartRouter (e.g., `https://eth-mainnet.example.com`). Each endpoint has its own health, block tracking, and latency metrics. The SmartRouter selects among endpoints using a scoring algorithm analogous to the consumer optimizer.

**`provider_address` label in SmartRouter** — unlike the consumer (where this is a Lava provider address), in the SmartRouter this label holds the resolved name of the backing RPC endpoint. It maps the endpoint URL to a human-readable name using the `urlToProviderName` map.

**`function` label** — the `end_to_end_latency_milliseconds` histogram uses a `function` label (not `method`) to carry the name of the internal relay handler invoked. Summing over `function` gives the router-wide aggregate; slicing by `function` gives a per-handler breakdown.

**Hedging, Consistency, Cross-Validation, Node/Protocol errors** — identical semantics to the consumer. See the [Consumer Metrics doc](./consumer_metrics.md#key-concepts) for detailed definitions.

---

## Counting Invariants

```
requests_success_total + requests_failed_total  == requests_total
requests_batch_total   + requests_read_total
                       + requests_write_total   == requests_total
```

`requests_debug_trace_total` and `requests_archive_total` are sub-categories of read/write, not a separate partition — they do not sum to `requests_total`.

---

## WebSocket

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_rpcsmartrouter_ws_connections_active` | Gauge | `spec`, `apiInterface` | Number of currently active WebSocket connections from callers to the SmartRouter. Incremented on open, decremented on close. |
| `lava_rpcsmartrouter_ws_subscriptions_total` | Counter | `spec`, `apiInterface` | Total WebSocket subscription requests received by the SmartRouter, including those that subsequently fail. |
| `lava_rpcsmartrouter_ws_subscription_errors_total` | Counter | `spec`, `apiInterface` | WebSocket subscription requests that failed to establish a subscription with a backing endpoint. |

---

## Requests

Counters incremented once per completed request. The `provider_address` label identifies which backing endpoint served the request (by its resolved name).

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_rpcsmartrouter_requests_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Total requests processed by the SmartRouter. |
| `lava_rpcsmartrouter_requests_success_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Requests that returned a successful response to the caller. |
| `lava_rpcsmartrouter_requests_failed_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Requests that returned an error to the caller. |
| `lava_rpcsmartrouter_requests_read_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Read (stateless, `stateful=0`) requests. Mutually exclusive with `batch`. |
| `lava_rpcsmartrouter_requests_write_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Write (stateful, `stateful=1`) requests. Mutually exclusive with `batch`. |
| `lava_rpcsmartrouter_requests_debug_trace_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Requests using the debug/trace addon. Subset of read or write — does **not** partition `requests_total`. |
| `lava_rpcsmartrouter_requests_archive_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Requests targeting archive data. Subset of read or write — does **not** partition `requests_total`. |
| `lava_rpcsmartrouter_requests_batch_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Batch (multi-call) requests. Mutually exclusive with `read` and `write`. |

---

## Incidents

### Errors

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_rpcsmartrouter_node_errors_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Node-level errors returned by a backing endpoint (e.g., `execution reverted`, unknown method). Indicates a problem with the endpoint's node, not the transport. |
| `lava_rpcsmartrouter_protocol_errors_total` | Counter | `spec`, `apiInterface`, `provider_address`, `method` | Transport/session-layer failures (connection refused, timeout, GRPC error). The backing endpoint was not reachable. |

### Retries

When a relay to a backing endpoint fails, the SmartRouter may retry against the same or a different endpoint.

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_rpcsmartrouter_retries_total` | Counter | `spec`, `apiInterface`, `method` | Total retry attempts triggered (extra relay attempts beyond the first). |
| `lava_rpcsmartrouter_retries_success_total` | Counter | `spec`, `apiInterface`, `method` | Retry sequences that eventually returned a successful response. |
| `lava_rpcsmartrouter_retries_failed_total` | Counter | `spec`, `apiInterface`, `method` | Retry sequences that exhausted all attempts without success. |
| `lava_rpcsmartrouter_retry_attempts` | Histogram | `spec`, `apiInterface`, `method` | Distribution of how many endpoint attempts were made per retried request. Buckets: 1–10. |

### Consistency Enforcement

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_rpcsmartrouter_consistency_total` | Counter | `spec`, `apiInterface`, `method` | Requests that required seenBlock enforcement — the SmartRouter filtered backing endpoints to those that have seen at least the block it has observed. |
| `lava_rpcsmartrouter_consistency_success_total` | Counter | `spec`, `apiInterface`, `method` | Consistency-enforced requests that succeeded. |
| `lava_rpcsmartrouter_consistency_failed_total` | Counter | `spec`, `apiInterface`, `method` | Consistency-enforced requests that failed (no endpoint was up-to-date, or the request failed after selection). |

### Hedging

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_rpcsmartrouter_hedge_total` | Counter | `spec`, `apiInterface`, `method` | Requests that triggered at least one hedge relay — a speculative parallel relay fired when the first endpoint is slow. |
| `lava_rpcsmartrouter_hedge_success_total` | Counter | `spec`, `apiInterface`, `method` | Hedged requests where at least one of the parallel relays succeeded. |
| `lava_rpcsmartrouter_hedge_failed_total` | Counter | `spec`, `apiInterface`, `method` | Hedged requests where all parallel relays failed. |
| `lava_rpcsmartrouter_hedge_attempts` | Histogram | `spec`, `apiInterface`, `method` | Distribution of how many hedge relays were fired per hedged request. Buckets: 1–10. |

---

## Latency

All latency histograms share the same bucket set: **1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000 ms**.

Use `histogram_quantile(0.99, rate(...[5m]))` in PromQL to compute percentiles.

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_rpcsmartrouter_end_to_end_latency_milliseconds` | Histogram | `spec`, `apiInterface`, `function` | Full latency across all endpoints on the SmartRouter, from relay dispatch to response return, in milliseconds. The `function` label identifies the internal relay handler (e.g., `SendRelay`, `SendParsedRelay`). Sum over `function` for the router-wide aggregate. |

> For **per-endpoint** latency, see `lava_rpc_endpoint_end_to_end_latency_milliseconds` in [`rpcendpoint_metrics.md`](./rpcendpoint_metrics.md).

---

## Cache

The SmartRouter maintains a local response cache for read requests, checked before dispatching to any endpoint.

> **Note:** `lava_rpcsmartrouter_cache_latency_milliseconds` is only recorded on **cache hits**. Misses still increment `cache_failed_total` but do not add a latency observation.

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_rpcsmartrouter_cache_requests_total` | Counter | `spec`, `apiInterface`, `method` | Total cache lookups attempted. |
| `lava_rpcsmartrouter_cache_success_total` | Counter | `spec`, `apiInterface`, `method` | Cache hits — a cached response was found and returned, no endpoint was contacted. |
| `lava_rpcsmartrouter_cache_failed_total` | Counter | `spec`, `apiInterface`, `method` | Cache misses — no cached response; the request proceeds to endpoint selection. |
| `lava_rpcsmartrouter_cache_latency_milliseconds` | Histogram | `spec`, `apiInterface`, `method` | Time spent in the cache lookup on hit, in milliseconds. |

**Derived:** `cache_success_total / cache_requests_total` = cache hit rate.

---

## Cross-Validation

The SmartRouter can dispatch the same request to several backing endpoints simultaneously and compare responses. Endpoints whose response hash matches the majority are "agreeing"; those that differ are "disagreeing".

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_rpcsmartrouter_cross_validation_requests_total` | Counter | `spec`, `apiInterface`, `method` | Total cross-validation rounds initiated. |
| `lava_rpcsmartrouter_cross_validation_success_total` | Counter | `spec`, `apiInterface`, `method` | Rounds that reached consensus (majority agreement on response hash). |
| `lava_rpcsmartrouter_cross_validation_failed_total` | Counter | `spec`, `apiInterface`, `method` | Rounds that failed to reach consensus. |
| `lava_rpcsmartrouter_cross_validation_provider_agreements_total` | Counter | `spec`, `apiInterface`, `method`, `provider_address` | Times a specific backing endpoint's response matched the consensus hash. High values = reliable endpoint. |
| `lava_rpcsmartrouter_cross_validation_provider_disagreements_total` | Counter | `spec`, `apiInterface`, `method`, `provider_address` | Times a specific backing endpoint's response conflicted with the consensus hash. High values = stale or misbehaving endpoint. |

---

## Health & Chain State

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_rpcsmartrouter_overall_health` | Gauge | — | `1` if the SmartRouter considers itself healthy (at least one backing endpoint is up); `0` otherwise. Also exposed as HTTP: `GET /metrics/overall-health` returns `200 OK` or `503`. |
| `lava_rpcsmartrouter_overall_health_breakdown` | Gauge | `spec`, `apiInterface` | Per-chain/interface health: `1` = at least one backing endpoint is healthy for this chain+interface, `0` = none are. |
| `lava_rpcsmartrouter_latest_block` | Gauge | `spec`, `apiInterface` | Latest block number the SmartRouter has observed from any backing endpoint for this chain/interface. |
| `lava_rpcsmartrouter_protocol_version` | Gauge | `version` | Running `lavap` binary version encoded as `major×1,000,000 + minor×1,000 + patch`. The `version` label carries the human-readable string (e.g. `"6.1.0"`). |

---

## Useful PromQL Examples

```promql
# Error rate across the SmartRouter
rate(lava_rpcsmartrouter_requests_failed_total[5m])
  / rate(lava_rpcsmartrouter_requests_total[5m])

# p99 end-to-end latency (all functions combined)
histogram_quantile(0.99,
  sum by (spec, le) (rate(lava_rpcsmartrouter_end_to_end_latency_milliseconds_bucket[5m]))
)

# Cache hit rate
rate(lava_rpcsmartrouter_cache_success_total[5m])
  / rate(lava_rpcsmartrouter_cache_requests_total[5m])

# Retry rate (what fraction of requests needed retries)
rate(lava_rpcsmartrouter_retries_total[5m])
  / rate(lava_rpcsmartrouter_requests_total[5m])

# Cross-validation disagreement rate per endpoint (data quality)
rate(lava_rpcsmartrouter_cross_validation_provider_disagreements_total[5m])

# Hedge rate (what fraction of requests triggered a hedge)
rate(lava_rpcsmartrouter_hedge_total[5m])
  / rate(lava_rpcsmartrouter_requests_total[5m])

# Which endpoints are unhealthy
lava_rpc_endpoint_overall_health == 0
```
