# Review: `14_STATE_MACHINE_ARCHITECTURE_AND_REFACTOR.md`

## Verdict

Not all review comments are fixed yet.

## Scope Reviewed

This review compares the document and branch changes against `main` in:

- `docs/retry-refactor-presentation/14_STATE_MACHINE_ARCHITECTURE_AND_REFACTOR.md`
- `protocol/relaycore/unified_relay_state_machine.go`
- `protocol/relaycore/relay_processor.go`
- `protocol/relaycore/relay_state.go`
- `protocol/relaypolicy/policy.go`
- `protocol/relaypolicy/classify.go`
- `protocol/relaypolicy/eligibility.go`
- `protocol/relaypolicy/types.go`
- `protocol/rpcconsumer/consumer_relay_state_machine.go`
- `protocol/rpcconsumer/rpcconsumer_server.go`
- `protocol/rpcsmartrouter/smartrouter_relay_state_machine.go`
- `protocol/rpcsmartrouter/rpcsmartrouter_server.go`

## Remaining Issues To Fix

### 1. `DecideEligibility()` is still not wired into worker paths

`relaypolicy.DecideEligibility()` exists, but it is still not called from either worker path.

- Consumer eligibility logic still remains in the existing worker/session flow.
- SmartRouter eligibility wiring is also still missing.

This means the doc should continue to treat eligibility centralization as future work until the actual worker integration is done.

### 2. SmartRouter worker classification wiring is still not implemented

Consumer node-error classification now uses `relaypolicy.ClassifyNodeError()`, but SmartRouter still does not use the new classification helpers.

So the worker-side refactor is only partial:

- Consumer node-error classification: partially migrated
- Consumer eligibility: not migrated
- SmartRouter classification: not migrated
- SmartRouter eligibility: not migrated

This should stay explicitly marked as incomplete in the doc until it is actually wired.

### 3. The doc still contains stale references to old API names and old config shape

There are still places in the design doc that refer to symbols or config fields that no longer match the code.

Examples that still need cleanup:

- `ClassifyError()` references should be updated to `ClassifyNodeError()` / `ClassifyProtocolError()` where appropriate.
- `PolicyConfig` examples still mention fields that no longer exist, such as `EnableUnsupportedCheck`.
- Later narrative sections still describe worker behavior as if `DecideEligibility()` were already executed in workers.

The document is much closer now, but it is not yet fully aligned with the current implementation.

### 4. The "current architecture" section still describes pre-refactor behavior as if it were live

The top of the document says Phase 3 is done, but the early architecture walkthrough still explains the old `HasRequiredNodeResults()` and `shouldRetryRelay()` interaction as current behavior.

That is no longer accurate.

Current code behavior is:

- no-success path in `HasRequiredNodeResults()` returns `false` unconditionally
- the state machine decides stop vs retry through `policy.Decide()`

This old description should be updated or clearly labeled as historical behavior.

### 5. The new `IsTickerHedge` and mutation-application paths do not have focused direct tests

The relevant protocol packages now pass, including the previously failing SmartRouter tests, which is good.

However, the newer logic added to fix the regression is still only indirectly covered:

- `DecisionInput.IsTickerHedge`
- explicit `DecisionOutput.Mutation` application in the unified state machine

Adding focused tests for those paths would reduce the chance of another regression.

## Validation

Current validation status:

- `go test ./protocol/relaypolicy`: pass
- `go test ./protocol/relaycore`: pass
- `go test ./protocol/rpcconsumer`: pass
- `go test ./protocol/rpcsmartrouter`: pass (including `TestConsumerStateMachineHappyFlow`)
