# lsq Logseq Query Design

Date: 2026-03-22
Repo: `./`
Status: Proposed, revised after review

## Goal

Extend `lsq` with query support in a way that is deliverable for the current codebase.

The immediate objective is not to build a full local Logseq query engine. The immediate objective is:

- let `lsq` execute real Logseq advanced queries through the local HTTP API when available
- add a small, explicit diagnostic surface so the user can tell whether query execution is possible
- preserve all existing `lsq` behaviors for journal/page opening, appending, and file-oriented search

## Current Reality

Today `lsq` is a small Go CLI centered on file operations.

It currently supports:

- opening or creating journal files
- opening a page by filename
- appending bullet content to journals and pages
- printing page or journal content
- filename and `alias::` prefix search
- regex scanning across raw files

Relevant code:

- `./main.go`
- `./system/journal.go`
- `./trie/trie.go`

It does not currently have:

- a block AST
- a page/block index
- a Datalog engine
- a local model for refs, tags, page properties, or query planning

That matters because a realistic v1 must fit this starting point.

## Product Direction

The design is split into phases.

### Phase 1

Deliver real advanced query support by delegating to Logseq's HTTP API.

Phase 1 includes:

- `lsq query doctor`
- `lsq query advanced --query ...`
- `lsq query advanced --file ...`
- `--backend http|auto`
- structured output for success, warnings, and backend selection

Phase 1 does not include a local query engine.

### Phase 2

Deliver remote simple-query support delegating directly to `logseq.DB.q`.
*Rationale: Phase 2 was actively pulled forward because `DB.q` for simple DSL execution is now firmly validated to deliver user-facing value immediately through the HTTP backend, avoiding the complexity of local parsing frameworks upfront.*

Phase 2 includes:
- `lsq query simple --expr '...'`
- raw simple DSL (`(task now)`, `(and [[logseq]] #TypeScript)`, `(page-property type project)`)
- HTTP execution strictly mapped to Logseq

### Phase 3

Add intermediate follow-up support to recognize and safely strip outer `{{query ...}}` wrappers surrounding pure simple DSL logic before submitting to `logseq.DB.q`.
*Rationale: Providing this macro stripping enables immediate user-visible value by allowing Logseq UI macro configurations to be pasted natively into the terminal without manually stripping the text syntax, while strictly enforcing boundaries against advanced EDN formats.*

Phase 3 includes:
- recognition of `{{query ...}}` wrapped macro strings for simple queries
- targeted wrapper stripping safely bypassing CLI layer for valid simple DSL only
- strict validation avoiding arbitrary `{:query ...}` shapes

### Phase 4

Add a tiny local file-backed simple-query DSL that works natively offline over fields that can be recovered reliably from filesystem files.
*Rationale: Phase 4 remains highly valuable because it provides offline/local capability and lays the crucial foundational parser/index work moving forward.*

Phase 4 includes:
- a small `simple` Markdown parser
- a local file index targeting `--graph-dir`
- local execution for a restricted set of simple predicates

Add optional compilation of a limited advanced-query subset into the local plan model.

This ordering is intentional. It avoids turning `lsq` into a half-finished query platform before the repo has the right primitives.

## Non-goals

The first implementation will not:

- fully reimplement Logseq's DataScript/Datalog engine
- guarantee parity between HTTP-backed results and file-backed results
- support every advanced query construct such as `:rules`, arbitrary `:in`, custom predicates, or unrestricted joins
- silently emulate unsupported advanced queries locally
- replace existing `-f` and `-r` behavior

## CLI Design

Add a query subcommand family rather than pushing more behavior into the existing flat flag interface.

Phase 1 commands:

```bash
lsq query doctor
lsq query advanced --query '[:find (pull ?b [*]) :where [?b :block/marker "TODO"]]'
lsq query advanced --file ./query.edn
lsq query advanced --query '[:find ?name :where [?p :block/name ?name]]' --backend http --format json
```

Phase 2 commands:

```bash
lsq query simple --expr '[[logseq]]' --backend http
lsq query simple --expr '(task now)' --backend http --format json
```

Phase 3 commands (Macro Stripping):

```bash
lsq query simple --expr '{{query (task now)}}'
lsq query simple --expr '{{query (and [[logseq]] #TypeScript)}}'
```

Phase 4 commands:

```bash
lsq query simple --expr 'ref:project-x and marker:TODO' --backend file --graph-dir ~/graph
lsq query simple --expr 'tag:reading and marker:DONE' --backend file --graph-dir ~/graph
```

Supported flags in Phase 1:

- `--backend auto|http`
- `--format text|json|ndjson`
- `--explain`
- `--api-url`
- `--api-token-env`

Supported flags added in Phase 2:

- `--expr`

Supported flags added in Phase 4:

- `--backend file`
- `--graph-dir`
- `--limit N`

## Input Model

The original draft mixed two incompatible meanings of "simple query". That is removed here.

Phase 1 supports one input kind only:
- raw advanced query text, passed through to `logseq.DB.datascriptQuery` via the Logseq HTTP API

Phase 2 supports raw simple DSL expressions passed through to `logseq.DB.q`:
- `[[logseq]]`
- `(task now)`
- `(between [[jan 1st, 2026]] [[mar 31st, 2026]])`

Phase 3 supports simple DSL wrapped inside macros by stripping the macros safely:
- `{{query [[logseq]]}}` -> `[[logseq]]`
- `{{query (task now)}}` -> `(task now)`
- Invalid outputs fail quickly (e.g. `{{query {:query [:find ...]}}}`) via local error handling.

Phase 4 introduces a separate local offline simple DSL parsing local filesystem targets.

The Phase 4 local simple DSL is intentionally not the same thing as `{{query ...}}` nor full Logseq query macros. It is an `lsq` offline DSL.

Phase 4 local offline DSL:
- `ref:<page-name>`
- `tag:<tag-name>`
- `marker:TODO|DOING|DONE`
- `text:"..."`
- `and`
- `or`
- `not`

Examples:

- `ref:project-x and marker:TODO`
- `tag:reading and not marker:DONE`
- `text:"distributed systems" and marker:DOING`

Not supported in the local DSL for the first local phase:

- `{{query ...}}`
- `scheduled:...`
- `deadline:...`
- date arithmetic
- `NOW`, `LATER`, `WAITING`
- page-level property filtering

The restriction to `TODO|DOING|DONE` is deliberate and matches the repo's existing TODO model in `./todo/todo.go`.

## Backend Strategy

### HTTP Backend

Phase 1 depends on the local Logseq HTTP API as the authoritative execution backend for advanced queries.

Responsibilities:

- detect whether the API is reachable
- detect whether database query methods are available
- execute `logseq.DB.datascriptQuery` for advanced Datalog queries
- return machine-readable results and explicit failure modes

The Phase 1 backend contract should be written against the observed request shape used by third-party integrations in this workspace:

- endpoint: `POST /api`
- request body:

```json
{
  "method": "logseq.DB.datascriptQuery",
  "args": ["[:find ?name :where [?p :block/name ?name]]"]
}
```

- auth: bearer token when configured

Advanced queries should be routed exclusively to `logseq.DB.datascriptQuery`, with `logseq.DB.q` reserved for simple DSL inputs.

### File Backend

The file backend is explicitly out of scope for Phase 1.

When it is introduced in Phase 4, it should only execute a restricted local DSL over fields that can be recovered reliably from files.

The file backend must not assume local availability of:

- `block.created_at`
- `block.updated_at`
- block UUIDs for every block
- full DB-style joinability

Those assumptions were removed from this revision because the current repo does not have a stable source for them.

## Query Plan Model

An internal plan model is still useful, but only for Phase 4 and beyond.

The plan model should remain intentionally narrow:

```go
type QueryPlan struct {
    Target  TargetKind
    Filters []Filter
    Limit   int
    RawInput string
    InputKind InputKind
}

type Filter struct {
    Op       FilterOp
    Field    FieldRef
    Value    string
    Children []Filter
}
```

Phase 4 local fields:

- `block.content`
- `block.marker`
- `block.refs`
- `block.tags`

Phase 4 local operators:

- `and`
- `or`
- `not`
- `eq`
- `contains`

No time fields are part of the local plan model until the codebase has a reliable source for them.

## Phase 1 Detailed Scope

Phase 1 is considered complete when all of the following exist:

1. `lsq query doctor`
Reports:
- whether HTTP API is reachable
- whether auth succeeded
- whether `logseq.DB.q` works
- whether `logseq.DB.datascriptQuery` works

Illustrative JSON response:

```json
{
  "backend": "http",
  "command": "doctor",
  "api_url": "http://127.0.0.1:12315/api",
  "reachable": true,
  "auth": {
    "configured": true,
    "succeeded": true
  },
  "capabilities": {
    "db_q": true,
    "datascript_query": true
  },
  "warnings": [],
  "error": null
}
```

2. `lsq query advanced`
Accepts either raw query text or a file path and executes it remotely.

3. `--backend auto`
Uses HTTP if healthy. If not healthy, it returns a clear error.

4. structured output
Every command can return `text`, `json`, or `ndjson`.

Illustrative JSON response:

```json
{
  "backend": "http",
  "input_kind": "advanced",
  "query_method": "logseq.DB.datascriptQuery",
  "results": [],
  "warnings": [],
  "error": null
}
```

Phase 1 explicitly does not attempt file fallback for advanced queries.

## Phase 2 Detailed Scope

Phase 2 relies entirely on the HTTP HTTP bindings established in Phase 1. It explicitly passes simple DSL syntax exclusively mapped to Logseq's `DB.q` API endpoint natively passing payloads upstream.

Explicit exclusions:
- `{{query ...}}` macro formatting natively unstripped. Stripping should be manually omitted from Phase 2.
- Advanced EDN/DataLog syntaxes.
- Offline File backend parses.
- Org files.

## Phase 3 Detailed Scope

Phase 3 intercepts remote `query simple` commands to examine input strings for `{{query \s*(.*?)\s*}}`. If found:
1. Strips the wrapper.
2. Evaluates if the inner payload resembles pure simple query DSL strings `(`, `[` and strictly avoids property or EDN-like tokens (`{`, `:inputs`, `:query`, etc).
3. If valid, forwards to Phase 2 HTTP execution `logseq.DB.q`.
4. If invalid, the CLI fails closed avoiding incorrect remote execution parameters.

Phase 3 avoids supporting:
- `#+BEGIN_QUERY ... #+END_QUERY`
- advanced query wrappers

## Phase 4 Detailed Scope

Phase 4 begins following HTTP simple validation completions in Phase 2.

Phase 4 builds a local file-backed offline query path for a tiny DSL only.

Local parsing requirements:
- parse block tree shape well enough to preserve parent-child structure
- extract `TODO`, `DOING`, `DONE` markers
- extract `[[page refs]]`
- extract `#tags`
- preserve source page and file path

Local block model should stay minimal:

```go
type Block struct {
    ID       string
    Content  string
    Marker   string
    Tags     []string
    Refs     []string
    PageName string
    Parent   string
    Children []string
    Order    int
}
```

This is intentionally smaller than the original draft and matches what can be recovered from local files with reasonable confidence.

## Package Layout

Phase 1 package additions:

- `./cmd/query.go`
- `./query/result.go`
- `./query/router.go`
- `./query/backend/httpapi/client.go`
- `./query/backend/httpapi/execute.go`

Phase 4 additions:

- `./query/types.go`
- `./query/parser/simple.go`
- `./query/compile/simple_to_plan.go`
- `./query/backend/file/model.go`
- `./query/backend/file/parser_markdown.go`
- `./query/backend/file/index.go`
- `./query/backend/file/execute.go`

Org support is not Phase 4 by default. It should be a later follow-up unless real user demand appears.

## Testing Strategy

### Phase 1 and 2

HTTP backend tests should use `httptest` and fixed fixtures.

Required coverage:

- API reachable and query succeeds
- auth failure
- timeout
- `logseq.DB.q` unavailable but `datascriptQuery` available
- malformed response
- `doctor` output in `text` and `json`

CLI routing coverage:

- `lsq query doctor`
- `lsq query advanced --query ...`
- `lsq query advanced --file ...`
- `lsq query simple --expr ...`
- `--backend auto`

### Phase 4

Simple parser tests:

- `ref:project-x and marker:TODO`
- `tag:reading and not marker:DONE`
- malformed DSL returns actionable errors

File parser tests:

- marker extraction
- ref extraction
- tag extraction
- nested blocks preserve hierarchy

File execution tests:

- ref filter
- tag filter
- marker filter
- logical combinations
- empty results

The repo's existing integration helper under `./tests/integration/integration.go` can help with temporary directories and env setup, but it is not by itself a complete integration framework. Phase 1 will need explicit CLI and HTTP test helpers.

## Risks

1. Logseq HTTP API response shapes may vary by version.
2. DB query methods may not behave identically across Logseq releases.
3. The current repo has no CLI subcommand infrastructure yet, so even Phase 1 requires careful refactoring of `main.go`.
4. A local query engine can easily overreach if it grows beyond the restricted DSL defined here.

## Delivery Order

Recommended order:

1. add `query doctor`
2. add HTTP client and advanced-query passthrough
3. add structured output
4. add `auto` backend for HTTP-only routing
5. map remote `query simple` targeting `DB.q` over HTTP (Phase 2)
6. implement intermediate follow-up simple macro stripping (Phase 3)
7. only then start Phase 4 local offline DSL execution (Phase 4)

This revision intentionally trades breadth for implementability.

## Appendix: Logseq API Routing Boundary

Based on verified observation of the active Logseq HTTP server, `lsq` must enforce a rigid boundary between simple and advanced execution entrypoints:

- **`logseq.DB.q`**: Operates on simple DSL shapes only (e.g., `[[logseq]]`, `(task now)`, `(between ...)`). It returns arrays of IDs/blocks when given simple DSL. It outputs `null` (or fails) on advanced EDN maps and most `{{query ...}}` forms.
- **`logseq.DB.datascriptQuery`**: Operates purely on DataScript/Datalog representations. It expects raw advanced arrays / EDN formats.
- **Macro Stripping (`{{query (...)}}`)**: Stripping exterior query braces and piping the inner token block is valid only if the innards definitively consist of **simple DSL** routed into `DB.q` (e.g. `{{query (task now)}}` → `(task now)`). Do not strip macros hoping `DB.q` or `datascriptQuery` can interpret arbitrary internal render properties—they belong exclusively to the frontend UI renderer, not the core DB interface.
