---
name: protocol-review
description: "Runs a comprehensive parallel review of the Lava protocol layer (rpcconsumer / rpcprovider / rpcsmartrouter / chainlib / lavasession / relaycore / statetracker / chaintracker / provideroptimizer / lavaprotocol / qos / metrics) with four specialized reviewers — architecture, security, performance, code style — and merges findings into a single prioritized report. Use when the user asks to review, audit, evaluate, inspect, critique, sanity-check, or analyze protocol code, a PR, a package, or a diff. Also triggers on: re-run review, update findings, re-review with feedback, expand scope, focus only on security/perf/style/architecture, regenerate report. Excludes on-chain x/ Cosmos modules."
---

# Protocol Review Orchestrator

Coordinates a 4-reviewer fan-out + 1-synthesizer fan-in over the Lava off-chain protocol layer. Produces a single prioritized review report.

## Execution Mode

**Sub-agent mode.** Four reviewers run in parallel via `Agent(subagent_type: "general-purpose", model: "opus", run_in_background: true)`, each loading its corresponding agent definition from `.claude/agents/`. After all four complete, a fifth agent merges the outputs.

Sub-agent mode is chosen because reviewers operate independently on the same input; cross-pollination is handled by the synthesizer rather than requiring live inter-agent chat. Simpler and cheaper than agent-team mode for this pattern.

## Agent Composition

| Agent | subagent_type | Role | Output file |
|-------|--------------|------|-------------|
| architecture-reviewer | general-purpose | Module boundaries, dependency direction, interfaces, concurrency structure | `_workspace/01_architecture_findings.md` |
| security-reviewer | general-purpose | Signing, sessions, trust boundaries, input validation, DoS, econ attacks | `_workspace/01_security_findings.md` |
| performance-reviewer | general-purpose | Hot-path allocations, goroutine lifecycle, locks, I/O, serialization | `_workspace/01_performance_findings.md` |
| style-reviewer | general-purpose | golangci-lint compliance, naming, error idioms, consistency | `_workspace/01_style_findings.md` |
| synthesis-reporter | general-purpose | Deduplicate, reconcile severity, prioritize, produce final report | `_workspace/02_synthesis_report.md` → copied to `REVIEW_REPORT.md` |

All agents run with `model: "opus"` per harness policy.

## Workflow

### Phase 0: Context Check

Before starting, determine execution mode:

1. Check if `_workspace/` exists in the project root.
2. Branch:
   - **No `_workspace/`** → initial run. Proceed to Phase 1.
   - **`_workspace/` exists + user asks partial re-review** (e.g., "just redo security", "tighten the performance findings") → partial re-invocation. Skip Phase 2 fan-out for the agents not being re-run; re-invoke only the specified reviewer(s); re-run the synthesizer in Phase 3.
   - **`_workspace/` exists + user asks full re-review with new scope or new code** → archive previous: move `_workspace/` to `_workspace_<YYYYMMDD_HHMMSS>/`, then proceed to Phase 1.
   - **`_workspace/` exists + user provides feedback on the report** → place feedback into `_workspace/00_input/feedback.md`, re-run affected reviewers (which read feedback per their re-invocation behavior), then synthesizer.

### Phase 1: Scope Capture

1. Create `_workspace/` and `_workspace/00_input/` if absent.
2. From user input, determine scope. Scope can be:
   - A diff range (e.g., `main..HEAD`, a commit, a PR number) — capture with `git diff <range> --stat` and list files.
   - A set of packages (e.g., `protocol/rpcconsumer/...`) — list files.
   - The whole protocol layer — list `protocol/` subdirs, excluding `loadtest/`, `performance/`, `monitoring/`, `integration/testdata/` by default.
   - A feature area (e.g., "session handling") — expand to the relevant packages.
3. Write scope to `_workspace/00_input/scope.md` with this shape:

```markdown
# Review Scope
**Kind:** diff | packages | full-protocol | feature
**Range / targets:** <specifics>
**Files in scope:** <list or glob>
**Out of scope:** <explicit exclusions — always include `x/**`, `app/**`, consensus code>
**Focus instructions:** <any user-specified emphasis>
**Previous reports (if any):** <paths>
```

4. If re-invocation with feedback: ensure `_workspace/00_input/feedback.md` is populated.

### Phase 2: Parallel Review

Spawn 4 reviewers in a **single message** with 4 `Agent` tool calls (parallel execution). Each uses `run_in_background: true`. Template per reviewer:

```
Agent(
  description: "<Role> review of Lava protocol",
  subagent_type: "general-purpose",
  model: "opus",
  run_in_background: true,
  prompt: """
  You are the <agent-name>. Read your full role definition at
  `.claude/agents/<agent-name>.md` and follow it strictly.

  Scope input: `_workspace/00_input/scope.md`
  Shared context:
    - `.claude/skills/protocol-review/references/protocol-map.md`
    - `.claude/skills/protocol-review/references/review-taxonomy.md`
    - `.claude/skills/<agent-specific>/SKILL.md` and references/

  {if re-invocation:}
  Prior findings: `_workspace/01_<category>_findings.md`
  User feedback:  `_workspace/00_input/feedback.md`
  Update behavior: preserve IDs, mark resolved, append new.

  Write your final report to `_workspace/01_<category>_findings.md`.
  Return a one-paragraph summary pointing at the output path.
  """
)
```

Agent-specific skill paths:
- architecture-reviewer → `.claude/skills/architecture-review/`
- security-reviewer → `.claude/skills/security-review/`
- performance-reviewer → `.claude/skills/performance-review/`
- style-reviewer → `.claude/skills/style-review/`

Orchestrator waits for all four background completions (notifications).

### Phase 3: Synthesis

Spawn synthesis-reporter as a single (foreground) agent:

```
Agent(
  description: "Synthesize Lava protocol review findings",
  subagent_type: "general-purpose",
  model: "opus",
  prompt: """
  You are synthesis-reporter. Read your role at
  `.claude/agents/synthesis-reporter.md` and skill at
  `.claude/skills/review-synthesis/SKILL.md`.

  Read the four reviewer outputs:
    - _workspace/01_architecture_findings.md
    - _workspace/01_security_findings.md
    - _workspace/01_performance_findings.md
    - _workspace/01_style_findings.md
  Read scope:
    - _workspace/00_input/scope.md
  {if exists} _workspace/00_input/feedback.md

  Produce:
    - _workspace/02_synthesis_report.md
  Copy the final report to <destination> (default: REVIEW_REPORT.md in repo root).

  Return a one-paragraph summary with the top 3 findings and the report path.
  """
)
```

### Phase 4: User Handoff

1. Read `_workspace/02_synthesis_report.md`.
2. Summarize to the user in ≤120 words: scope reviewed, top-3 critical/high findings, pointer to the full report path.
3. Prompt: "Would you like any finding expanded, a specific reviewer re-run with more depth, or a development pass (protocol-develop) to fix critical issues?"

### Phase 5: Cleanup

- Do NOT delete `_workspace/`. It's the audit trail.
- If the user asks for cleanup, move it to `_workspace_<timestamp>/` rather than deleting.

## Data Flow

```
[user request]
   └─► [orchestrator]
         ├─► Phase 1: scope.md
         ├─► Phase 2 (parallel):
         │     ├─► architecture-reviewer ─► 01_architecture_findings.md
         │     ├─► security-reviewer ───► 01_security_findings.md
         │     ├─► performance-reviewer ─► 01_performance_findings.md
         │     └─► style-reviewer ─────► 01_style_findings.md
         ├─► Phase 3:
         │     └─► synthesis-reporter ──► 02_synthesis_report.md
         └─► Phase 4: hand to user, copy to REVIEW_REPORT.md
```

## Error Handling

| Situation | Strategy |
|-----------|----------|
| 1 reviewer fails / times out | 1 retry. If still failing, synthesis proceeds with remaining 3; final report calls out missing coverage in the "Coverage" table. |
| 2+ reviewers fail | Pause. Surface the situation to the user with partial results; ask whether to retry, substitute with `general-purpose` fallback prompt, or accept partial. |
| Synthesizer fails | 1 retry. If still failing, surface raw reviewer files to user with a one-line warning. |
| Conflicting severity between reviewers | Keep both in the synthesis; final severity = max; rationale recorded in `Appendix B: Disagreements`. |
| Reviewer reports 0 findings | Valid outcome. Included in the Coverage table with count = 0. Do NOT re-run. |
| Scope too large (> 50 files in diff) | Orchestrator suggests narrowing before Phase 2; if user confirms, proceed but warn that time-boxing may skip files. |
| `_workspace/` contains stale results from an unrelated run | Phase 0 archives it to `_workspace_<timestamp>/` so history is preserved. |

## Test Scenarios

### Normal Flow
1. User: "Review the provider relay handler in protocol/rpcprovider/ for the last 3 commits"
2. Phase 1: scope.md lists files touched in `git log -3 -- protocol/rpcprovider/`
3. Phase 2: 4 reviewers run in parallel, each producing `01_*_findings.md`
4. Phase 3: synthesis-reporter produces `02_synthesis_report.md` and `REVIEW_REPORT.md`
5. Phase 4: orchestrator summarizes; user gets top-3 findings
6. Expected artifacts: `_workspace/{00_input/scope.md, 01_*.md, 02_synthesis_report.md}` + `REVIEW_REPORT.md`

### Partial Re-invocation
1. User (after initial run): "I need the security review to be more aggressive on the reward server"
2. Phase 0 detects existing `_workspace/`; user feedback → `_workspace/00_input/feedback.md`
3. Skip architecture/performance/style — re-invoke only security-reviewer with feedback + prior findings file path
4. Re-run synthesizer which picks up updated `01_security_findings.md`
5. Updated `REVIEW_REPORT.md` replaces previous; prior versions preserved in `_workspace/` history

### Error Flow
1. Phase 2: performance-reviewer times out on 2nd retry
2. Other 3 complete normally
3. Phase 3: synthesis-reporter called; produces report with Coverage table noting "performance: missing — timeout"
4. Phase 4: orchestrator surfaces to user, asks whether to retry performance-reviewer on a narrower scope
