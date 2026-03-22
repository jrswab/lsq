# lsq Query Phase 4 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a local file-backed `lsq query simple` command that executes a tiny DSL over Markdown Logseq files without depending on the HTTP API.

**Architecture:** Phase 4 adds a restricted local query path alongside the existing HTTP-backed `doctor` and `advanced` commands. The implementation is intentionally narrow: a simple DSL parser compiles into a tiny local plan model, a Markdown-only file parser builds a minimal block index, and a file executor evaluates the plan against that index. Remote simple-DSL support through `logseq.DB.q`, macro-stripping, Org support, and advanced-query compilation remain out of scope.

**Tech Stack:** Go 1.23+, standard library `flag`, `filepath`, `os`, `strings`, `regexp`, existing `cmd/query.go`, existing `query/result.go`, existing integration helpers under `./tests/integration`.

---

## File Map

- Modify: `./cmd/query.go`
  Purpose: add `query simple`, `--expr`, and `--backend file` validation while preserving existing `doctor` and `advanced` behavior.

- Modify: `./cmd/query_test.go`
  Purpose: command-level coverage for `query simple` argument parsing and result rendering.

- Modify: `./query/result.go`
  Purpose: add a machine-readable result type for local simple-query execution and render support for `text|json|ndjson`.

- Modify: `./query/result_test.go`
  Purpose: coverage for the new simple result format in `text|json|ndjson`.

- Create: `./query/types.go`
  Purpose: shared local query types such as `QueryPlan`, `Filter`, `FieldRef`, and `FilterOp`.

- Create: `./query/parser/simple.go`
  Purpose: tokenize and parse the local simple DSL into a small AST or intermediate form.

- Create: `./query/parser/simple_test.go`
  Purpose: parser coverage for valid expressions and actionable syntax errors.

- Create: `./query/compile/simple_to_plan.go`
  Purpose: compile parsed simple expressions into the narrow local `QueryPlan`.

- Create: `./query/compile/simple_to_plan_test.go`
  Purpose: verify field/operator mapping and unsupported-shape rejection.

- Create: `./query/backend/file/model.go`
  Purpose: define the minimal local `Block` and `Index` structures used by the file backend.

- Create: `./query/backend/file/parser_markdown.go`
  Purpose: parse Markdown pages and journals into minimal block records with hierarchy, refs, tags, and markers.

- Create: `./query/backend/file/parser_markdown_test.go`
  Purpose: extraction coverage for markers, refs, tags, nesting, and page names.

- Create: `./query/backend/file/index.go`
  Purpose: walk `pages/` and `journals/`, parse files, and build the in-memory local index.

- Create: `./query/backend/file/index_test.go`
  Purpose: verify graph discovery, ordering, and page/source metadata in the index.

- Create: `./query/backend/file/execute.go`
  Purpose: execute a compiled `QueryPlan` against the local index and return stable result rows.

- Create: `./query/backend/file/execute_test.go`
  Purpose: execution coverage for ref/tag/marker/content filters and logical combinations.

- Create: `./tests/integration/query_simple_cli_test.go`
  Purpose: built-binary integration tests for `lsq query simple`.

- Modify: `./README.md`
  Purpose: document the local `query simple` surface and its limitations.

- Modify: `./CHANGELOG.md`
  Purpose: record the new local simple-query capability.

## Scope Guardrails

Phase 4 includes only:

- `lsq query simple --expr '...' --backend file`
- Markdown-backed local execution
- a restricted DSL over:
  - `ref:<page-name>`
  - `tag:<tag-name>`
  - `marker:TODO|DOING|DONE`
  - `text:"..."`
  - `and`
  - `or`
  - `not`

Phase 4 explicitly excludes:

- remote simple query execution through `logseq.DB.q`
- `{{query ...}}` or macro stripping
- advanced-query compilation
- Org parsing
- `scheduled`, `deadline`, date arithmetic, or page properties
- markers beyond `TODO|DOING|DONE`

## Task 1: Add Result Model For Local Simple Queries

**Files:**
- Modify: `./query/result.go`
- Modify: `./query/result_test.go`

- [ ] **Step 1: Write the failing result-format tests**

Add tests for a new result shape:

```go
func TestRenderResult_SimpleJSON_Success(t *testing.T) {
    res := query.SimpleResult{
        Backend:   "file",
        InputKind: "simple",
        Results: []query.SimpleRow{
            {Page: "project-x", Content: "TODO review parser", Marker: "TODO"},
        },
        Warnings: []string{},
    }
    out, err := query.RenderResult("json", res)
    if err != nil {
        t.Fatal(err)
    }
    if !strings.Contains(string(out), `"backend":"file"`) {
        t.Fatalf("unexpected output: %s", out)
    }
}
```

Also cover:
- `text`
- `ndjson`
- nil warnings normalize to `[]`
- pointer receiver support

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./query -run TestRenderResult_Simple -v
```

Expected: FAIL because `SimpleResult` and its renderer support do not exist.

- [ ] **Step 3: Write minimal result model**

Add a result model equivalent to:

```go
type SimpleRow struct {
    Page    string `json:"page"`
    File    string `json:"file"`
    Content string `json:"content"`
    Marker  string `json:"marker,omitempty"`
    Refs    []string `json:"refs,omitempty"`
    Tags    []string `json:"tags,omitempty"`
}

type SimpleResult struct {
    Backend   string      `json:"backend"`
    InputKind string      `json:"input_kind"`
    Results   []SimpleRow `json:"results"`
    Warnings  []string    `json:"warnings"`
    Error     *string     `json:"error"`
}
```

Update `RenderResult` so:
- `json` emits one result envelope
- `ndjson` emits one row per result when there are rows and no warnings
- `text` prints one bullet-like line per row

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./query -run TestRenderResult_Simple -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add query/result.go query/result_test.go
git commit -m "feat: add local simple query result model"
```

## Task 2: Parse The Simple DSL

**Files:**
- Create: `./query/parser/simple.go`
- Create: `./query/parser/simple_test.go`

- [ ] **Step 1: Write the failing parser tests**

Cover at least:

```go
func TestParseSimple_RefAndMarker(t *testing.T) {
    expr, err := parser.ParseSimple(`ref:project-x and marker:TODO`)
    if err != nil {
        t.Fatal(err)
    }
    if expr == nil {
        t.Fatal("expected expression")
    }
}

func TestParseSimple_TagAndNot(t *testing.T) {
    expr, err := parser.ParseSimple(`tag:reading and not marker:DONE`)
    if err != nil {
        t.Fatal(err)
    }
    if expr == nil {
        t.Fatal("expected expression")
    }
}

func TestParseSimple_InvalidSyntax(t *testing.T) {
    _, err := parser.ParseSimple(`and marker:TODO`)
    if err == nil {
        t.Fatal("expected syntax error")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./query/parser -run TestParseSimple -v
```

Expected: FAIL because the parser package does not exist.

- [ ] **Step 3: Write the minimal parser**

Implement a small parser that recognizes only:
- `ref:<name>`
- `tag:<name>`
- `marker:TODO|DOING|DONE`
- `text:"..."`
- `and`
- `or`
- `not`

Recommended AST:

```go
type Expr interface{}

type Term struct {
    Kind  string
    Value string
}

type Unary struct {
    Op   string
    Expr Expr
}

type Binary struct {
    Op    string
    Left  Expr
    Right Expr
}
```

Use a tiny tokenizer plus precedence rules:
- `not` binds tightest
- `and` binds tighter than `or`

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./query/parser -run TestParseSimple -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add query/parser/simple.go query/parser/simple_test.go
git commit -m "feat: add simple query parser"
```

## Task 3: Compile Parsed Expressions Into A Local QueryPlan

**Files:**
- Create: `./query/types.go`
- Create: `./query/compile/simple_to_plan.go`
- Create: `./query/compile/simple_to_plan_test.go`

- [ ] **Step 1: Write the failing compile tests**

Cover:

```go
func TestCompileSimple_RefAndMarker(t *testing.T) {
    expr, _ := parser.ParseSimple(`ref:project-x and marker:TODO`)
    plan, err := compile.SimpleToPlan(expr, `ref:project-x and marker:TODO`)
    if err != nil {
        t.Fatal(err)
    }
    if plan.InputKind != "simple" {
        t.Fatalf("unexpected input kind: %s", plan.InputKind)
    }
}
```

Also verify:
- `tag:reading`
- `text:"distributed systems"`
- `not marker:DONE`

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./query/compile -run TestCompileSimple -v
```

Expected: FAIL because the compile package does not exist.

- [ ] **Step 3: Write the minimal plan model and compiler**

Add a narrow plan model:

```go
type QueryPlan struct {
    Target    string
    Filters   []Filter
    RawInput  string
    InputKind string
}

type Filter struct {
    Op       string
    Field    string
    Value    string
    Children []Filter
}
```

Compiler mapping:
- `ref:x` -> `contains` on `block.refs`
- `tag:x` -> `contains` on `block.tags`
- `marker:x` -> `eq` on `block.marker`
- `text:x` -> `contains` on `block.content`

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./query/compile -run TestCompileSimple -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add query/types.go query/compile/simple_to_plan.go query/compile/simple_to_plan_test.go
git commit -m "feat: compile simple DSL to local query plan"
```

## Task 4: Parse Markdown Files Into Minimal Blocks

**Files:**
- Create: `./query/backend/file/model.go`
- Create: `./query/backend/file/parser_markdown.go`
- Create: `./query/backend/file/parser_markdown_test.go`

- [ ] **Step 1: Write the failing markdown parser tests**

Cover:

```go
func TestParseMarkdown_ExtractsMarkerRefsTags(t *testing.T) {
    src := "- TODO review [[project-x]] #reading\n\t- child block\n"
    blocks, err := file.ParseMarkdown("Page Name", "/fake/path/project-x.md", src)
    if err != nil {
        t.Fatal(err)
    }
    if len(blocks) != 2 {
        t.Fatalf("expected 2 blocks, got %d", len(blocks))
    }
    if blocks[0].Marker != "TODO" || blocks[0].FilePath != "/fake/path/project-x.md" {
        t.Fatalf("unexpected block properties extracted")
    }
}
```

Also cover:
- parent/child relationship
- `DOING`
- `DONE`
- multiple refs
- multiple tags

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./query/backend/file -run TestParseMarkdown -v
```

Expected: FAIL because the file backend package does not exist.

- [ ] **Step 3: Write the minimal block model and Markdown parser**

Use a minimal model:

```go
type Block struct {
    ID       string
    Content  string
    Marker   string
    Tags     []string
    Refs     []string
    PageName string
    FilePath string
    Parent   string
    Children []string
    Order    int
}
```

Scope:
- Markdown bullets only
- indentation by tabs or 4-space groups
- marker extraction at block start
- `[[page refs]]`
- `#tags`

Do not parse page properties, deadlines, or timestamps.

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./query/backend/file -run TestParseMarkdown -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add query/backend/file/model.go query/backend/file/parser_markdown.go query/backend/file/parser_markdown_test.go
git commit -m "feat: parse markdown blocks for local query backend"
```

## Task 5: Build The Local File Index

**Files:**
- Create: `./query/backend/file/index.go`
- Create: `./query/backend/file/index_test.go`

- [ ] **Step 1: Write the failing index tests**

Cover:
- walks `pages/` and `journals/`
- preserves page name and file path
- stores blocks in stable order

Example:

```go
func TestBuildIndex_LoadsPagesAndJournals(t *testing.T) {
    root := t.TempDir()
    mustWriteFile(t, filepath.Join(root, "pages", "project-x.md"), "- TODO one\n")
    mustWriteFile(t, filepath.Join(root, "journals", "2026_03_22.md"), "- DONE two\n")
    idx, err := file.BuildIndex(root)
    if err != nil {
        t.Fatal(err)
    }
    if len(idx.Blocks) != 2 || idx.Blocks[0].FilePath == "" {
        t.Fatalf("expected 2 blocks with file path populated")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./query/backend/file -run TestBuildIndex -v
```

Expected: FAIL because the index builder does not exist.

- [ ] **Step 3: Write the minimal index builder**

Define an index like:

```go
type Index struct {
    Blocks []Block
}
```

Behavior:
- walk `pages/` and `journals/`
- skip non-Markdown files in Phase 4
- derive `PageName` from filename without extension
- parse file contents through `ParseMarkdown`

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./query/backend/file -run TestBuildIndex -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add query/backend/file/index.go query/backend/file/index_test.go
git commit -m "feat: build local markdown query index"
```

## Task 6: Execute Local Query Plans Against The Index

**Files:**
- Create: `./query/backend/file/execute.go`
- Create: `./query/backend/file/execute_test.go`

- [ ] **Step 1: Write the failing execution tests**

Cover:
- `ref:project-x`
- `tag:reading`
- `marker:TODO`
- `text:"parser"`
- `and`
- `or`
- `not`
- empty result set

Example:

```go
func TestRunPlan_MarkerAndRef(t *testing.T) {
    idx := file.Index{
        Blocks: []file.Block{
            {PageName: "project-x", Content: "TODO review parser", Marker: "TODO", Refs: []string{"project-x"}},
            {PageName: "reading", Content: "DONE write summary", Marker: "DONE"},
        },
    }
    plan := query.QueryPlan{
        Target:    "blocks",
        InputKind: "simple",
        Filters: []query.Filter{
            {Op: "and", Children: []query.Filter{
                {Op: "contains", Field: "block.refs", Value: "project-x"},
                {Op: "eq", Field: "block.marker", Value: "TODO"},
            }},
        },
    }
    rows, err := file.RunPlan(plan, idx)
    if err != nil {
        t.Fatal(err)
    }
    if len(rows) != 1 || rows[0].File == "" {
        t.Fatalf("expected 1 row with File populated")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./query/backend/file -run TestRunPlan -v
```

Expected: FAIL because the executor does not exist.

- [ ] **Step 3: Write the minimal executor**

Add:

```go
func RunPlan(plan query.QueryPlan, idx Index) ([]query.SimpleRow, error)
```

Implement only:
- `eq`
- `contains`
- `and`
- `or`
- `not`

Matching rules:
- `block.refs` and `block.tags` use exact string membership
- `block.content` uses case-insensitive substring match
- `block.marker` uses exact normalized match

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./query/backend/file -run TestRunPlan -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add query/backend/file/execute.go query/backend/file/execute_test.go
git commit -m "feat: execute local simple query plans"
```

## Task 7: Route `lsq query simple`

**Files:**
- Modify: `./cmd/query.go`
- Modify: `./cmd/query_test.go`

- [ ] **Step 1: Write the failing command tests**

Cover:
- `lsq query simple --expr 'ref:project-x and marker:TODO' --backend file --format json`
- missing `--expr`
- unsupported backend (`http`) for simple
- invalid DSL returns non-zero

Example:

```go
func TestRunQuery_SimpleJSON(t *testing.T) {
    var stdout, stderr bytes.Buffer
    code := cmd.RunQuery([]string{
        "simple",
        "--graph-dir", "/tmp",
        "--expr", `ref:project-x and marker:TODO`,
        "--backend", "file",
        "--format", "json",
    }, &stdout, &stderr)
    if code != 0 {
        t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./cmd -run TestRunQuery_Simple -v
```

Expected: FAIL because the `simple` subcommand does not exist.

- [ ] **Step 3: Write the minimal CLI routing**

Update `cmd/query.go` so:
- new subcommand: `simple`
- required flag: `--expr`
- directory flag: `--graph-dir` (provides a query-specific graph target override, bypassing top-level `-d` intercept boundaries cleanly)
- only `--backend file` is accepted in Phase 4 for `simple`
- graph root resolution: strictly parse `--graph-dir` if present, otherwise gracefully fallback loading `config.Load`.

Implementation detail:
- parse the `--graph-dir` flag dynamically during router initialization.
- add a helper that cascades config extraction deriving the required graph root for the file backend
- parse DSL -> compile plan -> build index -> execute plan -> render `SimpleResult`

If no graph path can be resolved, return a clear CLI error explicitly acknowledging the lack of valid `--graph-dir` or local fallback config.

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./cmd -run TestRunQuery_Simple -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/query.go cmd/query_test.go
git commit -m "feat: add query simple command"
```

## Task 8: Add Built-Binary Integration Coverage For `query simple`

**Files:**
- Create: `./tests/integration/query_simple_cli_test.go`

- [ ] **Step 1: Write the failing integration tests**

Use the built binary from `./tests/integration/integration.go`.

Cover:
- successful JSON query against a temp Markdown graph
- successful text query
- invalid DSL
- no matches returns empty results

Example:

```go
func TestQuerySimpleCLI_JSON(t *testing.T) {
    helper := NewTestHelper(t)
    defer helper.Cleanup()
    mustWriteConfig(t, helper, helper.LogseqDir)
    mustWriteFile(t, filepath.Join(helper.PagesDir, "project-x.md"), "- TODO review [[project-x]] #reading\n")

    res := RunCLI(lsqBinary, []string{
        "query", "simple",
        "--graph-dir", helper.LogseqDir,
        "--expr", "ref:project-x and marker:TODO",
        "--backend", "file",
        "--format", "json",
    })

    if res.ExitCode != 0 {
        t.Fatalf("exit %d: %s", res.ExitCode, res.Stderr)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./tests/integration -run TestQuerySimpleCLI -v
```

Expected: FAIL because the integration tests do not exist yet.

- [ ] **Step 3: Implement only missing test helpers**

If required, add tiny helpers to `./tests/integration/integration.go` for:
- writing config
- writing files in temp graphs

Do not refactor the integration package broadly.

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./tests/integration -run 'TestQueryCLI|TestLegacyCLI|TestQuerySimpleCLI' -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tests/integration/query_simple_cli_test.go tests/integration/integration.go
git commit -m "test: add query simple integration coverage"
```

## Task 9: Document The New Local Query Surface

**Files:**
- Modify: `./README.md`
- Modify: `./CHANGELOG.md`

- [ ] **Step 1: Write the failing doc check**

Manually verify the docs are missing:
- `lsq query simple`
- local-file backend requirement
- `--graph-dir` override flag and fallback behaviors
- supported simple DSL terms
- Phase 4 limitations

- [ ] **Step 2: Update README**

Add:
- `lsq query simple --expr '...' --backend file`
- an example utilizing the directory flag: `lsq query simple --graph-dir ~/my-graph --expr '...' --backend file`
- explanation defining what `--graph-dir` is for: overriding the graph path specific to simple queries, defaulting securely back to the configured graph path natively if omitted from the prompt.
- examples for `ref`, `tag`, `marker`, and `text`
- note that Phase 4 is Markdown-only and local-file-backed
- explicit exclusions:
  - no `{{query ...}}`
  - no simple remote `DB.q`
  - no Org support

- [ ] **Step 3: Update CHANGELOG**

Add an Unreleased entry summarizing:
- local `query simple`
- Markdown-backed file query execution
- restricted local DSL support

- [ ] **Step 4: Run verification**

Run:

```bash
go test ./...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add README.md CHANGELOG.md
git commit -m "docs: document local simple query support"
```

## Final Verification

- [ ] **Step 1: Run the full test suite**

```bash
go test ./...
```

Expected: PASS

- [ ] **Step 2: Manual smoke test on a local graph**

Run:

```bash
lsq query simple --graph-dir ~/my-graph --expr 'marker:TODO' --backend file --format json
lsq query simple --graph-dir ~/my-graph --expr 'ref:project-x and not marker:DONE' --backend file --format text
```

Expected:
- command exits `0`
- result envelope is machine-readable in JSON
- text output shows matching rows only

- [ ] **Step 3: Confirm no unintended HTTP dependency**

Run:

```bash
lsq query simple --graph-dir ~/my-graph --expr 'tag:reading' --backend file --format json
```

Expected:
- no requirement for Logseq desktop or HTTP API
- local graph files are sufficient

- [ ] **Step 4: Final commit if needed**

```bash
git status --short
```

Expected: clean working tree
