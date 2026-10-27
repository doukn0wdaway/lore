---
id: 2026-10-27-status-only-via-accept
status: accepted
zones: [adrs, changes]
tldr: "An ADR's status changes only through lore adr accept, never by hand; a draft inside a change supersedes old ADRs only when the change is archived."
rules:
  - "DO: lore adr accept <id> after the owner's yes"
  - "DON'T: edit status or superseded_by by hand"
  - "DON'T: mark ADRs superseded before the change with the replacing draft is archived"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# ADR status changes only through lore adr accept

## Context
ADR statuses form chains: `supersedes` on the new ADR, `status: superseded` and `superseded_by` on the old one. A change may carry ADR drafts that replace accepted ADRs, and a change can be abandoned.

## Decision
`lore adr accept` is the only way a status changes; it also marks the replaced ADRs. For a draft inside a change, accept only marks the draft; the old ADRs are superseded by `lore change archive --finish`.

## Why
Without a command, the agent changes statuses "however it turns out", and editing statuses by hand breaks the supersede chains. An abandoned change must not get to cancel a decision that is still in force.
