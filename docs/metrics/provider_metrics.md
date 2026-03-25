# Provider Metrics (`lava_provider_*`)

Emitted by the **rpcprovider** process. Tracks the provider's ability to serve relays, the health of the upstream node it proxies, and its on-chain standing (frozen/jailed status).

**Source:** `protocol/metrics/provider_metrics_manager.go`, `protocol/metrics/provider_metrics.go`

---

## Key Concepts

**Relay** — a single RPC request received from a consumer and forwarded to the upstream node. The provider receives relays, proxies them to its node, and returns the result.

**Function** — the internal handler that processed the relay (e.g., `SendRelay`, `SendParsedRelay`). Used as a label on relay-serving and latency metrics to break down activity by handler. Sum over `function` for chain-level aggregates.

**CU (Compute Unit)** — the cost weight of a request as defined by the chain spec. Providers are paid in CU at the end of an epoch. `total_cu_serviced` tracks how much work was done; `total_cu_paid` tracks how much was actually rewarded on-chain.

**Load rate** — `current_in_flight_relays / configured_rate_limit`. A value of `1.0` means the provider is at capacity. Configured via the provider's rate-limit flag.

**Block fetch** — the provider periodically polls its upstream node to track the latest block it has seen. These calls are distinct from user relays — they are internal health/state polls.

**Frozen** — an on-chain state where the provider has been suspended (e.g., for insufficient stake). Frozen providers cannot receive relays.

**Jailed** — an on-chain punishment state triggered by repeated QoS failures (bad latency, unavailability, or stale data). Jailed providers are temporarily barred from receiving relays.

**Virtual epoch** — an emergency-mode epoch counter used when the Lava chain is unreachable. The provider increments it on a timer to keep session rotation going. Normally stays at 0.

---

## Counting Invariant

> `lava_provider_total_relays_serviced` counts **all** relay attempts — including those that errored. Errored relays increment both `total_relays_serviced` and `total_errored`.

```
total_relays_serviced  =  successful relays  +  errored relays
total_errored          =  errored relays only

# Derived error rate:
rate(lava_provider_total_errored[5m]) / rate(lava_provider_total_relays_serviced[5m])
```

---

## Relay Serving

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_provider_total_relays_serviced` | Counter | `spec`, `apiInterface`, `function` | Total relay attempts handled by this provider, including errored ones (see counting invariant above). The `function` label identifies the internal handler. Sum over `function` for the chain-level total. |
| `lava_provider_total_errored` | Counter | `spec`, `apiInterface`, `function` | Relays that returned an error to the consumer. Always a subset of `total_relays_serviced`. |
| `lava_provider_requests_in_flight` | Gauge | `spec`, `apiInterface`, `function` | Number of relays currently being processed (real-time gauge). A sustained high value relative to `load_rate` indicates saturation. Incremented at relay start, decremented at relay end. |
| `lava_provider_total_cu_serviced` | Counter | `spec`, `apiInterface` | Cumulative Compute Units consumed across all served relays. Reflects workload volume — does not imply payment. |
| `lava_provider_total_cu_paid` | Counter | `spec` | Compute Units actually rewarded on-chain at epoch settlement. Has only `spec` label (no `apiInterface`) because on-chain payments are per-chain, not per-interface. Compare to `total_cu_serviced` to detect unpaid work. |
| `lava_provider_load_rate` | Gauge | `spec` | Current relay load as a fraction of the configured rate limit: `in_flight / rate_limit`. `1.0` = fully saturated, `>1.0` = overloaded. |

---

## Latency

All latency histograms share the same bucket set: **1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000 ms**.

There are three distinct latency measurements, each capturing a different scope:

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_provider_request_latency_milliseconds` | Histogram | `spec`, `apiInterface`, `function` | Time spent processing a relay inside a specific handler function, in milliseconds. Most granular — use this to identify which handler is slow. |
| `lava_provider_latency_milliseconds` | Histogram | `spec`, `apiInterface` | Round-trip latency to the upstream node (the time from forwarding the request to receiving the node's response), in milliseconds. Reflects node responsiveness. |
| `lava_provider_end_to_end_latency_milliseconds` | Histogram | `spec`, `apiInterface` | Full processing time from relay receipt to response sent back to the consumer, in milliseconds. Includes gRPC overhead, serialization, and node RTT. This is what the consumer experiences. |

**Relationship:** `end_to_end ≥ node_latency ≥ function_latency` — each scope adds more overhead.

---

## Node / Block Tracking

The provider periodically polls its upstream node to determine the latest block it has seen. These metrics track the health of that polling process.

> **Optional label `provider_endpoint`:** `lava_provider_latest_block` includes a `provider_endpoint` label **only** when the process is started with `--show-provider-endpoint-in-metrics`. By default this label is absent to reduce cardinality.

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_provider_latest_block` | Gauge | `spec`, `apiInterface` | Latest block number reported by the upstream node. Also set for the Lava chain itself with `spec=lava`, `apiInterface=lava`. |
| `lava_provider_last_serviced_block_update_time_seconds` | Gauge | `spec` | Unix timestamp (seconds) of the last successful block update received from the upstream node. A stale value indicates the node is not responding to block polls. Alert if `time() - lava_provider_last_serviced_block_update_time_seconds > threshold`. |
| `lava_provider_fetch_latest_success` | Counter | `spec`, `apiInterface` | Successful "get latest block" polls to the upstream node. |
| `lava_provider_fetch_latest_fails` | Counter | `spec`, `apiInterface` | Failed "get latest block" polls. A rising count means the node is unreachable or erroring on block queries. |
| `lava_provider_fetch_block_success` | Counter | `spec`, `apiInterface` | Successful fetches of a specific block number (used for consistency checks). |
| `lava_provider_fetch_block_fails` | Counter | `spec`, `apiInterface` | Failed fetches of a specific block number. |
| `lava_provider_virtual_epoch` | Gauge | `spec` | Current virtual epoch counter (always `spec=lava`). Increments on a timer when the Lava chain is unreachable. Normally stays at 0; a rising value indicates the provider has lost contact with the Lava chain. |

---

## Health & Availability

The provider exposes both Prometheus metrics and HTTP health endpoints:
- `GET /metrics/overall-health` → `200 OK` or `503 Service Unavailable` (also accessible at `/metrics/health-overall` for backward compatibility)
- `GET /metrics/{spec}/{apiInterface}/health` → per-chain health, driven by the RelaysMonitor for that chain

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_provider_overall_health` | Gauge | — | `1` if at least one chain endpoint is healthy; `0` if all are down. Mirrors the HTTP `/metrics/overall-health` response. |
| `lava_provider_overall_health_breakdown` | Gauge | `spec`, `apiInterface` | Per-chain/interface health: `1` = healthy, `0` = unhealthy. Use this to identify which specific chain is having issues without polling the HTTP endpoint. |
| `lava_provider_disabled_chains` | Gauge | `chainID`, `apiInterface` | `1` if this chain/interface is currently disabled on the provider, `0` if enabled. A chain can be disabled if its node is unreachable or misconfigured. |

---

## On-Chain Status

These metrics reflect the provider's standing on the Lava blockchain. They use a `chainID` label (not `spec`) because they are sourced from on-chain pairing data.

| Metric Name | Type | Labels | Description |
|---|---|---|---|
| `lava_provider_frozen_status` | Gauge | `chainID` | `1` if the provider is frozen on-chain for this chain (suspended, cannot receive relays); `0` if active. Freeze is typically caused by insufficient stake. |
| `lava_provider_jail_status` | Gauge | `chainID` | `1` if the provider is currently jailed for this chain; `0` if not. Jailing is triggered by repeated QoS failures (bad latency, unavailability, or stale data). |
| `lava_provider_jailed_count` | Gauge | `chainID` | Number of jail events in the last 24 hours. A rising count indicates persistent QoS problems. |
| `lava_provider_protocol_version` | Gauge | `version` | Running `lavap` binary version encoded as `major×1,000,000 + minor×1,000 + patch`. The `version` label carries the human-readable string (e.g. `"6.1.0"`). |

---

## Useful PromQL Examples

```promql
# Error rate per chain
rate(lava_provider_total_errored[5m])
  / rate(lava_provider_total_relays_serviced[5m])

# p99 end-to-end latency per chain (what consumers experience)
histogram_quantile(0.99,
  sum by (spec, apiInterface, le) (
    rate(lava_provider_end_to_end_latency_milliseconds_bucket[5m])
  )
)

# Node latency vs end-to-end latency (provider overhead)
histogram_quantile(0.99,
  sum by (spec, apiInterface, le) (rate(lava_provider_end_to_end_latency_milliseconds_bucket[5m]))
)
-
histogram_quantile(0.99,
  sum by (spec, apiInterface, le) (rate(lava_provider_latency_milliseconds_bucket[5m]))
)

# CU payment ratio (what fraction of served CU was actually paid)
rate(lava_provider_total_cu_paid[1h])
  / sum by (spec) (rate(lava_provider_total_cu_serviced[1h]))

# Load rate approaching saturation (alert if > 0.8)
lava_provider_load_rate > 0.8

# Block fetch failure rate (node health)
rate(lava_provider_fetch_latest_fails[5m])
  / (rate(lava_provider_fetch_latest_success[5m]) + rate(lava_provider_fetch_latest_fails[5m]))

# Time since last block update (detect stale node)
time() - lava_provider_last_serviced_block_update_time_seconds

# Providers currently frozen or jailed
lava_provider_frozen_status == 1
lava_provider_jail_status == 1

# Jail frequency (chains jailed more than once in 24h)
lava_provider_jailed_count > 1

# In-flight relay count per handler (saturation by function)
sum by (spec, apiInterface, function) (lava_provider_requests_in_flight)
```
