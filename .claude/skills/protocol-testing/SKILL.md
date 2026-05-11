---
name: protocol-testing
description: "How the protocol-tester writes and expands tests for the Lava protocol layer. Loaded when protocol-tester runs. Covers layer choice (unit / integration / e2e), table-driven patterns, race detection, fakes over mocks, deterministic time/randomness, and coverage reporting."
---

# Protocol Testing Skill

Method used by protocol-tester. Role at `.claude/agents/protocol-tester.md`; this file is the *how*.

## Method Overview

1. Read `_workspace/01_design.md` §7 Test Strategy (if development flow) or orchestrator-provided test scope.
2. Read `_workspace/02_implementation.md` to know what changed (Files Changed table).
3. For each behavior added/changed, decide the appropriate layer (unit / integration / e2e).
4. Write tests. Run them.
5. Produce `_workspace/03_testing.md`.

## Layer Decision

Push tests down. Prefer unit > integration > e2e.

| Layer | When | Path |
|-------|------|------|
| **unit** | logic inside a single fn/struct; no network; fakes for interfaces; <1s per test | `protocol/<pkg>/*_test.go` (same package or `_test` package) |
| **integration** | 2+ protocol packages together OR real chainlib proxy with fixture upstream | `protocol/integration/**` |
| **e2e** | full consumer↔provider behavior that can't be isolated | `testutil/e2e/**` |

Rule: each layer up is 10× slower and 10× flakier. e2e only for bugs that require the full stack to reproduce.

## Table-Driven Pattern (codebase dominant)

```go
func TestConsumer_RetriesOnProviderTimeout(t *testing.T) {
    tests := []struct {
        name        string
        providerErr error
        retries     int
        wantErr     bool
    }{
        {name: "timeout_then_success", providerErr: context.DeadlineExceeded, retries: 2, wantErr: false},
        {name: "persistent_timeout",   providerErr: context.DeadlineExceeded, retries: 3, wantErr: true},
        {name: "non_retryable",        providerErr: errPermission,             retries: 0, wantErr: true},
    }
    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            // ... arrange, act, assert
        })
    }
}
```

Notes:
- `tt := tt` inside the loop is defensive for Go <1.22; with Go 1.22+ `copyloopvar` it's no longer needed but harmless.
- `t.Parallel()` where the subtest has no shared state.
- Names are behavior-driven, not function-driven.

## Assertion Library

Use what the target package uses. If the package uses `testify/require`, use `testify/require`. Don't mix `require` and plain `t.Errorf` unless you have a reason.

## Fakes Over Mocks

Hand-written fake structs that implement the real interface are preferred. Mocks via mock frameworks only when the surrounding package already uses them for that interface.

Example fake for provider selection:

```go
type fakeProviderOptimizer struct {
    calls []string  // list of chainIDs called for Choose
    pick  string    // provider to return
}

func (f *fakeProviderOptimizer) Choose(chainID string) (string, error) {
    f.calls = append(f.calls, chainID)
    return f.pick, nil
}
```

Benefits: no reflection, IDE navigable, test reads like a story.

## Race Detector

Any test touching concurrency MUST be green under `go test -race`. Add it to the test run command and verify.

For tests that intentionally exercise races (to reproduce a reported race), consider `goleak` or `uber-go/goleak` equivalent for leak detection, or use a counter/channel barrier to force ordering.

## Deterministic Time

Don't use real `time.Now` in tests that assert on time.
- Inject a clock: `func(now func() time.Time)` constructor.
- Or use a test-only build tag that swaps a clock.

Don't `time.Sleep` to wait. Use:
- `require.Eventually(t, cond, timeout, tick)`
- Channels to signal completion
- `sync.WaitGroup` for goroutine join

## Fixtures

- Inline small fixtures (JSON strings <200 chars): inline literal.
- Medium fixtures: adjacent `testdata/` directory next to the test.
- Large recorded fixtures: `testdata/` with a comment on how they were recorded.

Don't fabricate upstream responses if a recorded fixture is realistic — drift risk.

## Negative Path Coverage

For every happy-path test, at least one error-path. Check:
- Returned error is the expected type (`errors.Is` / `errors.As`).
- Observable side effects (metric emitted, log line, state updated).
- Caller's expected behavior (retry, return, panic).

## Benchmarks

When the design or a perf finding calls for it:

```go
func BenchmarkRelayParse(b *testing.B) {
    req := newFixtureRequest()
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = parseRelay(req)
    }
}
```

Run with `go test -bench=. -benchmem ./protocol/<pkg>`. Capture before/after numbers in `03_testing.md`.

## Running the Tests

Before declaring done:

```bash
go test ./protocol/<changed-dirs>/... -race -timeout 15m
```

For the full protocol suite:
```bash
go test ./protocol/... -race -timeout 15m
```

For e2e:
```bash
go test ./testutil/e2e/ -run ^TestLavaProtocol$ -timeout 1200s
```

Note the test timeouts from `CLAUDE.md` — protocol tests need 15m recommended, e2e 20min.

## Flaky Risk Assessment

In `03_testing.md`, call out any test that has:
- Timing assumptions (`Eventually` timeouts)
- Order-dependent channel semantics
- Network (even fakes) — retry logic under test
- Shared state (global registry, package-level cache)

Mitigation options: tighter determinism (mock time), explicit ordering (WaitGroup), isolation (per-test instance).

## Discovering Bugs

If you discover a bug while writing tests:
1. Write the failing test first.
2. Note the bug in `03_testing.md` under "Bugs Found".
3. Do NOT silently fix production code as a tester — that's the implementer's job. Surface to orchestrator.

## Output

`_workspace/03_testing.md` per role template. Include run results, coverage notes, flaky risk, follow-ups.

## Revision Handling

When re-invoked to expand coverage:
1. Read feedback + prior `03_testing.md`.
2. Keep existing tests; only add.
3. Update summary with `Revision N` at top.
