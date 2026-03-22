# lsq Logseq Query Design

Date: 2026-03-22
Repo: `/Users/tr/Workspace/logseq/logseq-clis/lsq`
Status: Proposed

## Goal

Extend `lsq` from a file-oriented Logseq CLI into a query-capable CLI that can:

- execute real Logseq advanced queries when the Logseq HTTP API is available
- execute a useful subset of Logseq-style queries against local `pages/` and `journals/` files when HTTP API access is unavailable
- preserve existing `lsq` behavior for journal/page open, append, and search operations

This design deliberately separates:

- input compatibility with Logseq query syntax
- internal query planning
- execution backends

That separation is required because `lsq` currently has no graph database, no block model, and no Datalog engine.

## Non-goals

The first implementation will not:

- fully reimplement Logseq's complete DataScript/Datalog engine
- guarantee full parity between file graphs and DB graphs
- support every advanced query feature such as `:rules`, arbitrary `:in`, or custom Clojure predicates
- replace current `-f` and `-r` behavior

## Current State

`lsq` currently supports:

- open/create journal files
- open pages by filename
- append bullet content to journals/pages
- print page/journal content
- search page filenames and `alias::`
- regex scan raw file contents

Relevant code:

- `/Users/tr/Workspace/logseq/logseq-clis/lsq/main.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/trie/trie.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/system/journal.go`

`lsq` does not currently parse block structure, page properties, refs, tags, task markers, or advanced query syntax.

## Product Direction

The recommended direction is a dual-backend query system:

1. `http` backend
Calls Logseq's local HTTP API and uses native query capabilities such as `logseq.DB.q` and `logseq.DB.datascriptQuery` when available.

2. `file` backend
Builds a local index from `pages/` and `journals/` and executes a restricted subset of query semantics locally.

3. `auto` backend
Prefers HTTP for advanced query execution and falls back to the file backend only when compatible.

This gives `lsq` a fast path to real Logseq Query support without blocking on a full local engine.

## CLI Design

Introduce a new subcommand family instead of overloading the current flat flag interface.

Examples:

```bash
lsq query doctor
lsq query simple --expr '[[project-x]] and marker:TODO'
lsq query simple --expr '#reading and journal:true' --backend file --format json
lsq query advanced --query '[:find (pull ?b [*]) :where [?b :block/marker "TODO"]]'
lsq query advanced --file ./query.edn --backend http
lsq query auto --expr '{{query (and [[project-x]] (task todo))}}' --explain
```

Supported flags:

- `--backend auto|http|file`
- `--format text|json|ndjson`
- `--limit N`
- `--page <name>`
- `--journals-only`
- `--include-children`
- `--explain`
- `--api-url`
- `--api-token-env`

Behavior:

- `simple` accepts a compact query DSL and optionally `{{query ...}}`
- `advanced` accepts raw EDN/Datalog strings or a file path
- `auto` infers whether the input is simple or advanced and routes accordingly
- `doctor` validates backend availability and capability

## Query IR

The system will use an internal intermediate representation rather than executing each input syntax directly.

The IR is not a full Datalog AST. It is a constrained query plan that can be executed by either backend when possible.

Illustrative shape:

```go
type QueryPlan struct {
    Target      TargetKind
    Filters     []Filter
    Select      []FieldRef
    Sort        []SortSpec
    Limit       int
    Offset      int
    Include     IncludeSpec
    SourceHint  SourceHint
    RawInput    string
    InputKind   InputKind
    BackendHint BackendKind
}

type Filter struct {
    Op       FilterOp
    Field    FieldRef
    Value    Value
    Children []Filter
}
```

First-pass fields:

- `block.content`
- `block.marker`
- `block.refs`
- `block.tags`
- `block.properties.<key>`
- `block.page`
- `block.page_journal`
- `block.scheduled`
- `block.deadline`
- `block.created_at`
- `block.updated_at`
- `page.name`
- `page.original_name`
- `page.journal`
- `page.properties.<key>`

First-pass filter operators:

- `and`
- `or`
- `not`
- `eq`
- `contains`
- `in`
- `has-ref`
- `has-tag`
- `exists`
- `before`
- `after`
- `between`
- `regex`

## Input Compatibility

### Simple Query Input

The first version of the simple parser should support:

- `[[page]]`
- `#tag`
- `property:key=value`
- `marker:TODO|DOING|DONE|NOW|LATER|WAITING`
- `page:<name>`
- `journal:true|false`
- `scheduled:<date-expr>`
- `deadline:<date-expr>`
- `text:"..."`
- `and`
- `or`
- `not`

Examples:

- `[[project-x]] and marker:TODO`
- `#reading and journal:true and after:2026-03-01`
- `property:status=active and not marker:DONE`

### Advanced Query Input

Advanced queries split into two categories.

Category A: passthrough-compatible

- raw advanced query strings sent directly to Logseq HTTP API
- preferred whenever `http` backend is available

Category B: locally compilable subset

The file backend will only support advanced query forms that can be mapped to the IR, such as:

- `:find` with simple scalar or pull targets
- `:where` clauses over known block/page fields
- marker, ref, page, journal, and property filtering
- time filters
- basic `not`, `contains?`, and `re-find`

The following remain explicitly unsupported in file backend v1:

- `:rules`
- arbitrary `:in`
- custom predicates
- unrestricted joins
- arbitrary Clojure forms
- deep `pull` projections

Unsupported constructs must produce explicit errors. Silent partial execution is not acceptable.

## Backend Strategy

### HTTP Backend

Purpose:

- detect API availability
- detect `DB.q` capability
- run native Logseq advanced queries
- optionally run Logseq text search APIs

Responsibilities:

- `Ping()`
- `SupportsDBQuery()`
- `RunDBQuery(raw string)`
- `RunSearch(raw string, opts)`

The HTTP backend is the shortest path to genuine Logseq Query support and should be implemented before the local file backend.

### File Backend

Purpose:

- parse local Logseq files
- construct an in-memory index
- execute the restricted query subset without Logseq running

The file backend becomes the durable core of `lsq`, but should not claim full parity with Logseq DB behavior.

## File Graph Model

The file backend needs a real block/page model.

Illustrative model:

```go
type Page struct {
    Name       string
    Original   string
    FilePath   string
    Journal    bool
    JournalDay *time.Time
    Properties map[string]any
    BlockIDs   []string
}

type Block struct {
    ID         string
    UUID       string
    Content    string
    Marker     string
    Tags       []string
    Refs       []string
    Properties map[string]any
    PageName   string
    Journal    bool
    Scheduled  *time.Time
    Deadline   *time.Time
    CreatedAt  *time.Time
    UpdatedAt  *time.Time
    Parent     string
    Children   []string
    Order      int
}
```

Minimum indices:

- `pagesByName`
- `blocksByID`
- `blocksByPage`
- `blocksByRef`
- `blocksByTag`
- `blocksByMarker`
- `blocksByPropertyKey`
- `journalPagesByDay`

## Package Layout

Recommended package additions:

- `/Users/tr/Workspace/logseq/logseq-clis/lsq/cmd/query.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/types.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/plan.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/result.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/router.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/parser/simple.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/parser/advanced.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/compile/simple_to_ir.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/compile/advanced_to_ir.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/backend/httpapi/client.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/backend/httpapi/execute.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/backend/file/model.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/backend/file/parser_markdown.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/backend/file/parser_org.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/backend/file/index.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/backend/file/execute.go`
- `/Users/tr/Workspace/logseq/logseq-clis/lsq/query/backend/file/dates.go`

## Routing Rules

Execution rules:

1. parse input
2. classify as `simple` or `advanced`
3. choose backend
4. execute or fail with a precise unsupported message

Recommended behavior:

- `backend=http`
  - `advanced` queries go straight to `DB.q` or `datascriptQuery`
  - `simple` queries may be executed locally or translated later

- `backend=file`
  - `simple` queries execute locally
  - `advanced` queries must first compile to IR
  - unsupported advanced features fail explicitly

- `backend=auto`
  - use HTTP when healthy and the query benefits from native DB execution
  - otherwise use file backend
  - never silently drop unsupported semantics

Warnings should be visible in `--format text` and structured in `json`/`ndjson`.

## Output Contract

Provide stable machine-readable results.

Illustrative JSON shape:

```json
{
  "backend": "http",
  "input_kind": "advanced",
  "plan": {},
  "results": [],
  "warnings": [],
  "unsupported": []
}
```

Formats:

- `text`
- `json`
- `ndjson`

This makes the query system reusable by shell workflows, MCP wrappers, and other agents.

## Testing Strategy

### CLI Routing

- `lsq query doctor`
- `lsq query advanced --backend http --query ...`
- `lsq query simple --backend file --expr ...`
- `lsq query auto --expr '{{query ...}}'`

### Simple Parser

- `[[foo]]`
- `#foo and marker:TODO`
- `property:status=active`
- `not journal:true`
- invalid syntax returns actionable errors

### Advanced Parser and Classification

- detect `[:find ... :where ...]`
- detect `{{query ...}}`
- distinguish passthrough-only vs compilable subset
- reject `:rules`, complex `:in`, and arbitrary functions in file backend

### HTTP Backend

- health check
- `DB.q` success
- fallback to `datascriptQuery`
- auth failure
- timeout
- response shape differences

### File Parsing and Indexing

- page property extraction
- block property extraction
- task marker extraction
- `[[ref]]` extraction
- `#tag` extraction
- journal page detection
- nested block ordering

### File Execution

- `marker:TODO`
- `[[project-x]]`
- `#reading and journal:true`
- `property:status=active`
- sort and limit
- empty results
- unsupported operators return clear errors

### End-to-End Fixtures

Use temporary test graphs built in integration tests and reuse the repo's existing integration test infrastructure where possible.

## Risks

1. HTTP API result shapes may vary by Logseq version.
2. File graphs cannot guarantee parity with DB graphs.
3. Markdown and Org parsing can become brittle if v1 overreaches.
4. Future Logseq DB graph evolution may widen the gap between HTTP-native query support and file-native query support.

## Delivery Order

Recommended implementation order:

1. `query doctor`
2. HTTP backend and advanced query passthrough
3. query IR and result contract
4. simple query parser
5. file graph parser and index
6. file backend execution for simple query
7. advanced subset compilation to IR
8. auto backend routing and explain mode

This sequence delivers real query value early while keeping the local engine bounded and testable.
