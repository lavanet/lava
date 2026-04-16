---
name: review-synthesis
description: "How the synthesis-reporter merges findings from the 4 parallel reviewers (architecture, security, performance, style) into a single prioritized report. Loaded when synthesis-reporter runs at the end of protocol-review. Covers reading reviewer files, deduplication, severity reconciliation, clustering, prioritization, and final report production."
---

# Review Synthesis Skill

Method used by synthesis-reporter. Role at `.claude/agents/synthesis-reporter.md`; this file is the *how*.

## Method Overview

1. Read all four reviewer outputs from `_workspace/`:
   - `01_architecture_findings.md`
   - `01_security_findings.md`
   - `01_performance_findings.md`
   - `01_style_findings.md`
2. Read scope from `_workspace/00_input/scope.md`.
3. Read feedback (if re-synthesis): `_workspace/00_input/feedback.md`.
4. Parse each findings file into a structured list (in your head or via temp JSON).
5. **Deduplicate and cluster** per rules in `.claude/skills/protocol-review/references/review-taxonomy.md`.
6. **Reconcile severities** using max rule + rationale.
7. **Prioritize** using the fix-order heuristic.
8. Produce `_workspace/02_synthesis_report.md` using the template at `.claude/skills/protocol-review/references/report-template.md`.
9. Copy final report to user-specified path (default `REVIEW_REPORT.md` in repo root).
10. Return a one-paragraph summary to the orchestrator.

## Deduplication Test

Two findings are **duplicates** (merge into one entry in final) if:
- Fixing one fixes the other.
- Same file:line and same described defect.

Two findings are **clustered** (separate entries, grouped under a theme) if:
- Shared root cause but different angles.
- Same code location but different concerns.
- Different locations but symptomatic of the same pattern.

Never silently collapse both into one ID — each reviewer's finding retains its original ID.

## Severity Reconciliation

When two reviewers file findings that cluster:

- **Final severity = max(severities)** of the clustered findings.
- If the max is from security, security framing leads (title/recommendation).
- If the max is from architecture and the finding has a security angle, note "with security implications if left unresolved".
- Record the disagreement in Appendix B with a one-line rationale.

Example:
- SEC rates "session CU counter race" as critical.
- PERF rates same code as medium (locking inefficiency).
- ARCH rates same code as high (state management).
- Final: critical. Rationale: "SEC's attack scenario (CU double-charge) outweighs PERF's latency framing".

## Prioritization (Fix Order)

Default order:
1. critical security
2. critical correctness (architecture, concurrency)
3. high security
4. high perf
5. high arch
6. medium × (interleaved by category severity)
7. low style, etc.

Tie-breakers within tier:
- lower estimated effort first (cheap wins buy momentum)
- findings in hot path before cold
- findings in recently-changed code before stable code (probably fresh in reviewer's mind, lowest friction to fix)

## Effort Estimation

You produce effort estimates; reviewers don't. Use S/M/L:
- **S** — < ½ day: contained in one function, no API changes, tests trivial.
- **M** — 1–3 days: multi-file, contained in one package, tests substantial but not cross-package.
- **L** — > 3 days: cross-package API change, migration needed, or coordinated rollout.

Be honest — overestimating makes the report less actionable; underestimating erodes trust.

## Coverage Disclosure

Coverage table shows each reviewer's status:

| Status | Meaning |
|--------|---------|
| complete | reviewer file exists and covers the full scope |
| partial | reviewer explicitly noted time-boxing or skipped sections |
| missing | reviewer file absent (failure) |

If `partial` or `missing`, note it in Executive Summary prominently.

## Follow-up Investigation Section

Collect reviewer findings tagged `Confidence: low` or flagged as "needs measurement". These go in a dedicated section because they're not actionable until investigated.

## Report Version

- Initial synthesis: `Report version: 1`.
- Re-synthesis after feedback: increment version; add "Changes Since Previous Synthesis" section to the top of the report. Show resolved / new / re-rated findings.

## What NOT to Do

- Don't invent new findings. If synthesis reveals a gap, file under "Follow-up Investigation" or surface to the orchestrator.
- Don't delete a reviewer's finding, even if you think it's wrong. Annotate disagreement and include.
- Don't rewrite reviewer descriptions — cite them. You can paraphrase in the synthesized Top Findings but the Appendix A list shows raw titles.
- Don't recommend fixes the reviewers didn't — synthesize what they said; surface alternatives only if multiple reviewers suggested different fixes.

## Output

Per the role template. Primary: `_workspace/02_synthesis_report.md`. Secondary copy to user path. See report template at `.claude/skills/protocol-review/references/report-template.md`.
