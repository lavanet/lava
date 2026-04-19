# Report Template (Synthesis Output)

Used by synthesis-reporter. Reviewers use the template in their own agent definitions. This file shows the **merged** report that users see.

---

```markdown
# Protocol Code Review — Synthesized Report

**Scope:** <from _workspace/00_input/scope.md>
**Date:** <YYYY-MM-DD>
**Reviewers:** architecture, security, performance, style
**Report version:** <1 | 2 | ...>  (increments when re-synthesized)

## Executive Summary

- **Top risk:** <one-line headline>
- **Scope covered:** <high level>
- **Critical count:** N (fix before merge)
- **High count:** N
- **Fix order headline:** <which 2–3 findings to do first>
- **Deferred / info:** <count of low + info>
- <up to 5 more bullets>

## Coverage

| Reviewer | Status | Findings | Notes |
|----------|--------|----------|-------|
| architecture | complete | N | — |
| security | complete | N | — |
| performance | partial | N | timed out on chainlib/chainproxy |
| style | complete | N | — |

## Severity Counts

| Severity | Arch | Sec | Perf | Style | Total |
|----------|------|-----|------|-------|-------|
| critical | 0    | 1   | 0    | 0     | 1     |
| high     | 2    | 3   | 1    | 0     | 6     |
| medium   | 4    | 2   | 5    | 3     | 14    |
| low      | 1    | 0   | 2    | 7     | 10    |
| info     | 1    | 0   | 0    | 1     | 2     |

## Top Findings (Prioritized)

### 1. [CRITICAL] <synthesized title>
- **Sources:** SEC-003 (primary), ARCH-005 (structural overlap)
- **Severity rationale:** SEC rated critical, ARCH rated high on same code path — final critical.
- **Location:** `protocol/rpcprovider/foo.go:120–140`
- **What's wrong:** <plain-language synthesis>
- **Impact:** <aggregated from all source findings>
- **Recommended fix:** <concrete, ideally with code or API sketch>
- **Effort estimate:** S / M / L
- **Fix verification:** <how to prove the fix — test, benchmark, red-team scenario>

### 2. [HIGH] <title>
(same structure)

### 3. [HIGH] <title>
...

(continue for top 10 — remaining findings listed in Appendix A)

## Clusters / Themes

### Cluster A — session state races
**Findings:** SEC-003, SEC-007, PERF-002, ARCH-005
**Root cause:** CU counter and session metadata in `lavasession.Session` lack consistent synchronization; callers on both consumer and provider side modify concurrently.
**Suggested remediation:** Consolidate session mutation behind a single lock or replace the counter with `atomic.Int64`; document ownership in doc comments.
**Owned by:** lavasession package maintainer

### Cluster B — error wrapping drift
**Findings:** STYLE-004, STYLE-007, STYLE-012
**Root cause:** Recent PRs introduced `fmt.Errorf` where the rest of the codebase uses `utils.LavaFormatError`.
**Suggested remediation:** Audit recent commits; migrate to project helper; add CI grep to catch future drift.

(list all clusters)

## Follow-up Investigation

Items flagged by reviewers as requiring runtime measurement or further context:

- **PERF-009:** "chaintracker poll frequency may be high" — needs pprof profile under load. Reviewer confidence: low.
- **ARCH-007:** "possible overlap between rpcconsumer and rpcsmartrouter retry logic" — needs architect decision on whether convergence is desirable.

## Appendix A: Full Findings List

| ID | Sev | Title | Location | Source doc |
|----|-----|-------|----------|------------|
| SEC-001 | critical | <title> | `path.go:100` | 01_security_findings.md |
| ARCH-001 | high | <title> | `path.go:50` | 01_architecture_findings.md |
| PERF-001 | high | <title> | `path.go:80` | 01_performance_findings.md |
| ... |

## Appendix B: Disagreements & Reconciliation

| Finding | Arch | Sec | Perf | Style | Final | Rationale |
|---------|------|-----|------|-------|-------|-----------|
| Session CU counter race | — | critical | high | — | critical | SEC's attack scenario (double-spend) justifies critical over PERF's perf framing. |

## Appendix C: Methodology

- **Tools consulted:**
  - `.claude/skills/protocol-review/references/protocol-map.md`
  - `.claude/skills/protocol-review/references/review-taxonomy.md`
  - `.golangci.yml` (for style findings)
- **Reviewer prompts:** `.claude/agents/{architecture,security,performance,style,synthesis}-reviewer.md`
- **Scope excluded by policy:** `x/**`, `app/**`, generated proto, `protocol/loadtest/`, `protocol/performance/`, `protocol/monitoring/`.
- **Time-boxing:** <if any reviewer was time-boxed, note it here>

## Appendix D: Changes Since Previous Synthesis

Only present when `Report version >= 2`.

- **Resolved since v1:** SEC-002, STYLE-005
- **New since v1:** SEC-011 (introduced by feedback pass), PERF-014
- **Re-rated:** ARCH-004 (medium → high on further evidence)
```

---

## Rationale Notes (for synthesis-reporter)

- **Why keep IDs stable?** External references (commit messages, tickets) cite IDs. Renumbering breaks those.
- **Why cluster after prioritizing?** Prioritization is the engineer's 30-second scan; clusters are the deep-dive section.
- **Why disagreements appendix?** Keeps reviewer judgment visible rather than silently collapsing. Engineers reading later can see why a finding was rated critical.
- **Effort estimate is synthesizer's call?** Yes. Reviewers don't rate effort; synthesis has the cross-view to estimate.
- **Fix verification?** Every critical/high should say how to *prove* the fix works. For perf: benchmark name. For security: red-team scenario. For arch: refactor checklist. For style: linter green.
