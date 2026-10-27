---
id: 2026-10-27-zones-not-tags
status: accepted
zones: [adrs]
tldr: "An ADR is attached to zones through zones: [...] with existing zone ids or global, not through free-form tags."
rules:
  - "DO: use existing zone ids or global in zones of an ADR"
  - "DON'T: add free-form tags or keywords to ADRs for search"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# ADRs use zones, not free tags

## Context
ADRs have to be found for a task and shown next to the zones they constrain.

## Decision
Every ADR lists `zones`: existing zone ids, or `global` for project-wide decisions. Lint checks them. Finding ADRs by meaning is done by the decisions model.

## Why
Semantic search is the decisions model's job, and free-form tags multiply without control.

## Rejected alternatives
### Free-form tags
Why not: they multiply, and the model already does the semantic matching.
