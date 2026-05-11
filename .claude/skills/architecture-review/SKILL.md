---
name: architecture-review
description: "How the architecture-reviewer performs a structural review of the Lava protocol layer. Loaded when architecture-reviewer is invoked by the protocol-review orchestrator. Covers module boundaries, dependency direction, interface design, concurrency architecture, error model, request-path shape, and testability."
---

# Architecture Review Skill

Method used by architecture-reviewer. Keep this lean — the role file at `.claude/agents/architecture-reviewer.md` holds the persona and output contract. This file holds the *how*.

## Method Overview

1. Read the scope from `_workspace/00_input/scope.md`.
2. Read `.claude/skills/protocol-review/references/protocol-map.md` once to load topology into working memory.
3. Pass over the scope with the **structural lens** — not line-by-line, but package-by-package, then call-site-by-call-site.
4. Per package in scope, run the architecture checklist (see `references/checklist.md`).
5. Produce findings per taxonomy (`.claude/skills/protocol-review/references/review-taxonomy.md`).
6. Emit `_workspace/01_architecture_findings.md`.

## Two-Pass Strategy

**Pass 1 — Topology scan (30% of budget).**
- Walk the scope's package dependency graph using `go list -deps ./...` output or manual import tracing.
- Ask: does the dependency direction make sense given the layer diagram in protocol-map? Any upward imports? Any cycles?
- Ask: is each package's public surface minimal relative to its internal complexity?

**Pass 2 — Per-package inspection (70% of budget).**
- For each package, read the public types and their methods.
- Skim one-level-deep into private implementations when a public method hints at complex state (e.g., a `Start`/`Stop` pair → check goroutine lifecycle).
- Run the checklist (see references).

## What NOT to Do

- Don't line-comment on style — style-reviewer owns that.
- Don't flag perf unless it's *structural* (e.g., architecture forces a hot-path allocation that couldn't be fixed without restructuring).
- Don't flag security unless it's *structural* (e.g., trust boundary drawn in the wrong place).
- Don't propose rewrites without concrete reasons.

## Output

Per the role's template. See `.claude/agents/architecture-reviewer.md` § Input / Output Protocol.

## Detailed Checklist

See `references/checklist.md` for the structured list of things to check per package and per topic. Use it as a rubric — tick off as you review, file findings where the rubric fails.
