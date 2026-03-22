# lsq Query Phase 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Phase 2 Logseq remote simple query support to `lsq` by routing raw simple DSL expressions exclusively to `logseq.DB.q` across the active HTTP backend.

**Architecture:** Phase 2 explicitly leverages the existing HTTP-backend connection mapped rigorously in Phase 1. `lsq query simple` directly marshals simple DSL commands directly into the `logseq.DB.q` execution endpoint.

**Tech Stack:** Go 1.23+, existing `router`, `httpapi` mechanisms from Phase 1.

---

## File Map

- Modify: `./cmd/query.go`
  Purpose: Parse `simple` command flags targeting default remote HTTP operations.
- Modify: `./query/router.go`
  Purpose: Route `simple` query invocations passing raw input downward. 
- Modify: `./query/backend/httpapi/execute.go`
  Purpose: Expose `RunSimpleQuery` natively mapping executions directly targeting `logseq.DB.q`.
- Create: `./tests/integration/query_simple_cli_test.go`
  Purpose: Command regression smoke tests.

## Scope Guardrails

Phase 2 includes only:
- `lsq query simple --expr '...'` execution
- Execution mapped explicitly targeting Logseq HTTP API via `logseq.DB.q`.
- Valid expressions cleanly supported: `[[logseq]]`, `(and [[logseq]] #TypeScript)`, `(task now)`, `(page-property type project)`.

Phase 2 explicitly excludes:
- `{{query ...}}` generic parsing or macro stripping.
- Advanced EDN/Datalog map parsing.
- Offline/Local File backend operations (moved to Phase 4).
- Org parsing.

---

## Task 1: Add Remote `RunSimpleQuery` execution
**Files:**
- Modify: `./query/backend/httpapi/execute.go`
- Modify: `./query/backend/httpapi/client_test.go`

- [ ] **Step 1: Write failing client isolation tests**
Test `RunSimpleQuery` explicitly targeting `logseq.DB.q`. Require explicit error tracking for failure scopes.

- [ ] **Step 2: Run test to verify it fails**
Run: `go test ./query/backend/httpapi -v` (Fails)

- [ ] **Step 3: Implement `RunSimpleQuery`**
Add native `RunSimpleQuery` executing the raw simple DSL string against `logseq.DB.q` natively returning standard JSON payload envelopes structurally aligned identically to advanced targets. 

- [ ] **Step 4: Run test to verify it passes**
Run: `go test ./query/backend/httpapi -v` (Passes)

- [ ] **Step 5: Commit**

## Task 2: Route `lsq query simple` HTTP passthrough
**Files:**
- Modify: `./cmd/query.go`
- Modify: `./cmd/query_test.go`
- Modify: `./query/router.go`
- Modify: `./query/router_test.go`

- [ ] **Step 1: Write routing tests validating flags**
Require `--expr` validation. Validate parsing falls back gracefully to `http` logic since `file` backend is strictly excluded.

- [ ] **Step 2: Run test to verify fails**
Run: `go test ./cmd -v` (Fails)

- [ ] **Step 3: Build CLI dispatching logic**
Update `cmd/query.go` mapping `simple` -> HTTP parser.

- [ ] **Step 4: Verify test passes**
Run `go test ./cmd -v`

- [ ] **Step 5: Commit**

## Task 3: Build Smoke Integrations
**Files:**
- Modify: `./tests/integration/query_simple_cli_test.go`

- [ ] **Step 1: Define explicit JSON output bounds**
Utilize HTTP mock server verifying `simple --expr '(task now)' --format json` directly calls `DB.q` returning normalized success envelopes.

- [ ] **Step 2: Execute assertions**
Run `go test ./tests/integration -v` validating success mappings. Commit changes.

## Task 4: Document Final Phase Bounds
**Files:**
- Modify: `./README.md`
- Modify: `./CHANGELOG.md`

- [ ] **Step 1: Write Docs**
Expose `query simple` targeting remote execution manually detailing exclusions targeting embedded macro payloads unstripped. Provide raw simple valid targets. Ensure `CHANGELOG` surfaces HTTP bindings explicitly. Execute tests. Commit documentation payloads natively.

## Final Verification
Run `go test ./...`. Manually deploy `lsq query simple --expr '(task now)'` evaluating Logseq execution yields locally.
