---
id: 2026-10-27-owner-decides-contradictions
status: accepted
zones: [adrs, workflows]
tldr: "A contradiction with an accepted ADR is always decided by the owner — new decision or bug; a new decision supersedes the old ADR entirely, there is no partial replacement."
rules:
  - "DO: stop and ask — new decision (supersede) or a bug / keep the old one"
  - "DO: restate every rule of the old ADR that still holds in the superseding one"
  - "DON'T: decide yourself, work around the ADR, or 'fix' it silently"
  - "DON'T: supersede only part of an ADR"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# The owner decides contradictions with ADRs

## Context
Facts of a change, the code or a new idea may contradict an accepted ADR. The fact check at the gate looks for this, and an agent hits it during work.

## Decision
Every contradiction with an accepted ADR is a question to the owner: a new decision (a new ADR that supersedes the old one) or a mistake (keep the decision, change the plan or fix the code). A superseding ADR replaces the old one entirely and restates every rule that still holds.

## Why
A new decision and a bug in the code look the same from the outside — only the owner can tell them apart. Partial replacement forces readers to assemble a decision from pieces of several ADRs.

## Rejected alternatives
### Partial supersede
Why not: the current decision would have to be assembled from pieces.
