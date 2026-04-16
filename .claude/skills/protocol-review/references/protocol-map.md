# Lava Protocol Layer — Package Map

Shared reference for all review and development agents. Consult before reviewing or designing anything.

## Scope Boundary

**In scope (off-chain):** everything under `protocol/`.
**Out of scope (on-chain):** `x/`, `app/`, `cmd/lavad/consensus`, proto definitions in `proto/` (except for where they cross into protocol/).

## Layer Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        CONSUMER SIDE                                    │
│  ┌────────────┐   ┌──────────────┐   ┌──────────────────────────────┐   │
│  │rpcconsumer │   │rpcsmartrouter│   │ protocol/common, parser,     │   │
│  │(decentral) │   │(centralized) │   │ metrics, monitoring          │   │
│  └─────┬──────┘   └──────┬───────┘   └──────────────────────────────┘   │
│        │                 │                                              │
│        └────────┬────────┘                                              │
│                 ▼                                                       │
│  ┌──────────────────┐   ┌──────────────────┐   ┌──────────────────┐    │
│  │  provideroptimizer│  │   lavasession    │  │  lavaprotocol    │    │
│  │  (QoS, WRS)       │  │   (sessions, CU) │  │  (retry/finaliz) │    │
│  └──────────────────┘   └──────────────────┘   └──────────────────┘    │
│                 ▼                                                       │
│  ┌──────────────────┐   ┌──────────────────┐                            │
│  │    relaycore     │  │   statetracker   │                            │
│  │  (relay state    │  │  (chain state    │                            │
│  │   machine)       │  │   updates)       │                            │
│  └──────────────────┘   └──────────────────┘                            │
│                 ▼                                                       │
│                 gRPC RelayRequest ──────────────┐                       │
└─────────────────────────────────────────────────┼───────────────────────┘
                                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        PROVIDER SIDE                                    │
│  ┌────────────────┐                                                     │
│  │  rpcprovider   │───►  rpcprovider/rewardserver                       │
│  │  (serve relays)│                                                     │
│  └───────┬────────┘                                                     │
│          ▼                                                              │
│  ┌────────────────┐   ┌─────────────────┐                               │
│  │    chainlib    │   │  chaintracker   │                               │
│  │ (multi-proto)  │   │  (head blocks)  │                               │
│  └───────┬────────┘   └─────────────────┘                               │
│          ▼                                                              │
│  upstream blockchain node (JSON-RPC / REST / gRPC / TendermintRPC / WS) │
└─────────────────────────────────────────────────────────────────────────┘
```

## Package-by-Package

### `protocol/rpcconsumer`
Decentralized RPC gateway. Reads pairings from Lava chain via statetracker, sends signed RelayRequests to providers selected by provideroptimizer, returns signed RelayReplies to end-user.
- Key concerns: request validation, session management, retry/failover, QoS reporting, finalization.
- Primary entry: HTTP/WS server that speaks chain-native APIs.

### `protocol/rpcsmartrouter`
Centralized RPC gateway. Static provider list (no on-chain pairing). Shares much code with rpcconsumer but **different trust model** — operator trusts their own provider list, less cryptographic machinery needed between consumer and provider.
- Key concerns: still needs robust provider selection, health checks, upstream response handling.
- **Divergence hotspots** from rpcconsumer: skip signing/verification where both endpoints are operator-controlled, different failover policy.

### `protocol/rpcprovider`
Serves relays to consumers. Receives signed RelayRequests, authenticates via on-chain pairing, forwards to configured upstream via chainlib, signs replies.
- `rewardserver/`: accumulates proofs of relay for reward claims.
- Key concerns: pairing-based authZ, session state, CU accounting, upstream health, response sanitization.

### `protocol/chainlib`
Multi-chain protocol adapters. Each supported chain spec (JSON-RPC, REST, gRPC, TendermintRPC, WebSocket) has a parser + proxy here.
- `chainproxy/`: wraps the upstream node connection.
- `grpcproxy/`: gRPC-specific reflection and dynamic codec.
- `extensionslib/`: pluggable response augmentation.
- Key concerns: correct parsing, zero-copy passthrough where possible, response validation, finalization checks.

### `protocol/lavasession`
Consumer-provider session management. Session IDs, CU counters, session reuse, session metadata, badge/project context.
- Key concerns: race-free CU counting, session reuse correctness, badge validation.

### `protocol/provideroptimizer`
Provider selection engine. Weighted Random Selection (WRS) based on QoS metrics (latency, availability, freshness). Per-epoch, per-chain.
- Key concerns: selection math, probe updates, stale metrics, hot-path speed (called every relay).

### `protocol/statetracker`
Polls the Lava chain for updates: pairings, epochs, specs, policies, plans. Pushes updates to consumers.
- `updaters/`: modular updaters per state kind.
- `hybridclient/`: balances multiple chain endpoints for resilience.
- Key concerns: update correctness under epoch transitions, retry/backoff, fallback when chain is unreachable.

### `protocol/chaintracker`
Polls upstream blockchain nodes for latest block / finalized block. Used by provider for finalization checks and by consumer for freshness QoS.
- Key concerns: poll frequency vs upstream load, stale cache, reorg handling.

### `protocol/relaycore`
Core relay state machine and processing. Shared between rpcconsumer and rpcsmartrouter for the per-relay lifecycle.
- Key concerns: legibility of the hot path, error propagation, extensibility.

### `protocol/lavaprotocol`
Protocol-level abstractions above relaycore: retry managers, finalization verification, request construction.

### `protocol/qos`
QoS metric computation and reporting.

### `protocol/metrics`
Prometheus metrics collection for consumer, provider, smart-router.

### `protocol/monitoring`
Health monitoring surfaces.

### `protocol/parser`
Parsing utilities shared across chain adapters.

### `protocol/common`
Shared helpers: types used across packages.

### `protocol/integration`
Integration tests exercising multiple protocol packages together.

### `protocol/performance`
Performance utilities (connection validators, etc). Not critical path.

### `protocol/loadtest`
Load testing tool. Not critical path.

### `protocol/upgrade`
Protocol version governance liaison.

### `protocol/internal/chainqueries`
Internal-only chain query helpers.

## Request Path — Hot Path Reference

A single relay traverses (consumer-side):

1. **HTTP/WS ingress** (`rpcconsumer` or `rpcsmartrouter`): parse end-user request, identify chain/spec/method.
2. **Session acquire** (`lavasession`): get or create a session for the consumer-provider pair.
3. **Provider select** (`provideroptimizer`): pick 1 or more providers via WRS against live QoS.
4. **Relay construct** (`lavaprotocol` + `relaycore`): build signed RelayRequest.
5. **Network** (gRPC): send to provider.
6. **Retry / failover** (`lavaprotocol`): if provider errors or times out, pick another.
7. **Response parse & validate** (`chainlib`): parse reply, verify signature, check finalization.
8. **QoS update** (`qos`, `provideroptimizer`): record latency/outcome.
9. **Respond to end-user**.

Provider-side (receiving side of step 5):
1. **gRPC ingress** (`rpcprovider`).
2. **AuthZ** via pairing + session (`rpcprovider`, `lavasession`).
3. **CU accounting** (`lavasession`).
4. **Upstream call** (`chainlib/chainproxy`).
5. **Response signing** (`rpcprovider`).
6. **Proof-of-relay recording** (`rpcprovider/rewardserver`).

Performance bottlenecks cluster at: session acquire, provider select, relay construct, response parse. Zero-copy is desirable in chainlib passthrough.

## Trust Boundaries

| Boundary | Trust direction | Primary protection |
|----------|----------------|---------------------|
| end-user → consumer/smart-router HTTP | untrusted → trusted | input validation, auth if exposed |
| consumer ↔ provider gRPC | mutually untrusted | both sides sign + verify |
| provider ↔ upstream node | operator-trusted | validation of response shape, size, timing |
| provider → reward server | cryptographic | signed proofs, replay protection via epoch/session |

## Recent Direction (Perf)

Reviewers should align with recent hot-path perf work, not against it:
- `5921aa237`: send response bodies as bytes via `fiber.Ctx.Send` (reduce copy)
- `3f1baeba0`: skip upstream gzip auto-decode on smart-router HTTP
- `e542b716d`: configurable response compression
- `bd584797a`: zero-copy passthrough in `checkUTXOResponseAndFixReply` for non-UTXO chains
- `21fbf7559`: removed `SubCategoryUserError` — normal CU charge for user-input errors

## Things That Are NOT in Scope for Any Reviewer/Developer of This Harness

- Cosmos SDK keepers (`x/`)
- Consensus rules (`app/app.go`, validator set, slashing, downtime)
- Governance / params module internals
- Cometbft internals
- CLI command plumbing in `cmd/` except where it directly wires the protocol binaries
