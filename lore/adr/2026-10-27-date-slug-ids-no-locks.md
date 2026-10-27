---
id: 2026-10-27-date-slug-ids-no-locks
status: accepted
zones: [adrs, changes]
tldr: "ADR ids are <date>-<slug>, not sequential numbers, and parallel changes are never locked — two changes touching the same requirement only get a lint warning."
rules:
  - "DO: create ADRs with lore adr new (it builds the id)"
  - "DON'T: switch ADR ids to sequential numbers"
  - "DON'T: add locks between active changes"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# Date-slug ADR ids and no locks between changes

## Context
Several branches and changes may be in flight at once, each creating ADRs and editing zones.

## Decision
An ADR id is the creation date plus a slug. Active changes are not locked against each other; when two of them touch the same requirement, lint warns and the second merge is done carefully on top of the first.

## Why
Sequential numbers conflict between branches. Locks cost more than the rare overlaps they would prevent.

## Rejected alternatives
### Sequential ADR numbers
Why not: they conflict between branches.

### Locking a requirement while a change edits it
Why not: costs more than rare overlaps.
