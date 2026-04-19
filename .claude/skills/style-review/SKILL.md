---
name: style-review
description: "How the style-reviewer checks Go code style, idioms, and project conventions in the Lava protocol layer. Loaded when style-reviewer runs. Covers golangci-lint rules in use (.golangci.yml), naming, error idioms, comment hygiene, test style, nil handling, type assertions, consistency with the existing codebase."
---

# Style Review Skill

Method used by style-reviewer. Role at `.claude/agents/style-reviewer.md`; this file is the *how*.

## Method Overview

1. Read scope from `_workspace/00_input/scope.md`.
2. Load context:
   - `.golangci.yml` — authoritative linter config (enabled + disabled + excludes)
   - `references/checklist.md` in this skill — codebase-specific style rules
3. **Sample the codebase before filing** — for any pattern you want to flag, run a quick grep to confirm the codebase's *dominant* style is the other way.
4. Two-pass review:
   - **Pass 1 — Linter-hit simulation:** Walk through files mentally running the enabled linters. File findings only for hits that would be actual linter errors.
   - **Pass 2 — Idiom drift:** Check for inconsistencies with rest-of-codebase patterns (error wrapping, logging, naming).
5. **Cluster aggressively** — N occurrences of the same issue = 1 finding with a location list.
6. Emit `_workspace/01_style_findings.md`.

## Authoritative Signals

In priority order:

1. **`.golangci.yml` enabled set:**
   - `dogsled`, `copyloopvar`, `gocritic`, `gofumpt`, `gosimple`, `govet`, `ineffassign`, `misspell`, `nakedret`, `nolintlint`, `staticcheck`, `stylecheck`, `typecheck`, `unconvert`, `forcetypeassert`, `gofmt`, `goimports`, `importas`, `nilnil`, `whitespace`
   - `unused` is **disabled** — do not file unused-code findings
2. **`.golangci.yml` excludes — respect them:**
   - `gocritic` excludes: `singleCaseSwitch`, `ifElseChain`
   - `stylecheck` excludes: `ST1003`, `ST1016`
   - `staticcheck` excludes: `SA1019` for `github.com/golang/protobuf/proto`
   - Don't file findings in these rule/pattern spaces.
3. **Codebase's dominant patterns** — verify via `grep`/`rg` before filing.
4. **Effective Go / Go proverbs** — only when the codebase applies them; don't impose.

## Clustering Rule

- Same linter rule + same class of site → one finding, listed locations.
- Different linter rules on the same line → separate findings (different root cause).
- A class of idiom drift (e.g., "uses `fmt.Errorf` where rest uses `utils.LavaFormatError`") → one finding with count and sample sites.

## Severity Heuristics

- **high** — only for readability-breaking patterns that materially confuse readers or bypass observability (e.g., an error swallowed silently instead of `utils.LavaFormatError`).
- **medium** — inconsistencies across >5 sites; error-handling style drift; wrong receiver names; broad interfaces.
- **low** — individual linter hits, single-site naming issues.
- **info** — positive observations or FYI.

Never **critical** — style issues don't break production.

## Non-Findings

- Rules disabled in `.golangci.yml`.
- Code in excluded files (see `.golangci.yml` `issues.exclude-files`).
- Code under migrations/ (already excluded).
- Personal preference without codebase backing.
- Re-filing a linter hit in a file that's currently excluded.

## Output

Per role template. Every finding names the linter (if any) and shows the codebase-preferred form with a sample site from the repo.

## Detailed Checklist

See `references/checklist.md`.
