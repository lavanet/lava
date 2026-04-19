# Architecture Review Checklist

Detailed rubric for architecture-reviewer. Each item is a yes/no check; a "no" is a candidate finding.

## Table of Contents

1. [Module Boundaries](#1-module-boundaries)
2. [Dependency Direction](#2-dependency-direction)
3. [Interface Design](#3-interface-design)
4. [Concurrency Architecture](#4-concurrency-architecture)
5. [State Management](#5-state-management)
6. [Error Model](#6-error-model)
7. [Request-Path Shape](#7-request-path-shape)
8. [Testability](#8-testability)
9. [Smart-Router vs Consumer Divergence](#9-smart-router-vs-consumer-divergence)

---

## 1. Module Boundaries

Per package in scope:

- [ ] Package name reflects its single responsibility.
- [ ] Exports are minimal relative to imports — i.e., consumers don't reach into internals.
- [ ] No "util" or "helper" grab-bag. `protocol/common` is intentional shared types, not a dumping ground.
- [ ] Public symbols that are used by only one caller — candidate for unexport or move.
- [ ] Internal types leaking through public method signatures — either unexport them or add an abstraction.
- [ ] No circular responsibility (package A does X + part of Y, package B does Y + part of X).

**Signal for finding:** A change to one package forces mechanical edits across many others → boundary is fuzzy.

## 2. Dependency Direction

- [ ] Imports flow bottom-up per the layer diagram (chainlib is a leaf; relaycore sits on top of chainlib; rpcconsumer sits on top of relaycore).
- [ ] No cycles at package level (`go vet` catches Go-level cycles, but informal cycles via interfaces are sneaky).
- [ ] Statetracker and chaintracker are drivers (push updates) — they shouldn't import the packages they drive.
- [ ] Metrics / logging are dependencies of everything; they never depend on business packages.
- [ ] Packages don't import their own test helpers from outside (test helper lives in the package or under `testutil/`).

**Signal for finding:** An upward import exists (e.g., `chainlib` imports `rpcconsumer`) — structural leak.

## 3. Interface Design

- [ ] Interfaces are defined at the **consumer** side (the package that uses them), not the provider side (Go idiom).
- [ ] Interface methods are minimal — ISP. Flag interfaces with >6 methods unless they model a protocol with inherent surface.
- [ ] Return types are concrete structs unless a second implementation exists or is planned. (Returning interfaces by default bloats the type space.)
- [ ] Interface methods don't accept other interfaces without clear justification (wrapping chains are hard to reason about).
- [ ] No "interface-per-type" pattern where a test mock is the only second implementation. (Prefer hand-written fakes or real deps.)

**Signal for finding:** An interface exists solely to enable a mock → consider direct injection of a hand-written fake or a small FakeXxx struct.

## 4. Concurrency Architecture

- [ ] Every goroutine has a named owner — which function starts it, which stops it, which context governs its lifetime.
- [ ] Every channel has a defined closer. Receivers don't close; senders do (Go convention).
- [ ] `context.Context` is plumbed through every function that can block or loop.
- [ ] No `context.Background()` mid-request-path — only at true entry points.
- [ ] Goroutine fan-out is bounded (worker pool, semaphore, or rate-limited channel).
- [ ] Any `for { select { } }` loop has a `ctx.Done()` or `stop` channel case.
- [ ] No `time.Sleep` in goroutine bodies without context check.
- [ ] Shared state accesses are clearly protected (mutex, atomic, channel-based) — don't rely on "I think this is safe".

**Signal for finding:** A goroutine whose lifetime extends past the returning function, with no visible stop mechanism → leak.

## 5. State Management

- [ ] Each piece of mutable state has one owner.
- [ ] State-tracker snapshot semantics match readers (is the snapshot internally consistent? does it change under the reader's feet?).
- [ ] Epoch transitions — every component that cares about epochs has a defined update point.
- [ ] Session state (lavasession) has clear create / lookup / expire / destroy lifecycles.
- [ ] Caches have explicit eviction or size bounds.

**Signal for finding:** Two packages both mutate the "same" piece of state with no coordination → undefined behavior.

## 6. Error Model

- [ ] Errors are typed where callers branch on them (`errors.Is/As`).
- [ ] Errors carry context (chain ID, provider, epoch, session) via `utils.LavaFormatError` attrs.
- [ ] No silent swallowing — every `if err != nil` does *something* (return, log, retry, etc.). Note: the project sometimes uses `_ = someCall()` on purpose; flag only when the error seems material.
- [ ] Panic is reserved for programmer errors (impossible states). Runtime errors are wrapped and returned.
- [ ] Error wrap chains don't lose information (`%w` when `errors.Is/As` matters).

**Signal for finding:** A caller branches on an error's string → error not typed.

## 7. Request-Path Shape

- [ ] The hot path (consumer → optimizer → relaycore → network → provider → chainlib → upstream → return) is legible — a reader can trace one relay top-to-bottom without detours.
- [ ] Recent work has not stuffed unrelated logic into the hot path.
- [ ] Retry/failover logic lives in one place, not scattered.
- [ ] Finalization checks are at their intended layer (chainlib for parse-time, lavaprotocol for relay-level).

**Signal for finding:** Reading one hop of the hot path requires understanding 5 other packages → the abstraction is wrong.

## 8. Testability

- [ ] Key seams can be faked without mocking library reflection.
- [ ] Time sources are injectable or the package uses `time.Now` only in test-mockable helpers.
- [ ] Randomness sources (`math/rand`, `crypto/rand`) are injectable where behavior depends on them.
- [ ] Network boundaries abstracted behind an interface (e.g., gRPC client interface injectable).
- [ ] Tests live near the code (same package or `_test` package in same dir), not scattered under a parallel `tests/` tree.

**Signal for finding:** A test requires a real upstream node to run → integration test at unit layer.

## 9. Smart-Router vs Consumer Divergence

For code shared between rpcconsumer and rpcsmartrouter (relaycore, lavaprotocol, chainlib, etc.):

- [ ] Divergences are intentional — gated by a config flag or a clearly-named function suffix.
- [ ] Identical behavior isn't duplicated (flag copy-paste drift).
- [ ] Trust-model differences are reflected at the right layer (e.g., signature verification skip in smart-router is at a specific code point, not sprinkled).

**Signal for finding:** Near-duplicate functions in rpcconsumer and rpcsmartrouter with subtle differences → drift risk.

---

## Severity Heuristics for This Reviewer

- **critical** — Structural decision that will break production or block a known near-term change (e.g., a cycle that `go vet` will flag on the next merge).
- **high** — Structure will force expensive refactor when a plausible change arrives (e.g., adding a new chain family requires editing 8 packages).
- **medium** — Friction in everyday work (e.g., an interface so broad everyone has to scroll to find the method they want).
- **low** — Minor structural tidy (e.g., a helper that could move to a better package).
- **info** — Positive observation or FYI.
