# lsq Query Phase 3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add intermediate support for stripping simple `{{query ...}}` macros before remote execution, extending the existing Phase 2 path cleanly.

**Architecture:** Phase 3 introduces a small, strict text-normalization step in the CLI prior to delegating to `logseq.DB.q`. It recognizes the exact wrapper `{{query ...}}`, removes it if and only if the inner content matches the valid shapes of a simple DSL, and fails closed otherwise. It explicitly prevents executing unsupported macro inputs or configuration properties.

**Tech Stack:** Go 1.23+, existing `query` CLI command, regex/strings.

---

## Rationale
Why does this precede the local file-backed Phase 4 offline work?
Because it delivers immediate user-visible value by allowing users to paste literal Logseq simple query macros directly into the CLI (`lsq query simple --expr '{{query ...}}'`) without manually trimming the braces. Since Phase 2 effectively proved the remote execution path through `DB.q`, this is a very low-implementation-cost enhancement that smartly leverages the existing HTTP transport before we commit to heavy offline indexing efforts.

## File Map

- Modify: `./cmd/query.go`
  Purpose: add the stripping intercept layer during argument extraction.
- Modify: `./cmd/query_test.go`
  Purpose: validate supported vs ungraceful macro payloads natively returning expected errors or executing.

## Scope Guardrails

Phase 3 includes only text removal of `{{query ...}}` wrappers strictly surrounding simple DSL strings.

Phase 3 explicitly excludes:
- advanced EDN/datalog execution
- `{:query ...}` shapes
- `#+BEGIN_QUERY ... #+END_QUERY` shapes
- `{{query ...}}` wrappers around ungraceful/advanced elements or property modifications.
- `:inputs`, `:result-transform`, `:view`, `:collapsed?`, and all render-layer config properties.
- Generic input normalization layer implementations.
- Local offline File execution (delegated to Phase 4).

---

## Requirements and Rules

1. **Input Recognition**: Match inputs exactly wrapping simple logic: `^{{query\s+(.*?)\s*}}$`.
2. **Stripping Behavior**: Extract the inner content.
3. **Exact Validation**: Provide secondary safeguards validating the parsed interior consists solely of `()`, `[[]]`, or valid simple shapes. Only pass inner structures.
4. **Validation/Error Behavior**: Fail closed. If the extraction looks questionable, includes curly braces, or nested maps, exit cleanly with a non-zero code and actionable error output forbidding mixed syntax. Do not pass it remotely hoping Logseq silently fixes it. Ensure users cannot accidentally trigger advanced features via `query simple`.

## Supported Examples

- `{{query [[logseq]]}}`
- `{{query (task now)}}`
- `{{query (and [[logseq]] #TypeScript)}}`

## Unsupported Examples (Must return validation error)

- `{{query {:query [:find ?b ...]}}}`
- `#+BEGIN_QUERY ... #+END_QUERY`
- `{{query (do-something-else ...)}}` (if falling outside simple DSL subset)
- Any `:inputs` or configuration maps.

## Task 1: Add Macro Stripping Interceptor
**Files:**
- Modify: `./cmd/query.go`
- Modify: `./cmd/query_test.go`

- [ ] **Step 1: Write text normalization failing tests**
Require specific extraction strings dropping `{{query ` and `}}` cleanly while throwing actionable CLI errors for rejected inputs. Provide tests covering supported vs unsupported examples.

- [ ] **Step 2: Run test to verify it fails**
Run: `go test ./cmd -v` (Fails)

- [ ] **Step 3: Implement extraction validation**
Hook strict regex validation early within `runSimple` inside `cmd/query.go` before it invokes `query.RunSimple`. If macro tokens are present, extract them. Apply boundaries rejecting EDN/Config characters explicitly. Pass safe DSL representations along to HTTP.

- [ ] **Step 4: Verify test passes**
Run: `go test ./cmd -v` (Passes)

- [ ] **Step 5: Commit**

## Task 2: Docs and Final Verification
**Files:**
- Modify: `./README.md`
- Modify: `./CHANGELOG.md`

- [ ] **Step 1: Update documentation**
Detail the explicit list of supported `{{query ...}}` wrapper patterns inside README CLI simple instructions. Note error responses specifically for complex map configurations. Update the CHANGELOG.

- [ ] **Step 2: Commit and verify**
Execute the binary locally utilizing `lsq query simple --expr '{{query (task now)}}'` verifying the output successfully returns Logseq `DB.q` resolutions identical to `(task now)`.
