---
id: 2026-10-27-agent-writes-owner-approves
status: accepted
zones: [global]
tldr: "The agent writes every lore document; the owner approves at the gates — the proposal with its ADR drafts, each ADR, and the diff of lore/ at archive."
rules:
  - "DO: write zones, ADRs, deltas and product scenarios yourself"
  - "DO: get the owner's explicit yes before lore adr accept and before lore change archive --finish"
  - "DON'T: accept an ADR or archive a change on your own"
  - "DON'T: expect the owner to write docs by hand"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# The agent writes, the owner approves

## Context
Someone has to write and maintain the zones, ADRs and changes. Either the owner writes them, or the agent does.

## Decision
The agent writes everything. The owner approves at fixed points: the proposal with its ADR drafts, every ADR before it is accepted, and the diff of `lore/` before a change is archived (and each step of a bootstrap).

## Why
If the owner has to write the docs by hand, they will abandon it within a month. Without the owner's approval, the agent starts "legalising" its own mistakes by writing them down as ADRs.

## Rejected alternatives
### The owner writes the docs
Why not: they would stop doing it within a month.

### The agent writes without approval
Why not: the agent's own mistakes become recorded decisions.
