---
id: 2026-10-27-one-visible-lore-folder
status: accepted
zones: [global]
tldr: "All lore docs live in one visible folder named lore/ (configurable as root), not in docs/ and not in a hidden folder; per-machine state lives inside it in lore/.env and lore/.local/, both gitignored."
rules:
  - "DO: keep lore docs in the root folder from lore.yaml (lore/ by default)"
  - "DO: keep local state in lore/.local/ and the key in lore/.env"
  - "DON'T: move lore docs into docs/ or a hidden folder"
  - "DON'T: create a second top-level folder for lore state"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# One visible lore/ folder

## Context
lore needs a place for its docs in projects that may already have a `docs/` folder, and a place for per-machine state (the key, the call log, sessions, bootstrap progress) that must not be committed. Earlier versions kept that state in a separate top-level `.lore/`.

## Decision
One visible folder, `lore/` by default (`root` in `lore.yaml`). Local, uncommitted things live inside it: `lore/.env` and `lore/.local/`, both added to `.gitignore`. `lore upgrade` migrates an old `.lore/` into it.

## Why
`docs/` is often taken by other documents; a hidden folder gets skipped by people and agents. Two folders (`lore/` and `.lore/`) gave nothing but confusion.

## Rejected alternatives
### docs/
Why not: often holds other documents.

### A hidden .lore/
Why not: people and agents skip hidden folders.

### Docs in lore/, state in a separate .lore/
Why not: two folders gave nothing but confusion.
