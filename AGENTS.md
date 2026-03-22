# AGENTS

## Scope

This file governs `./` and its descendants unless a deeper `AGENTS.md` overrides part of it.

## Role

`lsq` is a small Go CLI for Logseq-oriented workflows. Its current stable surface is file-oriented: journals, pages, append flows, and simple search behavior.

## Inherits

- `../../AGENTS.md`
- `../AGENTS.md`

## Source Of Truth

Current query work is governed by:

- Spec: `./docs/superpowers/specs/2026-03-22-logseq-query-design.md`
- Plan: `./docs/superpowers/plans/2026-03-22-logseq-query-phase-1.md`

These documents are authoritative for current query expansion work. `AGENTS.md` should not restate their implementation detail.

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

- Feature-level decisions belong in `docs/superpowers/specs/` and `docs/superpowers/plans/`.
- Repo-wide usage belongs in `README.md`.
- This file only defines working constraints for contributors and agents.

## Deeper AGENTS

Do not add deeper `AGENTS.md` files inside this repo unless a subdirectory becomes a stable subsystem with rules that are meaningfully independent from the repo root.

Likely future candidates, only if justified:

- `query/`
- `docs/superpowers/`

## Overrides

None.
