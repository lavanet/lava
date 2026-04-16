# Relay State Machine: Architecture and Pure Go Refactor

**Date**: 2026-03-24
**Status**: Partially Implemented
**Audience**: Engineering team
**Decision**: All retry refactoring will use pure Go. No external libraries (failsafe-go or others).

### Implementation Status

| Area | Status |
|---|---|
| Phase 1: Unified Consumer + SmartRouter state machine | **Done** |
| Phase 2: `relaypolicy.Policy` with `Decide()` and `OnSendRelayResult()` | **Done** |
| Phase 2: `ClassifyNodeError` wired into consumer worker | **Done** — uses `ClassifyNodeErrorForRetry` from error registry. Sets `IsNonRetryable`, `IsUnsupportedMethod`. `IsUserError` removed (user-input errors charge normal CU). SmartRouter unchanged (preserving `main` behavior). |
| Phase 2: `IsNonRetryable` as the retry gate in `ResultsSummary` and `policy.Decide()` | **Done** — `GetResultsSummary()` scans for `IsNonRetryable` and sets `HasNonRetryableNodeError`. `policy.Decide()` checks `HasNonRetryableNodeError` as a single stop condition. |
| Phase 2: `DecideEligibility()` | **Done** — `used_providers.go:RemoveUsed()` computes eligibility inputs inline and calls `common.DecideEligibility(isUnsupportedMethod, isSyncLoss, isFirstSyncLoss)` directly. `relaypolicy.DecideEligibility` re-exports the same function. `shouldRetryWithThisError` was removed. |
| Phase 3: Simplified `HasRequiredNodeResults()` | **Done** |
| Phase 4: Provider-side `SendNodeMessage()` refactor | Not started |
| Hedge budget (`AllowHedge`) | Not started |
| Section 13: `Lava-Retry-Debug` trace header | Not started |

#### IsNonRetryable — implemented

`GetResultsSummary()` scans node errors for `IsNonRetryable == true` and sets `HasNonRetryableNodeError` in the summary. `policy.Decide()` checks `HasNonRetryableNodeError` as a single stop condition. `HasUnsupportedMethod` remains in the summary for the zero-CU carve-out and caching decisions. `IsUserError`/`HasUserError` have been removed — user-input errors now charge normal CU (providers do real work on every call since responses are not cached).

---

## 1. Overview

This document describes:
1. The current state machine architecture after Phase 1 (unified Consumer + SmartRouter)
2. How decision points are distributed across goroutines and why
3. The problems with this distribution
4. The refactored architecture where decisions are centralized in the state machine via a pure Go policy engine

---

## 2. Current Architecture: Goroutine Model

The relay lifecycle involves **four concurrent goroutines** communicating via channels. Each goroutine has a specific role, and retry decision points are currently scattered across all of them.

```
 Consumer/SmartRouter HTTP Request Arrives
              │
              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ RPCConsumerServer.SendRelay() / RPCSmartRouterServer.SendRelay()            │
│                                                                             │
│  1. Parse request → ProtocolMessage                                         │
│  2. Create RelayProcessor (result aggregator)                               │
│  3. Create UsedProviders (provider tracker)                                 │
│  4. Create UnifiedRelayStateMachine                                         │
│  5. relayTaskChannel := sm.GetRelayTaskChannel()  ← starts Goroutine A     │
│  6. Dispatch loop: for task := range relayTaskChannel  ← Goroutine B       │
│  7. Block on result                                                         │
│                                                                             │
└──────────────────┬──────────────────────────────────────────┬───────────────┘
                   │                                          │
         Goroutine A                                Goroutine B
         (STATE MACHINE)                            (DISPATCH LOOP)
                   │                                          │
                   ▼                                          ▼
┌──────────────────────────────────────┐  ┌──────────────────────────────────────┐
│ GetRelayTaskChannel()                 │  │ Dispatch Loop                         │
│                                       │  │                                       │
│ ►[SEND#1] INITIAL SEND               │  │ for task := range relayTaskChannel {  │
│ relayTaskCh <- instructions ─────────┼─▶│   if task.Done → close(ch), return    │
│   (1 provider, or N for CV)           │  │   sendRelay(task)                  ──┼──┐
│                                       │  │   sm.UpdateBatch(err)                │  │
│ go readResultsFromProcessor()         │  │ }                                    │  │
│ // starts Goroutine C                 │  └─────────────────────┬────────────────┘  │
│                                       │                        │                    │
│ for {                                 │                  batchUpdate channel         │
│   select {                            │                        │                    │
│   ┌─────────────────────────────────┐ │                        │                    │
│   │ case err := <-batchUpdate:      │◀┼────────────────────────┘                    │
│   │                                 │ │                                              │
│   │ if err == nil:                  │ │                                              │
│   │   reset counters, continue      │ │                                              │
│   │                                 │ │                                              │
│   │ if err != nil:                  │ │        Goroutine C                           │
│   │   consecutiveBatchErrors++      │ │        readResultsFromProcessor()            │
│   │                                 │ │        ┌────────────────────────────────┐    │
│   │   ►DP#11 Circuit breaker        │ │        │ WaitForResults() — see zoom ──┼─┐  │
│   │   PairingListEmpty >= 2?        │ │        │                                │ │  │
│   │     YES (trip):                 │ │        │ HasRequiredNodeResults()        │ │  │
│   │       validateReturnCondition() │ │        │   calls shouldRetryRelay()     │ │  │
│   │       → returnCondition channel │ │        │   gotResults <- result         │ │  │
│   │       continue (no send)        │ │        └────────────────────────────────┘ │  │
│   │     NO (continue):              │ │                                           │  │
│   │       fall through to DP#12     │ │        Goroutines D, E, F...              │  │
│   │                                 │ │        (RELAY WORKERS)                    │  │
│   │   ►DP#12 Batch send retry       │ │        ┌────────────────────────────────┐ │  │
│   │   consecutiveBatchErrors > 3?   │ │        │ Per-provider goroutines       ◀┼─┘──┘
│   │     YES (give up):              │ │        │                                │
│   │       validateReturnCondition() │ │        │ 1. Connect to provider         │
│   │       → returnCondition channel │ │        │ 2. Send relay request          │
│   │     NO (retry send):            │ │        │ 3. Receive response            │
│   │       ►[SEND#2] RETRY SEND      │ │        │                                │
│   │       relayTaskCh <- instr ─────┼─┼─▶ B    │ ►DP#6/#7/#8 Error classify     │
│   │                                 │ │        │ ►DP#9 Provider eligibility     │
│   └─────────────────────────────────┘ │        │                                │
│   ┌─────────────────────────────────┐ │        │ 4. relayProcessor              │
│   │ case success := <-gotResults:   │◀┼────────│      .SetResponse(resp) ──────┼──▶ C
│   │                                 │ │        │                                │
│   │   if success:                   │ │        │ 5. OnSessionFailure()          │
│   │     ►[SEND#3] DONE (success)    │ │        └────────────────────────────────┘
│   │     relayTaskCh <- {Done:true} ─┼─┼─▶ B (closes dispatch loop)
│   │     return                      │ │
│   │                                 │ │
│   │   if !success:                  │ │
│   │     ►DP#1/#2 shouldRetry()      │ │
│   │       if YES:                   │ │
│   │         stateTransition()       │ │
│   │         + archive mutation      │ │
│   │         ►[SEND#4] RETRY RELAY   │ │
│   │         relayTaskCh <- instr ───┼─┼─▶ B
│   │         go readResults... (new) │ │
│   │       if NO:                    │ │
│   │         validateReturnCondition │ │
│   │         → returnCondition ch    │ │
│   │         go readResults... (new) │ │
│   │                                 │ │
│   └─────────────────────────────────┘ │
│   ┌─────────────────────────────────┐ │
│   │ case <-ticker.C:                │ │
│   │                                 │ │
│   │   ►DP#1/#2 shouldRetry()        │ │
│   │     if YES:                     │ │
│   │       ►[SEND#5] HEDGE           │ │
│   │       relayTaskCh <- instr ─────┼─┼─▶ B
│   │     if NO:                      │ │
│   │       (do nothing, continue)    │ │
│   │   NO HEDGE BUDGET today         │ │
│   │                                 │ │
│   └─────────────────────────────────┘ │
│   ┌─────────────────────────────────┐ │
│   │ case returnErr := <-returnCond: │ │
│   │                                 │ │
│   │   ►[SEND#6] DONE (error)       │ │
│   │   relayTaskCh <- {Err, Done} ──┼─┼─▶ B (closes dispatch loop)
│   │   return                        │ │
│   │                                 │ │
│   └─────────────────────────────────┘ │
│   ┌─────────────────────────────────┐ │
│   │ case <-processingCtx.Done():    │ │
│   │                                 │ │
│   │   ►[SEND#7] DONE (timeout)     │ │
│   │   relayTaskCh <- {Err, Done} ──┼─┼─▶ B (closes dispatch loop)
│   │   return                        │ │
│   │                                 │ │
│   └─────────────────────────────────┘ │
│   }                                   │
│ }                                     │
└───────────────────────────────────────┘

  Additionally, SmartRouter's checkAndHandleTimeout() can send
  ►[SEND#8] relayTaskCh <- {Err, Done} from 4 locations:
    - Before select loop (priority check)
    - Inside batchUpdate error (before retry send)
    - Inside gotResults (before retry)
    - Inside ticker.C (before hedge)


PROVIDER SIDE (separate process):
┌────────────────────────────────────────┐
│ ►DP#13 SendNodeMessage()               │
│  sync for loop: retry to upstream node │
│  ►DP#10 CheckHashInCache              │
└────────────────────────────────────────┘


═══════════════════════════════════════════════════════════════════════════════
 ZOOM: WaitForResults() — what Goroutine C does inside
═══════════════════════════════════════════════════════════════════════════════

  readResultsFromProcessor() calls two functions sequentially:

  ┌─────────────────────────────────────────────────────────────────────────┐
  │ STEP 1: WaitForResults(ctx)                                            │
  │                                                                         │
  │ Blocks until this batch of workers is done.                             │
  │ Workers send responses via rp.responses channel (from SetResponse()).   │
  │                                                                         │
  │ for {                                                                   │
  │   select {                                                              │
  │   case response := <-rp.responses:      ← worker sent a result         │
  │     responsesCount++                                                    │
  │     handleResponse(response)            ← stores in success/nodeErr/   │
  │       │                                    protocolErr buckets,          │
  │       │                                    updates CV quorum hash map   │
  │       │                                                                 │
  │     checkEndProcessing(responsesCount)  ← "can I stop waiting?"        │
  │       │                                                                 │
  │       ├─ responsesCount >= SessionsLatestBatch()?                       │
  │       │    YES → return (all workers from this batch reported back)     │
  │       │                                                                 │
  │       ├─ CrossValidation: quorum >= agreementThreshold?                 │
  │       │    YES → return (enough matching responses for consensus)       │
  │       │                                                                 │
  │       └─ Stateless/Stateful: successCount >= 1?                         │
  │            YES → return (got at least one success, no need to wait      │
  │                          for remaining workers in this batch)           │
  │                                                                         │
  │   case <-ctx.Done():                    ← timeout                       │
  │     return                                                              │
  │   }                                                                     │
  │ }                                                                       │
  │                                                                         │
  │ NOTE: checkEndProcessing does NOT make retry decisions.                 │
  │ It only answers: "have enough workers from THIS BATCH reported back     │
  │ for me to stop blocking?" Either all reported, or we got an early       │
  │ exit condition (1 success or quorum met).                               │
  └─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
  ┌─────────────────────────────────────────────────────────────────────────┐
  │ STEP 2: HasRequiredNodeResults(batchNumber)                             │
  │                                                                         │
  │ Called immediately after WaitForResults() returns.                       │
  │ Determines what to signal to the state machine via gotResults channel.  │
  │                                                                         │
  │ ├─ CrossValidation:                                                     │
  │ │    quorumEqualResults >= agreementThreshold?                          │
  │ │      YES → gotResults <- true  (quorum met, DONE)                    │
  │ │      NO  → gotResults <- false (quorum not met)                      │
  │ │                                                                       │
  │ ├─ Stateless/Stateful:                                                  │
  │ │    resultsCount >= 1?                                                 │
  │ │      YES → gotResults <- true  (got a success, DONE)                 │
  │ │      NO  → calls shouldRetryRelay() (DP#3)  ← THE RETRY DECISION    │
  │ │              │                                                        │
  │ │              ├─ shouldRetryRelay returns true  (keep trying)          │
  │ │              │    → gotResults <- false                               │
  │ │              │    (tells state machine "we failed",                   │
  │ │              │     giving it a chance to run DP#1 and retry)          │
  │ │              │                                                        │
  │ │              └─ shouldRetryRelay returns false (give up)              │
  │ │                   → gotResults <- true                                │
  │ │                   (tells state machine "we're done",                  │
  │ │                    even though there's no success —                   │
  │ │                    state machine returns error to user)               │
  │ │                                                                       │
  │ │  NOTE: When shouldRetryRelay says "give up" (false), it sends        │
  │ │  gotResults <- true (which means "met requirements"). This is        │
  │ │  because metRequiredNodeResults = !shouldRetry. So "don't retry"     │
  │ │  becomes "requirements met" — meaning "we're done, return whatever   │
  │ │  we have (which is an error) to the user." This is the relay         │
  │ │  processor's way of telling the state machine to stop WITHOUT the    │
  │ │  state machine getting a chance to run DP#1.                         │
  │ └                                                                       │
  └─────────────────────────────────────────────────────────────────────────┘


═══════════════════════════════════════════════════════════════════════════════
 ALL relayTaskChannel SENDS — complete list
═══════════════════════════════════════════════════════════════════════════════

  ┌──────┬──────────────────────┬─────────────────────────────────────────────┐
  │ SEND │ Where                │ What / Why                                  │
  ├──────┼──────────────────────┼─────────────────────────────────────────────┤
  │ #1   │ Before select loop   │ INITIAL SEND. First relay attempt.          │
  │      │                      │ {RelayState, NumOfProviders: 1 or N for CV} │
  ├──────┼──────────────────────┼─────────────────────────────────────────────┤
  │ #2   │ batchUpdate err case │ RETRY SEND. GetSessions() failed but        │
  │      │ (DP#12, ≤3 errors)   │ under threshold — try getting sessions      │
  │      │                      │ again. {RelayState, NumOfProviders: 1}      │
  ├──────┼──────────────────────┼─────────────────────────────────────────────┤
  │ #3   │ gotResults case      │ DONE (success). Relay succeeded.            │
  │      │ (success == true)    │ {Done: true} — dispatch loop exits.         │
  ├──────┼──────────────────────┼─────────────────────────────────────────────┤
  │ #4   │ gotResults case      │ RETRY RELAY. Provider failed, DP#1 says     │
  │      │ (shouldRetry = true) │ retry with different provider.              │
  │      │                      │ {RelayState (possibly mutated with          │
  │      │                      │  archive), NumOfProviders: 1}               │
  ├──────┼──────────────────────┼─────────────────────────────────────────────┤
  │ #5   │ ticker.C case        │ HEDGE. Proactive parallel attempt while     │
  │      │ (shouldRetry = true) │ waiting for in-flight workers.              │
  │      │                      │ {RelayState, NumOfProviders: 1}             │
  ├──────┼──────────────────────┼─────────────────────────────────────────────┤
  │ #6   │ returnCondition case │ DONE (error). 15ms race guard confirmed     │
  │      │                      │ nothing else is coming.                     │
  │      │                      │ {Err: lastError, Done: true}                │
  ├──────┼──────────────────────┼─────────────────────────────────────────────┤
  │ #7   │ processingCtx.Done() │ DONE (timeout). Processing deadline         │
  │      │                      │ exceeded.                                   │
  │      │                      │ {Err: ctx.Err(), Done: true}                │
  ├──────┼──────────────────────┼─────────────────────────────────────────────┤
  │ #8   │ checkAndHandleTimeout│ DONE (timeout, SmartRouter only).           │
  │      │ (4 call sites)       │ Proactive timeout check before retry.       │
  │      │                      │ {Err: ctx.Err(), Done: true}                │
  └──────┴──────────────────────┴─────────────────────────────────────────────┘

  SENDS THAT CONTINUE: #1, #2, #4, #5 — send instructions, dispatch loop
    calls sendRelay(), workers launch, results flow back.

  SENDS THAT STOP: #3, #6, #7, #8 — send {Done: true}, dispatch loop
    sees task.Done, closes channel, returns. State machine returns.

  NOTE: DP#11 (circuit breaker trip) does NOT send to relayTaskChannel
    directly. It calls validateReturnCondition() which, after 15ms,
    sends to the returnCondition channel → triggers SEND#6.
```

---

## 3. Current Decision Points: Where They Live and Why

Decisions are currently scattered across all four goroutines. This section explains what each goroutine decides, why the decision ended up there, and what's wrong with this distribution.

### 3.1 State Machine (Goroutine A) — The Orchestrator

The state machine is the **central coordinator**. It runs a `select` loop that listens on five channels and decides what to do when events arrive. It owns the retry lifecycle — it's the only component that can send new instructions to the dispatch loop.

**What it knows**: Attempt number (batch number), selection mode (Stateless/Stateful/CrossValidation), whether the request is a batch.

**What it does NOT know**: How many errors accumulated across all workers, whether any of them were epoch mismatches, the error tolerance count. This information lives in the RelayProcessor (Goroutine C).

#### DP#1/DP#2 — `retryCondition()`: "Should I launch another retry attempt?"

**Where**: `gotResults` case and `ticker.C` case in the select loop.

**Why it's here**: The state machine owns the retry lifecycle. When a relay fails (`gotResults <- false`) or the hedge timer fires (`ticker.C`), the state machine is the component that decides whether to send a new instruction to the dispatch loop. No other goroutine can trigger a new relay attempt — only the state machine writes to `relayTaskChannel`.

**What it checks**:
- CrossValidation → NO (quorum mode, each provider gets one chance)
- Stateful → NO (all top providers already got the request)
- Batch + DisableBatchRetry → NO (batch requests too expensive to retry)
- Unsupported method errors detected → NO (permanent failure, Consumer only)
- Attempt number >= MaxRetries(10) → NO (hard ceiling)
- Otherwise → YES (try another provider)

**Problem**: It checks unsupported method errors by reaching into the RelayProcessor (`hasUnsupportedMethodErrorsInStateMachine()`), which is a cross-goroutine data access. It also doesn't know about error counts or epoch mismatch — those are checked separately by DP#3 in the RelayProcessor. This means a developer must read both DP#1 and DP#3 to understand "will this relay retry?", and the two can contradict each other (e.g., Stateful mode: DP#1 says "no retry", DP#3 says "keep waiting").

#### DP#11 — Circuit Breaker: "Are all providers exhausted?"

**Where**: `batchUpdate` case in the select loop.

**Why it's here**: When the dispatch loop calls `sendRelay()` and `GetSessions()` fails with `PairingListEmptyError`, the error arrives via the `batchUpdate` channel. The state machine tracks consecutive pairing failures. After 2 consecutive `PairingListEmptyError`, it trips the circuit breaker and stops retrying.

**What it checks**:
- PairingListEmpty + counter >= 2 → STOP (all providers exhausted)
- PairingListEmpty + counter < 2 → INCREMENT, continue
- Different error → RESET counter
- Success → RESET counter

**Note**: Currently SmartRouter-only (`EnableCircuitBreaker = true`). Consumer lacks this and falls through to DP#12, which catches all consecutive errors (not just PairingListEmpty) after 4 attempts. The circuit breaker's value is that it's **specific to PairingListEmpty** — it tracks the pairing exhaustion pattern separately, tripping after 2 consecutive pairing failures rather than 4 generic failures.

#### DP#12 — Batch Send Retry: "Retry the send operation or give up?"

**Where**: `batchUpdate` case in the select loop, after DP#11.

**Why it's here**: This handles **pre-relay** failures — the system couldn't even acquire sessions to start a relay. This is lifecycle coordination, not a retry policy decision. The distinction: relay retry = "provider returned bad result, try different provider". Send retry = "couldn't start the relay, try getting sessions again".

**What it checks**:
- consecutiveBatchErrors > SendRelayAttempts(3) → STOP
- Otherwise → send another instruction to dispatch loop

#### Archive Mutation — `UpgradeToArchiveIfNeeded()`: "Should I switch to/from archive?"

**Where**: Called from `stateTransition()` inside `shouldRetry()`, after `retryCondition()` returns true.

**Why it's here**: The archive upgrade happens during state transition — when the state machine prepares the next relay attempt. It modifies the protocol message (adds/removes the archive extension) based on the attempt number and node error count. This is a mutation of the relay instruction, which the state machine controls.

**What it does**:
- Attempt 1 (first retry) → Add archive extension (try archive provider)
- Attempt 2 (second retry) → Remove archive extension (back to regular)
- Archive tried + 2+ node errors → Cache request hashes as known-bad (6h TTL)

---

### 3.2 Result Aggregator (Goroutine C) — `readResultsFromProcessor()`

> **Historical note**: This section describes the **pre-refactor** behavior. After the refactor, `HasRequiredNodeResults()` no longer calls `shouldRetryRelay()`. It returns `false` unconditionally when there are no successes, and the state machine decides stop vs retry via `policy.Decide()`. See Section 6.1 for the current behavior.

The RelayProcessor runs in its own goroutine (`readResultsFromProcessor()`), calling `WaitForResults()` in a loop. It collects responses from all relay workers and determines when to signal the state machine via the `gotResults` channel.

**What it knows**: Aggregate results across all concurrent workers — how many successes, how many node errors, how many protocol errors, which errors are unsupported methods, which are epoch mismatches.

**What it does NOT know**: The attempt number, selection mode (it receives this at construction but doesn't track attempt progression), or the archive state.

#### DP#4 — `HasRequiredNodeResults()`: "Do I have enough results?"

**Where**: Inside `WaitForResults()` loop in Goroutine C.

**Why it's here**: The RelayProcessor is where all responses land (via `SetResponse()` called from relay workers). It's the only component with the aggregate view — how many successes, the CrossValidation quorum state, etc. Checking "do I have enough results?" requires this aggregate view.

**What it checks**:
- CrossValidation: quorum equal results >= agreement threshold → DONE
- Stateless/Stateful: resultsCount >= 1 → DONE
- No success → delegates to `shouldRetryRelay()` (DP#3)

**Problem**: When there are no successes, DP#4 delegates to DP#3, which makes a **retry decision**. This mixes two concerns in the same component: "do I have enough results?" (data question) and "should the system keep trying?" (policy question). The RelayProcessor should only answer the first question.

#### DP#3 — `shouldRetryRelay()`: "Should the system keep waiting for more results?"

**Where**: Called by `HasRequiredNodeResults()` when no successful results exist.

**Why it's here**: `shouldRetryRelay()` exists in the RelayProcessor because it needs information the state machine doesn't have — aggregate error counts and epoch mismatch detection. The state machine only knows the attempt number; the RelayProcessor knows *what happened* across all attempts.

**What it checks**:
- Stateful → KEEP WAITING (the outer timeout will stop it)
- CrossValidation → STOP (defensive guard)
- Unsupported method errors → STOP (**DUPLICATE of DP#1**)
- Batch + disabled → STOP (**DUPLICATE of DP#1**)
- Epoch mismatch + no results → KEEP WAITING (**UNIQUE** — not checked in DP#1)
- Total errors <= RelayRetryLimit(2) → KEEP WAITING (**UNIQUE** — not checked in DP#1)
- Default → STOP

**Problem**: This is the core of the "two-layer" issue. DP#3 and DP#1 both make retry decisions with different information, leading to:
- **Duplicated checks**: unsupported method and batch are checked in both DP#1 and DP#3
- **Hidden rules**: epoch mismatch and error tolerance exist ONLY in DP#3, invisible to anyone reading DP#1
- **Contradictions**: Stateful mode — DP#1 says "no retry" (don't launch new attempts), DP#3 says "keep waiting" (don't signal failure to the state machine). Both coexist because they control different things (launching vs waiting), but a developer reading either one alone gets an incomplete picture

#### DP#5 — `HasNonRetryableUserFacingErrors()`: "Do results contain non-retryable errors?"

**Where**: Called by DP#1 (via `hasUnsupportedMethodErrorsInStateMachine()` — name is historical, the function now calls `HasNonRetryableUserFacingErrors()` which covers all non-retryable cases).

**Why it's here**: The RelayProcessor has access to all stored results — it can scan node errors for the `IsNonRetryable` flag and protocol errors for error classification. This is fundamentally a **data query**, not a decision.

**Current implementation** (after error registry PR): Supersedes the old `HasUnsupportedMethodErrors()`. The check is based on `IsNonRetryable` — the umbrella flag set by the classifier in the worker. `IsUnsupportedMethod` is a strict subset of `IsNonRetryable` used only for the zero-CU carve-out and caching decisions, not for retry policy. (`IsUserError` has been removed — user-input errors charge normal CU.)

**How `IsNonRetryable` flows through the current architecture**:

```
Worker goroutine:
  ClassifyNodeErrorForRetry(family, transport, errorCode, message)
    → NodeErrorClassification{IsNonRetryable, IsUnsupportedMethod}
  localRelayResult.IsNonRetryable = classification.IsNonRetryable
  localRelayResult.IsUnsupportedMethod = classification.IsUnsupportedMethod  ← for zero-CU/caching
  SetResponse() → stored in RelayProcessor

RelayProcessor:
  HasNonRetryableUserFacingErrors():
    scans node errors for IsNonRetryable == true  ← THE RETRY GATE
    scans protocol errors for non-retryable types

State Machine (retryCondition):
  hasUnsupportedMethodErrorsInStateMachine()
    → calls relayProcessor.HasNonRetryableUserFacingErrors()
    → if true → return false (don't retry)
```

**Key point**: The retry decision uses `IsNonRetryable` (the umbrella), not `IsUnsupportedMethod`. If the error registry adds a new non-retryable subcategory (e.g., a chain-specific permanent failure), it will be caught by `IsNonRetryable` automatically — no code changes needed in the retry path.

**What it checks**:
- Any node error with `IsNonRetryable == true` → YES (covers unsupported method and any future non-retryable subcategory)
- Any protocol error matching `IsUnsupportedMethodError()` → YES
- Any protocol error that is epoch mismatch → SKIP (don't treat as non-retryable)
- Any protocol error that is non-retryable (via `IsNonRetryableUserFacingErrorType`) → YES
- None of the above → NO

---

### 3.3 Relay Workers (Goroutines D-N) — Per-Provider Execution

Each relay worker runs in its own goroutine, handles one provider, and is responsible for: connecting, sending the relay, receiving the response, classifying the error, updating provider eligibility, and storing the result in the RelayProcessor.

**What they know**: The raw response from one specific provider, including HTTP status codes, gRPC status, JSON-RPC error codes, and response body.

**What they do NOT know**: What other workers returned, the aggregate error picture, or whether the system should retry.

#### DP#6/DP#7/DP#8 — Error Classification: "What kind of error is this?"

**Where**: In the relay worker, immediately after receiving a provider response.

**Why it's here**: The worker has the **raw response data** — HTTP status codes, gRPC status codes, JSON-RPC error codes, response body. Classification requires understanding protocol-specific error formats. The classification must happen here because:

1. The result must be **labeled before storage** — the `IsUnsupportedMethod` flag is set on the `RelayResult` struct before `SetResponse()` stores it in the RelayProcessor. Later consumers read this flag rather than re-classifying.
2. **CU charging depends on classification** — the worker zeros `computeUnits` for unsupported methods before calling `OnSessionDone`.

**Current implementation** (after error registry PR): Classification uses the structured error registry in `protocol/common/`. Consumer and SmartRouter classify in different locations with different granularity:

| | Consumer | SmartRouter |
|---|---|---|
| **Where** | `rpcconsumer_server.go` worker goroutine | `direct_rpc_relay.go` (JSON-RPC, REST, gRPC handlers) |
| **Classification call** | `relaypolicy.ClassifyNodeError()` → wraps `common.ClassifyNodeErrorForRetry()` | `common.IsNonRetryableNodeErrorWithContext(family, transport, statusCode, errorMessage)` |
| **Sets `IsNonRetryable`** | Yes | Yes |
| **Sets `IsUnsupportedMethod`** | Yes | No |

**Consumer**: Uses `relaypolicy.ClassifyNodeError(chainID, apiInterface, statusCode, errorMessage, replyData)` which wraps `common.ClassifyNodeErrorForRetry()`. Returns two flags: `IsNonRetryable` (umbrella, read by the retry policy) and `IsUnsupportedMethod` (used for zero-CU carve-out and caching).

**SmartRouter**: Classification happens inside `direct_rpc_relay.go` where each transport handler (JSON-RPC, REST, gRPC) calls `CheckResponseError()` on the response, then `common.IsNonRetryableNodeErrorWithContext(family, transport, statusCode, errorMessage)` to set `IsNonRetryable`. The subcategory breakdown (`IsUnsupportedMethod`) is not computed — the SmartRouter only needs the umbrella flag for retry decisions. The `IsNonRetryable` flag is propagated up through `relayInnerDirect()` → `result.IsNonRetryable` → `relayResult.IsNonRetryable`.

**What they should NOT do**: Workers label the error; they don't decide whether the relay retries. The retry decision is made by `policy.Decide()` in the state machine via `HasNonRetryableNodeError` in the `ResultsSummary`.

#### DP#9 — `shouldRetryWithThisError()`: "Should this provider be excluded?"

**Where**: Called from `RemoveUsed()` in `used_providers.go`, after the response is stored. Now a method on `UsedProviders` (not a standalone function) with access to `chainID` for chain-aware classification.

**Why it's here**: Provider eligibility must be updated **immediately** — before the state machine sees the result. When the state machine wakes up on `gotResults` and decides to retry, it sends a new instruction to the dispatch loop, which calls `GetSessions()`. At that point, `UsedProviders` must already know which providers to exclude. If eligibility were decided in the state machine, there would be a race: the next hedge might select the same failing provider.

**What it checks** (chain-aware via `common.IsUnsupportedMethodError(chainID, 0, err.Error())`):
- Unsupported method → UNWANTED (no provider will succeed)
- Session sync loss, first time → ALLOW RETRY (one more chance with same provider)
- Session sync loss, second time → UNWANTED (second chance used up)
- Any other error → UNWANTED (try different provider)

**Relationship to policy**: `RemoveUsed()` calls `common.DecideEligibility()` directly with pre-computed boolean inputs. `relaypolicy.DecideEligibility()` re-exports the same function for API consistency. The old `shouldRetryWithThisError` method was removed.

---

### 3.4 Provider Side (Separate Process) — DP#13 and DP#10

These run on the **provider**, a completely separate process. They are independent of the consumer-side state machine.

#### DP#13 — `SendNodeMessage()`: "Should the provider retry to the upstream node?"

A synchronous `for` loop that retries sending to the blockchain node (up to 2 retries). Checks 7 conditions: gRPC status error, batch request, hash computation failure, success, unsupported method, hash in cache, retries exhausted.

#### DP#10 — `CheckHashInCache()`: "Is this request hash known to fail?"

Checks a shared cache (ristretto, 6h TTL) of request hashes that previously failed. If found, skip retrying. Used both provider-side (in DP#13) and consumer-side (in archive mutation).

---

## 4. The Problem: Why Decisions Are Scattered

The current distribution happened organically — each component got the decisions that seemed natural for its information scope. But this created three problems:

### 4.1 Duplicated Checks

The same condition is checked in multiple places because multiple components need to react to it:

| Check | Where It's Duplicated | Why |
|---|---|---|
| Unsupported method | DP#1 (state machine), DP#3 (relay processor), DP#5 (data query), DP#6/7 (worker classification), DP#9 (eligibility) — **6 places** | State machine needs it to stop retrying, relay processor needs it to stop waiting, worker needs it to classify and exclude provider |
| Batch + disabled | DP#1 (state machine), DP#3 (relay processor) — **2 places** | Both layers independently gate on this condition |
| CrossValidation no-retry | DP#1 (state machine), DP#3 (relay processor) — **2 places** | Both layers independently check selection mode |

### 4.2 Hidden Rules

Some rules exist in only one layer, invisible to anyone reading the other:

| Rule | Where It Lives | Hidden From |
|---|---|---|
| Epoch mismatch → always retry | DP#3 (relay processor only) | DP#1 — state machine has no epoch awareness |
| Error tolerance (RelayRetryLimit) | DP#3 (relay processor only) | DP#1 — state machine doesn't count errors |
| Provider eligibility (sync loss retry) | DP#9 (worker only) | DP#1, DP#3 — neither knows about per-provider eligibility |

### 4.3 Contradictions

The two retry layers can disagree:

| Scenario | DP#1 (State Machine) | DP#3 (Relay Processor) | Result |
|---|---|---|---|
| Stateful, no results | NO RETRY | KEEP WAITING | No new hedges launched, but processor keeps waiting. Timeout eventually stops it. Behavior is correct but confusing. |
| 3+ errors, over limit | RETRY (falls through) | STOP WAITING | DP#3 signals `gotResults <- false`, then DP#1 fires and may launch another attempt. DP#3's error tolerance is undermined. |

### 4.4 The Root Cause

**Decision-making is mixed with data collection.** The RelayProcessor should only collect results and report summaries. The workers should only classify errors and store results. But today, both make retry decisions — DP#3 decides "should we keep waiting?" and DP#9 decides "should this provider be excluded?" These decisions belong in the state machine, which is the retry lifecycle owner.

The state machine lacks the information to make these decisions today (error counts, epoch mismatch, error classification). The refactor fixes this by having the RelayProcessor and workers **report data** to the state machine, which then makes decisions via two policy entry points:
- `policy.Decide()` — post-relay retry decisions (provider responded, should we try another provider?)
- `policy.OnSendRelayResult(err, isPairingListEmpty)` — pre-relay retry decisions (couldn't get sessions, should we stop or retry the send?). Merges DP#11 (circuit breaker) and DP#12 (batch send retry) into one function. Returns `SendStop` / `SendRetry` / `SendSuccess`.

---

## 5. Refactored Architecture: Pure Go Policy Engine

### 5.1 Design Principle

**Separate data collection from decision-making.**

- **Workers**: Classify errors (retryable / non-retryable) and store classified results. No decisions.
- **RelayProcessor**: Accumulate results and provide a `ResultsSummary`. No decisions.
- **State Machine**: Read the summary, call `policy.Decide()` for post-relay decisions and `policy.OnSendRelayResult()` for pre-relay decisions (merges DP#11 + DP#12 into one function).

### 5.2 The Policy Engine — `relaypolicy.Policy`

A pure Go struct (no external libraries) with one main method:

```go
type Policy struct {
    config PolicyConfig  // selection mode, limits, flags (may differ between Consumer/SmartRouter)
}

func (p *Policy) Decide(input DecisionInput) DecisionOutput { ... }
```

**`DecisionInput`** — assembled by the state machine from two sources:

```go
type DecisionInput struct {
    // From state machine (already available)
    Selection      Selection   // Stateless / Stateful / CrossValidation
    AttemptNumber  int         // how many attempts launched so far
    IsBatch        bool        // is this a batch JSON-RPC request

    // From RelayProcessor (NEW — reported as data, not decisions)
    Summary        ResultsSummary

    // From archive status (already available)
    ArchiveStatus  *ArchiveStatus
    NodeErrors     uint64
}

type ResultsSummary struct {
    SuccessCount               int
    NodeErrors                 int
    SpecialNodeErrors          int
    ProtocolErrors             int
    HasNonRetryableNodeError   bool   // any node error with IsNonRetryable flag (umbrella — THE RETRY GATE)
    HasUnsupportedMethod       bool   // subset of non-retryable, used for zero-CU and caching decisions
    HasPermanentProtocolError  bool   // any non-retryable protocol error (excluding epoch mismatch)
    HasEpochMismatch           bool   // any protocol error is epoch mismatch
    HashErr                    error  // hash computation error (nil = hash OK)
}
```

**`DecisionOutput`** — tells the state machine exactly what to do:

```go
type DecisionOutput struct {
    Action      Action          // Retry / Stop
    Mutation    MutationOutput  // archive + cache side effects (can combine multiple)
    Reason      string          // "MaxRetriesReached", "UnsupportedMethod", "EpochMismatch", etc.
}

type MutationOutput struct {
    ArchiveAction  ArchiveAction  // AddArchive / RemoveArchive / NoChange
    CacheHashes    bool           // whether to cache request hashes as known-bad
}
```

This design supports combined mutations (e.g., `RemoveArchive + CacheHashes = true`) that a single enum cannot represent.

### 5.3 Error Classification — `relaypolicy.ClassifyNodeError()`

A pure Go function that wraps the structured error registry for worker-side classification:

```go
type ErrorClassification struct {
    IsNonRetryable      bool   // umbrella flag — THE RETRY GATE (policy reads this)
    IsUnsupportedMethod bool   // subset, used for zero-CU and caching decisions
}

func ClassifyNodeError(chainID string, apiInterface string, statusCode int, errorMessage string, replyData []byte) ErrorClassification { ... }
```

Wraps `common.ClassifyNodeErrorForRetry(family, transport, errorCode, errorMessage)` from the structured error registry. Resolves chain family and transport type from the endpoint config, and extracts JSON-RPC error codes from the response body for accurate code-based matching.

**Two flags, one retry gate**: `IsNonRetryable` is the umbrella — if the registry marks any error as non-retryable (unsupported method, Solana permanent, or any future subcategory), this flag is `true` and the policy stops retrying. `IsUnsupportedMethod` is a strict subset used only for the zero-CU carve-out and caching (cache unsupported responses). `IsUserError` has been removed — user-input errors charge normal CU since responses are not cached.

Called from the consumer worker in `rpcconsumer_server.go` after `CheckResponseError()` detects a node error. Sets all three flags on the `RelayResult` before storage.

**Protocol error classification** is not done in the worker — it happens in `GetResultsSummary()` which scans stored protocol errors using `chainlib.ShouldRetryError()` and `chainlib.IsUnsupportedMethodError()`.

**SmartRouter note**: The SmartRouter worker does not call `ClassifyNodeError()`. On `main`, the SmartRouter never sets `IsUnsupportedMethod` — it gets `IsNodeError` from the provider response but does not classify further. This preserves current behavior.

### 5.4 Provider Eligibility — `relaypolicy.DecideEligibility()`

A pure Go function that consolidates DP#9:

```go
type EligibilityResult struct {
    Action  EligibilityAction  // EligibilityMarkUnwanted / EligibilityAllowRetry
}

func DecideEligibility(isUnsupportedMethod bool, isSyncLoss bool, isFirstSyncLoss bool) EligibilityResult { ... }
```

**Status**: Defined in `common/eligibility.go`. Re-exported by `relaypolicy/eligibility.go` for API consistency. Called directly from `used_providers.go:RemoveUsed()` which computes the boolean inputs inline (`isUnsupportedMethod` via `common.IsUnsupportedMethodError(chainID, 0, err.Error())`, `isSyncLoss` via `IsSessionSyncLoss(err)`). The old `shouldRetryWithThisError` method was removed — its logic is now inline in `RemoveUsed()`.

The eligibility **execution** (updating `UsedProviders`) stays in the worker via `OnSessionFailure()` → `Free()` → `RemoveUsed()` for timing reasons — the next hedge must already see the updated exclusion list.

### 5.5 Refactored Flow Diagram

```
 Consumer/SmartRouter HTTP Request Arrives
              │
              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ RPCConsumerServer.SendRelay() / RPCSmartRouterServer.SendRelay()            │
│                                                                             │
│  1. Parse request → ProtocolMessage                                         │
│  2. Create policy := relaypolicy.NewPolicy(config)              ← NEW      │
│  3. Create RelayProcessor (SIMPLIFIED — pure data collector)                │
│  4. Create UsedProviders                                                    │
│  5. Create UnifiedRelayStateMachine(policy)                                │
│  6. relayTaskChannel := sm.GetRelayTaskChannel()                            │
│  7. Dispatch loop (unchanged)                                               │
│                                                                             │
└──────────────────┬──────────────────────────────────────────┬───────────────┘
                   │                                          │
         Goroutine A                                Goroutine B
         (STATE MACHINE)                            (DISPATCH — unchanged)
                   │                                          │
                   ▼                                          ▼
┌──────────────────────────────────────┐  ┌──────────────────────────────────────┐
│ GetRelayTaskChannel()                 │  │ Dispatch Loop (UNCHANGED)             │
│                                       │  │                                       │
│ relayTaskCh <- instructions ─────────┼─▶│ for task := range relayTaskChannel {  │
│                                       │  │   sendRelay(task)                     │
│ go readResultsFromProcessor()         │  │   sm.UpdateBatch(err)                 │
│                                       │  │ }                                     │
│ for {                                 │  └──────────────────────────────────────┘
│   select {                            │
│   ┌─────────────────────────────────┐ │
│   │ case err := <-batchUpdate:      │ │         Goroutine C
│   │                                 │ │         readResultsFromProcessor()
│   │ ►result :=                       │ │         ┌────────────────────────────────┐
│   │  policy.OnSendRelayResult(err, isPairingListEmpty)  │ │         │ relayProcessor.WaitForResults   │
│   │  REPLACES DP#11 + DP#12.        │ │         │                                │
│   │  One function, two counters:    │ │         │ ►HasRequiredNodeResults()       │
│   │   - PairingListEmpty (thresh 2) │ │         │  SIMPLIFIED — pure data check: │
│   │   - All batch errors (thresh 3) │ │         │  "do I have ≥1 success?"       │
│   │  Returns: SendStop / SendRetry  │ │         │  "does CV quorum agree?"       │
│   │           / SendSuccess         │ │         │                                │
│   │                                 │ │         │  NO shouldRetryRelay() call    │
│   └─────────────────────────────────┘ │         │  NO retry decision             │
│   ┌─────────────────────────────────┐ │         │  NO unsupported method check   │
│   │ case success := <-gotResults:   │◀┼─────────│                                │
│   │                                 │ │         │  If success → gotResults<-true │
│   │   if success → DONE             │ │         │  If no success:                │
│   │   if !success:                  │ │         │   Build ResultsSummary (data)   │
│   │                                 │ │         │   gotResults <- false           │
│   │ ►summary := rp.GetResultsSummary()     │ │         └────────────────────────────────┘
│   │  (pure data — no decisions)     │ │
│   │                                 │ │         Goroutines D-N (Relay Workers)
│   │ ►output := policy.Decide(input) │ │         ┌────────────────────────────────┐
│   │  ALL POST-RELAY RULES.          │ │         │ Per-provider goroutines         │
│   │  Pure Go. No external libs.     │ │         │                                │
│   │                                 │ │         │ 1. Connect to provider         │
│   │  REPLACES:                      │ │         │ 2. Send relay request          │
│   │    DP#1/#2 retryCondition       │ │         │ 3. Receive response            │
│   │    DP#3   shouldRetryRelay      │ │         │                                │
│   │    DP#5   unsupported check     │ │         │ ►ClassifyError(err)            │
│   │    (DP#10 stays provider-side)  │ │         │  Pure CLASSIFICATION only.     │
│   │    + archive mutation           │ │         │  Returns:                      │
│   │                                 │ │         │   {IsUnsupported,              │
│   │  RETURNS DecisionOutput:        │ │         │    IsSolanaNonRetryable,       │
│   │    Action: Retry / Stop         │ │         │    IsRetryable}                │
│   │    Mutation: {ArchiveAction,    │ │         │  NO retry decision.            │
│   │              CacheHashes}      │ │
│   │    Reason: "EpochMismatch"      │ │         │  Sets flags on RelayResult.    │
│   │                                 │ │         │                                │
│   │  if Retry:                      │ │         │ Eligibility:                   │
│   │    apply Mutation               │ │         │  common.DecideEligibility()    │
│   │    stateTransition()            │ │         │  called inline from            │
│   │    send instruction             │ │         │  RemoveUsed() in               │
│   │  if Stop:                       │ │         │  used_providers.go             │
│   │    done                         │ │         │                                │
│   │                                 │ │         │                                │
│   └─────────────────────────────────┘ │         │ 4. relayProcessor              │
│   ┌─────────────────────────────────┐ │         │      .SetResponse(resp) ──────┼──▶ C
│   │ case <-ticker.C:                │ │         │                                │
│   │                                 │ │         │ 5. OnSessionFailure()          │
│   │  (hedge budget: future work)     │ │         │    (session mgmt — untouched)  │
│   │                                 │ │         └────────────────────────────────┘
│   │                                 │ │
│   │                                 │ │  PROVIDER SIDE (separate process):
│   │ ►output := policy.Decide(input) │ │  ┌────────────────────────────────────────┐
│   │  (same as gotResults case)      │ │  │ ►DP#13 SendNodeMessage()               │
│   │                                 │ │  │  Pure Go retry loop (SIMPLIFIED).      │
│   └─────────────────────────────────┘ │  │  Same 7 rules, cleaner structure.      │
│   ┌─────────────────────────────────┐ │  │  ►DP#10 CheckHashInCache              │
│   │ case <-returnCondition:         │ │  │  (unchanged — ristretto, 6h TTL)       │
│   │   unchanged                     │ │  └────────────────────────────────────────┘
│   └─────────────────────────────────┘ │
│   ┌─────────────────────────────────┐ │
│   │ case <-processingCtx.Done():    │ │
│   │   unchanged                     │ │
│   └─────────────────────────────────┘ │
│   }                                   │
│ }                                     │
└───────────────────────────────────────┘
```

---

## 6. What Changed: Component by Component

### 6.1 RelayProcessor — From Decision-Maker to Data Collector

| Aspect | Before | After |
|---|---|---|
| `HasRequiredNodeResults()` (DP#4) | Checks for success, then delegates to `shouldRetryRelay()` for retry decision | Checks for success ONLY. If no success, signals `gotResults <- false` with no retry opinion. |
| `shouldRetryRelay()` (DP#3) | Makes retry decisions: checks unsupported method, batch, epoch mismatch, error tolerance | **ELIMINATED.** Logic absorbed into `policy.Decide()`. |
| `HasUnsupportedMethodErrors()` (DP#5) | Scans results and makes a "is this unsupported?" determination, called from DP#1 and DP#3. Also catches non-retryable protocol errors. | Superseded by `HasNonRetryableUserFacingErrors()` which checks the `IsNonRetryable` umbrella flag on node errors. For the policy, data is exposed via `ResultsSummary.HasNonRetryableNodeError` (the retry gate) and `HasPermanentProtocolError`. `HasUnsupportedMethod` remains for zero-CU/caching only. |
| New: `GetResultsSummary()` | Does not exist | Returns `ResultsSummary` struct with counts and flags. Pure data aggregation. |
| Role | Result collector + retry decision-maker | Result collector + data reporter |

**Key change**: `HasRequiredNodeResults()` no longer calls `shouldRetryRelay()`. When there are no successful results:
- **Before**: RelayProcessor decides whether to keep waiting (DP#3) and signals `gotResults <- !shouldRetry`
- **After**: RelayProcessor signals `gotResults <- false` unconditionally. The state machine calls `policy.Decide()` to determine whether to retry.

### 6.2 Relay Workers — From Classifier+Decider to Classifier Only

| Aspect | Before | After |
|---|---|---|
| Error classification (DP#6/7/8) | Ad-hoc pattern matching: `IsUnsupportedMethodMessage()`, REST 404/405 check, `IsSolanaNonRetryableError()` | `ClassifyNodeError(chainID, statusCode, errorMessage)` — uses structured error registry with chain-aware two-tier classification. Consumer wired. SmartRouter unchanged (does not classify). |
| User error detection | Not detected | Removed — user-input errors (Layer D) are `Retryable=false` in the registry (caught by `IsNonRetryable`) but charge normal CU (no zero-CU carve-out). |
| Provider eligibility (DP#9) | `shouldRetryWithThisError()` (standalone function, no chain awareness) | `shouldRetryWithThisError()` removed. `RemoveUsed()` computes eligibility inline and calls `common.DecideEligibility()` directly. `relaypolicy.DecideEligibility` re-exports the same function. |
| Role | Classify errors + decide provider eligibility | Classify errors via error registry (consumer) + eligibility chain-aware via `shouldRetryWithThisError` method |

**Key change**: Workers no longer have functions named `ShouldRetry*`. They classify and label. The "should" questions are answered by `policy.Decide()` in the state machine.

**Why eligibility execution stays in the worker**: The `UsedProviders` exclusion list must be updated immediately when a response arrives — before the state machine wakes up and launches the next hedge. If the worker just stored the classification and the state machine updated eligibility later, there would be a race window where the next hedge could select the same failing provider. Currently this still runs via `shouldRetryWithThisError()` in `used_providers.go`. When `DecideEligibility()` is wired, it will be called from the worker (policy logic) and the result immediately applied to `UsedProviders` (execution).

### 6.3 State Machine — Single Decision Point

| Aspect | Before | After |
|---|---|---|
| `gotResults` case | Calls `shouldRetry()` → `retryCondition()` (DP#1). Checks selection mode, batch, unsupported (via cross-goroutine call to RelayProcessor), max retries. | Reads `ResultsSummary` from RelayProcessor. Calls `policy.Decide(input)` — one function, all rules. |
| `ticker.C` case | Same `shouldRetry()` call. No hedge budget. | `policy.Decide(input)` with `IsTickerHedge=true`. Hedge budget is future work. |
| `batchUpdate` case | DP#11 circuit breaker (manual counter) + DP#12 batch retry (separate logic, separate counters) | `policy.OnSendRelayResult(err, isPairingListEmpty)` — one function merging DP#11 + DP#12. Tracks both counters internally, returns `SendStop` / `SendRetry` / `SendSuccess` |
| Archive mutation | Hidden inside `stateTransition()` → `UpgradeToArchiveIfNeeded()` | Returned as `MutationOutput{ArchiveAction, CacheHashes}` in `DecisionOutput`. Supports combined side effects (e.g., RemoveArchive + CacheHashes). State machine applies it explicitly. |
| Role | Partial decision-maker (depends on DP#3 for error counts, DP#5 for unsupported) | **Central decision authority** — post-relay rules in `policy.Decide()`, pre-relay rules in `policy.OnSendRelayResult()` (DP#11 + DP#12 merged) |

---

## 7. All Rules in `policy.Decide()` — Evaluation Order

```go
func (p *Policy) Decide(input DecisionInput) DecisionOutput {
    // 1. MODE CHECKS — does the selection mode allow retries?
    if input.Selection == CrossValidation → Stop("CrossValidation")
    if input.Selection == Stateful        → Stop("Stateful")

    // 2. PERMANENT FAILURE CHECKS — non-retryable errors
    //    Uses IsNonRetryable as the umbrella flag (same as main's
    //    HasNonRetryableUserFacingErrors). This is the single retry gate —
    //    if the error registry adds a new non-retryable subcategory,
    //    it is caught here automatically without policy changes.
    if input.Summary.HasNonRetryableNodeError   → Stop("NonRetryableNodeError")
    if input.Summary.HasPermanentProtocolError  → Stop("PermanentProtocolError")

    // 3. LIMIT CHECKS — have we exceeded retry limits?
    if input.AttemptNumber >= p.config.MaxRetries → Stop("MaxRetriesReached")
    if input.IsBatch && p.config.DisableBatchRetry → Stop("BatchDisabled")

    // 4. RESULT CHECKS — what do the aggregate results tell us?
    if input.Summary.HasEpochMismatch && input.Summary.SuccessCount == 0
        → Retry("EpochMismatch")  // always retry, bypasses error tolerance and hash check

    // 5. HASH ERROR CHECK — if hash computation failed, do not take the normal retry path.
    //    Current behavior: shouldRetryRelay() only retries when hashErr == nil.
    //    Without this check, hash computation failures would silently fall through
    //    to the default retry, changing behavior.
    if input.Summary.HashErr != nil → Stop("HashComputationFailed")

    totalErrors := input.Summary.NodeErrors + input.Summary.SpecialNodeErrors + input.Summary.ProtocolErrors
    if totalErrors > p.config.RelayRetryLimit → Stop("ErrorToleranceExceeded")

    // 6. ARCHIVE MUTATION
    mutation := p.decideMutation(input.AttemptNumber, input.ArchiveStatus, input.NodeErrors)

    // 7. DEFAULT: RETRY
    return DecisionOutput{Action: Retry, Mutation: mutation, Reason: "Default"}

    // NOTE: HashInCache (DP#10) is NOT checked here. In the current system,
    // CheckHashInCache() is enforced in the provider-side upstream retry loop
    // (DP#13 in SendNodeMessage()), not in the consumer-side state machine.
    // Adding it here would be a behavior change. The hash cache remains
    // provider-side only.
}
```

**All post-relay retry rules in one place.** No duplicates. No hidden rules in other goroutines. A developer reads this one function to understand all retry behavior for the `gotResults` and `ticker.C` cases.

Pre-relay retry decisions (the `batchUpdate` case) are handled by a separate function:

`policy.OnSendRelayResult(err, isPairingListEmpty)` merges DP#11 (circuit breaker) and DP#12 (batch send retry) into one function. It tracks two counters internally:
- Consecutive `PairingListEmptyError` count (trips at threshold 2)
- Consecutive batch error count of any type (trips at threshold 3)

It returns one of three outcomes:
- `SendSuccess` — send succeeded, reset counters
- `SendRetry` — send failed but under thresholds, retry getting sessions
- `SendStop` — send failed and a threshold was hit, stop retrying

This is separate from `policy.Decide()` because it handles a fundamentally different situation: the relay never reached a provider. There are no results to evaluate, no error classification, no archive mutation — just "the send operation itself failed."

The state machine's `batchUpdate` case becomes:
```go
case err := <-batchUpdate:
    switch policy.OnSendRelayResult(err, isPairingListEmpty) {
    case SendSuccess:
        // continue to select loop
    case SendStop:
        go validateReturnCondition(err)
    case SendRetry:
        relayTaskChannel <- instructions
    }
```

---

## 8. What Does NOT Change

- **Select loop** (5-case) — stays
- **Dispatch loop** (`for task := range relayTaskChannel`) — stays
- **All channels** (batchUpdate, gotResults, ticker, returnCondition, processingCtx) — stay
- **Provider selection** (WRS optimizer, UsedProviders) — untouched
- **Session management** (OnSessionFailure, blocking, reporting) — untouched
- **CrossValidation quorum consensus** — untouched
- **Hash cache** (ristretto, 6h TTL) — untouched; `CheckHashInCache()` remains provider-side only (DP#10 in `SendNodeMessage()`)
- **DP#12** (batch send retry) — merged with DP#11 into `policy.OnSendRelayResult()`
- **DP#13** (provider-side retry to upstream node) — stays, refactored to cleaner pure Go loop
- **Worker timing** — classification and eligibility execution stay in worker goroutines

---

## 9. Duplicate Eliminations

| Duplicate | Where It Exists Today | After Refactor |
|---|---|---|
| Unsupported method | DP#1 + DP#3 + DP#5 + DP#6/7 + DP#9 — **6 places** | `ClassifyNodeError()` sets flag once in consumer worker. `policy.Decide()` reads it once via `ResultsSummary`. SmartRouter + eligibility wiring is follow-up. |
| Batch + disabled | DP#1 + DP#3 — **2 places** | `policy.Decide()` checks once. |
| CrossValidation no-retry | DP#1 + DP#3 — **2 places** | `policy.Decide()` checks once. |
| Stateful handling | DP#1 (no retry) + DP#3 (keep waiting) — **contradiction** | `policy.Decide()` returns one explicit behavior. |

---

## 10. Consumer vs SmartRouter Configuration

The policy engine is shared, but Consumer and SmartRouter may use different configurations:

```go
type PolicyConfig struct {
    MaxRetries              int       // hard ceiling on retry attempts
    RelayRetryLimit         int       // error tolerance (total errors before giving up)
    DisableBatchRetry       bool      // whether batch requests can be retried
    EnableCircuitBreaker    bool      // provider exhaustion detection
    CircuitBreakerThreshold int       // consecutive pairing errors before tripping
    EnableTimeoutPriority   bool      // proactive timeout checking before retries
}
```

Whether each flag is the same or different for Consumer vs SmartRouter is a **product decision**, not an architectural constraint. The unified engine supports any combination without code duplication. See doc 13 (Product Spec), Part G for the current differences and options.

---

## 11. Migration from Current to Refactored

| Phase | What | Risk | Depends On |
|---|---|---|---|
| **Phase 1** | Unify Consumer + SmartRouter state machines | Low | **DONE** |
| **Phase 2** | Create `relaypolicy.Policy` with `Decide()`, `OnSendRelayResult()`, `ClassifyNodeError()`. Wire `ClassifyNodeError()` into consumer worker. `DecideEligibility()` wired via `common.DecideEligibility()` in `RemoveUsed()`. SmartRouter worker wiring is follow-up. | Medium (see note) | Phase 1 |
| **Phase 3** | Simplify `HasRequiredNodeResults()` — remove `shouldRetryRelay()` call | Low | Phase 2 |
| **Phase 4** | Refactor provider-side `SendNodeMessage()` to cleaner pure Go loop | Low | Independent |

**Phase 2 risk note**: The current two-layer retry logic has contradictions (see Section 4.3). Consolidating into one `policy.Decide()` forces **explicit resolution** of these contradictions. Each case must be reviewed and a behavior chosen. This is a policy decision, not just code movement. Phase 2 should include behavioral testing: compare relay success rates before and after.

---

## 12. E2E Test Scenarios

These scenarios trace the complete decision flow for specific provider configurations. Use them to verify that the refactored system produces identical behavior to the current system.

### 12.1 Scenario: 2 Providers, Both Return Rate Limit Error (After Refactor)

**Setup**: Providers A and B. Stateless mode. Single (non-batch) request. Rate limit error is a node error (HTTP 200, JSON-RPC `{"error": {"code": -32005, "message": "rate limit exceeded"}}`).

**Error classification**: Rate limit is **retryable** — doesn't match unsupported method patterns or Solana permanent error codes.

**Key config**: `MaxRetries=10`, `RelayRetryLimit=2`, `DisableBatchRetry=true` (irrelevant — not a batch).

```
USER SENDS: curl http://consumer:port -d '{"method":"eth_getBalance",...}'
│
▼
═══════════════════════════════════════════════════════════════════════════
 ATTEMPT 0 (initial)
═══════════════════════════════════════════════════════════════════════════
│
├─ SEND#1: relayTaskCh <- {NumOfProviders: 1}
├─ Dispatch: sendRelay() → GetSessions()
│    WRS selects Provider A
│    AddUsed(A) → sessionsLatestBatch=1, batchNumber=1
│    batchUpdate <- nil (success)
│
├─ State machine: policy.OnSendRelayResult(nil) → SendSuccess
│    Reset counters, continue
│
├─ Worker A: send to Provider A
│    ← Response: HTTP 200, body: {"error":{"code":-32005,"message":"rate limit exceeded"}}
│    CheckResponseError() → foundError=true → NODE ERROR
│
│    relaypolicy.ClassifyNodeError("rate limit exceeded", 200, "jsonrpc"):
│      IsUnsupported? → NO (no pattern match)
│      IsSolanaNonRetryable? → NO
│      → {IsUnsupported: false, IsSolanaNonRetryable: false, IsRetryable: true}
│
│    common.DecideEligibility(isUnsupportedMethod=false, isSyncLoss=false, isFirstSyncLoss=false):
│      → {Action: MarkUnwanted}
│    RemoveUsed() marks A as UNWANTED
│
│    SetResponse() → rp.responses <- {nodeError, providerA}
│
├─ Goroutine C: WaitForResults()
│    receives response, responsesCount=1
│    handleResponse() → stored in nodeResponseErrors
│    checkEndProcessing(1):
│      responsesCount(1) >= SessionsLatestBatch(1)? → YES
│    WaitForResults() returns
│
├─ Goroutine C: HasRequiredNodeResults() — SIMPLIFIED, no retry decision
│    resultsCount(success) = 0 → gotResults <- false
│    (no shouldRetryRelay call — just reports "no success")
│
├─ State machine: gotResults case, success=false
│    summary := relayProcessor.GetResultsSummary()
│      → {SuccessCount:0, NodeErrors:1, ProtocolErrors:0,
│         HasUnsupported:false, HasEpochMismatch:false}
│
│    output := policy.Decide(DecisionInput{
│      Selection:     Stateless,
│      AttemptNumber: 1,
│      IsBatch:       false,
│      Summary:       summary,   // totalErrors=1
│      ArchiveStatus: {isArchive:false, isUpgraded:false},
│      NodeErrors:    1,
│    })
│
│    policy.Decide evaluation:
│      1. MODE: Stateless → continue
│      2. PERMANENT: HasUnsupported=false → continue
│      3. LIMITS: AttemptNumber(1) < MaxRetries(10) → continue
│      4. RESULTS: totalErrors(1) <= RelayRetryLimit(2) → continue
│      5. ARCHIVE: attempt 1, not yet archive
│         → Mutation: AddArchive
│      6. DEFAULT: → Action: Retry
│
│    output = {Action: Retry, Mutation: {AddArchive, CacheHashes:false}, Reason: "Default"}
│
│    State machine applies mutation: add archive extension
│    SEND#4: relayTaskCh <- {RelayState (WITH archive), NumOfProviders: 1}
│    go readResultsFromProcessor() ← new Goroutine C-2
│
═══════════════════════════════════════════════════════════════════════════
 ATTEMPT 1 (first retry, with archive extension)
═══════════════════════════════════════════════════════════════════════════
│
├─ Dispatch: sendRelay() → GetSessions()
│    Provider A: UNWANTED (excluded)
│    WRS selects Provider B (only remaining)
│    batchUpdate <- nil (success)
│
├─ State machine: policy.OnSendRelayResult(nil) → SendSuccess
│
├─ Worker B: send to Provider B (with archive extension)
│    ← Response: rate limit exceeded (same error)
│    relaypolicy.ClassifyNodeError → {IsRetryable: true}
│    common.DecideEligibility(isUnsupportedMethod=false, ...) → {Action: MarkUnwanted}
│    RemoveUsed() marks B as UNWANTED
│    SetResponse() → rp.responses <- {nodeError, providerB}
│
├─ Goroutine C-2: WaitForResults() → checkEndProcessing → done
│    HasRequiredNodeResults(): success=0 → gotResults <- false
│
├─ State machine: gotResults case, success=false
│    summary: {SuccessCount:0, NodeErrors:2, HasUnsupported:false}
│
│    output := policy.Decide(DecisionInput{
│      AttemptNumber: 2,
│      Summary:       {NodeErrors:2},  // totalErrors=2
│      ArchiveStatus: {isArchive:true, isUpgraded:true},
│      NodeErrors:    2,
│    })
│
│    policy.Decide evaluation:
│      1. MODE: Stateless → continue
│      2. PERMANENT: HasUnsupported=false → continue
│      3. LIMITS: AttemptNumber(2) < MaxRetries(10) → continue
│      ┌─────────────────────────────────────────────────────────────┐
│      │ 4. RESULTS: totalErrors(2) <= RelayRetryLimit(2)           │
│      │    → YES (2 ≤ 2) → continue                                │
│      │                                                             │
│      │    KEY: This is the LAST attempt within error tolerance.    │
│      │    Next error pushes totalErrors to 3 > 2 → Stop.          │
│      └─────────────────────────────────────────────────────────────┘
│      5. ARCHIVE: attempt 2, isUpgraded=true
│         → ArchiveAction: RemoveArchive
│         Also: isUpgraded=true AND nodeErrors(2) >= 2
│         → CacheHashes: true (6h TTL)
│      6. DEFAULT: → Action: Retry
│
│    output = {Action: Retry, Mutation: {RemoveArchive, CacheHashes:true}, Reason: "Default"}
│
│    State machine: remove archive, cache hashes
│    SEND#4: relayTaskCh <- {RelayState (WITHOUT archive), NumOfProviders: 1}
│    go readResultsFromProcessor() ← Goroutine C-3
│
═══════════════════════════════════════════════════════════════════════════
 ATTEMPT 2 (second retry — PROVIDERS EXHAUSTED)
═══════════════════════════════════════════════════════════════════════════
│
├─ Dispatch: sendRelay() → GetSessions()
│    Provider A: UNWANTED, Provider B: UNWANTED
│    No providers available → PairingListEmptyError
│    ┌─────────────────────────────────────────────────────────────────┐
│    │ NOTE: ClearUnwanted is NOT called in the main dispatch path.   │
│    │ It's only called in the init relay path.                       │
│    │ The unwanted list stays as-is.                                 │
│    └─────────────────────────────────────────────────────────────────┘
│    sendRelay returns PairingListEmptyError
│    UpdateBatch(PairingListEmptyError) → batchUpdate channel
│
├─ State machine: batchUpdate case, err=PairingListEmptyError
│
│    result := policy.OnSendRelayResult(PairingListEmptyError)
│      consecutiveBatchErrors: 1
│      PairingListEmpty → consecutivePairingErrors: 1
│      pairingErrors(1) >= threshold(2)? → NO
│      batchErrors(1) > sendAttempts(3)? → NO
│      → SendRetry
│
│    SEND#2: relayTaskCh <- instructions (retry getting sessions)
│
═══════════════════════════════════════════════════════════════════════════
 ATTEMPT 2 RETRY SEND (second try — still exhausted)
═══════════════════════════════════════════════════════════════════════════
│
├─ Dispatch: → PairingListEmptyError (A,B still unwanted)
│
├─ State machine: batchUpdate, PairingListEmptyError
│
│    result := policy.OnSendRelayResult(PairingListEmptyError)
│      consecutiveBatchErrors: 2
│      consecutivePairingErrors: 2
│      ┌──────────────────────────────────────────────────────────────┐
│      │ pairingErrors(2) >= threshold(2)? → YES! STOP               │
│      │ → SendStop                                                   │
│      └──────────────────────────────────────────────────────────────┘
│
│    validateReturnCondition(err) → 15ms → returnCondition channel
│    SEND#6: relayTaskCh <- {Err: PairingListEmptyError, Done: true}
│
│    ════════════════════════════════════════════════════════════════════
│     STATE MACHINE EXITS with PairingListEmptyError
│     Total: 2 relay attempts (A, B) + 2 session retries
│    ════════════════════════════════════════════════════════════════════
│
═══════════════════════════════════════════════════════════════════════════
 Goroutine C-3 cleanup
═══════════════════════════════════════════════════════════════════════════

  Goroutine C-3 is blocked on WaitForResults() → rp.responses.
  No worker was ever launched for Attempt 2 (GetSessions failed).
  Cleaned up when processingCtx is cancelled (state machine exit
  triggers processingCtxCancel()).

═══════════════════════════════════════════════════════════════════════════
 USER RESPONSE
═══════════════════════════════════════════════════════════════════════════

  HTTP response to curl with Lava-Retry-Debug header (see Section 13):

  {
    "error": {"code": ..., "message": "No pairings available"}
  }

  Lava-Retry-Debug: {
    "attempts": 2,
    "providers_tried": ["providerA", "providerB"],
    "session_retries": 2,
    "worker_results": [
      {
        "attempt_id": 0,
        "batch_number": 1,
        "provider": "providerA",
        "error_type": "node_error",
        "error_message": "rate limit exceeded",
        "error_class": {"is_unsupported_method": false, "is_solana_non_retryable": false, "is_retryable": true},
        "eligibility": "unwanted"
      },
      {
        "attempt_id": 1,
        "batch_number": 2,
        "provider": "providerB",
        "error_type": "node_error",
        "error_message": "rate limit exceeded",
        "error_class": {"is_unsupported_method": false, "is_solana_non_retryable": false, "is_retryable": true},
        "eligibility": "unwanted"
      }
    ],
    "policy_decisions": [
      {
        "attempt_id": 0,
        "action": "retry",
        "mutation": {"archive_action": "add_archive", "cache_hashes": false},
        "reason": "Default"
      },
      {
        "attempt_id": 1,
        "action": "retry",
        "mutation": {"archive_action": "remove_archive", "cache_hashes": true},
        "reason": "Default"
      }
    ],
    "send_failures": [
      {"send_attempt_id": 0, "batch_number": 3, "error": "PairingListEmpty", "result": "retry", "pairing_errors": 1, "batch_errors": 1},
      {"send_attempt_id": 1, "batch_number": 3, "error": "PairingListEmpty", "result": "stop_pairing_exhausted", "pairing_errors": 2, "batch_errors": 2}
    ],
    "final_reason": "PairingListEmptyError",
    "archive_tried": true,
    "hashes_cached": true,
    "total_errors": {"node": 2, "protocol": 0, "special": 0}
  }
```

### 12.1.1 Decision Tree Summary

```
                    User Request
                        │
                        ▼
                   Provider A
                   rate limit
                        │
         ┌──────────────┴──────────────┐
         │ ClassifyError:              │
         │   {IsRetryable: true}       │
         │ DecideEligibility:          │
         │   {MarkUnwanted}            │
         │ policy.Decide:              │
         │   totalErrors(1) ≤ 2        │
         │   attempt(1) < 10           │
         │   → Retry, AddArchive       │
         └──────────────┬──────────────┘
                        ▼
                   Provider B
                  (with archive)
                   rate limit
                        │
         ┌──────────────┴──────────────┐
         │ ClassifyError:              │
         │   {IsRetryable: true}       │
         │ DecideEligibility:          │
         │   {MarkUnwanted}            │
         │ policy.Decide:              │
         │   totalErrors(2) ≤ 2        │
         │   attempt(2) < 10           │
         │   → Retry, RemoveArchive    │
         │     + CacheHashes           │
         └──────────────┬──────────────┘
                        ▼
                  GetSessions()
                  A,B both UNWANTED
                  PairingListEmptyError
                        │
      OnSendRelayResult(PairingListEmpty)
         1st: pairingErrors=1 → SendRetry
         2nd: pairingErrors=2 → SendStop
                        │
                        ▼
              Return to user:
              PairingListEmptyError
              "No pairings available"
```

### 12.1.2 Test Assertions

| Assertion | Expected Value | Verify Via |
|---|---|---|
| Provider A receives exactly 1 relay | 1 | `Lava-Retry-Debug.worker_results[0].provider` |
| Provider B receives exactly 1 relay | 1 | `Lava-Retry-Debug.worker_results[1].provider` |
| Total relay attempts to providers | 2 | `Lava-Retry-Debug.attempts` |
| Both classified as retryable | true | `Lava-Retry-Debug.worker_results[*].error_class.is_retryable` |
| Neither classified as unsupported | false | `Lava-Retry-Debug.worker_results[*].error_class.is_unsupported_method` |
| Both providers marked unwanted | true | `Lava-Retry-Debug.worker_results[*].eligibility == "unwanted"` |
| Worker and policy records correlate by attempt_id | match | `worker_results[0].attempt_id == policy_decisions[0].attempt_id` |
| Archive added on attempt 0 | true | `Lava-Retry-Debug.policy_decisions[0].mutation.archive_action == "add_archive"` |
| Archive removed on attempt 1 | true | `Lava-Retry-Debug.policy_decisions[1].mutation.archive_action == "remove_archive"` |
| Hashes cached on attempt 1 | true | `Lava-Retry-Debug.policy_decisions[1].mutation.cache_hashes == true` |
| `policy.Decide` returned Retry for both | true | `Lava-Retry-Debug.policy_decisions[*].action == "retry"` |
| GetSessions failures | 2 | `Lava-Retry-Debug.send_failures` length |
| `OnSendRelayResult` stopped on 2nd pairing error | true | `Lava-Retry-Debug.send_failures[1].result == "stop_pairing_exhausted"` |
| Request hash cached after archive failure | true | `Lava-Retry-Debug.hashes_cached` |
| Final error returned to user | PairingListEmptyError | HTTP response body |

---

## 13. Retry Debug Header: Implementation Requirements (FUTURE WORK)

> **Note**: This section describes planned functionality that is **not yet implemented**. The `Lava-Retry-Debug` header, retry trace model, and E2E assertions described below are design targets for a future phase. No trace recording or debug header logic exists in the current branch.

To enable E2E testing of retry behavior, add a `Lava-Retry-Debug` response header containing a JSON object with the full retry decision trace. This header is only included when the `lava-debug-relay` directive header is present in the request.

### 13.1 Existing Header Infrastructure

The system already has the infrastructure for this:

| Existing Component | Location | How It's Used |
|---|---|---|
| `lava-debug-relay` directive header | `common/endpoints.go:19` | Request header that enables verbose debug info in response |
| `Lava-Retries` header | `appendHeadersToRelayResult()` | Already returns retry count (attempts - 1) |
| `Lava-Errored-Providers` header | Debug-only | Already returns errored provider list |
| `Lava-Node-Errors-providers` header | Debug-only | Already returns per-provider error details |
| `appendHeadersToRelayResult()` | `rpcconsumer_server.go:1841`, `rpcsmartrouter_server.go:1980` | Constructs response metadata |
| `RelayReply.Metadata` | `pairing/relay.proto:97-105` | Protobuf field for response headers |

### 13.2 New Data to Collect

The policy engine and workers need to record their decisions into a shared trace structure that is attached to the relay context.

#### 13.2.1 RetryTrace struct (new, in `relaypolicy` package)

```go
type RetryTrace struct {
    mu               sync.Mutex
    WorkerResults    []WorkerRecord     // one per worker response, keyed by (attemptID, provider)
    PolicyDecisions  []PolicyRecord     // one per policy.Decide() call, keyed by attemptID
    SendFailures     []SendRecord       // one per OnSendRelayResult call, keyed by SendAttemptID
    ArchiveTried     bool
    HashesCached     bool
    TotalErrors      ErrorCounts
}

// WorkerRecord captures what a single worker observed and classified.
// Keyed by (AttemptID, Provider) — safe under concurrent workers because
// each worker has a unique provider address within a batch.
type WorkerRecord struct {
    AttemptID      int                 // stable identifier for this attempt
    BatchNumber    int                 // batch this worker belongs to
    Provider       string              // provider address (unique per batch)
    ErrorType      string              // "node_error", "protocol_error", "success"
    ErrorMessage   string              // raw error message (truncated to 200 chars)
    ErrorClass     ErrorClassification // result of ClassifyError
    Eligibility    string              // "unwanted", "allow_retry"
}

// PolicyRecord captures the output of one policy.Decide() call.
// Keyed by AttemptID — one per gotResults/ticker event.
// Separate from WorkerRecord to avoid race between worker writes and policy reads.
type PolicyRecord struct {
    AttemptID      int                 // matches WorkerRecord.AttemptID
    Action         string              // "retry", "stop"
    Mutation       MutationOutput      // structured: {ArchiveAction, CacheHashes}
    Reason         string              // "Default", "UnsupportedMethod", "MaxRetriesReached", etc.
}

type SendRecord struct {
    SendAttemptID  int                 // dedicated monotonic counter for send failures (unique key)
    BatchNumber    int                 // which batch this send attempt belongs to (NOT unique — see note)
    Error          string              // error type (e.g., "PairingListEmpty")
    Result         string              // "success", "retry", "stop_pairing_exhausted", "stop_batch_limit"
    PairingErrors  int                 // consecutive pairing error count at this point
    BatchErrors    int                 // consecutive batch error count at this point
}
// NOTE: BatchNumber is NOT a unique key for SendRecords. UsedProviders.BatchNumber()
// only increments when AddUsed() succeeds. Repeated GetSessions() failures do not
// increment it, so multiple send failures can share the same BatchNumber value.

type ErrorCounts struct {
    Node           int
    Protocol       int
    Special        int
}
```

#### 13.2.2 Where to Record

Each component records its part of the trace independently:

| Component | What It Records | Key | When |
|---|---|---|---|
| Worker goroutine | `WorkerRecord{AttemptID, BatchNumber, Provider, ErrorType, ErrorMessage, ErrorClass, Eligibility}` | `(AttemptID, Provider)` | After classification + eligibility, before SetResponse |
| `policy.Decide()` | `PolicyRecord{AttemptID, Action, Mutation, Reason}` | `AttemptID` | After decision, in the gotResults/ticker case |
| `policy.OnSendRelayResult()` | `SendRecord{SendAttemptID, BatchNumber, Error, Result, PairingErrors, BatchErrors}` | `SendAttemptID` | After send result decision, in the batchUpdate case |

**Attempt identity**: `AttemptID` is assigned by the state machine when it creates the dispatch instruction (before sending to `relayTaskChannel`). The ID is carried immutably inside the instruction into the worker goroutine. This avoids races where a slow worker reads a counter that the state machine has already incremented for a later attempt. Workers and policy records share `AttemptID` as a join key for correlation — but neither reads it from shared mutable state.

**Send failure identity**: `SendRecord` uses a dedicated `SendAttemptID` (monotonically incremented per `OnSendRelayResult()` call), not `batchNumber`. `batchNumber` is included for context but is not a unique key — repeated `GetSessions()` failures do not increment `batchNumber`, so multiple send failures can share the same value.

Worker records and policy records are **separate lists**, not patched together. They are written independently — no "update the last record" pattern.

#### 13.2.3 Flow: How the Trace is Built

```
1. Create RetryTrace at relay start (in SendRelay/ProcessRelaySend)
   Store in relay context or attach to RelayProcessor
   Initialize attemptCounter = 0, sendAttemptCounter = 0

2. State machine creates dispatch instruction:
   // Assign attempt ID BEFORE sending to relayTaskChannel
   instruction.AttemptID = attemptCounter
   attemptCounter++
   relayTaskCh <- instruction
   // The AttemptID is now immutable — carried into the worker goroutine

3. Worker goroutine completes:
   // Each worker reads AttemptID from its instruction (immutable, no race)
   retryTrace.RecordWorkerResult(WorkerRecord{
     AttemptID:    instruction.AttemptID,
     BatchNumber:  batchNumber,
     Provider:     providerAddress,
     ErrorType:    "node_error",
     ErrorMessage: truncate("rate limit exceeded", 200),
     ErrorClass:   classifyResult,
     Eligibility:  eligibilityResult.Action,
   })

4. policy.Decide() completes:
   // Uses the same AttemptID from the instruction that triggered this batch
   retryTrace.RecordPolicyDecision(PolicyRecord{
     AttemptID: currentAttemptID,
     Action:    output.Action,
     Mutation:  output.Mutation,
     Reason:    output.Reason,
   })

5. policy.OnSendRelayResult() completes:
   // Uses a dedicated send-attempt counter, NOT batchNumber
   retryTrace.RecordSendFailure(SendRecord{
     SendAttemptID: sendAttemptCounter,
     BatchNumber:   batchNumber,  // included for context, not as unique key
     Error:         "PairingListEmpty",
     Result:        "stop_pairing_exhausted",
     PairingErrors: 2,
     BatchErrors:   2,
   })
   sendAttemptCounter++

5. appendHeadersToRelayResult() serializes:
   if debugRelay {
     traceJSON, _ := json.Marshal(retryTrace)
     metadataReply = append(metadataReply, Metadata{
       Name:  "Lava-Retry-Debug",
       Value: string(traceJSON),
     })
   }
```

**Concurrency safety**: Worker records are appended under a mutex. Each worker has a unique `(AttemptID, Provider)` pair within a batch. Policy decisions are written only from the state machine goroutine (single writer). Send failures are written only from the state machine goroutine. The mutex protects cross-goroutine access (workers write, state machine reads for the final serialization).

### 13.3 Changes Required by File

| File | Change | Description |
|---|---|---|
| `relaypolicy/trace.go` | **NEW** | `RetryTrace`, `WorkerRecord`, `PolicyRecord`, `SendRecord` structs + `RecordWorkerResult()`, `RecordPolicyDecision()`, `RecordSendFailure()`, `ToJSON()` methods. Records keyed by `(AttemptID, Provider)` / `AttemptID` / `SendAttemptID` — no order-dependent patching. `AttemptID` is assigned at dispatch time and carried immutably into workers. |
| `relaypolicy/policy.go` | **MODIFY** | `Decide()` and `OnSendRelayResult()` accept `*RetryTrace` parameter (or trace is a field on `Policy`). Record decisions after each call. |
| `relaypolicy/classify.go` | **MODIFY** | `ClassifyNodeError(chainID, statusCode, errorMessage)` returns `ErrorClassification{IsNonRetryable, IsUnsupportedMethod}`. Worker records classification in the trace. |
| `relaycore/relay_processor.go` | **MODIFY** | Add `retryTrace *RetryTrace` field. Workers access it via `relayProcessor.GetRetryTrace()` to record their results. |
| `rpcconsumer/rpcconsumer_server.go` | **MODIFY** | (1) Create `RetryTrace` in `ProcessRelaySend()`. (2) In worker goroutine: record classification + eligibility. (3) In `appendHeadersToRelayResult()`: serialize trace to `Lava-Retry-Debug` header when `lava-debug-relay` is set. |
| `rpcsmartrouter/rpcsmartrouter_server.go` | **MODIFY** | Same changes as rpcconsumer. |
| `relaycore/unified_relay_state_machine.go` | **MODIFY** | After `policy.Decide()` and `policy.OnSendRelayResult()` calls, record decisions in the trace. |

### 13.4 How to Use in E2E Tests

```bash
# Send request with debug header to get retry trace
curl -s http://consumer:port \
  -H "lava-debug-relay: true" \
  -d '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x...","latest"],"id":1}' \
  | jq '.metadata["Lava-Retry-Debug"] | fromjson'

# Assert specific fields
TRACE=$(curl -s ... | jq -r '.metadata["Lava-Retry-Debug"]')

# Check number of attempts
echo $TRACE | jq '.attempts' # expect: 2

# Check all providers were tried (worker records, keyed by attempt_id + provider)
echo $TRACE | jq '.worker_results | length' # expect: 2

# Check classification was correct
echo $TRACE | jq '.worker_results[0].error_class.is_retryable' # expect: true
echo $TRACE | jq '.worker_results[0].error_class.is_unsupported_method' # expect: false

# Correlate worker and policy records by attempt_id
echo $TRACE | jq '.worker_results[0].attempt_id == .policy_decisions[0].attempt_id' # expect: true

# Check archive mutation (structured, not string)
echo $TRACE | jq '.policy_decisions[0].mutation.archive_action' # expect: "add_archive"
echo $TRACE | jq '.policy_decisions[1].mutation.cache_hashes' # expect: true

# Check archive was tried
echo $TRACE | jq '.archive_tried' # expect: true

# Check final reason
echo $TRACE | jq '.final_reason' # expect: "PairingListEmptyError"

# Check send failure pattern (keyed by send_attempt_id, batch_number is context only)
echo $TRACE | jq '.send_failures[-1].result' # expect: "stop_pairing_exhausted"
echo $TRACE | jq '.send_failures[-1].pairing_errors' # expect: 2
```

### 13.5 Header Size Considerations

The `Lava-Retry-Debug` header is only included when `lava-debug-relay` is set (test/debug environments). In production, no overhead is added. For typical retry scenarios (2-10 attempts), the JSON is ~1-3 KB — well within HTTP header limits. For pathological cases (10 attempts, long provider addresses), consider truncating `ErrorMessage` fields to 200 characters.
