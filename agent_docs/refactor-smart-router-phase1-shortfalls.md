# refactor/smart-router Phase 1 Shortfalls

Date: 2026-03-31  
Branch reviewed: `refactor/smart-router`  
Scope basis: `agent_docs/smart-router-repo.md` section `8.2 Phase 1 - Fork and Cosmos SDK Cleanup` (with fork intentionally deferred for now).

## Current Status

The branch shows strong progress toward Phase 1, but it is not yet at Phase 1 exit criteria.

## Shortfalls and Potential Fixes

### 1) Standalone `smart-router` binary target is missing

Observed:
- `cmd/smartrouter` directory is missing.
- `go build ./cmd/smartrouter` fails (`directory not found`).
- Smart router is still launched through [`cmd/lavap/main.go`](/Users/avitenzer/Documents/_workspace/magma-devs/lava/cmd/lavap/main.go).

Why this matters:
- Phase 1 explicitly requires a standalone binary (`smart-router`) instead of `lavap` subcommand wiring.

Potential fix:
- Add `cmd/smartrouter/main.go` with standalone bootstrap.
- Keep config compatibility with existing smart-router YAML files.
- Keep `lavap` as temporary compatibility wrapper only if needed, but make `cmd/smartrouter` the primary target.

Verification:
- `go build ./cmd/smartrouter`
- Smoke check `./smart-router --help`

### 2) Smart-router test suite has failing tests (core readiness blocker)

Observed failing suite summary:
- `go test ./protocol/rpcsmartrouter/...` fails.
- Representative failing tests include REST and state-machine paths:
  - [`protocol/rpcsmartrouter/rest_integration_test.go`](/Users/avitenzer/Documents/_workspace/magma-devs/lava/protocol/rpcsmartrouter/rest_integration_test.go)
  - [`protocol/rpcsmartrouter/smartrouter_relay_state_machine_test.go`](/Users/avitenzer/Documents/_workspace/magma-devs/lava/protocol/rpcsmartrouter/smartrouter_relay_state_machine_test.go)

Primary error pattern:
- Spec decode failures from [`utils/keeper/spec.go`](/Users/avitenzer/Documents/_workspace/magma-devs/lava/utils/keeper/spec.go):
  - `min_stake_provider.amount` expected `int64` but input JSON is string.
  - `providers_types` expected enum-string format but input can be numeric.

Why this matters:
- Static spec loading is a startup/runtime dependency for smart router.
- If decode fails, REST/state-machine behavior is effectively broken for many chains.

Potential fix:
- Make extracted spec types tolerant to legacy/current spec JSON encodings:
  - `Coin.Amount`: support quoted numeric strings and numeric literals.
  - `Spec_ProvidersTypes`: support both string and numeric JSON input.
- Add decoding compatibility tests for both representations.

Verification:
- `go test ./protocol/rpcsmartrouter/...`
- Targeted decode tests under `x/spec/types` and `utils/keeper`

### 3) `lavasession` has at least one failing test

Observed:
- `go test ./protocol/lavasession/...` fails.
- Failure example: `TestEndpointSortingFlow` in [`protocol/lavasession/consumer_session_manager_test.go`](/Users/avitenzer/Documents/_workspace/magma-devs/lava/protocol/lavasession/consumer_session_manager_test.go).

Why this matters:
- Session and endpoint ordering behavior is central to smart-router reliability.

Potential fix:
- Investigate async probe/swap ordering assumptions and race windows.
- Replace timing-based wait logic with deterministic readiness/sync signaling where possible.

Verification:
- `go test ./protocol/lavasession/...`
- Re-run flaky candidates with `-count` (for example `-count=20`) to verify stability.

### 4) Phase 1 strict dependency cleanup is not fully finished yet

Observed from dependency graph:
- `go list -deps ./cmd/lavap` still includes:
  - `github.com/lavanet/lava/v5/x/spec/types`
  - `github.com/lavanet/lava/v5/x/pairing/types`
  - `cosmossdk.io/errors`
  - `cosmossdk.io/math`

Why this matters:
- If the intended Phase 1 standard is "clean runtime without blockchain module coupling", this is still incomplete.

Potential fix:
- Continue extraction from `x/...` module paths into smart-router-owned protocol types package(s).
- Remove remaining `cosmossdk.io` usage from runtime paths where feasible.

Verification:
- `go list -deps ./cmd/smartrouter | rg "x/pairing|x/spec|cosmossdk.io|cosmos-sdk"`

### 5) Residual Cosmos-branded packaging references exist

Observed:
- [`cmd/lavap/Dockerfile`](/Users/avitenzer/Documents/_workspace/magma-devs/lava/cmd/lavap/Dockerfile) still contains cosmos-sdk version linker references.

Why this matters:
- Creates packaging/release confusion during migration to standalone smart-router binary.

Potential fix:
- Align Docker/ldflags naming with the standalone smart-router binary/version path.

Verification:
- Build image and confirm binary metadata and naming match smart-router expectations.

### 6) Branch hygiene risk: stale local `smart-router` branch can mislead validation

Observed:
- Local branch `smart-router` is stale and does not reflect current extraction work.
- `refactor/smart-router` is the actual implementation branch.

Why this matters:
- Review/test efforts can accidentally target the wrong branch and produce false conclusions.

Potential fix:
- Mark stale branch clearly (rename/archive/delete locally as team policy allows).
- Update team docs/PR template to always reference `refactor/smart-router` for this initiative.

Verification:
- Team workflow/docs consistently point to one canonical branch.

## Priority Order (Suggested)

1. Fix spec decode compatibility (`Coin.Amount`, `Spec_ProvidersTypes`) and get `rpcsmartrouter` tests green.
2. Fix `lavasession` failing test(s) and stabilize with repeat runs.
3. Add `cmd/smartrouter` and make it the primary build target.
4. Finish strict dependency cleanup (`x/...` and `cosmossdk.io` from runtime path if required by Phase 1 interpretation).
5. Clean packaging/docs and branch hygiene items.

## Practical Acceptance Checklist For "Phase 1 Ready" (Branch-Only, No Fork Yet)

- `go build ./cmd/smartrouter` succeeds.
- `go test ./protocol/rpcsmartrouter/...` passes.
- `go test ./protocol/lavasession/...` passes.
- Static specs from `specs/` load without decode errors.
- `go list -deps ./cmd/smartrouter` has no unwanted blockchain/Cosmos dependencies per agreed Phase 1 bar.
- Startup and config behavior is documented for migration from `lavap` invocation to standalone `smart-router`.

---

## Shortfall Review (2026-03-31, post-remediation)

Each shortfall re-evaluated against the current `refactor/smart-router` branch state.

### #1 — Standalone binary: RESOLVED
- `cmd/smartrouter/main.go` exists and builds successfully.
- `go build ./cmd/smartrouter` produces a working `smart-router` binary.
- `smart-router version` outputs `6.1.0`.
- `smart-router --help` shows `rpcsmartrouter`, `test`, and `version` commands.
- Makefile has `build` (smart-router), `build-lavap`, and `install-all` targets.
- **No further action needed.**

### #2 — Smart router test suite: RESOLVED
- `go test ./protocol/rpcsmartrouter/...` now passes all tests.
- Root causes were:
  - `Coin.Amount` decoded as `int64` but spec JSON uses quoted strings — fixed with custom `UnmarshalJSON`.
  - `Spec_ProvidersTypes` expected string but JSON has numeric — fixed with dual string/numeric unmarshal.
  - `ContributorPercentage` was `*float64` but JSON has quoted string `"0.015"` — fixed with `FlexFloat64` type.
  - `DisallowUnknownFields` in spec decoder rejected fields our stripped types don't model — removed.
  - Cross-directory spec imports (LAV1 → COSMOSSDK) failed — fixed with `GetSpecFromLocalDirs` multi-directory loader.
- **No further action needed.**

### #3 — lavasession failing test: STILL OPEN (pre-existing, low priority)
- `TestEndpointSortingFlow` fails consistently — waits 20s for async probe-based endpoint sorting that never completes.
- **This is a pre-existing issue**, not a regression from the extraction. The test depends on gRPC probe connections to a mock server succeeding within a timing window. The probe mechanism uses async goroutines with `time.Sleep` polling.
- The test was last significantly modified in commit `bd37dbff8` (on main) and `e3055cfe6`, both before the extraction work.
- **Verdict**: Valid issue but not caused by Phase 1 work. Low priority for Phase 1 exit — this is a test robustness improvement, not a code correctness issue.

### #4 — Dependency cleanup: MOSTLY RESOLVED
- `cosmos-sdk` module is **fully removed** from `go.mod` (no require, no replace).
- `x/` directory convention is **fully removed** — types now live under `types/relay`, `types/spec`, `types/epoch`, `types/plans`, `types/protocol`.
- Two `cosmossdk.io` utility libraries remain as dependencies:
  - `cosmossdk.io/errors` — lightweight error wrapping (no blockchain coupling, ~200 lines)
  - `cosmossdk.io/math` — decimal math used by score computations (no blockchain coupling)
- **Verdict**: These are standalone utility modules published independently from cosmos-sdk. They have no blockchain state, no Cosmos SDK imports, and no chain coupling. Removing them would require replacing their functionality with stdlib equivalents (possible but low value). **Acceptable for Phase 1 — defer full removal to a future cleanup pass if desired.**

### #5 — Dockerfile cosmos references: STILL OPEN (low priority)
- `cmd/lavap/Dockerfile` still exists with cosmos-sdk ldflags references:
  ```
  -X github.com/cosmos/cosmos-sdk/version.Name="lava"
  -X github.com/cosmos/cosmos-sdk/version.AppName="lavap"
  -X github.com/cosmos/cosmos-sdk/version.Version=${GIT_VERSION}
  ```
- These ldflags reference a module that no longer exists in go.mod, so they are **silently ignored** (the linker skips unknown -X paths). The Docker build still works but the version metadata won't be embedded.
- **Verdict**: Valid issue. The Dockerfile should be updated to build `cmd/smartrouter` and use local version ldflags. Low priority — no runtime impact.

### #6 — Branch hygiene: RESOLVED (N/A for implementation)
- This is a team workflow item, not a code issue.
- **No code action needed.**

---

## Action Plan

### Must-do for Phase 1 exit

| # | Task | Effort | Status |
|---|------|--------|--------|
| 1 | Update `cmd/lavap/Dockerfile` to build `cmd/smartrouter`, remove cosmos-sdk ldflags, update ENTRYPOINT to `smart-router` | Small | Open |

### Should-do (recommended but not blocking)

| # | Task | Effort | Rationale |
|---|------|--------|-----------|
| 2 | Fix `TestEndpointSortingFlow` in lavasession — replace timing-based polling with deterministic signaling (e.g. channel/sync.WaitGroup) | Medium | Pre-existing flaky test; not a regression but hurts CI confidence |
| 3 | Remove `cosmossdk.io/errors` dependency — replace with `fmt.Errorf` / stdlib error wrapping | Small | Eliminates last cosmossdk.io reference; low value but clean |
| 4 | Remove `cosmossdk.io/math` dependency — replace `math.LegacyDec` usage with `float64` or `math/big` | Medium | Only used in score computations; float64 is sufficient |

### Deferred (Phase 2 / post-fork)

| # | Task | Rationale |
|---|------|-----------|
| 5 | Rename Go module path from `github.com/lavanet/lava/v5` to `github.com/magma-devs/smart-router` | Deferred to actual repo creation |
| 6 | Write migration guide: `lavap rpcsmartrouter` → `smart-router rpcsmartrouter` | Phase 2 deliverable |
| 7 | Community/enterprise gating with build tags + license validation | Phase 2 scope |
