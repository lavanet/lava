# RPC Endpoint Metrics (`lava_rpc_endpoint_*`)

Emitted by the **rpcsmartrouter** process, scoped to individual backing RPC endpoints rather than the router as a whole. Each endpoint represents one upstream RPC URL the SmartRouter can route to.

**Source:** `protocol/metrics/smartrouter_metrics_manager.go`

> For router-level aggregates, see [`smartrouter_metrics.md`](./smartrouter_metrics.md). These metrics let you drill down from a router-wide anomaly to the specific endpoint causing it.

---

## Key Concepts

**Endpoint** — a single backing RPC URL registered with the SmartRouter (e.g., `https://eth-mainnet.example.com`). Identified in metrics by the `endpoint_id` label.

**`endpoint_id` label** — a stable identifier for the endpoint within a chain/interface pair. Use this to isolate one upstream node's behavior.

**`function` label** — appears on relay-serving and latency metrics. It carries the name of the internal relay handler that processed the request (e.g., `SendRelay`, `SendParsedRelay`). This lets a single metric cover both the per-handler breakdown and, via `sum by (spec, apiInterface, endpoint_id)`, the per-endpoint aggregate — without needing duplicate metrics.

**`score_type` label** — appears on `lava_rpc_endpoint_selection_score` and `lava_rpc_optimizer_selection_score`. The SmartRouter scores each endpoint along five dimensions: `availability`, `latency`, `sync`, `stake`, and `composite`. `composite` is the combined score that drives endpoint selection. Values are normalized to the range [0, 1].

**Block fetch** — the SmartRouter periodically polls each backing endpoint to determine the latest block it has seen. "latest" fetches ask for the chain head; "block" fetches ask for a specific block number (used during consistency checks). Both have success and failure counters.

---

## Relay Serving

These metrics track requests dispatched to a specific backing endpoint. The `function` label identifies which internal handler processed the relay.

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_rpc_endpoint_total_relays_serviced` | Counter | `spec`, `apiInterface`, `endpoint_id`, `function` | Total relays successfully completed by this endpoint, broken down by handler function. Sum over `function` for the per-endpoint total. |
| `lava_rpc_endpoint_total_errored` | Counter | `spec`, `apiInterface`, `endpoint_id`, `function` | Total relays that returned an error from this endpoint, broken down by handler function. |
| `lava_rpc_endpoint_requests_in_flight` | Gauge | `spec`, `apiInterface`, `endpoint_id`, `function` | Number of relays currently executing against this endpoint (real-time gauge). A sustained high value indicates the endpoint is slow or saturated. Sum over `function` for the per-endpoint in-flight total. |

**Derived:** `total_errored / (total_relays_serviced + total_errored)` = per-endpoint error rate.

---

## Latency

Latency histogram bucket set: **1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000 ms**.

Use `histogram_quantile(0.99, rate(...[5m]))` in PromQL to compute percentiles.

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_rpc_endpoint_end_to_end_latency_milliseconds` | Histogram | `spec`, `apiInterface`, `endpoint_id`, `function` | End-to-end relay latency for this specific backing endpoint, in milliseconds, broken down by handler function. Aggregate over `function` to get the per-endpoint latency distribution. Compare across `endpoint_id` to identify slow nodes. |

---

## Block Tracking

The SmartRouter periodically polls each endpoint to track which block it has seen. These counters capture the health of that polling process.

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_rpc_endpoint_latest_block` | Gauge | `spec`, `apiInterface`, `endpoint_id` | Latest block number this endpoint reported during its most recent successful poll. Compare across endpoints to detect lag. |
| `lava_rpc_endpoint_fetch_latest_success` | Counter | `spec`, `apiInterface`, `endpoint_id` | Successful "get latest block" polls to this endpoint's upstream node. |
| `lava_rpc_endpoint_fetch_latest_fails` | Counter | `spec`, `apiInterface`, `endpoint_id` | Failed "get latest block" polls. A rising count indicates the endpoint's node is unreachable or returning errors during block tracking. |
| `lava_rpc_endpoint_fetch_block_success` | Counter | `spec`, `apiInterface`, `endpoint_id` | Successful fetches of a specific block number (used during consistency enforcement). |
| `lava_rpc_endpoint_fetch_block_fails` | Counter | `spec`, `apiInterface`, `endpoint_id` | Failed fetches of a specific block number. |

**Derived:** `fetch_latest_fails / (fetch_latest_success + fetch_latest_fails)` = latest-block poll failure rate.

---

## Health & Selection

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_rpc_endpoint_overall_health` | Gauge | `spec`, `apiInterface`, `endpoint_id` | Health of this specific endpoint: `1` = healthy, `0` = unhealthy. Updated by the SmartRouter's health-check loop. Use this to pinpoint which endpoint is down. |
| `lava_rpc_endpoint_overall_health_breakdown` | Gauge | `spec`, `apiInterface` | Aggregate health across **all** endpoints for this chain/interface: `1` = at least one endpoint is healthy, `0` = all endpoints are unhealthy. This is the endpoint-scoped analogue of `lava_rpcsmartrouter_overall_health_breakdown`. |
| `lava_rpc_endpoint_selection_score` | Gauge | `spec`, `apiInterface`, `endpoint_id`, `score_type` | Per-relay selection score for this endpoint, broken down by scoring dimension. Updated each time an endpoint is selected for a relay, so it carries the correct `apiInterface`. The `score_type` label takes one of five values: `availability`, `latency`, `sync`, `stake`, `composite`. All values are normalized to [0, 1]. `composite` is the combined score used to rank endpoints during selection. |
| `lava_rpc_optimizer_selection_score` | Gauge | `spec`, `endpoint_id`, `score_type` | Periodic optimizer selection score for this endpoint (chain-level, not scoped to `apiInterface`). Updated on a timer from the optimizer's internal state, so scores stay fresh even when no relays are flowing. Same `score_type` values as `endpoint_selection_score`. |

---

## Useful PromQL Examples

```promql
# Per-endpoint error rate
rate(lava_rpc_endpoint_total_errored[5m])
  / (rate(lava_rpc_endpoint_total_relays_serviced[5m]) + rate(lava_rpc_endpoint_total_errored[5m]))

# p99 latency per endpoint (all functions combined)
histogram_quantile(0.99,
  sum by (spec, apiInterface, endpoint_id, le) (
    rate(lava_rpc_endpoint_end_to_end_latency_milliseconds_bucket[5m])
  )
)

# Block lag per endpoint (relative to the best-known block across all endpoints)
max by (spec, apiInterface) (lava_rpc_endpoint_latest_block)
  - lava_rpc_endpoint_latest_block

# Endpoints currently unhealthy
lava_rpc_endpoint_overall_health == 0

# Latest-block poll failure rate per endpoint
rate(lava_rpc_endpoint_fetch_latest_fails[5m])
  / (rate(lava_rpc_endpoint_fetch_latest_success[5m]) + rate(lava_rpc_endpoint_fetch_latest_fails[5m]))

# Composite selection score per endpoint (higher = preferred)
lava_rpc_endpoint_selection_score{score_type="composite"}

# In-flight requests per endpoint (saturation check)
sum by (spec, apiInterface, endpoint_id) (lava_rpc_endpoint_requests_in_flight)
```
