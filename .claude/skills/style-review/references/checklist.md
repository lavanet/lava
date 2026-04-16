# Style Review Checklist

Detailed rubric for style-reviewer. Grounded in `.golangci.yml` + codebase conventions.

## Table of Contents

1. [Linter Coverage](#1-linter-coverage)
2. [Naming](#2-naming)
3. [Error Idioms](#3-error-idioms)
4. [Comments](#4-comments)
5. [Package Layout](#5-package-layout)
6. [Imports](#6-imports)
7. [Test Style](#7-test-style)
8. [Nil Handling](#8-nil-handling)
9. [Type Assertions](#9-type-assertions)
10. [Consistency with Codebase](#10-consistency-with-codebase)

---

## 1. Linter Coverage

Simulate each enabled linter from `.golangci.yml` (respecting excludes):

### `gofumpt` — stricter formatter
- Consecutive blank lines collapsed.
- No space after `func (`.
- Standard composite literals use field names or don't (consistently).

### `gofmt` / `goimports`
- Standard Go formatting.
- Imports grouped: stdlib, external, internal; ordered alphabetically within group.

### `gocritic` (respecting excludes `singleCaseSwitch`, `ifElseChain`)
- No named returns just to skip a line.
- No `if x == nil` where the type is guaranteed non-nil.
- No unnecessary deref in receiver.

### `gosimple`
- `if err != nil { return err } return nil` → `return err`.
- `len(m) == 0` for map existence where `_, ok := m[k]` is meant.

### `staticcheck` (respecting `SA1019` for golang/protobuf)
- Deprecated APIs other than explicitly-allowed ones.

### `stylecheck` (respecting `ST1003`, `ST1016`)
- `ST1000`: package has doc (low priority on existing files).
- `ST1005`: error strings don't start with capital or end with punctuation.

### `govet`
- Shadowed declarations that actually change meaning.
- Bad struct tags.

### `ineffassign`
- `x := 5; x = 6` without reading `x = 5`.

### `misspell`
- Common English misspellings.

### `copyloopvar` (Go 1.22+)
- `for _, v := range xs { go use(v) }` — ensure no loopvar bug (Go 1.22+ copy semantics help but audit explicit).

### `forcetypeassert`
- `x := y.(T)` without `, ok` — forbidden. Must be `x, ok := y.(T); if !ok { ... }`.

### `nilnil`
- `return nil, nil` forbidden where the caller can't disambiguate.

### `nakedret`
- Naked returns only in very short functions.

### `importas`
- Aliases match project policy (check `.golangci.yml` importas config if defined).

### `whitespace`
- No trailing whitespace; no leading blank line after `{`.

### `typecheck`
- Doesn't compile → not a style issue; compile errors surface separately.

### `nolintlint`
- Every `//nolint` directive has a reason and specific rules; no blanket suppression.

## 2. Naming

- **Exported identifiers:** start with capital, CamelCase. Acronyms: `URL`, `ID`, `RPC`, `API`, `JSON`, `HTTP`, `TLS` stay all-caps.
- **Receivers:** short, consistent across methods of a struct (`c *Consumer`, not `c1` / `this` / `self`).
- **Variable scope proportional to length:** `i` in loops, `err` universally, descriptive in long scopes.
- **Interfaces:** `-er` suffix where it reads naturally (`Signer`, `Validator`); noun where it doesn't (`Session`, `Pairing`).
- **Test functions:** `TestXxx` for public tests; subtest names as full sentences in snake_case via `t.Run`.

## 3. Error Idioms

**Project helper:** `utils.LavaFormatError`, `utils.LavaFormatWarning`, `utils.LavaFormatInfo`, `utils.LavaFormatDebug`.

Dominant pattern in the codebase:

```go
return utils.LavaFormatError("failed to verify reply", err,
    utils.Attribute{Key: "chainID", Value: chainID},
    utils.Attribute{Key: "provider", Value: provider},
)
```

Flag:
- `fmt.Errorf(...)` in files that otherwise use the project helper.
- `errors.New("failed X")` where caller branches on the error → should be a typed sentinel.
- `err.Error()` string comparisons — use `errors.Is` / `errors.As`.
- Missing `%w` when wrapping with `fmt.Errorf`.
- Error attributes that don't include useful context (chainID, provider, epoch, session).

## 4. Comments

- Exported identifiers SHOULD have godoc-style comment starting with the identifier name. Don't file against every missing doc — prioritize:
  - New exports in this diff
  - Public APIs across package boundaries
- Flag misleading or stale comments more aggressively than missing comments.
- Flag code that's commented-out in place of a decision (should either be deleted or have a reason).

## 5. Package Layout

- Package name = directory name.
- One primary type per file usually; related types can share a file.
- `internal/` for non-exported dependencies.
- Test files in the same package (`package foo`) or `_test` package when black-box.
- No "util.go" / "helpers.go" if the file contains unrelated things — split.

## 6. Imports

- Group order: stdlib, external, internal.
- Alphabetical within group.
- No unused imports (compiler catches).
- Aliases only when necessary (name conflict or ambiguous); match project's `importas` config.

## 7. Test Style

- Table-driven tests are the dominant pattern:
  ```go
  tests := []struct{
      name string
      input ...
      wantErr bool
  }{ ... }
  for _, tt := range tests {
      t.Run(tt.name, func(t *testing.T) { ... })
  }
  ```
- Assertion library: whatever the package already uses (typically `testify/require` or `testify/assert`). Don't mix.
- `t.Parallel()` on tests with no shared state.
- `t.Helper()` in helper functions that call assertion methods.
- No real network / real DB in unit tests — use fakes.
- No `time.Sleep` to wait — use `require.Eventually` or channels.

## 8. Nil Handling

- `nilnil` rule: don't `return nil, nil`. If the second value can be nil without meaning an error, restructure.
- `nil` checks before method calls on interface variables where the interface could hold a typed nil.
- Guard against nil receiver where the method is exported and may be called via interface.

## 9. Type Assertions

- `forcetypeassert` rule: always use `x, ok := y.(T); if !ok { ... }`.
- For type switches, default arm handles unexpected types explicitly (no silent pass-through).
- Don't assert inside hot loops — hoist the assertion out.

## 10. Consistency with Codebase

- Check dominant pattern before filing. Example greps:
  - `rg "utils\.LavaFormatError\(" protocol | wc -l` vs `rg "fmt\.Errorf\(" protocol | wc -l`
  - `rg "var " protocol` for var-block style
  - `rg "^func \([a-z] \*" protocol` for receiver-name conventions
- If the pattern is split ~50/50, it's ambiguous; don't file.
- If one side clearly dominates (>5×), the minority occurrences are candidates.

---

## Reporting Pattern

When filing:

```markdown
### STYLE-003 — fmt.Errorf used where utils.LavaFormatError is the codebase standard
- Severity: medium
- Category: error-idiom
- Linter: (none — idiom drift)
- Locations:
  - `protocol/rpcprovider/foo.go:55`
  - `protocol/rpcprovider/foo.go:112`
  - `protocol/chainlib/bar.go:88`
  - ... (12 total)
- Description: These sites use `fmt.Errorf(...)` to return errors. The rest of the `protocol/` codebase overwhelmingly uses `utils.LavaFormatError(msg, err, attrs...)` (rg count: 1400 vs 85).
- Evidence: (excerpt from foo.go:55)
- Codebase-preferred form (from `protocol/rpcconsumer/consumer.go:210`):
    `utils.LavaFormatError("failed to x", err, utils.Attribute{Key:"chainID", Value:chainID})`
- Recommendation: Migrate the 12 listed sites. Add a CI grep to catch future drift (e.g., `rg "fmt.Errorf" protocol` count should stay in check).
- Confidence: high
```
