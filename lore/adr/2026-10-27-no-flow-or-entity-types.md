---
id: 2026-10-27-no-flow-or-entity-types
status: accepted
zones: [docs]
tldr: "lore has only two kinds of knowledge docs — zones and ADRs; there is no flow type (a scenario's route lives in product.md, its rules in zones) and no entity type (an entity is a zone or a section of one)."
rules:
  - "DO: put a scenario's route in product.md and its rules in the zones it passes through"
  - "DO: make an entity a zone or a section of a zone"
  - "DON'T: add flow or entity document types"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# No flow or entity document types

## Context
Scenarios cross several zones, and some areas are centred on an entity (`users`, `orders`). Both could get their own document type.

## Decision
No `flow` type: the rules of a scenario live in the zones, the logic "between steps" almost always has an owning zone, and the route of the scenario is a line in `product.md` with `[live]` / `[planned]`. No `entity` type: an entity is either a zone or a section inside one. An action lives in the zone of the object acted upon, a reaction in the zone that reacts.

## Why
A flow has no rules of its own — it is a projection of the zones, and it goes stale; keeping scenarios in `product.md` also keeps the product layer apart from the technical one. Without the actor rule, a zone built around an actor (`users`) becomes a god-zone that swallows half the app.

## Rejected alternatives
### A flow document per scenario
Why not: a projection without rules of its own that goes stale.

### An entity document type
Why not: an entity is already a zone or a section; actor-centred docs turn into god-zones.
