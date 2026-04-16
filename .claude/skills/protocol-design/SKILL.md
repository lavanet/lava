---
name: protocol-design
description: "How the protocol-architect designs changes to the Lava protocol layer. Loaded when protocol-architect runs. Covers intent clarification, code exploration, API design, data-flow tracing, concurrency strategy, failure modes, compatibility, test strategy, and alternative consideration."
---

# Protocol Design Skill

Method used by protocol-architect. Role at `.claude/agents/protocol-architect.md`; this file is the _how_.

## Method Overview

1. Read `_workspace/00_input/requirements.md`.
2. Restate intent in your own words — put this at the top of §1 Problem Statement. If unclear, populate §10 Open Questions and mark status `draft`.
3. Load `.claude/skills/protocol-review/references/protocol-map.md` for topology.
4. **Explore before designing** — 30% of time: read the packages the change will touch. Cite files with line anchors as you read.
5. Design with the template in §Output of the role file. Budget ~1500 lines max; if exceeding, probably over-designed or scope too big.
6. Consider at least two alternatives (§9). If you can't think of two, you haven't thought enough.
7. Produce `_workspace/01_design.md`.

## Design Priorities (in order)

1. **Correctness over elegance** — the boring correct design beats the clever broken one.
2. **Extend existing seams** — if a similar feature exists, mimic its pattern. Consistency is a feature.
3. **Small first PR** — phased plans mergeable independently.
4. **Testable from the start** — design with tests in mind, not as an afterthought.
5. **Observability hooks** — log fields, metrics labels, traces — built in, not bolted on.

## Exploration Routine

For a typical protocol change, read:

1. The package most directly affected (e.g., `protocol/lavasession/`).
2. Its direct callers (grep imports).
3. The interface it implements or that wraps it.
4. The test file for the target — reveals what the current tests cover (and don't).

Cite each reading:

```
Current state (lavasession.Session) — `protocol/lavasession/session.go:48–92`:
- ProviderAddress, SessionID, CUSum fields
- Lock is a sync.Mutex guarding CUSum and metadata
- Called from rpcconsumer/consumer_session_manager.go:212 (Acquire)
```

## Smart-Router vs Consumer Design Decisions

For every design, answer:

- Does this change apply to rpcconsumer only, rpcsmartrouter only, or both?
- If both: does it behave identically? If not, table out the divergence in §5.
- What shared code (relaycore, lavaprotocol, chainlib, lavasession) needs to know?

If the decision is "rpcconsumer only" — be explicit. Drift between the two stacks is a recurring cost.

## Failure Mode Enumeration

In §4.5, list every way the feature can fail. Template:

```
| Failure | Observed at | Handled how | Telemetry |
|---------|-------------|-------------|-----------|
| upstream timeout | provider.handler | retry via relaycore, log warn with chainID | metric: relay_upstream_timeout_total |
| signature mismatch | rpcprovider.verify | reject + return signed error | metric: relay_invalid_signature_total |
| pairing stale | consumer pre-dispatch | refresh via statetracker, 1 retry | metric: pairing_refresh_total |
| ... | | | |
```

If the table is small (< 5 rows) for a non-trivial feature, you've missed cases. Rethink.

## Concurrency Strategy

In §4.4, specify:

- Goroutines spawned: owner, lifetime, context, stop channel.
- Channels: capacity (with rationale), direction, closer.
- Locks: what they protect, ordering if multiple.
- Cancellation: how context propagates; what state is cleaned on cancel.
- Race safety: which accesses are atomic / channel-gated / mutex-gated.

## Test Strategy (§7)

Per layer decision rule (push tests down):

| Layer       | Include when                                                |
| ----------- | ----------------------------------------------------------- |
| unit        | always — every new function exposed needs at least one test |
| integration | cross-package behavior, or real chainlib adapter            |
| e2e         | only when unit+integration genuinely can't cover            |

List specific test files to add/touch. Include bench if perf-sensitive.

## Alternative Consideration

§9 must include ≥2 alternatives. Common ones:

- Alt 1: "smaller scope — do X only" (phased)
- Alt 2: "different placement — put the logic in Y instead of Z"
- Alt 3: "different data shape — use this existing type instead of a new one"

For each: 1-line description, pros, cons, why rejected (or why deferred to a later phase).

## Deciding Design Status

- **`ready-for-implementation`** — all sections filled; no Open Questions that block; at least one alternative considered.
- **`draft`** — Open Questions remain, or reviewer hasn't completed a key exploration.
- **`superseded`** — only when an earlier design is replaced after major shift.

## Output

`_workspace/01_design.md` per the role file's template. Keep it legible; sections with nothing to say can be one line ("N/A — change doesn't affect wire format").

## Revision Discipline

On re-invocation:

- Add a `## Revision History` section at the top if not present.
- Log each revision with date + change summary.
- Don't delete old alternatives even if they were rejected — history is useful.
