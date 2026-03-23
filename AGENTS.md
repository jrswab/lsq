# AGENTS

## Scope

This file governs `./` and its descendants unless a deeper `AGENTS.md` overrides part of it.

## Role

`lsq` is a small Go CLI for Logseq-oriented workflows. Its current stable surface is file-oriented: journals, pages, append flows, and simple search behavior.

## Inherits

None.

## Source Of Truth

For upstream-facing work in this repository:

- shipped behavior is defined by the code, tests, `README.md`, and `CHANGELOG.md`
- internal planning docs may exist out of tree and should not be referenced here unless they are committed into this repository

`AGENTS.md` should not restate implementation detail that already lives in code, tests, or user-facing docs.

## Rules

- Keep `lsq` small and explicit. Do not silently turn it into a generic platform.
- Protect existing behavior. Query work must not casually regress current journal/page open, append, print, filename search, alias search, or regex search flows.
- Follow current phase boundaries:
  - Phase 1 is HTTP-only query support
  - file backend work is later-phase work unless the spec/plan changes
- Any change to query scope should update the spec and plan before code diverges.
- Prefer narrow files with clear boundaries:
  - command entrypoint
  - transport/client
  - execution
  - result rendering
  - tests
- Do not claim local parity with Logseq DB behavior unless that claim is explicitly justified in the owning spec.

## Documentation

- Repo-wide usage belongs in `README.md`.
- Changelog entries belong in `CHANGELOG.md`.
- This file only defines working constraints for contributors and agents.

## Deeper AGENTS

Do not add deeper `AGENTS.md` files inside this repo unless a subdirectory becomes a stable subsystem with rules that are meaningfully independent from the repo root.

Likely future candidates, only if justified:

- `query/`

## Overrides

None.
