# Review Taxonomy

Shared taxonomy for all reviewers. Ensures consistent severity, IDs, categories, and output shape so the synthesis-reporter can merge cleanly.

## Severity Scale

| Severity | Definition | Examples |
|----------|-----------|----------|
| **critical** | Exploitable now, or will cause production incidents imminently. Must fix before merge/deploy. | Unverified RelayRequest signature accepted; goroutine leak that exhausts fds under normal load; panic on malformed upstream response. |
| **high** | Plausible exploit, material correctness risk, or notable perf regression on hot path. Fix this sprint. | Racey CU counter allowing double-spend of a relay; O(N²) loop in provider selection; missing ctx propagation on retry. |
| **medium** | Quality/maintainability drag, likely to bite within a quarter. Schedule in backlog. | Over-broad interface across 3 packages; repeated allocation in a warm path; inconsistent error wrapping. |
| **low** | Readability, style, minor cleanup. Nice to have. | Naming inconsistency; stale comment; minor linter nit. |
| **info** | Noted observation, not an action item. | Architectural note, reference to upcoming change, positive observation. |

**Severity reconciliation rule (synthesis only):** when two reviewers rate the same issue differently, the final severity = max; record both in the disagreements appendix with a one-line rationale.

## ID Format

`<CATEGORY>-NNN` where CATEGORY ∈ {`ARCH`, `SEC`, `PERF`, `STYLE`}, `NNN` is zero-padded starting at `001` per reviewer.

- Reviewers **never** renumber their own IDs across runs. If a finding is resolved, mark `Status: resolved`; the ID persists.
- Synthesis may group findings under a "cluster" with a theme name — this is narrative, not a new ID.

## Category Vocabulary

Each reviewer has a fixed category vocabulary. Use only from the list; if none fits, use `other` and describe. Synthesis relies on these for grouping.

### ARCH categories
- `module-boundary`
- `dependency-direction`
- `interface-design`
- `concurrency`
- `state-management`
- `error-model`
- `request-path`
- `testability`
- `other`

### SEC categories
- `auth`
- `signature`
- `input-validation`
- `crypto`
- `secret-handling`
- `dos`
- `tls-transport`
- `logging`
- `economic`
- `concurrency-security`
- `other`

### PERF categories
- `allocation`
- `goroutine-lifecycle`
- `locking`
- `channels`
- `io-pattern`
- `serialization`
- `caching`
- `memory-footprint`
- `gc-pressure`
- `blocking-boundary`
- `other`

### STYLE categories
- `linter-hit`
- `naming`
- `error-idiom`
- `comment`
- `package-layout`
- `imports`
- `test-style`
- `nil-handling`
- `type-assertion`
- `consistency`
- `other`

## Required Fields (per finding)

Every finding in every reviewer output must have:

| Field | Required | Notes |
|-------|----------|-------|
| ID (`<CAT>-NNN`) | yes | reviewer-owned, stable |
| Severity | yes | one of the 5 levels |
| Category | yes | from the category vocabulary |
| Location | yes | `file.go:LINE` format; multiple locations allowed as bullet list |
| Description | yes | what is wrong |
| Evidence | yes | code excerpt, call path, or config snippet |
| Recommendation | yes | concrete change direction |
| Confidence | yes | high / medium / low |

Reviewer-specific additional fields:
- SEC: `Attack Scenario`, `Impact`
- PERF: `Estimated impact`, `Verification`
- ARCH: `Impact`
- STYLE: `Linter` (if applicable), `Codebase-preferred form`

## Confidence Scale

- **high** — direct code evidence; reviewer can cite the concrete path from trigger to consequence.
- **medium** — pattern suggests the issue but reviewer could not fully trace (e.g., hot-path claim without benchmark).
- **low** — speculative or requires runtime data to confirm.

Synthesis should down-weight low-confidence findings in "top findings" but still include them.

## Clustering Rules (for synthesis)

Two findings belong to the same cluster when:
- They share a root cause (fixing one reasonably fixes both), or
- They live in the same function/struct and are symptomatic of the same design choice, or
- They're different angles on the same code (e.g., PERF sees allocation, SEC sees resource exhaustion, ARCH sees the locking structure).

Clusters are **not** merged into single findings — each original finding keeps its ID. The cluster is a narrative grouping in the synthesis report.

## Report Shape (per reviewer)

Every reviewer's output file follows:

```markdown
# <Category> Review

**Scope:** <what was reviewed>
**Date:** <YYYY-MM-DD>
**Reviewer:** <agent name>

## Summary
<3–6 bullets>

## [reviewer-specific section if any, e.g., Threat Model Assumptions for SEC]

## Findings

### <CAT>-001 — <title>
- Severity: ...
- Category: ...
- Location: ...
- Description: ...
- Evidence: ...
- [reviewer-specific fields]
- Recommendation: ...
- Confidence: ...
- Related: <optional; other IDs this relates to>
- Status: <optional; `resolved (verified <YYYY-MM-DD>)` on re-runs>
```

Empty outcome is valid — emit the file with `## Findings\n\n_No issues found in scope._`.

## Machine-Readable Variant (optional)

When the orchestrator asks for JSON, reviewers may also emit `01_<category>_findings.json` alongside the markdown:

```json
{
  "reviewer": "security-reviewer",
  "scope": "protocol/rpcprovider/...",
  "date": "2026-04-16",
  "findings": [
    {
      "id": "SEC-001",
      "severity": "high",
      "category": "signature",
      "location": "protocol/rpcprovider/foo.go:120",
      "description": "...",
      "evidence": "...",
      "attack_scenario": "...",
      "impact": "...",
      "recommendation": "...",
      "confidence": "high",
      "related": ["ARCH-003"],
      "status": null
    }
  ]
}
```

Field naming is snake_case in JSON. Keep parity with markdown.

## Re-invocation Discipline

When a reviewer is re-invoked with feedback:
1. Read prior findings file.
2. Preserve IDs and content for findings still valid. If resolved, set `Status: resolved (verified <YYYY-MM-DD>)` and keep the finding body.
3. For revised findings (e.g., more evidence), edit in place; do NOT add a new ID.
4. For genuinely new findings, use the next sequential ID.
5. NEVER renumber historical findings.

This gives stable references across runs — a reviewer's finding cited in a commit message or issue stays valid forever.
