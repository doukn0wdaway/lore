---
id: 2026-10-27-product-has-no-dropped
status: accepted
zones: [docs, workflows]
tldr: "product.md describes the present: there is no [dropped] status — a changed scenario is rewritten in place, an abandoned one is deleted, and a non-obvious reason for dropping it becomes an ADR."
rules:
  - "DO: rewrite a changed scenario in place; delete one that is no longer wanted"
  - "DO: record a non-obvious reason for dropping a scenario as an ADR"
  - "DON'T: add [dropped], [deprecated] or similar statuses"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# product.md has no [dropped] status

## Context
Scenarios in `product.md` change or get abandoned. Their history could be kept in the file with a status, or elsewhere.

## Decision
Only `[live]` and `[planned]`. A changed scenario is rewritten, an abandoned one deleted. If the reason it was dropped is non-obvious and an agent might propose it again, the reason becomes an ADR ("we don't do X because …").

## Why
`product.md` describes the present; history lives in archived changes (and git).
