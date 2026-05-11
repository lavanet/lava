# Performance Review Checklist

Detailed rubric for performance-reviewer. "No" on a hot-path item is a candidate finding.

## Table of Contents

1. [Allocations](#1-allocations)
2. [Goroutine Lifecycle](#2-goroutine-lifecycle)
3. [Locking](#3-locking)
4. [Channels](#4-channels)
5. [I/O Patterns](#5-io-patterns)
6. [Serialization](#6-serialization)
7. [Caching & Dedup](#7-caching--dedup)
8. [Memory Footprint](#8-memory-footprint)
9. [GC Pressure & Escape](#9-gc-pressure--escape)
10. [Blocking Boundaries](#10-blocking-boundaries)

---

## 1. Allocations

Hot-path specific:

- [ ] No `[]byte(string)` or `string([]byte)` unless truly needed; prefer bytes throughout where the downstream accepts bytes.
- [ ] No `fmt.Sprintf` in a per-relay loop for error attribution — build with `strconv.AppendXxx` or similar.
- [ ] No `json.Marshal/Unmarshal` of a value that already has the right bytes.
- [ ] No `strings.Split` / `strings.Join` in hot loops — consider manual indexing if called at per-relay frequency.
- [ ] Slices pre-allocated with capacity where the size is known (`make([]T, 0, n)`).
- [ ] `sync.Pool` considered for hot buffer-like types (read: balance — Pool adds complexity, use only when profile shows benefit).
- [ ] No closure captures that force heap allocation in a per-relay goroutine spawn.

Recent wins to preserve (per `protocol-map.md`):
- zero-copy passthrough in chainlib — don't undo with re-encoding
- bytes via `fiber.Ctx.Send` — don't revert to string intermediates
- skip gzip auto-decode on smart-router — don't re-enable

## 2. Goroutine Lifecycle

- [ ] Every spawn has a bounded lifetime (ctx, stop channel, or synchronous join).
- [ ] No `go func() { for { ... } }()` without a `ctx.Done()` in the inner loop.
- [ ] Fan-out has fan-in — the spawner waits (`WaitGroup` or channel close) or tracks completion.
- [ ] No panic-in-goroutine that the main flow can't recover from.
- [ ] Per-relay goroutines are bounded (pool or per-request semaphore); not unlimited per request rate.
- [ ] Leaked goroutines don't accumulate under cancellation (verified via `leaktest` or `goleak` in tests where possible).

## 3. Locking

- [ ] Mutex vs RWMutex choice matches read/write ratio.
- [ ] Critical sections don't include I/O (network, disk).
- [ ] Defer-unlock pattern used consistently; no early returns leaving locks held.
- [ ] Shared counters on hot path prefer `atomic` over mutex.
- [ ] No lock ordering that could deadlock (A holds L1 and tries L2 while B holds L2 trying L1).
- [ ] Sharded locking considered for hot shared maps.

## 4. Channels

- [ ] Buffered channels have capacity justified by back-pressure math, not guessed.
- [ ] No send without timeout or `ctx.Done()` branch on potentially-blocked channel.
- [ ] `select { case <-ch: ... default: ... }` used only with a clear reason (flag busy-loop defaults).
- [ ] Receivers don't close channels; senders do (leak-safe).
- [ ] Single producer / multi-consumer and multi-producer / single-consumer patterns are clearly labeled.

## 5. I/O Patterns

- [ ] Outbound HTTP/gRPC uses a pooled client (not `http.Get`, not per-request dial).
- [ ] Keepalive configured on connections to upstream and to providers.
- [ ] DNS is cached or resolved via an async mechanism; not resolved on hot path.
- [ ] Response bodies are streamed where possible, not buffered in full.
- [ ] Multiple small writes to upstream coalesced (if the protocol allows).
- [ ] WebSocket writes batched where protocol-appropriate.

## 6. Serialization

- [ ] Proto marshal/unmarshal happens at most once per transition (no redundant re-encode).
- [ ] JSON-RPC passthrough uses bytes where possible.
- [ ] Compression decisions are configurable and their defaults align with recent direction.
- [ ] `proto.Marshal` results reused instead of re-marshalled for each downstream call (when safe).

## 7. Caching & Dedup

- [ ] Per-epoch or per-chain-ID work is cached where it changes rarely (pairings, specs).
- [ ] Cache invalidation on epoch boundary is explicit.
- [ ] Cache stampede mitigated (singleflight or probabilistic refresh).
- [ ] TTLs sized for change rate — too short causes churn, too long causes staleness.

## 8. Memory Footprint

- [ ] Long-lived buffers are explicit (not accidentally accumulating via slice appends).
- [ ] Maps keyed by unbounded input (session IDs, consumer addrs) have size bounds or eviction.
- [ ] Per-connection slices (subscription lists, metadata) don't grow without bound.

## 9. GC Pressure & Escape

- [ ] Hot types don't escape unnecessarily (`go build -gcflags='-m'` for suspected cases).
- [ ] `interface{}` / `any` boxing minimized on hot path.
- [ ] Short-lived objects favored over pooled when allocation is cheap; pooled for medium-size buffers.

## 10. Blocking Boundaries

- [ ] Mutex + network call combinations are flagged (network under lock = tail latency spike).
- [ ] Disk I/O (config read, log write) not on the relay request path.
- [ ] `time.Sleep` never used to "wait" for a condition — use channel or callback.
- [ ] Rate limiters non-blocking or with timeout (no unbounded wait).

---

## Finding Format Example

```markdown
### PERF-007 — Per-relay allocation in chainlib JSON parse
- Severity: high
- Category: allocation
- Location: `protocol/chainlib/chainproxy/rpcInterfaceMessages/jsonRPCMessage.go:180–195`
- Description: `parseJSONRPC` re-marshals the request to compute a digest. The same bytes were already unmarshaled. On a 10k rps gateway this is ~10k mallocs/s of 1–10 KB, directly taxing GC.
- Evidence: <code excerpt>
- Estimated impact: ~5% of consumer CPU at 10k rps based on `pprof heap` shape (needs measurement via `go test -bench=ParseRelay -benchmem`).
- Recommendation: Compute digest over original bytes; plumb bytes through and skip remarshal. Alternative: use `json.RawMessage` in the struct.
- Verification: `go test -bench=ParseRelay -benchmem ./protocol/chainlib/chainproxy/rpcInterfaceMessages` — expect allocs/op drop from N to 0.
- Confidence: medium — hot claim needs bench to quantify exact CPU %.
```
