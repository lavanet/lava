---
name: performance-review
description: "How the performance-reviewer audits the Lava protocol layer hot path. Loaded when performance-reviewer runs. Covers allocations, goroutine lifecycle, locking, channels, I/O patterns, serialization, caching, GC pressure. Focuses on per-relay latency and throughput under concurrent load."
---

# Performance Review Skill

Method used by performance-reviewer. Role at `.claude/agents/performance-reviewer.md`; this file holds the *how*.

## Method Overview

1. Read scope from `_workspace/00_input/scope.md`.
2. Load context:
   - `.claude/skills/protocol-review/references/protocol-map.md` § "Request Path — Hot Path Reference" and § "Recent Direction (Perf)".
   - `references/checklist.md` in this skill.
3. **Classify every file in scope as hot / warm / cold** — hot files get micro-scrutiny, warm get a scan, cold get only obvious-issue attention.
4. Two-pass review:
   - **Pass 1 — Walk the hot path** on consumer and provider sides. At each hop, identify per-relay work.
   - **Pass 2 — Cross-cutting** checks (goroutine inventory, allocation hotspots, locking patterns).
5. For each finding, estimate per-relay impact (even as "μs — needs measurement").
6. Emit `_workspace/01_performance_findings.md`.

## Hot / Warm / Cold Classification

| Hot | Called per-relay on the critical path | Examples |
|-----|--------------------------------------|----------|
| **Hot** | called for every relay | `relaycore` state machine, `chainlib` parse/proxy, `provideroptimizer.Choose`, `lavasession.Get`, `rpcprovider` handler, `rpcconsumer` handler |
| **Warm** | called per session or per epoch | `statetracker` updates, `chaintracker` polls, session setup, pairing refresh |
| **Cold** | startup or rare operations | CLI, config loading, daemon bootstrap, loadtest, monitoring exports |

Report severity should roughly follow: hot-path bugs start high; warm-path bugs start medium; cold-path bugs start low (unless they're structural).

## Measurement Language

- Prefer quantitative framing: "~X allocs per relay", "lock held for Y μs", "called O(providers × relays)".
- When you can't measure from reading, use "needs measurement" and suggest the measurement tool/benchmark.
- Don't make up numbers. A bad estimate is worse than no estimate.

## Common Suggestions to Know

- `go test -bench=. -benchmem ./protocol/<pkg>` — allocation and ns/op per op
- `pprof` CPU and heap profiles — for live load
- `go tool trace` — for contention and scheduling issues
- `go test -race` — for ordering bugs; not a perf tool but perf-adjacent

## Non-Findings

- Startup allocation that happens once.
- A `fmt.Sprintf` in an error path that runs rarely.
- A defer in a non-loop context (trivial cost).
- Fixing allocations in `loadtest/`, `monitoring/`, cold code.

## Respecting Recent Direction

Recent commits prioritize zero-copy and skipping unnecessary decompress. **Do not** propose "add a copy for safety" unless you can prove the zero-copy version is unsafe. Align findings with the existing direction; call out regressions from that direction as high.

## Output

Per role template. Each finding must include:
- **Estimated impact** (with units or "needs measurement")
- **Verification** (how to prove the fix helps)

## Detailed Checklist

See `references/checklist.md`.
