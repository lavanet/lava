---
name: protocol-implementation
description: "How the protocol-implementer writes Go code for the Lava protocol layer, following a design doc from protocol-architect. Loaded when protocol-implementer runs. Covers matching codebase voice, error wrapping with utils.LavaFormatError, concurrency discipline, lint/build/race cleanliness, and revision handling."
---

# Protocol Implementation Skill

Method used by protocol-implementer. Role at `.claude/agents/protocol-implementer.md`; this file is the *how*.

## Method Overview

1. Read `_workspace/01_design.md` — the source of truth.
2. Read adjacent code in the target packages to internalize the codebase voice.
3. Implement the design, section by section.
4. Run `make lint` and `go build ./protocol/...` (or scoped equivalent) between meaningful edits, not only at the end.
5. Run `go test -race ./protocol/<changed-dirs>/...` before declaring done.
6. Produce `_workspace/02_implementation.md`.

## Codebase Voice Calibration

Before writing new code in a package, read:

1. The nearest existing file to where your change lands.
2. The package's top-level `doc.go` or package comment (if any).
3. One or two test files for style reference.
4. Usages of project helpers (`utils.LavaFormatError`, etc.) in that package.

Match:
- Error wrapping style (project helper vs stdlib).
- Logging style (structured attributes).
- Naming conventions (receiver names, fn name patterns).
- Receiver types (value vs pointer) — match the struct's existing methods.

## Error Handling Pattern

Default to:
```go
return utils.LavaFormatError("descriptive message", err,
    utils.Attribute{Key: "chainID", Value: chainID},
    utils.Attribute{Key: "provider", Value: providerAddr},
)
```

Deviations:
- `utils.LavaFormatWarning` for recoverable issues worth logging.
- `utils.LavaFormatInfo` / `LavaFormatDebug` for non-error telemetry.
- Typed errors (sentinels or custom types) when callers need to branch via `errors.Is/As`.

Never `_ = someCall()` silently — if you truly want to drop the error, name the fn as `mustXxx` or comment the reason.

## Concurrency Implementation

- **Every** goroutine spawn gets a `context.Context` input or a tied stop channel.
- **Every** `for { select { } }` has a `ctx.Done()` case.
- **Every** channel close has a clear owner (sender-side).
- **Every** lock acquire has a `defer unlock` unless there's a deliberate branching reason.
- Prefer `atomic.Int64` / `atomic.Value` for single counters/flags over mutex.

When in doubt, run `go test -race` with tests that exercise the new path.

## Adherence to Design

The design is the contract. Deviations are allowed but must be documented:

- **Minor deviation** (impl detail that doesn't change API or behavior) → note in `02_implementation.md` under Deviations.
- **API deviation** (change to signatures, types, flow) → stop, surface to orchestrator, update design, then resume.
- **Scope creep** (adding something not in design) → avoid. If truly needed, surface to orchestrator.

## Lint Discipline

Run `make lint` incrementally. Fix:
- All hits on lines you edited.
- Hits on lines adjacent that are cheap to fix while you're there (but don't silently refactor unrelated files).

`//nolint` is a last resort. If you add one:
- Target the specific rule: `//nolint:staticcheck // SA1019 — legacy proto API, migration tracked in TICKET-XXX`.
- Reason required.

## Test Discipline

You're not the tester, but don't leave untestable code:
- Add tests for the happy path of every new function you write.
- Add tests for error paths if the error is a typed sentinel or affects callers.
- If a test file doesn't exist yet, create one (match existing test style in the package).

The protocol-tester will expand — your bar is "it works and basic tests pass".

## Metric / Log Integration

If the design calls for new metrics:
- Follow the pattern in `protocol/metrics`.
- Use the same metric naming conventions (snake_case, `_total` for counters, `_seconds` for histograms).
- Labels: chain, provider, method — low cardinality.

If adding new log fields, use `utils.Attribute` consistently.

## Protobuf Changes (only if design calls for it)

1. Edit `proto/lavanet/lava/<package>/*.proto`.
2. Run `make proto-gen`.
3. Commit generated files (don't hand-edit generated code).
4. Update Go call sites.

## Fiber / HTTP Integration

- Recent direction: send bytes via `fiber.Ctx.Send`, avoid string conversions.
- Use `ctx.BodyRaw()` when you need the raw bytes without allocation.

## Output

`_workspace/02_implementation.md` per role template. Be honest about deviations, lint results, test results.

## Revision Handling

When re-invoked with feedback:
1. Read `_workspace/00_input/feedback.md`.
2. Read prior `02_implementation.md` — don't undo prior good work.
3. Make minimal changes to address feedback.
4. Update `02_implementation.md` with `Revision N` section at top.

## Common Pitfalls

- Copy-pasting a block from elsewhere without verifying the target package's style → creates inconsistency.
- Adding a new config flag not in the design → scope creep.
- Forgetting to plumb `context.Context` through a new goroutine chain.
- Using `fmt.Errorf` when the file otherwise uses `utils.LavaFormatError`.
- Holding a lock across a network/gRPC call.
