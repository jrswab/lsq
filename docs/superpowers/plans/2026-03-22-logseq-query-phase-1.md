# lsq Query Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Phase 1 Logseq query support to `lsq` by validating the Logseq HTTP API, implementing `query doctor`, and executing advanced queries through `logseq.DB.q` or `logseq.DB.datascriptQuery` with structured output.

**Architecture:** Phase 1 is HTTP-only. It adds a small query command surface on top of the existing CLI, a focused HTTP API client, and stable result formatting. It does not implement a local query engine or file backend fallback.

**Tech Stack:** Go 1.23+, standard library `flag`, `net/http`, `httptest`, existing `config` package, existing integration test helpers.

---

## File Map

- Modify: `/Users/tr/Workspace/logseq/logseq-clis/lsq/main.go`
  Purpose: route `lsq query ...` before existing flat-flag behavior while preserving all current non-query flows.

- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/cmd/query.go`
  Purpose: importable query command entrypoint so tests do not need to call package `main` directly.

- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/result.go`
  Purpose: shared result and doctor output types for `text|json|ndjson`.

- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/router.go`
  Purpose: parse query subcommands and validate arguments, including `--explain`.

- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/backend/httpapi/client.go`
  Purpose: HTTP API transport, auth handling, probe methods, and raw request helpers.

- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/backend/httpapi/execute.go`
  Purpose: doctor probing and advanced-query execution built on top of the HTTP client.

- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/backend/httpapi/client_test.go`
  Purpose: `httptest` coverage for reachability, auth, fallback, malformed responses, and timeouts.

- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/router_test.go`
  Purpose: unit coverage for `query doctor`, `query advanced`, backend parsing, and `--explain`.

- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/cmd/query_test.go`
  Purpose: command-level coverage without involving package `main`.

- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/tests/integration/query_cli_test.go`
  Purpose: CLI-level integration tests using `os/exec`, a mock HTTP server, and the built binary.

- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/tests/integration/legacy_cli_smoke_test.go`
  Purpose: regression coverage for existing non-query CLI behavior after main entrypoint changes.

- Modify: `/Users/tr/Workspace/logseq/logseq-clis/lsq/README.md`
  Purpose: document `query doctor` and `query advanced`.

- Modify: `/Users/tr/Workspace/logseq/logseq-clis/lsq/CHANGELOG.md`
  Purpose: record the new Phase 1 query capability.

## Task 1: Validate the Logseq HTTP API Contract

**Files:**
- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/backend/httpapi/client_test.go`
- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/backend/httpapi/client.go`
- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/backend/httpapi/execute.go`

- [ ] **Step 1: Write the failing probe tests**

Add tests for:
- reachable API + valid `logseq.DB.q`
- `logseq.DB.q` failure with `datascriptQuery` fallback success
- bearer auth missing/invalid
- malformed JSON response
- request timeout

Use `httptest.NewServer` with handlers that inspect:

```go
type apiRequest struct {
    Method string        `json:"method"`
    Args   []interface{} `json:"args"`
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./query/backend/httpapi -run TestClient -v
```

Expected: FAIL because the package and client do not exist yet.

- [ ] **Step 3: Write the minimal HTTP client**

Implement a client with methods equivalent to:

```go
type Client struct {
    BaseURL    string
    Token      string
    HTTPClient *http.Client
}

func (c *Client) ProbeDBQ(ctx context.Context) error
func (c *Client) ProbeDatascriptQuery(ctx context.Context) error
func (c *Client) DoRaw(ctx context.Context, method string, args []any) (json.RawMessage, error)
```

Implement execution helpers in `execute.go`:

```go
func RunDoctor(ctx context.Context, c *Client) (query.DoctorResult, error)
func RunAdvancedQuery(ctx context.Context, c *Client, query string) (query.AdvancedResult, error)
```

Behavior:
- send `POST /api`
- add `Authorization: Bearer <token>` only when configured
- try `logseq.DB.q` first
- fallback to `logseq.DB.datascriptQuery`
- preserve raw response bytes for later formatting

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./query/backend/httpapi -run TestClient -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add query/backend/httpapi/client.go query/backend/httpapi/execute.go query/backend/httpapi/client_test.go
git commit -m "feat: add Logseq HTTP query client"
```

## Task 2: Add Stable Result Types

**Files:**
- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/result.go`
- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/result_test.go`

- [ ] **Step 1: Write the failing result-format tests**

Cover:
- doctor result to `text`
- doctor result to `json`
- doctor result to `ndjson`
- advanced query result to `json`
- advanced query result to `ndjson`
- unsupported format rejected

Target public surface:

```go
func RenderResult(format string, result any) ([]byte, error)
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./query -run TestRenderResult -v
```

Expected: FAIL because result types and renderers do not exist.

- [ ] **Step 3: Write minimal result and doctor models**

Include explicit machine-readable fields from the revised spec:

```go
type DoctorResult struct {
    Backend      string            `json:"backend"`
    Command      string            `json:"command"`
    APIURL       string            `json:"api_url"`
    Reachable    bool              `json:"reachable"`
    Auth         DoctorAuth        `json:"auth"`
    Capabilities DoctorCapabilities `json:"capabilities"`
    Warnings     []string          `json:"warnings"`
    Error        *string           `json:"error"`
}
```

Also define an advanced query result type with:
- `backend`
- `input_kind`
- `query_method`
- `results`
- `warnings`
- `error`

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./query -run 'TestRenderResult|TestParseQueryArgs' -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add query/result.go query/result_test.go
git commit -m "feat: add query result models and renderers"
```

## Task 3: Route `lsq query doctor`

**Files:**
- Modify: `/Users/tr/Workspace/logseq/logseq-clis/lsq/main.go`
- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/cmd/query.go`
- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/cmd/query_test.go`
- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/router.go`
- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/router_test.go`

- [ ] **Step 1: Write the failing router tests**

Cover:
- `lsq query doctor`
- `lsq query doctor --format json`
- `lsq query doctor --backend http`
- unsupported subcommand returns usage error

Use an importable command entrypoint shaped like:

```go
func RunQuery(args []string, stdout io.Writer, stderr io.Writer) int
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./query -run TestRunDoctor -v
```

Expected: FAIL because the command entrypoint and router do not exist.

- [ ] **Step 3: Implement query subcommand dispatch**

Implementation rules:
- detect `os.Args[1] == "query"` before existing flag parsing
- keep current non-query flows unchanged
- parse `doctor`, `advanced`
- accept `--backend auto|http`, `--format`, `--api-url`, `--api-token-env`, `--explain`

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./cmd -run TestRunQueryDoctor -v
go test ./query -run TestRunDoctor -v
go test ./... 
```

Expected: PASS, with existing tests still green.

- [ ] **Step 5: Commit**

```bash
git add main.go cmd/query.go cmd/query_test.go query/router.go query/router_test.go
git commit -m "feat: add query doctor command routing"
```

## Task 4: Implement `lsq query advanced`

**Files:**
- Modify: `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/router.go`
- Modify: `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/router_test.go`
- Modify: `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/backend/httpapi/client.go`

- [ ] **Step 1: Write the failing advanced-query tests**

Cover:
- `--query <raw>`
- `--file <path>`
- `--backend http`
- `--backend auto`
- `--explain`
- missing `--query` and `--file` rejected
- HTTP unhealthy under `auto` returns explicit error instead of silent fallback

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./query -run TestRunAdvanced -v
```

Expected: FAIL because advanced execution path does not exist.

- [ ] **Step 3: Implement advanced passthrough**

Requirements:
- read query from `--query` or `--file`
- probe backend health when `auto`
- execute through `RunAdvancedQuery`
- emit stable `text|json|ndjson`
- include backend/method/warnings in `--explain` text output
- do not add file-backend fallback in Phase 1

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./query -run TestRunAdvanced -v
go test ./...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add query/router.go query/router_test.go query/backend/httpapi/execute.go
git commit -m "feat: add advanced query passthrough"
```

## Task 5: Add CLI Integration Coverage

**Files:**
- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/tests/integration/query_cli_test.go`
- Modify: `/Users/tr/Workspace/logseq/logseq-clis/lsq/tests/integration/integration.go`

- [ ] **Step 1: Write failing CLI integration tests**

Cover:
- full `doctor` command against mock HTTP server
- full `advanced --query` command against mock HTTP server
- JSON output shape assertions

Use `os/exec` to run the built CLI, not direct calls into package `main`.

Use the existing temp-dir helper for env setup, but add any missing helpers for:
- temporary API URL
- temporary token env var
- building the test binary once per package
- capturing stdout/stderr from `exec.Cmd`

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./tests/integration -run TestQueryCLI -v
```

Expected: FAIL because query CLI integration harness and executable test path do not exist.

- [ ] **Step 3: Implement only the missing test helpers**

Keep changes small:
- do not redesign the integration package
- add helper functions only if required by query CLI tests
- prefer one binary-build helper over package-wide refactors

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./tests/integration -run TestQueryCLI -v
go test ./...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tests/integration/integration.go tests/integration/query_cli_test.go
git commit -m "test: add query CLI integration coverage"
```

## Task 6: Add Regression Coverage for Existing CLI Behavior

**Files:**
- Create: `/Users/tr/Workspace/logseq/logseq-clis/lsq/tests/integration/legacy_cli_smoke_test.go`

- [ ] **Step 1: Write the failing regression tests**

Cover at least:
- existing default journal open path still resolves without `query`
- `-c` still prints file content
- `-f` still returns filename/alias search results

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./tests/integration -run TestLegacyCLI -v
```

Expected: FAIL because the regression tests do not exist yet.

- [ ] **Step 3: Implement the minimal smoke coverage**

Use the same built-binary helper from query CLI tests so these checks exercise the real entrypoint.

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./tests/integration -run 'TestQueryCLI|TestLegacyCLI' -v
go test ./...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tests/integration/legacy_cli_smoke_test.go tests/integration/query_cli_test.go tests/integration/integration.go
git commit -m "test: cover legacy CLI after query routing"
```

## Task 7: Document the Feature

**Files:**
- Modify: `/Users/tr/Workspace/logseq/logseq-clis/lsq/README.md`
- Modify: `/Users/tr/Workspace/logseq/logseq-clis/lsq/CHANGELOG.md`

- [ ] **Step 1: Write the failing doc check**

Manually verify the docs are missing:
- `query doctor`
- `query advanced`
- HTTP API prerequisites
- output formats

- [ ] **Step 2: Update README**

Add:
- setup requirements for Logseq HTTP API
- examples for `doctor` and `advanced`
- `--explain`
- `ndjson` output support
- note that Phase 1 is HTTP-only

- [ ] **Step 3: Update CHANGELOG**

Add an Unreleased entry summarizing:
- `query doctor`
- advanced query passthrough
- structured query output

- [ ] **Step 4: Run final verification**

Run:

```bash
go test ./...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add README.md CHANGELOG.md
git commit -m "docs: document query phase 1"
```

## Final Verification

- [ ] **Step 1: Run the full test suite**

```bash
go test ./...
```

Expected: PASS

- [ ] **Step 2: Manual smoke test against a real Logseq instance**

Run:

```bash
lsq query doctor --format json
lsq query advanced --query '[:find ?name :where [?p :block/name ?name]]' --format json
```

Expected:
- `doctor` reports reachability/auth/capabilities
- advanced query returns a machine-readable result envelope

- [ ] **Step 3: Final commit if needed**

```bash
git status --short
```

Expected: clean working tree
