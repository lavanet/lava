---
name: protocol-develop
description: "Designs, implements, tests, and (optionally) reviews changes to the Lava protocol layer (rpcconsumer / rpcprovider / rpcsmartrouter / chainlib / lavasession / relaycore / statetracker / chaintracker / provideroptimizer / lavaprotocol / qos / metrics). Triggers on: add a feature to consumer/provider/smart-router, implement X in the protocol, refactor a protocol package, fix a bug in chainlib/relaycore/etc., build this change with design+tests, end-to-end dev task on the off-chain layer. Also covers follow-up: update the design, revise the implementation, expand tests, re-run after review findings. Does NOT touch on-chain x/ Cosmos modules — for those, do not invoke this skill."
---

# Protocol Develop Orchestrator

Coordinates a design → implement → test pipeline for changes to the Lava off-chain protocol layer. Optionally chains into `protocol-review` for post-implementation audit.

## Execution Mode

**Sub-agent mode.** Pipeline: `protocol-architect` → `protocol-implementer` → `protocol-tester`. Each waits on the previous artifact. Optionally followed by `protocol-review`.

Sub-agent mode is chosen because the pipeline is sequential; there's little value in live chat across stages.

## Agent Composition

| Agent | subagent_type | Role | Output |
|-------|--------------|------|--------|
| protocol-architect | general-purpose | Design doc before code | `_workspace/01_design.md` |
| protocol-implementer | general-purpose | Go implementation following the design | code edits + `_workspace/02_implementation.md` |
| protocol-tester | general-purpose | Unit / integration tests | test files + `_workspace/03_testing.md` |
| (optional) synthesis-reporter via `protocol-review` | — | Post-impl review | `_workspace/04_post_review/...` |

All agents run with `model: "opus"`.

## Workflow

### Phase 0: Context Check

1. Check if `_workspace/` exists.
2. Branch:
   - **No `_workspace/`** → initial run → Phase 1.
   - **`_workspace/` exists + user asks to update the design** → partial: re-run protocol-architect only, then implementer and tester receive revised design.
   - **`_workspace/` exists + user asks to revise implementation from review feedback** → write feedback to `_workspace/00_input/feedback.md`, re-run implementer (picks up feedback per its re-invocation behavior), then re-run tester.
   - **`_workspace/` exists + user starts a new task** → archive previous: `mv _workspace _workspace_<YYYYMMDD_HHMMSS>`, then Phase 1.

### Phase 1: Intake

1. Create `_workspace/00_input/requirements.md` capturing the user's intent. Include:
   - What the change does (in user's words + orchestrator's rephrase)
   - Why (motivation, ticket, stakeholder ask if known)
   - Constraints (backwards compatibility, rollout, protocol version)
   - Non-goals

2. Decide pipeline shape:
   - **Full pipeline** (design + implement + test + optional review) — default for new features or non-trivial changes
   - **Skip design** — only if user explicitly says "I know what to do, just implement X" AND the change is localized (one file, obvious)
   - **Skip tests** — forbidden. A change without tests is unfinished. If the user insists, challenge once; if they re-confirm, note it in the final summary and let it stand.
   - **Include review** — if the user asked for "code review after", or if the change touches >3 packages or security-sensitive code (`lavasession`, `rewardserver`, signing paths).

### Phase 2: Design

Spawn protocol-architect (foreground):

```
Agent(
  description: "Design <feature>",
  subagent_type: "general-purpose",
  model: "opus",
  prompt: """
  You are protocol-architect. Role: `.claude/agents/protocol-architect.md`.
  Skill: `.claude/skills/protocol-design/SKILL.md` and references/.
  Shared context: `.claude/skills/protocol-review/references/protocol-map.md`.

  Read `_workspace/00_input/requirements.md`.
  {if re-invocation} Read `_workspace/00_input/feedback.md` and prior `_workspace/01_design.md`.

  Produce `_workspace/01_design.md` per the template in your role.
  Status: mark `ready-for-implementation` when confident; `draft` with `Open Questions` if blocked.
  """
)
```

**Gate:** if the architect marks status = `draft` with Open Questions, the orchestrator surfaces them to the user and pauses. User answers → re-invoke architect → re-check gate.

### Phase 3: Implementation

Spawn protocol-implementer (foreground):

```
Agent(
  description: "Implement <feature>",
  subagent_type: "general-purpose",
  model: "opus",
  prompt: """
  You are protocol-implementer. Role: `.claude/agents/protocol-implementer.md`.
  Skill: `.claude/skills/protocol-implementation/SKILL.md`.

  Read `_workspace/01_design.md` as source of truth.
  {if re-invocation} Read `_workspace/00_input/feedback.md` and prior `_workspace/02_implementation.md`.

  Implement the change. Run `make lint` and `go build ./protocol/...` before declaring done.
  Produce `_workspace/02_implementation.md` per the template in your role.
  """
)
```

**Gate:** if `02_implementation.md` reports lint/build failures, surface to user; ask whether to retry implementer with the failing output as feedback or to stop.

### Phase 4: Testing

Spawn protocol-tester (foreground):

```
Agent(
  description: "Test <feature>",
  subagent_type: "general-purpose",
  model: "opus",
  prompt: """
  You are protocol-tester. Role: `.claude/agents/protocol-tester.md`.
  Skill: `.claude/skills/protocol-testing/SKILL.md`.

  Read `_workspace/01_design.md` §7 Test Strategy.
  Read `_workspace/02_implementation.md` to know what changed.
  {if re-invocation} Read feedback + prior `_workspace/03_testing.md`.

  Add/update tests. Run them with `-race`. Produce `_workspace/03_testing.md`.
  """
)
```

**Gate:** if tests fail (and they're your new tests), surface; ask whether to re-invoke implementer to fix or tester to adjust.

### Phase 5: (Optional) Post-Implementation Review

If review was requested or if conditions in Phase 1 triggered it, run the review **inline** (do not nest the `protocol-review` skill — this would collide on `_workspace/`). The orchestrator runs the same fan-out + fan-in pattern as `protocol-review`, but with `_workspace/04_post_review/` as the base path.

1. Create `_workspace/04_post_review/00_input/scope.md` listing the files touched (from `_workspace/02_implementation.md` Files Changed table).
2. Spawn the 4 reviewers in a **single message** (parallel `Agent` tool calls, each with `run_in_background: true` and `model: "opus"`). Each prompt mirrors the review orchestrator's Phase 2 template but substitutes `_workspace/04_post_review/` for every `_workspace/` path and substitutes `_workspace/04_post_review/01_<category>_findings.md` for the output.
3. Wait for all 4 completions, then spawn `synthesis-reporter` (foreground) with paths pointed at `_workspace/04_post_review/01_*_findings.md` → output at `_workspace/04_post_review/02_synthesis_report.md`.
4. Copy the synthesis to `_workspace/04_post_review/REVIEW_REPORT.md`.

If the post-review returns **critical** or **high** findings, do not declare the dev task done. Offer the user:
- "Re-run protocol-implementer with these findings as feedback (Phase 3)" — default recommendation for critical.
- "Address in a follow-up PR" — acceptable for non-critical with user approval.

Per-reviewer prompt skeleton (use the same structure for all four):

```
Agent(
  description: "<Role> review — post-implementation",
  subagent_type: "general-purpose",
  model: "opus",
  run_in_background: true,
  prompt: """
  You are <agent-name>. Role: .claude/agents/<agent-name>.md.
  Skill: .claude/skills/<corresponding-review-skill>/SKILL.md and references/.
  Shared context: .claude/skills/protocol-review/references/{protocol-map,review-taxonomy}.md

  Scope: _workspace/04_post_review/00_input/scope.md
  Output: _workspace/04_post_review/01_<category>_findings.md

  This is a post-implementation review of a specific change — focus on files
  in scope, do not re-review the whole protocol. Follow the re-invocation
  behavior in your role file if a prior file exists at the output path.
  """
)
```

### Phase 6: User Handoff

1. Summarize in ≤150 words: what was designed, what was implemented (files & LoC), what was tested, and any deferred items.
2. Point to: `_workspace/01_design.md`, `_workspace/02_implementation.md`, `_workspace/03_testing.md`, and (if applicable) `_workspace/04_post_review/REVIEW_REPORT.md`.
3. If `status: ready-for-implementation` never reached (design still `draft`): surface Open Questions, do not proceed.

### Phase 7: Cleanup

- Preserve `_workspace/` as audit trail.
- Do not `git add` / commit automatically — the user decides when to commit.

## Data Flow

```
[user requirements]
   ↓
[architect] ──► _workspace/01_design.md
   ↓
[implementer] ──► code edits + _workspace/02_implementation.md
   ↓
[tester] ──► test files + _workspace/03_testing.md
   ↓ (optional)
[protocol-review] ──► _workspace/04_post_review/REVIEW_REPORT.md
   ↓
[summary to user]
```

## Error Handling

| Situation | Strategy |
|-----------|----------|
| Architect blocked on Open Questions | Surface to user, pause pipeline, resume after answers. |
| Architect repeatedly returns `draft` after 2 feedback loops | Surface — the task may be ill-defined; recommend breaking it down or gathering more context. |
| Implementer lint/build failure | Re-invoke once with lint output as feedback. If still failing, surface — possibly design-level issue. |
| Test failures on code under implementer control | Offer user a choice: re-invoke implementer with test output as feedback, or re-invoke tester to adjust test expectations (only if the test itself is wrong). |
| Tester reveals a bug in implementation | Stop forward progress. Go back to implementer with the failing test. |
| Post-review finds critical issues | Do not close the task. Offer implementer re-invocation with review findings as feedback. |
| Protocol version / wire format change detected | Orchestrator notes it in final summary; ideally architect surfaces early with the `x/protocol` flag, since governance changes are out of this skill's scope. |

## Test Scenarios

### Normal Flow
1. User: "Add per-provider circuit breaker in rpcconsumer when error rate > 30% for 10s"
2. Phase 1: requirements captured
3. Phase 2: architect reads current optimizer + lavasession, produces design with API sketch + test strategy; status=ready-for-implementation
4. Phase 3: implementer writes code in `protocol/provideroptimizer/` and `protocol/lavasession/`; lint clean
5. Phase 4: tester adds unit tests + one integration test; race-clean
6. Phase 5 (auto-triggered because lavasession touched): protocol-review runs on changed files; no critical
7. Phase 6: summary to user + 4 artifacts

### Re-invocation Flow
1. Post-review flags SEC-002 (racey CU counter) as high
2. User: "Fix the SEC-002 finding"
3. Phase 0 detects `_workspace/` → re-invocation
4. Write feedback → re-invoke implementer → re-invoke tester → re-run protocol-review
5. Updated artifacts; finding resolved

### Error Flow
1. Phase 2: architect raises 3 Open Questions
2. Orchestrator pauses, surfaces to user
3. User answers 2, defers 1
4. Architect re-invoked with answers; produces design noting the deferred question explicitly
5. Pipeline proceeds
