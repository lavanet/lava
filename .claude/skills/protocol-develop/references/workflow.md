# Development Workflow Reference

Extension for `protocol-develop` orchestrator. Covers workflow nuances that don't fit the lean SKILL.md.

## When to Skip the Design Phase

The architect phase adds real value but also takes time. Skip it only when:
- The change is truly localized (single function, single file) AND
- The user explicitly says they know what to do AND
- The change does not touch trust boundaries, signing, or cross-package contracts.

Example **do skip:** "rename the `latency` field to `latencyMillis` in the QoS struct and update callers."

Example **do not skip:** "add a per-provider circuit breaker."

When in doubt, run the architect briefly (produces a shorter design doc); the cost is low.

## When to Auto-Trigger Post-Implementation Review

Per the SKILL.md gate, trigger when:
- User asked for review explicitly
- Change touches `lavasession`, `rewardserver`, or any signing path
- Change touches >3 packages
- Change touches the hot path (`rpcconsumer`, `rpcprovider`, `rpcsmartrouter`, `chainlib`, `relaycore`)
- Change adds/modifies a concurrency primitive (goroutines, channels, mutexes, atomics)

Don't trigger for:
- Pure test additions
- Comment-only changes
- Config/default value changes with no behavior impact
- Renames covered by gofmt/gofumpt

## Handling Protocol Version / Wire Format Changes

If the architect discovers that a design requires a wire format change:
1. Architect marks Status = `draft` with Open Question "requires protocol version bump?".
2. Orchestrator surfaces to user — this is a governance question outside the harness's scope (it needs `x/protocol` interaction).
3. User decides: proceed with additive change (backward-compat), schedule a version bump, or drop the design.

## Rollout Order Convention

For changes that span consumer and provider:
- **Additive changes:** implement provider first (it must accept the new format); deploy providers first.
- **Breaking changes:** require a version bump and coordinated rollout.
- **Consumer-only changes:** no rollout concern on provider side.

The architect must decide this and put it in §6 Compatibility & Migration of the design doc.

## Handling `make proto-gen`

If the design requires proto changes:
- Architect proposes the proto diff in §4.2 API / Interface Changes.
- Implementer runs `make proto-gen` after editing `proto/lavanet/lava/*.proto` and commits generated files.
- Tester ensures tests for the new proto types pass.
- Warning: generated files are large and show up in diffs — don't reflow them unnecessarily.

## Handling Lint Failures

Implementer must:
1. Run `make lint` before declaring implementation complete.
2. Fix all lint issues that are on lines they touched.
3. If pre-existing unrelated lint issues appear (due to a file they modified), leave them alone but note it in `02_implementation.md` under Follow-up Items.
4. Never add `//nolint:...` except with a comment explaining why, and only if the rule genuinely doesn't apply.

## Handling Test Failures

If tests fail during Phase 4:
- **Failing test is new** (tester just added it): likely a real bug in Phase 3 implementation. Go back to implementer.
- **Failing test is existing** and unrelated to the change: flaky? Run `-race` and retry once. If still failing, the change may have broken something subtle — investigate.
- **Existing flaky test** masking new issues: document and proceed, but call out in final summary.

## Handling Discovered Bugs During Development

If the implementer or tester discovers a bug unrelated to the current task:
1. Do NOT fix it silently inside this PR.
2. Note it in `02_implementation.md` or `03_testing.md` under Follow-up Items.
3. The user can open a separate task for it via `protocol-develop` again.

## Re-invocation Patterns

| User says | Orchestrator does |
|-----------|------------------|
| "update the design to also handle X" | re-run architect with feedback, then implementer, then tester |
| "fix the SEC-003 finding" | write feedback, re-run implementer (skipping architect if design unchanged), then tester |
| "the tests don't cover the retry case" | re-run only tester with feedback; implementer unchanged |
| "start over, different approach" | archive `_workspace/`, Phase 1 initial run |
| "implement the deferred open question" | re-run architect with the previously-deferred question as primary, then pipeline |
